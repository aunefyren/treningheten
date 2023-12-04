package models

import (
	"time"

	"github.com/google/uuid"
)

type ExerciseDay struct {
	GormModel
	Date             time.Time `json:"date" gorm:"not null"`
	Note             string    `json:"note"`
	Enabled          bool      `json:"enabled" gorm:"not null; default: true"`
	GoalID           uuid.UUID `json:"goal_id" gorm:"type:varchar(100);"`
	Goal             Goal      `json:"goal" gorm:"not null"`
	ExerciseInterval int       `json:"exercise_interval" gorm:"not null; default: 0"`
}

type ExerciseDayObject struct {
	GormModel
	Date             time.Time  `json:"date"`
	Note             string     `json:"note"`
	Enabled          bool       `json:"enabled"`
	Goal             GoalObject `json:"goal"`
	ExerciseInterval int        `json:"exercise_interval"`
}

type Exercise struct {
	GormModel
	Note          string      `json:"note"`
	Enabled       bool        `json:"enabled" gorm:"not null; default: true"`
	On            bool        `json:"on" gorm:"not null; default: true"`
	ExerciseDayID uuid.UUID   `json:"" gorm:"type:varchar(100);"`
	ExerciseDay   ExerciseDay `json:"goal" gorm:"not null"`
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
