package database

import (
	"github.com/aunefyren/treningheten/models"

	"github.com/google/uuid"
)

// GetActivityTypeActions returns the curated set of importable "activity type" actions — the
// enabled catalog actions that carry a Strava sport name (Run, Walk, Hike, Ride, Swim, …). This
// is the manageable list surfaced on the goal-settings screen, rather than the full action
// catalog (which includes hundreds of per-exercise strength actions from Hevy).
func GetActivityTypeActions() (actions []models.Action, err error) {
	actions = []models.Action{}
	record := Instance.
		Where("`actions`.enabled = ?", 1).
		Where("`actions`.strava_name <> ?", "").
		Order("`actions`.name asc").
		Find(&actions)
	if record.Error != nil {
		return []models.Action{}, record.Error
	}
	return actions, nil
}

// GetActivityGoalSettingsForUserID returns every per-activity-type goal setting a user has
// saved. Absence of a row for an action means that action counts (the default), so callers
// only need the rows that exist.
func GetActivityGoalSettingsForUserID(userID uuid.UUID) (settings []models.UserActivityGoalSetting, err error) {
	settings = []models.UserActivityGoalSetting{}
	record := Instance.Where("`user_activity_goal_settings`.user_id = ?", userID).Find(&settings)
	if record.Error != nil {
		return []models.UserActivityGoalSetting{}, record.Error
	}
	return settings, nil
}

// GetActivityGoalSettingForUserAndAction returns the single (user, action) setting, or nil
// when none is stored (i.e. the action counts by default).
func GetActivityGoalSettingForUserAndAction(userID uuid.UUID, actionID uuid.UUID) (*models.UserActivityGoalSetting, error) {
	setting := models.UserActivityGoalSetting{}
	record := Instance.
		Where("`user_activity_goal_settings`.user_id = ?", userID).
		Where("`user_activity_goal_settings`.action_id = ?", actionID).
		First(&setting)
	if record.Error != nil {
		if record.Error.Error() == "record not found" {
			return nil, nil
		}
		return nil, record.Error
	}
	return &setting, nil
}

// UpsertActivityGoalSettingInDB creates or updates the (user, action) goal setting. The bool
// is written with an explicit column update because GORM omits a false zero value that carries
// a default:true tag on Create — a "doesn't count" row would otherwise persist as true.
func UpsertActivityGoalSettingInDB(userID uuid.UUID, actionID uuid.UUID, countsTowardGoal bool) error {
	existing, err := GetActivityGoalSettingForUserAndAction(userID, actionID)
	if err != nil {
		return err
	}
	if existing != nil {
		return Instance.Model(&models.UserActivityGoalSetting{}).
			Where("`user_activity_goal_settings`.id = ?", existing.ID).
			Update("counts_toward_goal", countsTowardGoal).Error
	}

	setting := models.UserActivityGoalSetting{UserID: userID, ActionID: actionID, CountsTowardGoal: countsTowardGoal}
	setting.ID = uuid.New()
	if err := Instance.Create(&setting).Error; err != nil {
		return err
	}
	return Instance.Model(&models.UserActivityGoalSetting{}).
		Where("`user_activity_goal_settings`.id = ?", setting.ID).
		Update("counts_toward_goal", countsTowardGoal).Error
}
