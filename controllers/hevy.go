package controllers

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/aunefyren/treningheten/database"
	"github.com/aunefyren/treningheten/files"
	"github.com/aunefyren/treningheten/logger"
	"github.com/aunefyren/treningheten/middlewares"
	"github.com/aunefyren/treningheten/models"
	"github.com/aunefyren/treningheten/utilities"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Hevy uses a single per-user API key (UUID) sent as the 'api-key' header. The key
// is only available to Hevy PRO subscribers. There is no OAuth/refresh flow, so we
// simply store the key (encrypted at rest) and validate it against /user/info.
const hevyAPIBaseURL = "https://api.hevyapp.com/v1"

// hevyStravaOverlapWindow is how close a Strava activity and a Hevy workout must start to
// be treated as the same session for de-duplication.
const hevyStravaOverlapWindow = 15 * time.Minute

// hevyLocation returns the app's configured timezone, falling back to UTC.
func hevyLocation() *time.Location {
	if loc, err := time.LoadLocation(files.ConfigFile.Timezone); err == nil && loc != nil {
		return loc
	}
	return time.UTC
}

// hevyAPIGet performs an authenticated GET against the Hevy API and returns the body.
// path must include a leading slash and any query string (e.g. "/workouts?page=1").
func hevyAPIGet(apiKey string, path string) (body []byte, err error) {
	if strings.TrimSpace(apiKey) == "" {
		return nil, errors.New("empty Hevy API key")
	}

	req, err := http.NewRequest("GET", hevyAPIBaseURL+path, nil)
	if err != nil {
		return nil, errors.New("failed to build Hevy request: " + err.Error())
	}
	req.Header.Set("api-key", apiKey)
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, errors.New("failed to reach Hevy: " + err.Error())
	}
	defer resp.Body.Close()

	body, err = io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.New("failed to read Hevy response: " + err.Error())
	}

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return nil, errors.New("the Hevy API key was rejected")
	} else if resp.StatusCode != http.StatusOK {
		return nil, errors.New("unexpected response from Hevy: " + resp.Status)
	}

	return body, nil
}

// hevyValidateAPIKey checks an API key by calling GET /user/info. A successful call
// confirms the key is valid (and that the account exists).
func hevyValidateAPIKey(apiKey string) (userInfo models.HevyUserInfo, err error) {
	body, err := hevyAPIGet(apiKey, "/user/info")
	if err != nil {
		return userInfo, err
	}

	var response models.HevyUserInfoResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return userInfo, errors.New("failed to parse Hevy response: " + err.Error())
	}

	return response.Data, nil
}

// hevyFetchExerciseTemplates pages through GET /exercise_templates (pageSize max 100)
// and returns a template_id -> template map. This is the lookup used to classify each
// exercise in a workout (the workout payload only carries the template id + title, not
// the is_custom flag or muscle group).
func hevyFetchExerciseTemplates(apiKey string) (map[string]models.HevyExerciseTemplate, error) {
	templates := map[string]models.HevyExerciseTemplate{}

	page := 1
	for {
		body, err := hevyAPIGet(apiKey, fmt.Sprintf("/exercise_templates?page=%d&pageSize=100", page))
		if err != nil {
			return nil, err
		}

		var response models.HevyExerciseTemplatesResponse
		if err := json.Unmarshal(body, &response); err != nil {
			return nil, errors.New("failed to parse Hevy exercise templates: " + err.Error())
		}

		for _, template := range response.ExerciseTemplates {
			templates[template.ID] = template
		}

		if page >= response.PageCount || response.PageCount == 0 {
			break
		}
		page++
	}

	return templates, nil
}

// hevyTypeToActionType maps a Hevy exercise-template type onto Treningheten's
// Action/Operation type vocabulary (lifting/timing/moving). Hevy is strength-centric,
// so unknown/most types default to "lifting".
func hevyTypeToActionType(hevyType string) string {
	return models.HevyActionType(hevyType)
}

// getOrCreateHevyAction resolves the global Action for a Hevy exercise template.
// Custom exercises (is_custom=true) are private to a single user, so they never create
// a global Action — the caller represents those on the Operation itself. Official-catalog
// templates are looked up by HevyTemplateID and auto-created on miss.
func getOrCreateHevyAction(template models.HevyExerciseTemplate) (*models.Action, error) {
	if template.IsCustom || strings.TrimSpace(template.ID) == "" {
		return nil, nil
	}

	action, err := database.GetActionByHevyTemplateID(template.ID)
	if err != nil {
		return nil, err
	}
	if action != nil {
		return action, nil
	}

	// Auto-create a global Action keyed by the (stable, unique) Hevy template id.
	// Shares models.HevyExerciseTemplate.ToAction with the catalog seeder so a template
	// imported on-demand is identical to one seeded at startup.
	newAction := template.ToAction()
	newAction.ID = uuid.New()

	created, err := database.CreateActionInDB(newAction)
	if err != nil {
		return nil, err
	}
	return &created, nil
}

// encryptHevyAPIKey encrypts the API key at rest using the configured Hevy token key.
func encryptHevyAPIKey(apiKey string) (string, error) {
	if apiKey == "" {
		return "", errors.New("empty Hevy API key")
	}
	return utilities.EncryptString(apiKey, files.ConfigFile.HevyTokenKey)
}

// decryptHevyAPIKey returns the plaintext API key from its stored ciphertext.
func decryptHevyAPIKey(stored string) (string, error) {
	if stored == "" {
		return "", errors.New("no Hevy API key stored")
	}
	return utilities.DecryptString(stored, files.ConfigFile.HevyTokenKey)
}

// hevyDisableExerciseChildren soft-disables every operation (and its sets) under an
// exercise. Used to rebuild a workout's contents on re-import — Hevy exercises/sets have
// no stable ids, so the whole tree is recreated rather than diffed.
func hevyDisableExerciseChildren(exerciseID uuid.UUID) error {
	operations, err := database.GetOperationsByExerciseID(exerciseID)
	if err != nil {
		return err
	}

	for _, operation := range operations {
		sets, err := database.GetOperationSetsByOperationID(operation.ID)
		if err != nil {
			return err
		}
		for _, set := range sets {
			set.Enabled = false
			if _, err := database.UpdateOperationSetInDB(set); err != nil {
				return err
			}
		}

		operation.Enabled = false
		if _, err := database.UpdateOperationInDB(operation); err != nil {
			return err
		}
	}

	return nil
}

// HevySyncWorkoutForUser imports one Hevy workout into the exercise tree (workout →
// Exercise, each Hevy exercise → Operation, each set → OperationSet), idempotent by the
// workout id stored on the exercise. templates is the template_id -> template map used to
// classify each exercise; pass the result of hevyFetchExerciseTemplates.
func HevySyncWorkoutForUser(user models.User, workout models.HevyWorkout, templates map[string]models.HevyExerciseTemplate) error {
	now := time.Now()
	workoutID := workout.ID

	exercise, err := database.GetExerciseForUserWithHevyWorkoutID(user.ID, workoutID)
	if err != nil {
		return errors.New("failed to look up exercise for Hevy workout: " + err.Error())
	}

	// Hevy reports start_time in UTC with no local-time field, so resolve the calendar
	// day in the app's configured timezone before bucketing (otherwise late-night
	// workouts land on the wrong day).
	localStart := workout.StartTime.In(hevyLocation())

	isNewExercise := exercise == nil
	if exercise == nil {
		// Find or create the exercise day for the workout's local date.
		exerciseDay, err := database.GetExerciseDayByDateAndUserID(user.ID, localStart)
		if err != nil {
			return errors.New("failed to get exercise day: " + err.Error())
		} else if exerciseDay == nil {
			exerciseDay = &models.ExerciseDay{}
			exerciseDay.ID = uuid.New()
			exerciseDay.Enabled = true
			exerciseDay.CreatedAt = now
			exerciseDay.UpdatedAt = now
			exerciseDay.UserID = &user.ID
			// Stamp the day at midnight UTC using the local calendar date so it is
			// deterministic and matches the date-string lookups.
			exerciseDay.Date = time.Date(localStart.Year(), localStart.Month(), localStart.Day(), 0, 0, 0, 0, time.UTC)

			if err := database.CreateExerciseDayInDB(*exerciseDay); err != nil {
				return errors.New("failed to create exercise day: " + err.Error())
			}
		}

		exercise = &models.Exercise{}
		exercise.ID = uuid.New()
		exercise.CreatedAt = now
		exercise.UpdatedAt = now
		exercise.ExerciseDayID = exerciseDay.ID
		exercise.HevyWorkoutID = &workoutID
	} else {
		// Existing workout: rebuild its contents from scratch.
		if err := hevyDisableExerciseChildren(exercise.ID); err != nil {
			return errors.New("failed to clear existing operations: " + err.Error())
		}
	}

	exercise.Enabled = true
	exercise.IsOn = true
	// Default a fresh import to counting (Save writes the struct's zero value rather than the
	// DB default, so set it explicitly); the per-type opt-out below may flip it to false once
	// the workout's activity types are known. Re-syncs keep whatever the user set.
	if isNewExercise {
		exercise.CountsTowardGoal = true
	}
	startTime := workout.StartTime
	exercise.Time = &startTime

	// Session note from the workout title; duration from start/end span (stored as a
	// raw seconds count, per the duration convention).
	exercise.Note = strings.TrimSpace(workout.Title)
	if workout.EndTime.After(workout.StartTime) {
		duration := int64(workout.EndTime.Sub(workout.StartTime).Seconds())
		exercise.Duration = &duration
	}

	finalExercise, err := database.UpdateExerciseInDB(*exercise)
	if err != nil {
		return errors.New("failed to save exercise: " + err.Error())
	}

	// Hevy wins: if enabled, supersede any already-imported Strava activity that
	// overlaps this workout's start by disabling it (and its operations/sets).
	if user.StravaSkipHevyDuplicates != nil && *user.StravaSkipHevyDuplicates {
		stravaExercise, overlapErr := database.GetStravaExerciseForUserNearTime(user.ID, workout.StartTime, hevyStravaOverlapWindow)
		if overlapErr != nil {
			logger.Log.Warn("Failed to check Strava overlap for Hevy workout " + workoutID + ". Error: " + overlapErr.Error())
		} else if stravaExercise != nil {
			if err := hevyDisableExerciseChildren(stravaExercise.ID); err != nil {
				logger.Log.Warn("Failed to disable Strava operations superseded by Hevy. Error: " + err.Error())
			} else {
				stravaExercise.Enabled = false
				if _, err := database.UpdateExerciseInDB(*stravaExercise); err != nil {
					logger.Log.Warn("Failed to disable Strava exercise superseded by Hevy. Error: " + err.Error())
				}
			}
		}
	}

	actionIDs := []uuid.UUID{}
	for _, hevyExercise := range workout.Exercises {
		template := templates[hevyExercise.ExerciseTemplateID]

		action, err := getOrCreateHevyAction(template)
		if err != nil {
			return errors.New("failed to resolve Hevy action: " + err.Error())
		}
		if action != nil {
			actionIDs = append(actionIDs, action.ID)
		}

		operation := models.Operation{}
		operation.ID = uuid.New()
		operation.CreatedAt = now
		operation.UpdatedAt = now
		operation.Enabled = true
		operation.ExerciseID = finalExercise.ID
		operation.WeightUnit = "kg"
		operation.DistanceUnit = "km"

		notes := strings.TrimSpace(hevyExercise.Notes)
		if action != nil {
			// Official catalog: the Action carries the exercise name; only attach the
			// user's per-exercise note when present.
			operation.ActionID = &action.ID
			operation.Type = action.Type
			if notes != "" {
				operation.Note = &notes
			}
		} else {
			// Custom (or unknown) exercise: no global Action, so preserve the title on
			// the operation itself (the frontend uses note as the title fallback).
			operation.Type = hevyTypeToActionType(template.Type)
			title := strings.TrimSpace(hevyExercise.Title)
			if title == "" {
				title = "Custom exercise"
			}
			operation.Note = &title
			if notes != "" {
				operation.Description = &notes
			}
		}

		createdOperation, err := database.CreateOperationInDB(operation)
		if err != nil {
			return errors.New("failed to create operation: " + err.Error())
		}

		for _, set := range hevyExercise.Sets {
			operationSet := models.OperationSet{}
			operationSet.ID = uuid.New()
			operationSet.CreatedAt = now
			operationSet.UpdatedAt = now
			operationSet.Enabled = true
			operationSet.OperationID = createdOperation.ID
			operationSet.Repetitions = set.Reps
			operationSet.Weight = set.WeightKg

			if set.DistanceMeters != nil {
				km := *set.DistanceMeters / 1000
				operationSet.Distance = &km
			}
			if set.DurationSeconds != nil {
				// Stored as a raw seconds count, per the duration convention.
				duration := int64(*set.DurationSeconds)
				operationSet.Time = &duration
			}

			if _, err := database.CreateOperationSetInDB(operationSet); err != nil {
				return errors.New("failed to create operation set: " + err.Error())
			}
		}
	}

	// Snapshot goal-counting from the user's per-activity-type settings, but only for a fresh
	// import so a manual builder toggle survives later re-syncs. Now that every action in the
	// workout is known, a Hevy session counts unless every one of its types is flagged off.
	if isNewExercise {
		offActions, err := loadOffCountActions(user.ID)
		if err != nil {
			logger.Log.Warn("Failed to load activity goal settings for Hevy import. Error: " + err.Error())
		} else if counts := countsTowardGoalForActions(actionIDs, offActions); !counts {
			finalExercise.CountsTowardGoal = counts
			if _, err := database.UpdateExerciseInDB(finalExercise); err != nil {
				logger.Log.Warn("Failed to persist Hevy session goal-counting flag. Error: " + err.Error())
			}
		}
	}

	return nil
}

// HevyBackfillForUser imports the user's full Hevy workout history. It fetches the
// exercise-template catalog once, then pages through GET /workouts (pageSize max 10),
// importing each workout idempotently. Errors on individual workouts are logged and
// skipped so one bad workout doesn't abort the whole backfill.
func HevyBackfillForUser(user models.User) error {
	if user.HevyAPIKey == nil || *user.HevyAPIKey == "" {
		return errors.New("user has no Hevy API key")
	}

	apiKey, err := decryptHevyAPIKey(*user.HevyAPIKey)
	if err != nil {
		return errors.New("failed to decrypt Hevy API key: " + err.Error())
	}

	// Baseline for incremental sync: any workout changed during the backfill is after
	// this instant, so the next /events poll re-fetches it. Recorded only on success.
	syncStart := time.Now()

	templates, err := hevyFetchExerciseTemplates(apiKey)
	if err != nil {
		return errors.New("failed to fetch Hevy exercise templates: " + err.Error())
	}

	// Give user the "Influencer" achievement for connecting Hevy, ignore outcome
	go GiveUserAnAchievement(user.ID, uuid.MustParse("fb4f6c1f-dfad-4df7-8007-4cfd6f351b17"), time.Now(), 5)

	page := 1
	for {
		body, err := hevyAPIGet(apiKey, fmt.Sprintf("/workouts?page=%d&pageSize=10", page))
		if err != nil {
			return err
		}

		var response models.HevyWorkoutsResponse
		if err := json.Unmarshal(body, &response); err != nil {
			return errors.New("failed to parse Hevy workouts: " + err.Error())
		}

		for _, workout := range response.Workouts {
			if err := HevySyncWorkoutForUser(user, workout, templates); err != nil {
				logger.Log.Warn("Failed to import Hevy workout " + workout.ID + ". Error: " + err.Error())
			}
		}

		if page >= response.PageCount || response.PageCount == 0 {
			break
		}
		page++
	}

	// Record the baseline so the hourly /events sync takes over from here.
	user.HevyLastSync = &syncStart
	if _, err := database.UpdateUser(user); err != nil {
		return errors.New("failed to record Hevy sync baseline: " + err.Error())
	}

	return nil
}

// hevyDeleteWorkoutForUser soft-disables the exercise (and its operations/sets) that a
// now-deleted Hevy workout was imported into. A no-op if the workout was never imported.
func hevyDeleteWorkoutForUser(user models.User, workoutID string) error {
	exercise, err := database.GetExerciseForUserWithHevyWorkoutID(user.ID, workoutID)
	if err != nil {
		return errors.New("failed to look up exercise for deleted Hevy workout: " + err.Error())
	} else if exercise == nil {
		return nil
	}

	if err := hevyDisableExerciseChildren(exercise.ID); err != nil {
		return errors.New("failed to disable operations for deleted Hevy workout: " + err.Error())
	}

	exercise.Enabled = false
	if _, err := database.UpdateExerciseInDB(*exercise); err != nil {
		return errors.New("failed to disable exercise for deleted Hevy workout: " + err.Error())
	}

	return nil
}

// HevyEventsSyncForUser applies workout changes since the user's last sync via
// GET /workouts/events: "updated" events upsert the workout, "deleted" events disable it.
// It only runs once a backfill has recorded a baseline (HevyLastSync), so it never races
// the initial import. On success it advances the baseline.
func HevyEventsSyncForUser(user models.User) error {
	if user.HevyAPIKey == nil || *user.HevyAPIKey == "" {
		return nil
	}
	if user.HevyLastSync == nil {
		// Backfill hasn't completed yet; let it own the import.
		return nil
	}

	apiKey, err := decryptHevyAPIKey(*user.HevyAPIKey)
	if err != nil {
		return errors.New("failed to decrypt Hevy API key: " + err.Error())
	}

	syncStart := time.Now()
	since := user.HevyLastSync.UTC().Format(time.RFC3339)

	templates, err := hevyFetchExerciseTemplates(apiKey)
	if err != nil {
		return errors.New("failed to fetch Hevy exercise templates: " + err.Error())
	}

	// Give user the "Influencer" achievement for connecting Hevy, ignore outcome
	go GiveUserAnAchievement(user.ID, uuid.MustParse("fb4f6c1f-dfad-4df7-8007-4cfd6f351b17"), time.Now(), 5)

	page := 1
	for {
		body, err := hevyAPIGet(apiKey, fmt.Sprintf("/workouts/events?since=%s&page=%d&pageSize=10", url.QueryEscape(since), page))
		if err != nil {
			return err
		}

		var response models.HevyWorkoutEventsResponse
		if err := json.Unmarshal(body, &response); err != nil {
			return errors.New("failed to parse Hevy workout events: " + err.Error())
		}

		for _, event := range response.Events {
			switch event.Type {
			case "updated":
				if event.Workout == nil {
					continue
				}
				if err := HevySyncWorkoutForUser(user, *event.Workout, templates); err != nil {
					logger.Log.Warn("Failed to apply updated Hevy workout " + event.Workout.ID + ". Error: " + err.Error())
				} else if exercise, lookupErr := database.GetExerciseForUserWithHevyWorkoutID(user.ID, event.Workout.ID); lookupErr == nil && exercise != nil {
					// A freshly synced Hevy workout carries a real start time, so kick off the
					// soundtrack overlay immediately (like Strava). Bulk backfill skips this
					// and leans on the reconcile cron so it doesn't fan out a goroutine per
					// historical workout.
					TriggerMediaSyncForExercise(user, exercise.ID)
				}
			case "deleted":
				if err := hevyDeleteWorkoutForUser(user, event.ID); err != nil {
					logger.Log.Warn("Failed to apply deleted Hevy workout " + event.ID + ". Error: " + err.Error())
				}
			}
		}

		if page >= response.PageCount || response.PageCount == 0 {
			break
		}
		page++
	}

	// Advance the baseline only after the whole run succeeds.
	user.HevyLastSync = &syncStart
	if _, err := database.UpdateUser(user); err != nil {
		return errors.New("failed to advance Hevy sync baseline: " + err.Error())
	}

	return nil
}

// HevyEventsSyncForAllUsers is the hourly cron entrypoint: it runs the incremental events
// sync for every user with a stored Hevy API key.
func HevyEventsSyncForAllUsers() {
	users, err := database.GetHevyUsers()
	if err != nil {
		logger.Log.Error("Failed to get Hevy users. Error: " + err.Error())
		return
	}

	logger.Log.Debug("Hevy sync: got '" + strconv.Itoa(len(users)) + "' users.")

	for _, user := range users {
		if err := HevyEventsSyncForUser(user); err != nil {
			logger.Log.Error("Hevy events sync for user returned error. Error: " + err.Error())
		}
	}

	logger.Log.Info("Hevy sync task finished.")
}

// APISetHevyAPIKey validates a user-supplied Hevy API key and stores it encrypted.
func APISetHevyAPIKey(context *gin.Context) {
	var request models.UserHevyAPIKeyUpdateRequest

	if err := context.ShouldBindJSON(&request); err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse request."})
		context.Abort()
		return
	}

	if !files.ConfigFile.HevyEnabled {
		context.JSON(http.StatusBadRequest, gin.H{"error": "Hevy is not enabled."})
		context.Abort()
		return
	}

	apiKey := strings.TrimSpace(request.HevyAPIKey)
	if apiKey == "" {
		context.JSON(http.StatusBadRequest, gin.H{"error": "A Hevy API key is required."})
		context.Abort()
		return
	}

	userID, err := middlewares.GetAuthUsername(context.GetHeader("Authorization"))
	if err != nil {
		logger.Log.Info("Failed to get user ID. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user ID."})
		context.Abort()
		return
	}

	user, err := database.GetAllUserInformation(userID)
	if err != nil {
		logger.Log.Info("Failed to get user object. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user object."})
		context.Abort()
		return
	}

	// Validate the key against Hevy before storing it
	userInfo, err := hevyValidateAPIKey(apiKey)
	if err != nil {
		logger.Log.Info("Failed to validate Hevy API key. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		context.Abort()
		return
	}

	// Store the public profile URL so it can be shown (gated by HevyPublic) on the
	// user's profile, mirroring the Strava athlete link.
	if strings.TrimSpace(userInfo.URL) != "" {
		profileURL := strings.TrimSpace(userInfo.URL)
		user.HevyProfileURL = &profileURL
	}

	encrypted, err := encryptHevyAPIKey(apiKey)
	if err != nil {
		logger.Log.Info("Failed to encrypt Hevy API key. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to store Hevy API key."})
		context.Abort()
		return
	}

	user.HevyAPIKey = &encrypted

	// On first connect, default to skipping Strava activities that duplicate Hevy
	// workouts (Hevy carries the richer set/rep data). A later explicit choice is kept.
	if user.StravaSkipHevyDuplicates == nil {
		skip := true
		user.StravaSkipHevyDuplicates = &skip
	}

	// Default to showing the Hevy link on the profile (mirrors Strava). The gorm default
	// only applies on row insert, so set it explicitly for users connecting later.
	if user.HevyPublic == nil {
		public := true
		user.HevyPublic = &public
	}

	if _, err := database.UpdateUser(user); err != nil {
		logger.Log.Info("Failed to update user object. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user object."})
		context.Abort()
		return
	}

	// Backfill the user's workout history in the background so the connect response stays
	// fast; the full history can span many pages (pageSize max 10).
	go func(u models.User) {
		if err := HevyBackfillForUser(u); err != nil {
			logger.Log.Warn("Hevy backfill failed for user " + u.ID.String() + ". Error: " + err.Error())
		}
	}(user)

	context.JSON(http.StatusOK, gin.H{"message": "Hevy connected! Your workouts are importing in the background."})
}

// APIDeleteHevyAPIKey disconnects Hevy by clearing the stored API key.
func APIDeleteHevyAPIKey(context *gin.Context) {
	userID, err := middlewares.GetAuthUsername(context.GetHeader("Authorization"))
	if err != nil {
		logger.Log.Info("Failed to get user ID. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user ID."})
		context.Abort()
		return
	}

	user, err := database.GetAllUserInformation(userID)
	if err != nil {
		logger.Log.Info("Failed to get user object. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user object."})
		context.Abort()
		return
	}

	user.HevyAPIKey = nil
	user.HevyProfileURL = nil
	if _, err := database.UpdateUser(user); err != nil {
		logger.Log.Info("Failed to update user object. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user object."})
		context.Abort()
		return
	}

	context.JSON(http.StatusOK, gin.H{"message": "Hevy disconnected."})
}

// APISyncHevyForUser runs an on-demand Hevy sync for the authenticated user only (the
// user is resolved from the token, not the URL param). If the initial backfill never
// recorded a baseline it runs a full backfill, otherwise an incremental events sync.
func APISyncHevyForUser(context *gin.Context) {
	if !files.ConfigFile.HevyEnabled {
		context.JSON(http.StatusBadRequest, gin.H{"error": "Hevy is not enabled."})
		context.Abort()
		return
	}

	userID, err := middlewares.GetAuthUsername(context.GetHeader("Authorization"))
	if err != nil {
		logger.Log.Info("Failed to get user ID. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user ID."})
		context.Abort()
		return
	}

	user, err := database.GetAllUserInformation(userID)
	if err != nil {
		logger.Log.Info("Failed to get user object. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user object."})
		context.Abort()
		return
	}

	if user.HevyAPIKey == nil || *user.HevyAPIKey == "" {
		context.JSON(http.StatusBadRequest, gin.H{"error": "User does not have a Hevy connection."})
		context.Abort()
		return
	}

	go func(u models.User) {
		var syncErr error
		if u.HevyLastSync == nil {
			syncErr = HevyBackfillForUser(u)
		} else {
			syncErr = HevyEventsSyncForUser(u)
		}
		if syncErr != nil {
			logger.Log.Warn("Manual Hevy sync failed for user " + u.ID.String() + ". Error: " + syncErr.Error())
		}
	}(user)

	context.JSON(http.StatusOK, gin.H{"message": "Hevy sync started!"})
}
