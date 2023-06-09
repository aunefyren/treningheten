package models

import (
	"time"

	"gorm.io/gorm"
)

type Debt struct {
	gorm.Model
	Date    time.Time `json:"date" gorm:"not null"`
	Season  int       `json:"season" gorm:"not null"`
	Loser   int       `json:"loser" gorm:"not null"`
	Winner  int       `json:"winner" gorm:"default: null"`
	Paid    bool      `json:"paid" gorm:"not null"`
	Enabled bool      `json:"enabled" gorm:"not null;default: true"`
}

type DebtObject struct {
	gorm.Model
	Date    time.Time    `json:"date"`
	Season  SeasonObject `json:"season"`
	Loser   User         `json:"loser"`
	Winner  User         `json:"winner"`
	Paid    bool         `json:"paid"`
	Enabled bool         `json:"enabled"`
}

type DebtOverview struct {
	UnviewedDebt      []WheelviewObject `json:"debt_unviewed"`
	UnspunLostDebt    []DebtObject      `json:"debt_lost"`
	UnreceivedWonDebt []DebtObject      `json:"debt_won"`
	UnpaidLostDebt    []DebtObject      `json:"debt_unpaid"`
}

type DebtCreationRequest struct {
	Date time.Time `json:"date" gorm:"not null"`
}
