package models

import (
	"time"

	"gorm.io/gorm"
)

type Exercise struct {
	gorm.Model
	Date             time.Time `json:"date" gorm:"not null"`
	Note             string    `json:"note"`
	Enabled          bool      `json:"enabled" gorm:"not null; default: true"`
	Goal             int       `json:"goal" gorm:"not null"`
	ExerciseInterval int       `json:"exercise_interval" gorm:"not null; default: 0"`
}

type ExerciseCreationRequest struct {
	Date             time.Time `json:"date"`
	Note             string    `json:"note"`
	ExerciseInterval int       `json:"exercise_interval"`
}

type Week struct {
	Days []Exercise `json:"days"`
}

type WeekCreationRequest struct {
	Days []ExerciseCreationRequest `json:"days"`
}

type WeekFrequency struct {
	Monday    int `json:"monday"`
	Tuesday   int `json:"tuesday"`
	Wednesday int `json:"wednesday"`
	Thursday  int `json:"thursday"`
	Friday    int `json:"friday"`
	Saturday  int `json:"saturday"`
	Sunday    int `json:"sunday"`
}
