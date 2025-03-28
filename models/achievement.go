package models

import (
	"time"

	"github.com/google/uuid"
)

type Achievement struct {
	GormModel
	Enabled             bool   `json:"enabled" gorm:"not null; default: true;"`
	Name                string `json:"name" gorm:"not null;"`
	Description         string `json:"description" gorm:"not null;"`
	Category            string `json:"category" gorm:"default: default;"`
	CategoryColor       string `json:"category_color" gorm:"default: #778da9;"`
	AchievementOrder    int    `json:"achievement_order" gorm:"default: 1;"`
	MultipleDelegations *bool  `json:"multiple_delegations" gorm:"default: 0;"`
	HiddenDescription   *bool  `json:"hidden_description" gorm:"default: 0;"`
}

type AchievementDelegation struct {
	GormModel
	Enabled       bool        `json:"enabled" gorm:"not null; default: true;"`
	UserID        uuid.UUID   `json:"user_id" gorm:"type: varchar(100);"`
	User          User        `json:"user" gorm:"not null; foreignkey: UserID;"`
	AchievementID uuid.UUID   `json:"" gorm:"type: varchar(100);"`
	Achievement   Achievement `json:"achievement" gorm:"not null; foreignkey: AchievementID;"`
	GivenAt       time.Time   `json:"given_at" gorm:"not null;"`
	Seen          bool        `json:"seen" gorm:"not null; default: false;"`
}

type AchievementUserObject struct {
	GormModel
	Enabled               bool                     `json:"enabled"`
	Name                  string                   `json:"name"`
	Description           string                   `json:"description"`
	Category              string                   `json:"category"`
	CategoryColor         string                   `json:"category_color"`
	AchievementOrder      int                      `json:"achievement_order"`
	MultipleDelegations   *bool                    `json:"multiple_delegations"`
	HiddenDescription     *bool                    `json:"hidden_description"`
	AchievementDelegation *[]AchievementDelegation `json:"achievement_delegations"`
	LastGivenAt           *time.Time               `json:"last_given_at"`
}

type AchievementDelegationCreationRequest struct {
	AchievementID uuid.UUID  `json:"achievement_id"`
	GivenAt       *time.Time `json:"given_at"`
}
