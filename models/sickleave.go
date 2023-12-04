package models

import (
	"time"

	"github.com/google/uuid"
)

type Sickleave struct {
	GormModel
	Enabled bool      `json:"enabled" gorm:"not null; default: true"`
	GoalID  uuid.UUID `json:"" gorm:"type:varchar(100);"`
	Goal    Goal      `json:"goal" gorm:"not null"`
	Used    bool      `json:"used" gorm:"not null;default: false"`
	Date    time.Time `json:"date" gorm:"default: null"`
}
