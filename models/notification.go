package models

import (
	"time"

	"gorm.io/gorm"
)

type Subscription struct {
	gorm.Model
	Enabled          bool       `json:"enabled" gorm:"not null; default: true"`
	User             int        `json:"user" gorm:"not null"`
	Endpoint         string     `json:"endpoint" gorm:"not null"`
	ExpirationTime   *time.Time `json:"expiration_time"`
	P256Dh           string     `json:"p256dh" gorm:"not null"`
	Auth             string     `json:"auth" gorm:"not null"`
	SundayAlert      bool       `json:"sunday_alert" gorm:"not null; default: false"`
	AchievementAlert bool       `json:"achievement_alert" gorm:"not null; default: false"`
}

type SubscriptionOriginal struct {
	Endpoint       string                   `json:"endpoint"`
	ExpirationTime *time.Time               `json:"expirationTime"`
	Keys           SubscriptionOriginalKeys `json:"keys"`
}

type SubscriptionOriginalKeys struct {
	Auth   string `json:"auth"`
	P256Dh string `json:"p256dh"`
}

type SubscriptionCreationRequest struct {
	Subscription SubscriptionOriginal `json:"subscription"`
	Settings     struct {
		SundayAlert      bool `json:"sunday_alert"`
		AchievementAlert bool `json:"achievement_alert"`
	}
}

type NotificationCreationRequest struct {
	Title    string `json:"title"`
	Body     string `json:"body"`
	UserID   int    `json:"user"`
	Category string `json:"category"`
}
