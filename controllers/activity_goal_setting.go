package controllers

import (
	"net/http"

	"github.com/aunefyren/treningheten/database"
	"github.com/aunefyren/treningheten/logger"
	"github.com/aunefyren/treningheten/middlewares"
	"github.com/aunefyren/treningheten/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// countsTowardGoalForActions decides the default goal-counting flag for a freshly imported
// session given the activity types it contains. The session counts unless every one of its
// types is flagged off in the user's per-type settings (offActions). An empty action list
// (e.g. a session whose only activities are custom exercises with no catalog action) counts —
// unknown types are never silently excluded.
func countsTowardGoalForActions(actionIDs []uuid.UUID, offActions map[uuid.UUID]bool) bool {
	if len(actionIDs) == 0 {
		return true
	}
	for _, id := range actionIDs {
		if !offActions[id] {
			return true
		}
	}
	return false
}

// loadOffCountActions returns the set of action IDs a user has flagged to not count toward
// their goal (UserActivityGoalSetting.CountsTowardGoal = false). Used at import time to choose
// the default for a new session.
func loadOffCountActions(userID uuid.UUID) (map[uuid.UUID]bool, error) {
	settings, err := database.GetActivityGoalSettingsForUserID(userID)
	if err != nil {
		return nil, err
	}
	off := map[uuid.UUID]bool{}
	for _, setting := range settings {
		if !setting.CountsTowardGoal {
			off[setting.ActionID] = true
		}
	}
	return off, nil
}

// mergeActivityGoalSettings builds the settings-screen view: each curated activity-type action
// annotated with the user's stored preference, defaulting to "counts" when no row exists.
func mergeActivityGoalSettings(actions []models.Action, settings []models.UserActivityGoalSetting) []models.ActivityGoalSettingObject {
	stored := map[uuid.UUID]bool{}
	for _, setting := range settings {
		stored[setting.ActionID] = setting.CountsTowardGoal
	}

	objects := []models.ActivityGoalSettingObject{}
	for _, action := range actions {
		countsTowardGoal := true
		if value, ok := stored[action.ID]; ok {
			countsTowardGoal = value
		}
		objects = append(objects, models.ActivityGoalSettingObject{
			ActionID:         action.ID,
			ActionName:       action.Name,
			ActionType:       action.Type,
			ActionHasLogo:    action.HasLogo,
			CountsTowardGoal: countsTowardGoal,
		})
	}
	return objects
}

// APIGetActivityGoalSettings returns the per-activity-type goal-counting preferences for the
// authenticated user — one entry per curated activity type, each with its current state.
func APIGetActivityGoalSettings(context *gin.Context) {
	userID, err := middlewares.GetAuthUsername(context.GetHeader("Authorization"))
	if err != nil {
		logger.Log.Error("Failed to verify user ID. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to verify user ID."})
		context.Abort()
		return
	}

	objects, err := buildActivityGoalSettingObjects(userID)
	if err != nil {
		logger.Log.Error("Failed to build activity goal settings. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get activity goal settings."})
		context.Abort()
		return
	}

	context.JSON(http.StatusOK, gin.H{"message": "Activity goal settings retrieved.", "activity_goal_settings": objects})
}

// buildActivityGoalSettingObjects loads the curated activity types and the user's stored
// preferences and merges them for the settings screen.
func buildActivityGoalSettingObjects(userID uuid.UUID) ([]models.ActivityGoalSettingObject, error) {
	actions, err := database.GetActivityTypeActions()
	if err != nil {
		return nil, err
	}
	settings, err := database.GetActivityGoalSettingsForUserID(userID)
	if err != nil {
		return nil, err
	}
	return mergeActivityGoalSettings(actions, settings), nil
}

// APISetActivityGoalSetting upserts one activity type's goal-counting preference for the
// authenticated user and returns the refreshed list.
func APISetActivityGoalSetting(context *gin.Context) {
	userID, err := middlewares.GetAuthUsername(context.GetHeader("Authorization"))
	if err != nil {
		logger.Log.Error("Failed to verify user ID. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to verify user ID."})
		context.Abort()
		return
	}

	var request models.ActivityGoalSettingUpdateRequest
	if err := context.ShouldBindJSON(&request); err != nil {
		logger.Log.Info("Failed to parse request. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse request."})
		context.Abort()
		return
	}

	// Validate the action exists (and is enabled) before storing a preference against it.
	if _, err := database.GetActionByID(request.ActionID); err != nil {
		logger.Log.Info("Activity goal setting referenced an unknown action. Error: " + err.Error())
		context.JSON(http.StatusNotFound, gin.H{"error": "Activity type not found."})
		context.Abort()
		return
	}

	if err := database.UpsertActivityGoalSettingInDB(userID, request.ActionID, request.CountsTowardGoal); err != nil {
		logger.Log.Error("Failed to save activity goal setting. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save activity goal setting."})
		context.Abort()
		return
	}

	objects, err := buildActivityGoalSettingObjects(userID)
	if err != nil {
		logger.Log.Error("Failed to rebuild activity goal settings. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get activity goal settings."})
		context.Abort()
		return
	}

	context.JSON(http.StatusOK, gin.H{"message": "Activity goal setting saved.", "activity_goal_settings": objects})
}
