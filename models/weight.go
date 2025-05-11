package models

import (
	"time"

	"github.com/google/uuid"
)

type WeightValue struct {
	GormModel
	Enabled bool      `json:"enabled" gorm:"not null; default: true"`
	Date    time.Time `json:"date" gorm:"not null"`
	Weight  float64   `json:"weight" gorm:"not null"`
	UserID  uuid.UUID `json:"" gorm:"type:varchar(100);"`
	User    User      `json:"user" gorm:"not null;foreignKey:UserID"`
}

type WeightValueCreationRequest struct {
	Date   time.Time `json:"date"`
	Weight float64   `json:"weight"`
	UserID uuid.UUID `json:"user_id" `
}
