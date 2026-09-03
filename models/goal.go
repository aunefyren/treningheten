package models

import (
	"github.com/google/uuid"
)

type Goal struct {
	GormModel
	SeasonID         uuid.UUID `json:"" gorm:"type:varchar(100);"`
	Season           Season    `json:"season" gorm:"not null"`
	ExerciseInterval int       `json:"exercise_interval" gorm:"not null; default: 3"`
	Competing        bool      `json:"competing" gorm:"not null; default: true"`
	UserID           uuid.UUID `json:"" gorm:"type:varchar(100);"`
	// Never serialized: this is the GORM association, and rows like this one reach response
	// bodies embedded in other structures (models.Week carries []Goal, for one). It marshals
	// as an empty object today only because nothing preloads it — accidental safety. The
	// read path hands out models.PublicUser instead. See docs/wip.md, S15.
	User    User `json:"-" gorm:"not null"`
	Enabled bool `json:"enabled" gorm:"not null; default: true"`
}

type GoalCreationRequest struct {
	ExerciseInterval int       `json:"exercise_interval"`
	Competing        bool      `json:"competing"`
	SeasonID         uuid.UUID `json:"season_id"`
}

type GoalObject struct {
	GormModel
	SeasonID         uuid.UUID  `json:"season"`
	ExerciseInterval int        `json:"exercise_interval"`
	Competing        bool       `json:"competing"`
	User             PublicUser `json:"user"`
	Enabled          bool       `json:"enabled"`
	SickleaveLeft    int        `json:"sickleave_left"`
}
