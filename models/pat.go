package models

import (
	"time"

	"github.com/google/uuid"
)

// PATPrefix identifies a Personal Access Token so the auth middleware can tell
// it apart from a stateless access-token JWT.
const PATPrefix = "trh_pat_"

// PATMaxLifetimeDays is the maximum allowed expiry for a Personal Access Token.
const PATMaxLifetimeDays = 365

// PersonalAccessToken is a long-lived, user-created, revocable API credential.
// Only the SHA-256 hash of the token is stored.
type PersonalAccessToken struct {
	GormModel
	UserID     uuid.UUID  `json:"user_id" gorm:"not null;type:varchar(100)"`
	Name       string     `json:"name" gorm:"not null"`
	TokenHash  string     `json:"-" gorm:"unique;not null;type:varchar(100)"`
	Scope      string     `json:"scope" gorm:"not null"`
	ExpiresAt  time.Time  `json:"expires_at" gorm:"not null"`
	LastUsedAt *time.Time `json:"last_used_at" gorm:"default:null"`
	RevokedAt  *time.Time `json:"-" gorm:"default:null"`
}

// PATCreationRequest is the body for creating a Personal Access Token.
type PATCreationRequest struct {
	Name          string `json:"name"`
	Scope         string `json:"scope"`           // "api:read" or "api:write"
	Admin         bool   `json:"admin"`           // include the admin scope (admins only)
	ExpiresInDays int    `json:"expires_in_days"` // 1..PATMaxLifetimeDays
}

// PATCreationResponse returns the one-time plaintext token plus its metadata.
type PATCreationResponse struct {
	Token string              `json:"token"`
	PAT   PersonalAccessToken `json:"pat"`
}
