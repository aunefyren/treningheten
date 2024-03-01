package models

import (
	"time"

	"github.com/google/uuid"
)

type Operation struct {
	GormModel
	Enabled      bool      `json:"enabled" gorm:"not null; default: true"`
	ExerciseID   uuid.UUID `json:"" gorm:"type:varchar(100);"`
	Exercise     Exercise  `json:"exercise" gorm:"not null;"`
	Action       string    `json:"action" gorm:""`
	Type         string    `json:"type" gorm:"not null; default: lifting"`
	WeightUnit   string    `json:"weight_unit" gorm:"not null; default: kg"`
	DistanceUnit string    `json:"distance_unit" gorm:"not null; default: km"`
}

type OperationCreationRequest struct {
	ExerciseID   uuid.UUID `json:"exercise_id"`
	Action       string    `json:"action"`
	Type         string    `json:"type"`
	WeightUnit   string    `json:"weight_unit"`
	DistanceUnit string    `json:"distance_unit"`
}

type OperationObject struct {
	GormModel
	Enabled       bool                 `json:"enabled"`
	Exercise      uuid.UUID            `json:"exercise"`
	OperationSets []OperationSetObject `json:"operation_sets"`
	Action        string               `json:"action"`
	Type          string               `json:"type"`
	WeightUnit    string               `json:"weight_unit"`
	DistanceUnit  string               `json:"distance_unit"`
}

type OperationSet struct {
	GormModel
	Enabled     bool          `json:"enabled" gorm:"not null; default: true;"`
	OperationID uuid.UUID     `json:"" gorm:"type:varchar(100);"`
	Operation   Operation     `json:"operation" gorm:"not null"`
	Repetitions float64       `json:"repetitions" gorm:"default: 10"`
	Weight      float64       `json:"weight" gorm:"default: 10"`
	Distance    float64       `json:"distance" gorm:"default: 10"`
	Time        time.Duration `json:"time" gorm:"default: 1"`
}

type OperationSetCreationRequest struct {
	GormModel
	OperationID uuid.UUID     `json:"operation_id"`
	Repetitions float64       `json:"repetitions"`
	Weight      float64       `json:"weight"`
	Distance    float64       `json:"distance"`
	Time        time.Duration `json:"time"`
}

type OperationSetObject struct {
	GormModel
	Enabled     bool          `json:"enabled"`
	Operation   uuid.UUID     `json:"operation"`
	Repetitions float64       `json:"repetitions"`
	Weight      float64       `json:"weight"`
	Distance    float64       `json:"distance"`
	Time        time.Duration `json:"time"`
}
