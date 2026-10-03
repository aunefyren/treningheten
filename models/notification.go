package models

import (
	"time"

	"github.com/google/uuid"
)

type Subscription struct {
	GormModel
	Enabled bool      `json:"enabled" gorm:"not null; default: true"`
	UserID  uuid.UUID `json:"" gorm:"type:varchar(100);"`
	// Never serialized: the GORM association, not part of any response shape. The read path
	// hands out models.PublicUser. See docs/wip.md, S15.
	User             User       `json:"-" gorm:"not null"`
	Endpoint         string     `json:"endpoint" gorm:"not null"`
	ExpirationTime   *time.Time `json:"expiration_time"`
	P256Dh           string     `json:"p256dh" gorm:"not null"`
	Auth             string     `json:"auth" gorm:"not null"`
	SundayAlert      bool       `json:"sunday_alert" gorm:"not null; default: false"`
	AchievementAlert bool       `json:"achievement_alert" gorm:"not null; default: false"`
	NewsAlert        bool       `json:"news_alert" gorm:"not null; default: false"`
	// AccountAlert covers notices about the user's own account, such as a broken
	// integration connection. On by default (existing rows included): these are things
	// the user has to act on. Mind the insert-drops-false trap when creating a row.
	AccountAlert bool `json:"account_alert" gorm:"not null; default: true"`
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
		// A pointer so an older client that doesn't send it gets the default (on)
		// rather than silently opting out.
		AccountAlert *bool `json:"account_alert"`
	}
}

// PushNotificationPayload is the JSON body delivered to the service worker's push
// handler. It is marshalled with encoding/json so values containing quotes, newlines
// or other special characters are escaped correctly.
type PushNotificationPayload struct {
	Title          string  `json:"title"`
	Body           string  `json:"body"`
	AdditionalData *string `json:"additional_data"`
	Category       string  `json:"category"`
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
	// Nil leaves the stored value alone (an older client that doesn't know the field).
	AccountAlert *bool `json:"account_alert"`
}
