package models

import (
	"time"

	"github.com/google/uuid"
)

type Achievement struct {
	GormModel
	Enabled          bool   `json:"enabled" gorm:"not null; default: true"`
	Name             string `json:"name" gorm:"not null"`
	Description      string `json:"description" gorm:"not null"`
	Category         string `json:"category" gorm:"default: Default;"`
	CategoryColor    string `json:"category_color" gorm:"default: #778da9;"`
	AchievementOrder int    `json:"achievement_order" gorm:"default: 1;"`
}

type AchievementDelegation struct {
	GormModel
	Enabled       bool        `json:"enabled" gorm:"not null; default: true"`
	UserID        uuid.UUID   `json:"" gorm:"type:varchar(100);"`
	User          User        `json:"user" gorm:"not null"`
	AchievementID uuid.UUID   `json:"" gorm:"type:varchar(100);"`
	Achievement   Achievement `json:"achievement" gorm:"not null"`
	GivenAt       time.Time   `json:"given_at" gorm:"default:CURRENT_TIMESTAMP()"`
	Seen          bool        `json:"seen" gorm:"not null; default: false"`
}

type AchievementUserObject struct {
	GormModel
	Enabled               bool                   `json:"enabled"`
	Name                  string                 `json:"name"`
	Description           string                 `json:"description"`
	Category              string                 `json:"category"`
	CategoryColor         string                 `json:"category_color"`
	AchievementOrder      int                    `json:"achievement_order"`
	AchievementDelegation *AchievementDelegation `json:"achievement_delegation"`
}
