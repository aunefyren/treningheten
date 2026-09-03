package models

import (
	"time"

	"github.com/google/uuid"
)

type Debt struct {
	GormModel
	Date     time.Time `json:"date" gorm:"not null"`
	SeasonID uuid.UUID `json:"" gorm:"type:varchar(100);"`
	Season   Season    `json:"season" gorm:"not null"`
	LoserID  uuid.UUID `json:"" gorm:"type:varchar(100);"`
	// Never serialized: this is the GORM association, and rows like this one reach response
	// bodies embedded in other structures (models.Week carries []Goal, for one). It marshals
	// as an empty object today only because nothing preloads it — accidental safety. The
	// read path hands out models.PublicUser instead. See docs/wip.md, S15.
	Loser    User       `json:"-" gorm:"not null"`
	WinnerID *uuid.UUID `json:"" gorm:"type:varchar(100);"`
	// Never serialized: this is the GORM association, and rows like this one reach response
	// bodies embedded in other structures (models.Week carries []Goal, for one). It marshals
	// as an empty object today only because nothing preloads it — accidental safety. The
	// read path hands out models.PublicUser instead. See docs/wip.md, S15.
	Winner  *User `json:"-" gorm:"default: null"`
	Paid    bool  `json:"paid" gorm:"not null"`
	Enabled bool  `json:"enabled" gorm:"not null;default: true"`
}

type DebtObject struct {
	GormModel
	Date    time.Time    `json:"date"`
	Season  SeasonObject `json:"season"`
	Loser   PublicUser   `json:"loser"`
	Winner  *PublicUser  `json:"winner"`
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
	Date       time.Time  `json:"date" gorm:"not null"`
	TargetUser *uuid.UUID `json:"target_user"`
}
