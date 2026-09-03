package models

import (
	"github.com/google/uuid"
)

type Invite struct {
	GormModel
	Code        string     `json:"code" gorm:"unique;not null"`
	Used        bool       `json:"used" gorm:"not null;default: false"`
	RecipientID *uuid.UUID `json:"" gorm:"type:varchar(100);"`
	// Never serialized: this is the GORM association, and rows like this one reach response
	// bodies embedded in other structures (models.Week carries []Goal, for one). It marshals
	// as an empty object today only because nothing preloads it — accidental safety. The
	// read path hands out models.PublicUser instead. See docs/wip.md, S15.
	Recipient *User `json:"-" gorm:"default: null"`
	Enabled   bool  `json:"enabled" gorm:"not null;default: true"`
}

type InviteObject struct {
	GormModel
	Code      string      `json:"code"`
	Used      bool        `json:"used"`
	Recipient *PublicUser `json:"recipient"`
	Enabled   *bool       `json:"enabled"`
}
