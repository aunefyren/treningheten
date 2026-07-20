package controllers

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync"
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

const stravaAPIBaseURL = "https://www.strava.com/api/v3"

// Strava's OAuth token endpoint is NOT under /api/v3.
const stravaOAuthTokenURL = "https://www.strava.com/oauth/token"

// ErrStravaSessionInvalid signals that Strava rejected the stored credential as
// permanently invalid (an already-used authorization code or a revoked refresh
// token), as opposed to a transient failure (rate limiting, 5xx, network). The
// caller uses this to clear the connection so the user is prompted to reconnect.
var ErrStravaSessionInvalid = errors.New("Strava session invalid.")

// Strava's default read rate limit is ~100 requests per 15-minute window (plus a
// daily cap). A flat per-minute ticker can exceed the 15-minute window under load, so
// we enforce the window directly with a sliding-window limiter, kept safely under the
// cap. stravaWait() blocks until a request slot is free.
const (
	stravaRateWindow = 15 * time.Minute
	stravaRateLimit  = 90
)

// The detailed-activity endpoint is the only source of an activity's description,
// so the hourly weekly sync only re-fetches it when missing or older than this.
const stravaDetailRefreshInterval = 7 * 24 * time.Hour

// stravaDerivedTags maps the Strava attributes Treningheten can read (commute +
// workout_type) onto tag slugs. Strava's public API exposes nothing for the other
// tags (for-a-cause, recovery, with-pet, with-kid), so those stay user-managed.
func stravaDerivedTags(activity models.StravaGetActivitiesRequestReply) []string {
	tags := []string{}
	if activity.Commute {
		tags = append(tags, models.TagCommute)
	}
	if activity.WorkoutType != nil {
		switch *activity.WorkoutType {
		case 1, 11: // race (run / ride)
			tags = append(tags, models.TagRace)
		case 2: // long run (run)
			tags = append(tags, models.TagLongRun)
		case 3, 12: // workout (run / ride)
			tags = append(tags, models.TagWorkout)
		}
	}
	return tags
}

// mergeStravaTags refreshes the Strava-managed tags from a sync while preserving
// any user-managed tags already on the operation. Strava is the source of truth
// for its subset, so a user removing e.g. "commute" sees it re-added on the next
// sync if Strava still reports it.
func mergeStravaTags(existing []string, derived []string) models.TagList {
	merged := []string{}
	// Keep still-valid user-managed tags.
	for _, tag := range existing {
		if !models.IsStravaManagedTag(tag) && models.IsValidTag(tag) {
			merged = append(merged, tag)
		}
	}
	// Add the freshly derived Strava-managed tags, avoiding duplicates.
	for _, tag := range derived {
		if !slices.Contains(merged, tag) {
			merged = append(merged, tag)
		}
	}
	return models.TagList(merged)
}

var (
	stravaRateMu    sync.Mutex
	stravaRateTimes []time.Time
)

func stravaWait() {
	for {
		stravaRateMu.Lock()
		now := time.Now()
		cutoff := now.Add(-stravaRateWindow)

		// Drop timestamps that have aged out of the window.
		kept := stravaRateTimes[:0]
		for _, t := range stravaRateTimes {
			if t.After(cutoff) {
				kept = append(kept, t)
			}
		}
		stravaRateTimes = kept

		if len(stravaRateTimes) < stravaRateLimit {
			stravaRateTimes = append(stravaRateTimes, now)
			stravaRateMu.Unlock()
			return
		}

		// Window is full; wait until the oldest request falls out of it.
		wait := stravaRateTimes[0].Add(stravaRateWindow).Sub(now)
		stravaRateMu.Unlock()
		if wait < 10*time.Millisecond {
			wait = 10 * time.Millisecond
		}
		time.Sleep(wait)
	}
}

func StravaAuthorize(code string) (authorization models.StravaAuthorizeRequestReply, err error) {
	err = nil
	authorization = models.StravaAuthorizeRequestReply{}
	url := stravaOAuthTokenURL

	var jsonStr = []byte(``)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonStr))
	if err != nil {
		logger.Log.Error("URL request generation threw error. Error: " + err.Error())
		return authorization, errors.New("URL request generation threw error.")
	}

	// Headers
	req.Header.Set("Content-Type", "application/json")

	// Params
	q := req.URL.Query()
	q.Add("client_id", files.ConfigFile.StravaClientID)
	q.Add("client_secret", files.ConfigFile.StravaClientSecret)
	q.Add("code", code)
	q.Add("grant_type", "authorization_code")
	req.URL.RawQuery = q.Encode()

	client := &http.Client{}

	resp, err := client.Do(req)
	if err != nil {
		logger.Log.Error("URL request threw error. Error: " + err.Error())
		return authorization, errors.New("URL request threw error.")
	}
	defer resp.Body.Close()

	logger.Log.Trace("Authorize gave HTTP code: " + resp.Status)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Log.Error("Failed to read reply body. Error: " + err.Error())
		return authorization, errors.New("Failed to read reply body.")
	}

	if resp.StatusCode != 200 {
		logger.Log.Error("Strava authorize returned non-200. Body: " + string(body))
		if resp.StatusCode == http.StatusBadRequest || resp.StatusCode == http.StatusUnauthorized {
			return authorization, ErrStravaSessionInvalid
		}
		return authorization, errors.New("Strava authorize returned non-200 status: " + resp.Status)
	}

	err = json.Unmarshal(body, &authorization)
	if err != nil {
		logger.Log.Error("Failed to parse reply body. Error: " + err.Error())
		return authorization, errors.New("Failed to parse reply body.")
	}

	return
}

func StravaReauthorize(code string) (authorization models.StravaReauthorizationRequestReply, err error) {
	err = nil
	authorization = models.StravaReauthorizationRequestReply{}
	url := stravaOAuthTokenURL

	var jsonStr = []byte(``)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonStr))
	if err != nil {
		logger.Log.Error("URL request generation threw error. Error: " + err.Error())
		return authorization, errors.New("URL request generation threw error.")
	}

	// Headers
	req.Header.Set("Content-Type", "application/json")

	// Params
	q := req.URL.Query()
	q.Add("client_id", files.ConfigFile.StravaClientID)
	q.Add("client_secret", files.ConfigFile.StravaClientSecret)
	q.Add("refresh_token", code)
	q.Add("grant_type", "refresh_token")
	req.URL.RawQuery = q.Encode()

	client := &http.Client{}

	resp, err := client.Do(req)
	if err != nil {
		logger.Log.Error("URL request threw error. Error: " + err.Error())
		return authorization, errors.New("URL request threw error.")
	}
	defer resp.Body.Close()

	logger.Log.Trace("Reauthorize gave HTTP code: " + resp.Status)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Log.Error("Failed to read reply body. Error: " + err.Error())
		return authorization, errors.New("Failed to read reply body.")
	}

	if resp.StatusCode != 200 {
		logger.Log.Error("Strava reauthorize returned non-200. Body: " + string(body))
		if resp.StatusCode == http.StatusBadRequest || resp.StatusCode == http.StatusUnauthorized {
			return authorization, ErrStravaSessionInvalid
		}
		return authorization, errors.New("Strava reauthorize returned non-200 status: " + resp.Status)
	}

	err = json.Unmarshal(body, &authorization)
	if err != nil {
		logger.Log.Error("Failed to parse reply body. Error: " + err.Error())
		return authorization, errors.New("Failed to parse reply body.")
	}

	return
}

func StravaGetActivities(token string, before int, after int) (activities []models.StravaGetActivitiesRequestReply, err error) {
	stravaWait() // wait until a slot is available

	err = nil
	activities = []models.StravaGetActivitiesRequestReply{}
	url := stravaAPIBaseURL + "/athlete/activities"

	var jsonStr = []byte(``)
	req, err := http.NewRequest("GET", url, bytes.NewBuffer(jsonStr))
	if err != nil {
		logger.Log.Info("URL request generation threw error. Error: " + err.Error())
		return activities, errors.New("URL request generation threw error.")
	}

	// Debug lines
	// logger.Log.Info(strconv.Itoa(before))
	// logger.Log.Info(strconv.Itoa(after))

	// Headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	// Params
	q := req.URL.Query()
	q.Add("before", strconv.Itoa(before))
	q.Add("after", strconv.Itoa(after))
	q.Add("page", "1")
	q.Add("per_page", "30")
	req.URL.RawQuery = q.Encode()

	client := &http.Client{}

	resp, err := client.Do(req)
	if err != nil {
		logger.Log.Info("URL request threw error. Error: " + err.Error())
		return activities, errors.New("URL request threw error.")
	}
	defer resp.Body.Close()

	logger.Log.Info("Get activities gave HTTP code: " + resp.Status)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Log.Info("Failed to read reply body. Error: " + err.Error())
		return activities, errors.New("Failed to read reply body.")
	}

	if resp.StatusCode != 200 {
		logger.Log.Info("Get activities returned non-200. Body: " + string(body))
		return activities, errors.New("Strava returned non-200 status: " + resp.Status)
	}

	err = json.Unmarshal(body, &activities)
	if err != nil {
		logger.Log.Info("Failed to parse reply body. Error: " + err.Error())
		return activities, errors.New("Failed to parse reply body.")
	}

	return
}

func StravaSyncWeekForAllUsers() {
	users, err := database.GetStravaUsers()
	if err != nil {
		logger.Log.Error("Failed to get Strava users.")
		return
	}

	logger.Log.Debug("Got '" + strconv.Itoa(len(users)) + "' users.")

	for _, user := range users {
		err = StravaSyncWeekForUser(user, time.Now())
		if err != nil {
			logger.Log.Error("Sync Strava for user returned error. Error: " + err.Error())
		}
	}

	logger.Log.Info("Strava sync task finished.")
}

func StravaSyncWeekForUser(user models.User, pointInTime time.Time) (err error) {
	err = nil
	logger.Log.Debug("Strava sync for user '" + user.FirstName + " " + user.LastName + "'.")

	monday, err := utilities.FindEarlierMonday(pointInTime)
	if err != nil {
		logger.Log.Error("Failed to find monday. ID: " + user.ID.String())
		return errors.New("Failed to find monday.")
	}

	sunday, err := utilities.FindNextSunday(pointInTime)
	if err != nil {
		logger.Log.Error("Failed to find sunday. ID: " + user.ID.String())
		return errors.New("Failed to find sunday.")
	}

	token, err := StravaGetAuthorizationForUser(user)
	if err != nil {
		logger.Log.Error("failed to find authorize user toward Strava. error: " + err.Error())
		return errors.New("failed to find authorize user toward Strava")
	}

	activities, err := StravaGetActivities(token, int(sunday.Unix()), int(monday.Unix()))
	if err != nil {
		logger.Log.Error("Failed to get activities. ID: " + user.ID.String())
		return errors.New("Failed to get activities.")
	}

	logger.Log.Debug("Got '" + strconv.Itoa((len(activities))) + "' activities for user.")

	// Give user achievement for connecting Strava, ignore outcome
	go GiveUserAnAchievement(user.ID, uuid.MustParse("fb4f6c1f-dfad-4df7-8007-4cfd6f351b17"), time.Now(), 5)

	for _, activity := range activities {
		err = StravaSyncActivityForUser(activity, user, token, false)
		if err != nil {
			logger.Log.Errorf("failed to sync activity '%d' for user '%s %s'. error: %s", activity.ID, user.FirstName, user.LastName, err.Error())
		}
	}

	go OllamaAsyncRefreshCacheForUser(user.ID)

	return
}

func StravaGetAuthorizationForUser(user models.User) (token string, err error) {
	token = ""
	err = nil

	if user.StravaCode == nil {
		logger.Log.Error("no Strava code")
		return token, errors.New("no Strava code")
	}

	// The code field is "<prefix>:<value>"; split only on the first colon so the
	// value (an encrypted, base64 token) is preserved intact.
	stravaCodeData := strings.SplitN(*user.StravaCode, ":", 2)
	if len(stravaCodeData) != 2 {
		logger.Log.Error("Invalid Strava code format for user. ID: " + user.ID.String())
		return token, errors.New("Invalid Strava code format for user.")
	}

	switch strings.ToLower(stravaCodeData[0]) {
	case "c":
		// Fresh one-time authorization code from the OAuth callback.
		authorization, err := StravaAuthorize(stravaCodeData[1])
		if err != nil {
			// A transient error (rate limiting, 5xx, network) must not brick the
			// connection, so it is left intact. An explicitly invalid session
			// (already-used code / revoked access) is cleared so the user is
			// prompted to reconnect on the account page.
			if errors.Is(err, ErrStravaSessionInvalid) {
				clearStravaConnection(user.ID)
				return token, ErrStravaSessionInvalid
			}
			logger.Log.Error("Failed to authorize user. ID: " + user.ID.String() + ". Error: " + err.Error())
			return token, errors.New("Failed to authorize user.")
		}
		if authorization.AccessToken == "" || authorization.RefreshToken == "" {
			return token, errors.New("Strava authorize returned empty tokens.")
		}

		// The authorization-code exchange is the only Strava response that carries the
		// athlete object, so this is where the user's Strava ID is captured. Persist it
		// in the same write as the refresh token so a connection always records the ID,
		// regardless of whether the user has any activities to sync.
		if user.StravaID == nil && authorization.Athlete.ID != 0 {
			stravaID := strconv.Itoa(authorization.Athlete.ID)
			user.StravaID = &stravaID
		}

		if err := storeStravaRefreshToken(user, authorization.RefreshToken); err != nil {
			logger.Log.Error("Failed to store Strava refresh token. ID: " + user.ID.String() + ". Error: " + err.Error())
			return token, errors.New("Failed to store Strava refresh token.")
		}

		token = authorization.AccessToken
	case "r":
		// Stored refresh token (encrypted; legacy values may still be plaintext).
		refreshToken := decryptStravaRefreshToken(stravaCodeData[1])

		authorization, err := StravaReauthorize(refreshToken)
		if err != nil {
			// Same policy as the "c" branch: clear only on an explicitly invalid
			// session (revoked / invalid refresh token), keep it on transient errors.
			if errors.Is(err, ErrStravaSessionInvalid) {
				clearStravaConnection(user.ID)
				return token, ErrStravaSessionInvalid
			}
			logger.Log.Error("Failed to re-authorize user. ID: " + user.ID.String() + ". Error: " + err.Error())
			return token, errors.New("Failed to re-authorize user.")
		}
		if authorization.AccessToken == "" || authorization.RefreshToken == "" {
			return token, errors.New("Strava reauthorize returned empty tokens.")
		}

		if err := storeStravaRefreshToken(user, authorization.RefreshToken); err != nil {
			logger.Log.Error("Failed to store Strava refresh token. ID: " + user.ID.String() + ". Error: " + err.Error())
			return token, errors.New("Failed to store Strava refresh token.")
		}

		token = authorization.AccessToken
	default:
		logger.Log.Error("Invalid Strava code format for user. ID: " + user.ID.String())
		return token, errors.New("Invalid Strava code format for user.")
	}

	return
}

// storeStravaRefreshToken encrypts the refresh token at rest and persists it on the
// user as "r:<ciphertext>". It refuses to store an empty token.
func storeStravaRefreshToken(user models.User, refreshToken string) error {
	if refreshToken == "" {
		return errors.New("empty Strava refresh token")
	}

	encrypted, err := utilities.EncryptString(refreshToken, files.ConfigFile.StravaTokenKey)
	if err != nil {
		return errors.New("failed to encrypt Strava refresh token: " + err.Error())
	}

	newCode := "r:" + encrypted
	user.StravaCode = &newCode
	if _, err := database.UpdateUser(user); err != nil {
		return errors.New("failed to update user: " + err.Error())
	}

	return nil
}

// clearStravaConnection removes the stored Strava credential and athlete id for a
// user, disconnecting them. Failures are logged but not surfaced — the caller is
// already returning an error for the failed authorization.
func clearStravaConnection(userID uuid.UUID) {
	logger.Log.Warn("Clearing invalid Strava connection for user. ID: " + userID.String())
	if err := database.ClearStravaConnectionForUser(userID); err != nil {
		logger.Log.Error("Failed to clear Strava connection. ID: " + userID.String() + ". Error: " + err.Error())
	}
}

// APIDeleteStravaConnection disconnects Strava by clearing the stored authorization
// code / refresh token and athlete id for the requesting user.
func APIDeleteStravaConnection(context *gin.Context) {
	userID, err := middlewares.GetAuthUsername(context.GetHeader("Authorization"))
	if err != nil {
		logger.Log.Info("Failed to get user ID. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user ID."})
		context.Abort()
		return
	}

	if err := database.ClearStravaConnectionForUser(userID); err != nil {
		logger.Log.Info("Failed to clear Strava connection. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to disconnect Strava."})
		context.Abort()
		return
	}

	context.JSON(http.StatusOK, gin.H{"message": "Strava disconnected."})
}

// decryptStravaRefreshToken returns the plaintext refresh token. Values stored before
// encryption was introduced are plaintext and fail to decrypt — those are returned
// as-is and get re-encrypted on the next successful exchange.
func decryptStravaRefreshToken(stored string) string {
	plaintext, err := utilities.DecryptString(stored, files.ConfigFile.StravaTokenKey)
	if err != nil {
		return stored
	}
	return plaintext
}

// stravaActivityCountsTowardGoal returns whether a newly imported session of the given Strava
// sport type should count toward the user's goal, per their per-activity-type settings. It
// fails open (counts) on any lookup error or an unresolved sport type — silently excluding a
// real workout is more surprising than a missed opt-out.
func stravaActivityCountsTowardGoal(userID uuid.UUID, sportType string) bool {
	offActions, err := loadOffCountActions(userID)
	if err != nil {
		logger.Log.Warn("Failed to load activity goal settings for Strava import. Error: " + err.Error())
		return true
	}
	action, err := database.GetActionByStravaName(sportType)
	if err != nil || action == nil {
		return true
	}
	return countsTowardGoalForActions([]uuid.UUID{action.ID}, offActions)
}

// StravaSyncActivityForUser imports one Strava activity. hasDetail tells whether
// activity already came from the detailed endpoint (so its Description is
// authoritative); when false, the activity is a list-sync summary and the detailed
// activity is fetched lazily behind the StravaDetailRetrievedAt staleness guard.
func StravaSyncActivityForUser(activity models.StravaGetActivitiesRequestReply, user models.User, token string, hasDetail bool) (err error) {
	err = nil
	now := time.Now()

	logger.Log.Tracef("strava activity action: %s", activity.SportType)

	// Walks (and every other sport type) are always imported now — whether they count toward
	// the goal is decided per-activity-type from the user's goal settings and snapshotted onto
	// the new exercise below, rather than dropping the activity and losing its streams/media.
	logger.Log.Trace("Sport type is: " + activity.SportType)

	// skip activities that duplicate an imported Hevy workout (Hevy wins), when enabled
	if user.StravaSkipHevyDuplicates != nil && *user.StravaSkipHevyDuplicates {
		hevyExercise, hevyErr := database.GetHevyExerciseForUserNearTime(user.ID, activity.StartDate, hevyStravaOverlapWindow)
		if hevyErr != nil {
			logger.Log.Warn("Failed to check Hevy overlap for Strava activity. Error: " + hevyErr.Error())
		} else if hevyExercise != nil {
			logger.Log.Trace("Skipping Strava activity because it overlaps an imported Hevy workout.")
			return nil
		}
	}

	// The user's Strava ID is captured at authorization time (see
	// StravaGetAuthorizationForUser), not here — deriving it in the activity loop
	// coupled it to having activities and risked clobbering strava_code by saving a
	// stale user struct.

	// check for data streams
	var stravaStreams *models.StravaActivityStreams
	stravaActivityStreams, err := StravaGetActivityStreams(token, strconv.FormatInt(activity.ID, 10))
	if err != nil {
		logger.Log.Warn("Failed to get activity streams. Error: " + err.Error())
	} else {
		stravaStreams = &stravaActivityStreams
	}

	// The description is only available from the detailed-activity endpoint, which
	// the list-sync payload does not include. When the caller already has detail,
	// use it directly; otherwise fetch it lazily, but only when missing or stale so
	// the hourly sync doesn't make an extra rate-limited call every run.
	detailRetrieved := hasDetail
	if !hasDetail {
		existingSet, setErr := database.GetOperationSetByStravaIDAndUserID(user.ID, int(activity.ID))
		if setErr != nil {
			logger.Log.Warn("Failed to look up operation set for detail guard. Error: " + setErr.Error())
		}
		needDetail := existingSet == nil ||
			existingSet.StravaDetailRetrievedAt == nil ||
			time.Since(*existingSet.StravaDetailRetrievedAt) > stravaDetailRefreshInterval
		if needDetail {
			detailedActivity, detailErr := StravaGetActivity(token, strconv.FormatInt(activity.ID, 10))
			if detailErr != nil {
				logger.Log.Warn("Failed to fetch detailed activity for description. Error: " + detailErr.Error())
			} else {
				activity.Description = detailedActivity.Description
				detailRetrieved = true
			}
		}
	}

	isNewExercise := false
	exercise, err := database.GetExerciseForUserWithStravaID(user.ID, strconv.Itoa(int(activity.ID)))
	if err != nil {
		logger.Log.Error("Failed to get exercise. ID: " + user.ID.String())
		return errors.New("Failed to get exercise.")
	} else if exercise == nil {
		isNewExercise = true
		// Get exercise day
		exerciseDay, err := database.GetExerciseDayByDateAndUserID(user.ID, activity.StartDateLocal)
		if err != nil {
			logger.Log.Error("Failed to get exercise day. ID: " + user.ID.String())
			return errors.New("Failed to get exercise day.")
		} else if exerciseDay == nil {
			logger.Log.Trace("Creating new exercise day.")

			exerciseDay = &models.ExerciseDay{}
			exerciseDay.ID = uuid.New()
			exerciseDay.Enabled = true
			exerciseDay.CreatedAt = now
			exerciseDay.UpdatedAt = now
			exerciseDay.UserID = &user.ID

			logger.Log.Trace("activity start: " + activity.StartDateLocal.String())

			// StartDateLocal carries the athlete's local wall-clock date. Stamp the day
			// at midnight UTC (not the server's zone) so it is deterministic and matches
			// the date-string lookups regardless of where the server runs.
			dateObject := time.Date(activity.StartDateLocal.Year(), activity.StartDateLocal.Month(), activity.StartDateLocal.Day(), 0, 0, 0, 0, time.UTC)
			exerciseDay.Date = dateObject

			logger.Log.Trace("exercise day date: " + dateObject.String())

			err = database.CreateExerciseDayInDB(*exerciseDay)
			if err != nil {
				logger.Log.Error("Failed to create exercise day. ID: " + user.ID.String())
				return errors.New("Failed to create exercise day.")
			}
		}
		logger.Log.Trace("Creating new exercise.")

		exercise = &models.Exercise{}
		exercise.ID = uuid.New()
		exercise.CreatedAt = now
		exercise.UpdatedAt = now
		exercise.ExerciseDayID = exerciseDay.ID
	}

	// Note and Duration are derived from the operations by
	// SyncStravaOperationsToExerciseSession below, so they are not set here.
	exercise.Enabled = true
	exercise.IsOn = true
	exercise.Time = &activity.StartDate

	// Snapshot whether this counts toward the goal from the user's per-activity-type settings —
	// but only for a fresh import, so a manual builder toggle survives later re-syncs.
	if isNewExercise {
		exercise.CountsTowardGoal = stravaActivityCountsTowardGoal(user.ID, activity.SportType)
	}

	logger.Log.Tracef("Strava activity start time %s for Strava ID %d", activity.StartDate, activity.ID)
	logger.Log.Tracef("Strava activity local start time %s for Strava ID %d", activity.StartDateLocal, activity.ID)

	finalExercise, err := database.UpdateExerciseInDB(*exercise)
	if err != nil {
		logger.Log.Error("Failed to get exercise. ID: " + user.ID.String())
		return errors.New("Failed to get exercise.")
	}

	logger.Log.Trace("Updated exercise.")

	operation, err := StravaSyncOperationForActivity(activity, user, finalExercise, stravaStreams, detailRetrieved, token)
	if err != nil {
		logger.Log.Error("Failed to sync operation. Error: " + err.Error())
		logger.Log.Error("Sport type was: " + activity.SportType)
	} else if operation == nil {
		logger.Log.Error("Failed to sync operation. No error.")
		logger.Log.Error("Sport type was: " + activity.SportType)
	} else {
		// A Strava activity carries a fully known time window, so this is the prime
		// trigger to overlay listening history (async; appears on next load). The
		// soundtrack attaches to the session, and each Strava activity is its own
		// session, so trigger once per exercise.
		TriggerMediaSyncForExercise(user, finalExercise.ID)
	}

	logger.Log.Trace("Synced operations.")
	return nil
}

func StravaSyncOperationForActivity(activity models.StravaGetActivitiesRequestReply, user models.User, exercise models.Exercise, streams *models.StravaActivityStreams, detailRetrieved bool, token string) (finalOperation *models.Operation, err error) {
	err = nil
	finalOperation = nil

	// Get action by Strava activity
	action, err := database.GetActionByStravaName(activity.SportType)
	if err != nil {
		return finalOperation, err
	} else if action == nil {
		// not known Strava exercise, get general action
		action, err = database.GetActionByStravaName("Workout")
		if err != nil {
			return finalOperation, err
		}
	}

	// Get or create operation
	var operation models.Operation = models.Operation{}
	oldOperation, err := database.GetOperationByStravaIDAndUserIDAndExerciseID(user.ID, int(activity.ID), exercise.ID)
	if err != nil {
		return finalOperation, err
	} else if oldOperation == nil {
		logger.Log.Info("Creating new operation.")
		operation = models.Operation{}
		operation.ID = uuid.New()
	} else {
		logger.Log.Info("Updating operation.")
		operation = *oldOperation
	}

	operation.ExerciseID = exercise.ID
	operation.ActionID = &action.ID
	operation.Type = action.Type

	// Map the activity's Strava gear onto a local gear row. Only re-link when the
	// activity actually carries a gear id, so a user-set gear on an activity with
	// no Strava gear is preserved.
	gear, err := resolveStravaGearForUser(activity.GearID, user.ID, token)
	if err != nil {
		logger.Log.Warn("Failed to resolve Strava gear, leaving operation gear unchanged. Error: " + err.Error())
	} else if gear != nil {
		operation.GearID = &gear.ID
	}

	stravaID := strconv.Itoa(int(activity.ID))
	durationTime := int64(activity.ElapsedTime)
	operation.Duration = &durationTime
	operation.Note = &activity.Name

	// Refresh the Strava-derived tags while preserving user-managed ones.
	operation.Tags = mergeStravaTags(operation.Tags, stravaDerivedTags(activity))

	// Only overwrite the description when we actually fetched detail this run; a
	// list-only sync carries no description and must not blank an existing one.
	if detailRetrieved {
		if activity.Description != nil && strings.TrimSpace(*activity.Description) != "" {
			operation.Description = activity.Description
		} else {
			operation.Description = nil
		}
	}

	newOperation, err := database.UpdateOperationInDB(operation)
	if err != nil {
		return finalOperation, err
	}

	logger.Log.Info("Updated operation.")

	finalOperation = &newOperation

	// Get or create operation set
	var operationSet models.OperationSet = models.OperationSet{}
	oldOperationSet, err := database.GetOperationSetByStravaIDAndUserIDAndOperationID(user.ID, int(activity.ID), operation.ID)
	if err != nil {
		return finalOperation, err
	} else if oldOperationSet == nil {
		logger.Log.Info("Creating new operation set.")
		operationSet = models.OperationSet{}
		operationSet.ID = uuid.New()
	} else {
		logger.Log.Info("Updating operation set.")
		operationSet = *oldOperationSet
	}

	operationSet.StravaID = &stravaID
	operationSet.OperationID = operation.ID
	movingTime := int64(activity.MovingTime)
	operationSet.MovingTime = &movingTime
	totalTime := int64(activity.ElapsedTime)
	operationSet.Time = &totalTime

	now := time.Now()
	operationSet.StravaDataRetrievedAt = &now

	// Record when detail (description) was last fetched so the guard can throttle it.
	if detailRetrieved {
		operationSet.StravaDetailRetrievedAt = &now
	}

	if streams != nil {
		operationSet.StravaStreams = &models.StravaStreamsJSON{StravaActivityStreams: *streams}
	}

	if activity.Distance != 0.0 {
		var newFloat float64
		var newDistance float64
		newFloat = activity.Distance
		newDistance = (newFloat / 1000)
		operationSet.Distance = &newDistance
	}

	_, err = database.UpdateOperationSetInDB(operationSet)
	if err != nil {
		return finalOperation, err
	}

	logger.Log.Info("Updated operation set.")

	// Sync duration of operations to exercise
	err = SyncStravaOperationsToExerciseSession(exercise.ID, user.ID)
	if err != nil {
		return finalOperation, err
	}

	return
}

func SyncStravaOperationsToExerciseSession(exerciseID uuid.UUID, userID uuid.UUID) (err error) {
	err = nil

	logger.Log.Info("Syncing Strava operations to exercise.")

	exercise, err := database.GetExerciseByIDAndUserID(exerciseID, userID)
	if err != nil {
		logger.Log.Info("Failed to get exercise object. Error: " + err.Error())
		return errors.New("Failed to get exercise object.")
	} else if exercise == nil {
		logger.Log.Info("Failed to find exercise object.")
		return errors.New("Failed to find exercise object.")
	}

	exerciseObject, err := ConvertExerciseToExerciseObject(*exercise)
	if err != nil {
		logger.Log.Info("Failed to convert exercise to exercise object. Error: " + err.Error())
		return errors.New("Failed to convert exercise to exercise object.")
	}

	var newDuration int64 = 0
	for _, operation := range exerciseObject.Operations {
		if operation.Duration != nil {
			newDuration += *operation.Duration
		}
	}

	var newNote string = ""
	for index, operation := range exerciseObject.Operations {
		if operation.Note != nil {
			if index != 0 {
				newNote += " + "
			}
			newNote += *operation.Note
		}
	}

	exercise.Duration = &newDuration
	exercise.Note = newNote

	_, err = database.UpdateExerciseInDB(*exercise)
	if err != nil {
		logger.Log.Info("Failed to update exercise. Error: " + err.Error())
		return errors.New("Failed to update exercise.")
	}

	logger.Log.Info("Updated exercise with operations.")

	return
}

func StravaGetActivityStreams(token string, activityID string) (streams models.StravaActivityStreams, err error) {
	stravaWait() // wait until a slot is available

	err = nil
	streams = models.StravaActivityStreams{}

	url := stravaAPIBaseURL + "/activities/" + activityID + "/streams"
	var jsonStr = []byte(``)

	req, err := http.NewRequest("GET", url, bytes.NewBuffer(jsonStr))
	if err != nil {
		logger.Log.Info("URL request generation threw error. Error: " + err.Error())
		return streams, errors.New("URL request generation threw error.")
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	q := req.URL.Query()
	q.Add("keys", "time,latlng,altitude,heartrate,cadence,watts,temp,velocity_smooth")
	q.Add("key_by_type", "true")

	req.URL.RawQuery = q.Encode()
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		logger.Log.Info("URL request threw error. Error: " + err.Error())
		return streams, errors.New("URL request threw error.")
	}

	defer resp.Body.Close()

	logger.Log.Info("Get activity streams gave HTTP code: " + resp.Status)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Log.Info("Failed to read reply body. Error: " + err.Error())
		return streams, errors.New("Failed to read reply body.")
	}

	if resp.StatusCode != 200 {
		logger.Log.Info("HTTP code was not 200. Body:")
		logger.Log.Info(string(body))
		return streams, errors.New("Strava returned non-200 status: " + resp.Status)
	}

	err = json.Unmarshal(body, &streams)
	if err != nil {
		logger.Log.Info("Failed to parse reply body. Error: " + err.Error())
		return streams, errors.New("Failed to parse reply body.")
	}

	return
}

func APISyncStravaActivitiesForUsers(context *gin.Context) {
	var syncRequest models.StravaSyncActivitiesForUsersRequest

	if !files.ConfigFile.StravaEnabled {
		context.JSON(http.StatusBadRequest, gin.H{"error": "Strava is not enabled"})
		context.Abort()
		return
	}

	if err := context.ShouldBindJSON(&syncRequest); err != nil {
		logger.Log.Error("failed to parse request. error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "failed to parse request"})
		context.Abort()
		return
	}

	usersToSync := []models.User{}
	if len(syncRequest.UserIDs) > 0 {
		for _, userID := range syncRequest.UserIDs {
			parsedID, err := uuid.Parse(userID)
			if err != nil {
				context.JSON(http.StatusBadRequest, gin.H{"error": "failed to parse user ID"})
				context.Abort()
				return
			}

			user, err := database.GetAllUserInformation(parsedID)
			if err != nil {
				context.JSON(http.StatusNotFound, gin.H{"error": "failed to find user by ID"})
				context.Abort()
				return
			}

			if user.Enabled && user.StravaCode != nil && *user.StravaCode != "" {
				logger.Log.Debugf("adding to sync queue %s", user.ID.String())
				usersToSync = append(usersToSync, user)
			} else {
				context.JSON(http.StatusBadRequest, gin.H{"error": "invalid user to sync " + user.ID.String()})
				context.Abort()
				return
			}
		}
	} else {
		users, err := database.GetUsersInformation()
		if err != nil {
			logger.Log.Info("Failed to get user objects. Error: " + err.Error())
			context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user objects."})
			context.Abort()
			return
		}

		for _, user := range users {
			if user.Enabled && user.StravaCode != nil && *user.StravaCode != "" {
				usersToSync = append(usersToSync, user)
			}
		}
	}

	go SyncStravaActivitiesForUsers(usersToSync, syncRequest.StravaIDs)

	context.JSON(http.StatusAccepted, gin.H{"message": "Strava sync started!"})
}

func SyncStravaActivitiesForUsers(usersToSync []models.User, stravaIDs []string) {
	for _, user := range usersToSync {
		userExercises, err := database.GetStravaExercisesByUserID(user.ID)
		if err != nil {
			logger.Log.Info("failed to get user operation sets. error: " + err.Error())
			continue
		}

		if len(userExercises) < 1 {
			continue
		}

		userExerciseObjects, err := ConvertExercisesToExerciseObjects(userExercises)
		if err != nil {
			logger.Log.Error("Failed to convert user exercises to exercise objects. error: " + err.Error())
			continue
		}

		stravaToken, err := StravaGetAuthorizationForUser(user)
		if err != nil {
			logger.Log.Info("failed to authorize user toward Strava. error: " + err.Error())
			continue
		}

		logger.Log.Debugf("syncing activities for user '%s %s'", user.FirstName, user.LastName)

		sort.Slice(userExercises, func(i, j int) bool {
			return userExercises[i].CreatedAt.After(userExercises[j].CreatedAt)
		})

		for _, userExercise := range userExerciseObjects {
			for _, operation := range userExercise.Operations {
				for _, operationSet := range operation.OperationSets {
					if len(stravaIDs) > 0 && operationSet.StravaID != nil && !slices.Contains(stravaIDs, *operationSet.StravaID) {
						continue
					}

					logger.Log.Debugf("getting activity from Strava '%s'", *operationSet.StravaID)

					if operationSet.StravaDataRetrievedAt != nil && time.Since(*operationSet.StravaDataRetrievedAt) < time.Duration(time.Hour*24*7) {
						logger.Log.Debugf("Strava activity '%s' data is too new for full refresh", *operationSet.StravaID)
						continue
					}

					stravaActivity, err := StravaGetActivity(stravaToken, *operationSet.StravaID)
					if err != nil {
						logger.Log.Info("failed to authorize user toward Strava. error: " + err.Error())
						continue
					}

					err = StravaSyncActivityForUser(stravaActivity, user, stravaToken, true)
					if err != nil {
						logger.Log.Info("failed to update Strava activity. error: " + err.Error())
						continue
					}

					logger.Log.Infof("updated Strava activity '%d' for user '%s %s'", stravaActivity.ID, user.FirstName, user.LastName)
				}
			}
		}
	}

	logger.Log.Infof("Strava activities synced for %d users", len(usersToSync))
}

// StravaGetGear fetches the detail for one piece of equipment (GET /gear/{id}),
// used to name gear the first time it is seen during sync.
func StravaGetGear(token string, gearID string) (gear models.StravaGear, err error) {
	stravaWait()
	err = nil
	gear = models.StravaGear{}
	url := stravaAPIBaseURL + "/gear/" + gearID

	var jsonStr = []byte(``)
	req, err := http.NewRequest("GET", url, bytes.NewBuffer(jsonStr))
	if err != nil {
		logger.Log.Info("URL request generation threw error. Error: " + err.Error())
		return gear, errors.New("URL request generation threw error.")
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		logger.Log.Info("URL request threw error. Error: " + err.Error())
		return gear, errors.New("URL request threw error.")
	}
	defer resp.Body.Close()

	logger.Log.Info("Get gear gave HTTP code: " + resp.Status)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Log.Info("Failed to read reply body. Error: " + err.Error())
		return gear, errors.New("Failed to read reply body.")
	}

	if resp.StatusCode != 200 {
		logger.Log.Info("HTTP code was not 200. Body:")
		logger.Log.Info(string(body))
		return gear, errors.New("Strava returned non-200 status: " + resp.Status)
	}

	err = json.Unmarshal(body, &gear)
	if err != nil {
		logger.Log.Info("Failed to parse reply body. Error: " + err.Error())
		return gear, errors.New("Failed to parse reply body.")
	}

	return
}

func StravaGetActivity(token string, activityID string) (activity models.StravaGetActivitiesRequestReply, err error) {
	stravaWait()
	err = nil
	activity = models.StravaGetActivitiesRequestReply{}
	url := stravaAPIBaseURL + "/activities/" + activityID

	var jsonStr = []byte(``)
	req, err := http.NewRequest("GET", url, bytes.NewBuffer(jsonStr))
	if err != nil {
		logger.Log.Info("URL request generation threw error. Error: " + err.Error())
		return activity, errors.New("URL request generation threw error.")
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		logger.Log.Info("URL request threw error. Error: " + err.Error())
		return activity, errors.New("URL request threw error.")
	}
	defer resp.Body.Close()

	logger.Log.Info("Get activity gave HTTP code: " + resp.Status)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Log.Info("Failed to read reply body. Error: " + err.Error())
		return activity, errors.New("Failed to read reply body.")
	}

	if resp.StatusCode != 200 {
		logger.Log.Info("HTTP code was not 200. Body:")
		logger.Log.Info(string(body))
		return activity, errors.New("Strava returned non-200 status: " + resp.Status)
	}

	err = json.Unmarshal(body, &activity)
	if err != nil {
		logger.Log.Info("Failed to parse reply body. Error: " + err.Error())
		return activity, errors.New("Failed to parse reply body.")
	}

	return
}
