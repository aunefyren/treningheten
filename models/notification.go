package models

import (
	"time"

	"github.com/google/uuid"
)

type Subscription struct {
	GormModel
	Enabled          bool       `json:"enabled" gorm:"not null; default: true"`
	UserID           uuid.UUID  `json:"" gorm:"type:varchar(100);"`
	User             User       `json:"user" gorm:"not null"`
	Endpoint         string     `json:"endpoint" gorm:"not null"`
	ExpirationTime   *time.Time `json:"expiration_time"`
	P256Dh           string     `json:"p256dh" gorm:"not null"`
	Auth             string     `json:"auth" gorm:"not null"`
	SundayAlert      bool       `json:"sunday_alert" gorm:"not null; default: false"`
	AchievementAlert bool       `json:"achievement_alert" gorm:"not null; default: false"`
	NewsAlert        bool       `json:"news_alert" gorm:"not null; default: false"`
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
		NewsAlert        bool `json:"news_alert"`
	}
}

type NotificationCreationRequest struct {
	Title          string    `json:"title"`
	Body           string    `json:"body"`
	UserID         uuid.UUID `json:"user"`
	AdditionalData *string   `json:"additional_data"`
	Category       string    `json:"category"`
}

type SubscriptionGetRequest struct {
	Endpoint string `json:"endpoint"`
}

type SubscriptionUpdateRequest struct {
	Endpoint         string `json:"endpoint"`
	SundayAlert      bool   `json:"sunday_alert"`
	AchievementAlert bool   `json:"achievement_alert"`
	NewsAlert        bool   `json:"news_alert"`
}
