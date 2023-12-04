package models

import (
	"github.com/google/uuid"
)

type Wheelview struct {
	GormModel
	UserID  uuid.UUID `json:"" gorm:"type:varchar(100);"`
	User    User      `json:"user" gorm:"not null"`
	DebtID  uuid.UUID `json:"" gorm:"type:varchar(100);"`
	Debt    Debt      `json:"debt" gorm:"not null"`
	Viewed  bool      `json:"viewed" gorm:"not null; default: false"`
	Enabled bool      `json:"enabled" gorm:"not null; default: true"`
}

type WheelviewObject struct {
	GormModel
	User    User       `json:"user" `
	Debt    DebtObject `json:"debt" `
	Viewed  bool       `json:"viewed" `
	Enabled bool       `json:"enabled" `
}
