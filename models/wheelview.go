package models

import (
	"github.com/google/uuid"
)

type Wheelview struct {
	GormModel
	UserID uuid.UUID `json:"" gorm:"type:varchar(100);"`
	// Never serialized: this is the GORM association, and rows like this one reach response
	// bodies embedded in other structures (models.Week carries []Goal, for one). It marshals
	// as an empty object today only because nothing preloads it — accidental safety. The
	// read path hands out models.PublicUser instead. See docs/wip.md, S15.
	User    User      `json:"-" gorm:"not null"`
	DebtID  uuid.UUID `json:"" gorm:"type:varchar(100);"`
	Debt    Debt      `json:"debt" gorm:"not null"`
	Viewed  bool      `json:"viewed" gorm:"not null; default: false"`
	Enabled bool      `json:"enabled" gorm:"not null; default: true"`
}

type WheelviewObject struct {
	GormModel
	User    PublicUser `json:"user" `
	Debt    DebtObject `json:"debt" `
	Viewed  bool       `json:"viewed" `
	Enabled bool       `json:"enabled" `
}
