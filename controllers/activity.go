package controllers

import (
	"errors"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/aunefyren/treningheten/database"
	"github.com/aunefyren/treningheten/logger"
	"github.com/aunefyren/treningheten/middlewares"
	"github.com/aunefyren/treningheten/models"
	"github.com/aunefyren/treningheten/utilities"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// activityFeedSorts is the whitelist of sort keys the feed accepts. Anything else is a 400,
// so the DB layer never sees an unmapped column.
var activityFeedSorts = map[string]bool{
	"date":     true,
	"distance": true,
	"duration": true,
	"weight":   true,
	"reps":     true,
}

// parseActivityFeedTime accepts either a full RFC3339 timestamp or a bare YYYY-MM-DD date
// (treated as that day at midnight), which is what the date-range inputs on the page send.
func parseActivityFeedTime(value string) (time.Time, error) {
	if parsed, err := time.Parse(time.RFC3339, value); err == nil {
		return parsed, nil
	}
	return time.Parse("2006-01-02", value)
}

// parseActivityFeedFilter reads and validates the feed query parameters into an
// ActivityFeedFilter. A returned error carries a client-facing message and always maps to a
// 400. Limit is clamped to [1, 100] and an out-of-range offset falls back to 0, so the DB
// layer never sees a nonsensical page window.
func parseActivityFeedFilter(context *gin.Context) (models.ActivityFeedFilter, error) {
	filter := models.ActivityFeedFilter{
		Sort:   "date",
		Order:  "desc",
		Limit:  30,
		Offset: 0,
	}

	if value := strings.TrimSpace(context.Query("action_id")); value != "" {
		actionID, err := uuid.Parse(value)
		if err != nil {
			return filter, errors.New("Invalid action id.")
		}
		filter.ActionID = &actionID
	}

	if value := strings.TrimSpace(context.Query("start")); value != "" {
		parsed, err := parseActivityFeedTime(value)
		if err != nil {
			return filter, errors.New("Invalid start date.")
		}
		filter.Start = &parsed
	}

	if value := strings.TrimSpace(context.Query("end")); value != "" {
		parsed, err := parseActivityFeedTime(value)
		if err != nil {
			return filter, errors.New("Invalid end date.")
		}
		filter.End = &parsed
	}

	filter.Query = strings.TrimSpace(context.Query("q"))

	if strings.EqualFold(strings.TrimSpace(context.Query("has_distance")), "true") {
		filter.HasDistance = true
	}

	if value := strings.ToLower(strings.TrimSpace(context.Query("sort"))); value != "" {
		if !activityFeedSorts[value] {
			return filter, errors.New("Invalid sort.")
		}
		filter.Sort = value
	}

	if value := strings.ToLower(strings.TrimSpace(context.Query("order"))); value != "" {
		if value != "asc" && value != "desc" {
			return filter, errors.New("Invalid order.")
		}
		filter.Order = value
	}

	if value := strings.TrimSpace(context.Query("limit")); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil {
			filter.Limit = parsed
		}
	}
	if filter.Limit < 1 {
		filter.Limit = 1
	}
	if filter.Limit > 100 {
		filter.Limit = 100
	}

	if value := strings.TrimSpace(context.Query("offset")); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil && parsed >= 0 {
			filter.Offset = parsed
		}
	}

	return filter, nil
}

// APIGetActivityFeed powers the /exercises timeline: a filtered, sorted, paginated list of
// activities (operations) with per-activity metrics aggregated from their sets. See
// database.GetActivityFeedForUser and docs/exercises.md.
func APIGetActivityFeed(context *gin.Context) {
	userID, err := middlewares.GetAuthUsername(context.GetHeader("Authorization"))
	if err != nil {
		logger.Log.Info("Failed to verify user ID. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		context.Abort()
		return
	}

	filter, err := parseActivityFeedFilter(context)
	if err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		context.Abort()
		return
	}

	items, total, err := database.GetActivityFeedForUser(userID, filter)
	if err != nil {
		logger.Log.Info("Failed to get activity feed. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get activities."})
		context.Abort()
		return
	}

	if items == nil {
		items = []models.ActivityFeedItem{}
	}
	hasMore := int64(filter.Offset+len(items)) < total

	context.JSON(http.StatusOK, gin.H{
		"message":    "Activities retrieved.",
		"activities": items,
		"total":      total,
		"has_more":   hasMore,
	})
}

// sharedActivityFeedLimit caps the front-page feed. It is a glanceable weekly digest, not a
// browsable archive — /exercises is the searchable feed for that. Without a cap the module
// grows with every season a user has ever played.
const sharedActivityFeedLimit = 50

// buildActivitiesFromExerciseDays flattens exercise days into the front-page activity shape,
// newest first. Only enabled, switched-on, non-private sessions become activities. Strava
// links are included per the session owner's StravaPublic setting, not the viewer's; a
// session whose operations resolve no Action falls back to the general "Workout" action so
// an activity is never actionless.
//
// Private sessions are dropped unconditionally, including for the owner's own view of their
// own feed: this builder backs three endpoints and every one of them is a social surface, so
// one rule beats a viewer-aware branch. Your private sessions stay in the /exercises timeline
// and the builder.
func buildActivitiesFromExerciseDays(exerciseDays []models.ExerciseDay) ([]models.Activity, error) {
	exerciseDayObjects, err := ConvertExerciseDaysToExerciseDayObjects(exerciseDays)
	if err != nil {
		return nil, errors.New("Failed to convert exercise day to exercise day objects.")
	}

	generalAction, err := database.GetActionByStravaName("Workout")
	if err != nil {
		return nil, errors.New("Failed to get general action object.")
	} else if generalAction == nil {
		return nil, errors.New("Failed to find general action object.")
	}

	activities := []models.Activity{}
	for _, exerciseDayObject := range exerciseDayObjects {
		for _, exercise := range exerciseDayObject.Exercises {
			if !exercise.IsOn || !exercise.Enabled || exercise.Private {
				continue
			}

			activity := models.Activity{}
			activity.ExerciseID = exercise.ID
			activity.User = exerciseDayObject.User
			activity.Time = exercise.Time
			activity.Actions = []models.Action{}

			if exerciseDayObject.User.StravaPublic != nil && *exerciseDayObject.User.StravaPublic {
				activity.StravaIDs = exercise.StravaID
			} else {
				activity.StravaIDs = []string{}
			}

			activity.HevyWorkoutID = exercise.HevyWorkoutID

			if len(exercise.Operations) > 0 {
				for _, operation := range exercise.Operations {
					if operation.Action != nil {
						activity.Actions = append(activity.Actions, *operation.Action)
					} else {
						activity.Actions = append(activity.Actions, *generalAction)
					}
				}
			} else {
				activity.Actions = append(activity.Actions, *generalAction)
			}

			activities = append(activities, activity)
		}
	}

	sort.Slice(activities, func(i, j int) bool {
		return activities[j].Time.Before(activities[i].Time)
	})

	return activities, nil
}

// APIGetSharedActivities returns this week's activities from everyone the authenticated user
// has ever shared a season with (plus their own). Co-membership of a season is the app's
// implicit social relation — there is no friend-request system — and the season need not be
// ongoing, so the feed keeps working between seasons. Appearing here still requires the
// session owner's `share_activities` opt-in.
func APIGetSharedActivities(context *gin.Context) {
	userID, err := middlewares.GetAuthUsername(context.GetHeader("Authorization"))
	if err != nil {
		logger.Log.Info("Failed to get user ID. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to get user ID."})
		context.Abort()
		return
	}

	now := time.Now()
	mondayStart, err := utilities.FindEarlierMonday(now)
	if err != nil {
		logger.Log.Info("Failed to find Monday. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to find Monday."})
		context.Abort()
		return
	}

	sundayEnd, err := utilities.FindNextSunday(now)
	if err != nil {
		logger.Log.Info("Failed to find Sunday. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to find Sunday."})
		context.Abort()
		return
	}

	peerIDs, err := database.GetSeasonPeerUserIDsForUser(userID)
	if err != nil {
		logger.Log.Info("Failed to get season peers. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get season peers."})
		context.Abort()
		return
	}

	exerciseDays, err := database.GetExerciseDaysForSharingUsersInListUsingDates(peerIDs, mondayStart, sundayEnd)
	if err != nil {
		logger.Log.Info("Failed to get exercise days from time frame. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get exercise days from time frame."})
		context.Abort()
		return
	}

	activities, err := buildActivitiesFromExerciseDays(exerciseDays)
	if err != nil {
		logger.Log.Info("Failed to build activities. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		context.Abort()
		return
	}

	if len(activities) > sharedActivityFeedLimit {
		activities = activities[:sharedActivityFeedLimit]
	}

	context.JSON(http.StatusOK, gin.H{"message": "Activities retrieved.", "activities": activities})
}
