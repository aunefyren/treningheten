package models

import "github.com/google/uuid"

// UserSportMaxHeartrate is a user's explicit maximum heart rate for one sport (an activity-type
// Action such as Run or Bicycling). It anchors the HR zones of that sport's activities ahead
// of the global User.MaxHeartrate — max HR differs by sport, cycling typically sitting a few
// beats under running. Absence of a row means the sport follows the global anchors. Rows are
// replaced as a set with the rest of the training profile.
type UserSportMaxHeartrate struct {
	GormModel
	UserID       uuid.UUID `json:"user_id" gorm:"type:varchar(100);not null;uniqueIndex:idx_user_sport_max_heartrate"`
	ActionID     uuid.UUID `json:"action_id" gorm:"type:varchar(100);not null;uniqueIndex:idx_user_sport_max_heartrate"`
	MaxHeartrate int       `json:"max_heartrate" gorm:"not null"`
}

// SportMaxHeartrate is one per-sport max in a UserTrainingProfileRequest.
type SportMaxHeartrate struct {
	ActionID     uuid.UUID `json:"action_id"`
	MaxHeartrate int       `json:"max_heartrate"`
}

// SportHeartrateObject is the /account view of one sport: an activity-type Action with the
// user's explicit max for it (nil = follows the global anchors) and the highest heart rate
// seen in that sport's activities (nil = none recorded). The observed value is informational
// only — it is offered as a one-click pin, never used as a zone anchor on its own, because a
// sport with a handful of easy sessions would otherwise get inflated zones.
type SportHeartrateObject struct {
	ActionID             uuid.UUID `json:"action_id"`
	ActionName           string    `json:"action_name"`
	MaxHeartrate         *int      `json:"max_heartrate"`
	ObservedMaxHeartrate *int      `json:"observed_max_heartrate"`
}
