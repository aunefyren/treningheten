package models

import (
	"gorm.io/gorm"
)

type Wheelview struct {
	gorm.Model
	User    int  `json:"user" gorm:"not null"`
	Debt    int  `json:"debt" gorm:"not null"`
	Viewed  bool `json:"viewed" gorm:"not null; default: false"`
	Enabled bool `json:"enabled" gorm:"not null; default: true"`
}

type WheelviewObject struct {
	gorm.Model
	User    User       `json:"user" `
	Debt    DebtObject `json:"debt" `
	Viewed  bool       `json:"viewed" `
	Enabled bool       `json:"enabled" `
}
