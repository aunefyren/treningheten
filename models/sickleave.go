package models

import (
	"time"

	"gorm.io/gorm"
)

type Sickleave struct {
	gorm.Model
	Enabled       bool      `json:"enabled" gorm:"not null; default: true"`
	Goal          int       `json:"goal" gorm:"not null"`
	SickleaveUsed bool      `json:"used" gorm:"not null;default: false"`
	Date          time.Time `json:"date" gorm:"default: null"`
}
