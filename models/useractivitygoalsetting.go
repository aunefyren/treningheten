package models

import "github.com/google/uuid"

// UserActivityGoalSetting is a per-user, per-activity-type opt-out from goal counting. A row
// with CountsTowardGoal=false means "sessions of this activity type should not count toward my
// weekly goal by default when imported" — from Strava, Hevy or any future sync. Absence of a
// row means the type counts (the default). The flag is applied at import time and snapshotted
// onto Exercise.CountsTowardGoal; changing this setting later does not retro-apply to
// already-imported sessions (see docs/data-model.md). It is deliberately sync-agnostic —
// keyed on Action, not on the source — so one setting governs every importer. Replaces the
// old per-user StravaIgnoreWalks flag.
type UserActivityGoalSetting struct {
	GormModel
	UserID           uuid.UUID `json:"user_id" gorm:"type:varchar(100);not null;uniqueIndex:idx_user_activity_goal"`
	ActionID         uuid.UUID `json:"action_id" gorm:"type:varchar(100);not null;uniqueIndex:idx_user_activity_goal"`
	CountsTowardGoal bool      `json:"counts_toward_goal" gorm:"not null;default:true"`
}

// ActivityGoalSettingObject is the read/edit view for one activity type on the settings screen:
// an activity-type Action annotated with the user's current preference (defaulting to counts
// when no row is stored).
type ActivityGoalSettingObject struct {
	ActionID         uuid.UUID `json:"action_id"`
	ActionName       string    `json:"action_name"`
	ActionType       string    `json:"action_type"`
	ActionHasLogo    bool      `json:"action_has_logo"`
	CountsTowardGoal bool      `json:"counts_toward_goal"`
}

// ActivityGoalSettingUpdateRequest is the client payload to set one activity type's preference.
type ActivityGoalSettingUpdateRequest struct {
	ActionID         uuid.UUID `json:"action_id"`
	CountsTowardGoal bool      `json:"counts_toward_goal"`
}
