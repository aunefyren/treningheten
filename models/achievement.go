package models

import (
	"time"

	"gorm.io/gorm"
)

type Achievement struct {
	gorm.Model
	Enabled     bool   `json:"enabled" gorm:"not null; default: true"`
	Name        string `json:"name" gorm:"not null"`
	Description string `json:"description" gorm:"not null"`
}

type AchievementDelegation struct {
	gorm.Model
	Enabled     bool      `json:"enabled" gorm:"not null; default: true"`
	User        int       `json:"user" gorm:"not null"`
	Achievement int       `json:"achievement" gorm:"not null"`
	GivenAt     time.Time `json:"given_at" gorm:"default:CURRENT_TIMESTAMP()"`
}

type AchievementObject struct {
	Enabled     bool      `json:"enabled"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	GivenAt     time.Time `json:"given_at"`
	GivenTo     User      `json:"user"`
	ID          uint      `json:"id"`
}
