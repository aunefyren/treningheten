package models

import (
	"time"

	"github.com/google/uuid"
)

// CarryOverUse records that a participant chose "Use saved workouts" for one week of one
// season (one goal). It only says the week *may* spend saved workouts: the season walk
// decides when the week is judged whether they cover the whole shortfall
// (nextCarryBank). Undoing it while the week runs disables the row. Modelled on Sickleave;
// see docs/flexible-workouts.md.
type CarryOverUse struct {
	GormModel
	Enabled bool      `json:"enabled" gorm:"not null; default: true"`
	GoalID  uuid.UUID `json:"goal_id" gorm:"type:varchar(100);"`
	Goal    Goal      `json:"-" gorm:"not null"`
	Date    time.Time `json:"date" gorm:"not null"`
}
