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
	User             User      `json:"user" gorm:"not null"`
	Enabled          bool      `json:"enabled" gorm:"not null; default: true"`
}

type GoalCreationRequest struct {
	ExerciseInterval int  `json:"exercise_interval"`
	Competing        bool `json:"competing"`
}

type GoalObject struct {
	GormModel
	SeasonID         uuid.UUID `json:"season"`
	ExerciseInterval int       `json:"exercise_interval"`
	Competing        bool      `json:"competing"`
	User             User      `json:"user"`
	Enabled          bool      `json:"enabled"`
	SickleaveLeft    int       `json:"sickleave_left"`
}
