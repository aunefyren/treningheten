package models

import (
	"time"

	"github.com/google/uuid"
)

type ExerciseDay struct {
	GormModel
	Date    time.Time `json:"date" gorm:"not null"`
	Note    string    `json:"note"`
	Enabled bool      `json:"enabled" gorm:"not null; default: true"`
	GoalID  uuid.UUID `json:"" gorm:"type:varchar(100);"`
	Goal    Goal      `json:"goal" gorm:"not null"`
}

type ExerciseDayObject struct {
	GormModel
	Date             time.Time        `json:"date"`
	Note             string           `json:"note"`
	Enabled          bool             `json:"enabled"`
	Goal             GoalObject       `json:"goal"`
	ExerciseInterval int              `json:"exercise_interval"`
	Exercises        []ExerciseObject `json:"exercises"`
}

type ExerciseDayUpdateRequest struct {
	Note string `json:"note"`
}

type Exercise struct {
	GormModel
	Note          string         `json:"note"`
	Duration      *time.Duration `json:"duration"`
	Enabled       bool           `json:"enabled" gorm:"not null; default: true"`
	On            bool           `json:"on" gorm:"not null; default: true"`
	ExerciseDayID uuid.UUID      `json:"" gorm:"type:varchar(100);"`
	ExerciseDay   ExerciseDay    `json:"exercise_day" gorm:"not null"`
	StravaID      string         `json:"strava_id" gorm:"default: null;"`
}

type ExerciseUpdateRequest struct {
	Note     string         `json:"note"`
	On       bool           `json:"on"`
	Duration *time.Duration `json:"duration"`
}

type ExerciseCreationRequest struct {
	ExerciseDayID uuid.UUID      `json:"exercise_day_id"`
	Note          string         `json:"note"`
	On            bool           `json:"on"`
	Duration      *time.Duration `json:"duration"`
}

type ExerciseObject struct {
	GormModel
	Note        string            `json:"note"`
	Duration    *time.Duration    `json:"duration"`
	Enabled     bool              `json:"enabled"`
	On          bool              `json:"on"`
	ExerciseDay uuid.UUID         `json:"exercise_day"`
	Operations  []OperationObject `json:"operations"`
	StravaID    string            `json:"strava_id"`
}

type ExerciseDayCreationRequest struct {
	Date             time.Time `json:"date"`
	Note             string    `json:"note"`
	ExerciseInterval int       `json:"exercise_interval"`
}

type Week struct {
	Days []ExerciseDayObject `json:"days"`
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

type WeekResult struct {
	WeekNumber            int        `json:"week_number"`
	WeekYear              int        `json:"week_year"`
	WeekDate              time.Time  `json:"week_date"`
	UserID                uuid.UUID  `json:"user_id"`
	GoalID                uuid.UUID  `json:"goal_id"`
	DebtID                *uuid.UUID `json:"debt_id"`
	ExercisePercentage    float64    `json:"exercise_percentage"`
	SickLeave             bool       `json:"sick_leave"`
	FullWeekParticipation bool       `json:"full_week_participation"`
}
