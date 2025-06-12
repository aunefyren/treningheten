package models

import (
	"time"

	"github.com/google/uuid"
)

type Operation struct {
	GormModel
	Enabled      bool           `json:"enabled" gorm:"not null; default: true"`
	ExerciseID   uuid.UUID      `json:"" gorm:"type:varchar(100);"`
	Exercise     Exercise       `json:"exercise" gorm:"not null;"`
	ActionID     *uuid.UUID     `json:"" gorm:"type:varchar(100);"`
	Action       *Action        `json:"action" gorm:""`
	Type         string         `json:"type" gorm:"not null; default: lifting"`
	WeightUnit   string         `json:"weight_unit" gorm:"not null; default: kg"`
	DistanceUnit string         `json:"distance_unit" gorm:"not null; default: km"`
	Equipment    *string        `json:"equipment" gorm:""`
	StravaID     *string        `json:"strava_id" gorm:"default: null;"`
	Note         *string        `json:"note" gorm:"default: null;"`
	Duration     *time.Duration `json:"duration"`
}

type OperationCreationRequest struct {
	ExerciseID   uuid.UUID  `json:"exercise_id"`
	Action       *uuid.UUID `json:"action"`
	Type         string     `json:"type"`
	WeightUnit   string     `json:"weight_unit"`
	DistanceUnit string     `json:"distance_unit"`
	Equipment    *string    `json:"equipment"`
}

type OperationUpdateRequest struct {
	Action       string `json:"action"`
	Type         string `json:"type"`
	WeightUnit   string `json:"weight_unit"`
	DistanceUnit string `json:"distance_unit"`
	Equipment    string `json:"equipment"`
}

type OperationObject struct {
	GormModel
	Enabled       bool                 `json:"enabled"`
	Exercise      uuid.UUID            `json:"exercise"`
	OperationSets []OperationSetObject `json:"operation_sets"`
	Action        *Action              `json:"action"`
	Type          string               `json:"type"`
	WeightUnit    string               `json:"weight_unit"`
	DistanceUnit  string               `json:"distance_unit"`
	Equipment     *string              `json:"equipment"`
	StravaID      *string              `json:"strava_id"`
	Note          *string              `json:"note"`
	Duration      *time.Duration       `json:"duration"`
}

type OperationSet struct {
	GormModel
	Enabled     bool           `json:"enabled" gorm:"not null; default: true;"`
	OperationID uuid.UUID      `json:"" gorm:"type:varchar(100);"`
	Operation   Operation      `json:"operation" gorm:"not null"`
	Repetitions *float64       `json:"repetitions" gorm:"default: null"`
	Weight      *float64       `json:"weight" gorm:"default: null"`
	Distance    *float64       `json:"distance" gorm:"default: null"`
	Time        *time.Duration `json:"time" gorm:"default: null"`
	StravaID    *string        `json:"strava_id" gorm:"default: null;"`
}

type OperationSetCreationRequest struct {
	OperationID uuid.UUID      `json:"operation_id"`
	Repetitions *float64       `json:"repetitions"`
	Weight      *float64       `json:"weight"`
	Distance    *float64       `json:"distance"`
	Time        *time.Duration `json:"time"`
}

type OperationSetUpdateRequest struct {
	Repetitions *float64       `json:"repetitions"`
	Weight      *float64       `json:"weight"`
	Distance    *float64       `json:"distance"`
	Time        *time.Duration `json:"time"`
}

type OperationSetObject struct {
	GormModel
	Enabled     bool           `json:"enabled"`
	Operation   uuid.UUID      `json:"operation"`
	Repetitions *float64       `json:"repetitions"`
	Weight      *float64       `json:"weight"`
	Distance    *float64       `json:"distance"`
	Time        *time.Duration `json:"time"`
	StravaID    *string        `json:"strava_id" gorm:"default: null;"`
}

type Action struct {
	GormModel
	Enabled       bool    `json:"enabled" gorm:"not null; default: true;"`
	Name          string  `json:"name"`
	NorwegianName string  `json:"norwegian_name"`
	Description   string  `json:"description"`
	Type          string  `json:"type"`
	BodyPart      string  `json:"body_part"`
	StravaName    string  `json:"strava_name"`
	PastTenseVerb *string `json:"past_tense_verb"`
	HasLogo       bool    `json:"has_logo" gorm:"not null; default: false;"`
}

type ActionCreationRequest struct {
	Name          string `json:"name"`
	NorwegianName string `json:"norwegian_name"`
	Description   string `json:"description"`
	Type          string `json:"type"`
	BodyPart      string `json:"body_part"`
}

type ActionStatistics struct {
	Action     Action                `json:"action"`
	Statistics StatisticsCompilation `json:"statistics"`
	Operations []OperationObject     `json:"operations"`
}

type StatisticsCompilation struct {
	Sums     StatisticsSumCompilation     `json:"sums"`
	Averages StatisticsAverageCompilation `json:"averages"`
	Tops     StatisticsTopCompilation     `json:"tops"`
}

type StatisticsSumCompilation struct {
	Distance   float64       `json:"distance"`
	Time       time.Duration `json:"time"`
	Repetition float64       `json:"repetition"`
	Weight     float64       `json:"weight"`
	Operations int64         `json:"operations"`
}

type StatisticsAverageCompilation struct {
	Distance   float64 `json:"distance"`
	Time       int     `json:"time"`
	Repetition float64 `json:"repetition"`
	Weight     float64 `json:"weight"`
}

type StatisticsTopCompilation struct {
	Distance   *OperationObject `json:"distance"`
	Time       *OperationObject `json:"time"`
	Repetition *OperationObject `json:"repetition"`
	Weight     *OperationObject `json:"weight"`
}
