package models

import (
	"time"

	"gorm.io/gorm"
)

type ExerciseDay struct {
	gorm.Model
	Date             time.Time `json:"date" gorm:"not null"`
	Note             string    `json:"note"`
	Enabled          bool      `json:"enabled" gorm:"not null; default: true"`
	Goal             int       `json:"goal" gorm:"not null"`
	ExerciseInterval int       `json:"exercise_interval" gorm:"not null; default: 0"`
}

type Exercise struct {
	gorm.Model
	Note        string `json:"note"`
	Enabled     bool   `json:"enabled" gorm:"not null; default: true"`
	On          bool   `json:"enabled" gorm:"not null; default: true"`
	ExerciseDay int    `json:"goal" gorm:"not null"`
}

type ExerciseDayCreationRequest struct {
	Date             time.Time `json:"date"`
	Note             string    `json:"note"`
	ExerciseInterval int       `json:"exercise_interval"`
}

type Week struct {
	Days []ExerciseDay `json:"days"`
}

type WeekCreationRequest struct {
	Days     []ExerciseDayCreationRequest `json:"days"`
	TimeZone string                       `json:"timezone"`
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
