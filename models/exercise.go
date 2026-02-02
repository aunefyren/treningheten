package models

import (
	"time"

	"github.com/google/uuid"
)

type ExerciseDay struct {
	GormModel
	Date    time.Time  `json:"date" gorm:"not null"`
	Note    string     `json:"note"`
	Enabled bool       `json:"enabled" gorm:"not null; default: true"`
	GoalID  *uuid.UUID `json:"" gorm:"type:varchar(100);default: null; null;"`
	Goal    *Goal      `json:"goal" gorm:""`
	UserID  *uuid.UUID `json:"" gorm:"type:varchar(100);"`
	User    *User      `json:"user" gorm:"not null"`
}

type ExerciseDayObject struct {
	GormModel
	Date             time.Time        `json:"date"`
	Note             string           `json:"note"`
	Enabled          bool             `json:"enabled"`
	Goal             *GoalObject      `json:"goal"`
	User             User             `json:"user"`
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
	IsOn          bool           `json:"is_on" gorm:"not null; default: true"`
	ExerciseDayID uuid.UUID      `json:"" gorm:"type:varchar(100);"`
	ExerciseDay   ExerciseDay    `json:"exercise_day" gorm:"not null"`
	StravaID      *string        `json:"strava_id" gorm:"default: null;"`
	Time          *time.Time     `json:"time"`
}

type ExerciseUpdateRequest struct {
	Note     string         `json:"note"`
	IsOn     bool           `json:"is_on"`
	Duration *time.Duration `json:"duration"`
}

type ExerciseCreationRequest struct {
	ExerciseDayID uuid.UUID      `json:"exercise_day_id"`
	Note          string         `json:"note"`
	IsOn          bool           `json:"is_on"`
	Duration      *time.Duration `json:"duration"`
}

type ExerciseObject struct {
	GormModel
	Note        string            `json:"note"`
	Duration    *time.Duration    `json:"duration"`
	Enabled     bool              `json:"enabled"`
	IsOn        bool              `json:"is_on"`
	ExerciseDay uuid.UUID         `json:"exercise_day"`
	Operations  []OperationObject `json:"operations"`
	StravaID    []string          `json:"strava_id"`
	Time        time.Time         `json:"time"`
}

type ExerciseDayCreationRequest struct {
	Date             time.Time `json:"date"`
	Note             string    `json:"note"`
	ExerciseInterval int       `json:"exercise_interval"`
}

type Week struct {
	Days  []ExerciseDayObject `json:"days"`
	Goals []Goal              `json:"goals"`
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
	WeekNumber            int         `json:"week_number"`
	WeekYear              int         `json:"week_year"`
	WeekDate              time.Time   `json:"week_date"`
	UserID                uuid.UUID   `json:"user_id"`
	Goals                 []uuid.UUID `json:"goals"`
	DebtID                *uuid.UUID  `json:"debt_id"`
	ExercisePercentage    float64     `json:"exercise_percentage"`
	SickLeave             bool        `json:"sick_leave"`
	FullWeekParticipation bool        `json:"full_week_participation"`
}

type Activity struct {
	ExerciseID uuid.UUID `json:"id"`
	User       User      `json:"user"`
	Time       time.Time `json:"time"`
	StravaIDs  []string  `json:"strava_ids"`
	Actions    []Action  `json:"actions"`
}
