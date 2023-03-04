package models

import (
	"gorm.io/gorm"
)

type Goal struct {
	gorm.Model
	Season           int  `json:"season" gorm:"not null"`
	ExerciseInterval int  `json:"exercise_interval" gorm:"not null; default: 3"`
	Competing        bool `json:"competing" gorm:"not null; default: true"`
	User             int  `json:"user" gorm:"not null"`
	Enabled          bool `json:"enabled" gorm:"not null; default: true"`
}

type GoalCreationRequest struct {
	ExerciseInterval int  `json:"exercise_interval"`
	Competing        bool `json:"competing"`
}

type GoalObject struct {
	gorm.Model
	Season           int  `json:"season"`
	ExerciseInterval int  `json:"exercise_interval"`
	Competing        bool `json:"competing"`
	User             User `json:"user"`
	Enabled          bool `json:"enabled"`
	SickleaveLeft    int  `json:"sickleave_left"`
}
