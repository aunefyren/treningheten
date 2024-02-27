package models

import (
	"time"

	"github.com/google/uuid"
)

type Operation struct {
	GormModel
	Enabled    bool      `json:"enabled" gorm:"not null; default: true"`
	ExerciseID uuid.UUID `json:"" gorm:"type:varchar(100);"`
	Exercise   Exercise  `json:"exercise" gorm:"not null;"`
}

type OperationObject struct {
	GormModel
	Enabled       bool                 `json:"enabled"`
	Exercise      uuid.UUID            `json:"exercise"`
	OperationSets []OperationSetObject `json:"operation_sets"`
}

type OperationSet struct {
	GormModel
	Enabled      bool          `json:"enabled" gorm:"not null; default: true;"`
	OperationID  uuid.UUID     `json:"" gorm:"type:varchar(100);"`
	Operation    Operation     `json:"operation" gorm:"not null"`
	Type         string        `json:"type" gorm:"not null; default: lifting"`
	Action       string        `json:"action" gorm:""`
	Repetitions  float64       `json:"repetitions" gorm:"default: 10"`
	Weight       float64       `json:"weight" gorm:"default: 10"`
	WeightUnit   string        `json:"weight_unit" gorm:"not null; default: kg"`
	Distance     float64       `json:"distance" gorm:"default: 10"`
	DistanceUnit string        `json:"distance_unit" gorm:"not null; default: km"`
	Time         time.Duration `json:"time" gorm:"default: 1"`
}

type OperationSetObject struct {
	GormModel
	Enabled      bool          `json:"enabled"`
	Operation    uuid.UUID     `json:"operation"`
	Type         string        `json:"type"`
	Action       string        `json:"action"`
	Repetitions  float64       `json:"repetitions"`
	Weight       float64       `json:"weight"`
	WeightUnit   string        `json:"weight_unit"`
	Distance     float64       `json:"distance"`
	DistanceUnit string        `json:"distance_unit"`
	Time         time.Duration `json:"time"`
}
