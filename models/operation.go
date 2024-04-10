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
}

type Action struct {
	GormModel
	Enabled       bool   `json:"enabled" gorm:"not null; default: true;"`
	Name          string `json:"name"`
	NorwegianName string `json:"norwegian_name"`
	Description   string `json:"description"`
	Type          string `json:"type"`
	BodyPart      string `json:"body_part"`
	StravaName    string `json:"strava_name"`
}

type ActionCreationRequest struct {
	Name          string `json:"name"`
	NorwegianName string `json:"norwegian_name"`
	Description   string `json:"description"`
	Type          string `json:"type"`
	BodyPart      string `json:"body_part"`
}
