package models

import (
	"time"

	"github.com/google/uuid"
)

type Debt struct {
	GormModel
	Date     time.Time  `json:"date" gorm:"not null"`
	SeasonID uuid.UUID  `json:"" gorm:"type:varchar(100);"`
	Season   Season     `json:"season" gorm:"not null"`
	LoserID  uuid.UUID  `json:"" gorm:"type:varchar(100);"`
	Loser    User       `json:"loser" gorm:"not null"`
	WinnerID *uuid.UUID `json:"" gorm:"type:varchar(100);"`
	Winner   *User      `json:"winner" gorm:"default: null"`
	Paid     bool       `json:"paid" gorm:"not null"`
	Enabled  bool       `json:"enabled" gorm:"not null;default: true"`
}

type DebtObject struct {
	GormModel
	Date    time.Time    `json:"date"`
	Season  SeasonObject `json:"season"`
	Loser   User         `json:"loser"`
	Winner  *User        `json:"winner"`
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
