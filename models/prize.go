package models

import (
	"gorm.io/gorm"
)

type Prize struct {
	gorm.Model
	Name      string `json:"name" gorm:"not null"`
	Quanitity int    `json:"quantity" gorm:"not null"`
	Enabled   bool   `json:"enabled" gorm:"not null;default: true"`
}
