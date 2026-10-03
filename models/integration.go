package models

import (
	"time"

	"github.com/google/uuid"
)

// Integration providers tracked by IntegrationStatus. The media providers reuse their
// MediaProvider* identifiers so the two never drift apart.
const (
	IntegrationProviderPlex = MediaProviderPlex
)

// Integration health states. A connection with no IntegrationStatus row is healthy;
// rows only exist while something is wrong.
const (
	// IntegrationStatusOK means the provider answers. A row can sit in this state while
	// a transient failure is still inside its grace period (FailingSince set).
	IntegrationStatusOK = "ok"
	// IntegrationStatusAuthFailed means the provider rejected the stored credential
	// (revoked or expired token). Only the user can fix it, by reconnecting.
	IntegrationStatusAuthFailed = "auth_failed"
	// IntegrationStatusUnavailable means the provider has not answered for longer than
	// the grace period. It usually comes back on its own.
	IntegrationStatusUnavailable = "unavailable"
)

// IntegrationStatus records that a user's connection to an external service is failing,
// so the user can be told once and the gap can be re-pulled when it recovers. It is a
// table of its own rather than columns on User / MediaConnection because the
// connections live in different places (Strava and Hevy on User, media on
// MediaConnection) and the health logic is shared. See docs/integration-health.md.
//
// Rows are hard-deleted on recovery and disconnect, which is what lets (user_id,
// provider) be unique.
type IntegrationStatus struct {
	GormModel
	UserID uuid.UUID `json:"" gorm:"type:varchar(100); not null; uniqueIndex:idx_integration_status_user_provider"`
	// Never serialized: the GORM association, not part of any response shape.
	User     User   `json:"-" gorm:"foreignKey:UserID; references:ID"`
	Provider string `json:"provider" gorm:"type:varchar(50); not null; uniqueIndex:idx_integration_status_user_provider"`
	Status   string `json:"status" gorm:"type:varchar(50); not null"`
	// FailingSince is when the current run of failures began: the start of the gap that
	// is re-pulled on recovery.
	FailingSince *time.Time `json:"failing_since" gorm:"default: null"`
	// NotifiedAt is set when the user was told about this run of failures, so they are
	// told once per breakage rather than on every failed sync.
	NotifiedAt *time.Time `json:"notified_at" gorm:"default: null"`
}
