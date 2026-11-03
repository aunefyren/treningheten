package models

import (
	"time"
)

type News struct {
	GormModel
	Title   string    `json:"title" gorm:"not null"`
	Body    string    `json:"body"`
	Enabled bool      `json:"enabled" gorm:"not null; default: true"`
	Date    time.Time `json:"date" gorm:"not null"`
}

type NewsCreationRequest struct {
	Title string    `json:"title" gorm:"not null" sanitize:"trim"`
	Body  string    `json:"body" sanitize:"trim"`
	Date  time.Time `json:"date" gorm:"not null"`
}
