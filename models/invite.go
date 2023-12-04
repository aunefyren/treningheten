package models

import (
	"github.com/google/uuid"
)

type Invite struct {
	GormModel
	Code        string     `json:"code" gorm:"unique;not null"`
	Used        bool       `json:"used" gorm:"not null;default: false"`
	RecipientID *uuid.UUID `json:"" gorm:"type:varchar(100);"`
	Recipient   *User      `json:"recipient" gorm:"default: null"`
	Enabled     bool       `json:"enabled" gorm:"not null;default: true"`
}

type InviteObject struct {
	GormModel
	Code      string `json:"code"`
	Used      bool   `json:"used"`
	Recipient *User  `json:"recipient"`
	Enabled   *bool  `json:"enabled"`
}
