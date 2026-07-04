package models

import (
	"time"

	"github.com/google/uuid"
)

// Media provider identifiers. The integration is built one provider at a time,
// each behind its own config flag (see MediaSettings); Plex is the first.
const (
	MediaProviderPlex           = "plex"
	MediaProviderSpotify        = "spotify"
	MediaProviderAudiobookshelf = "audiobookshelf"
)

// Media item types. Plex covers all three from its audio libraries, which is why
// it is the first provider — one connection exercises the full data model.
const (
	MediaTypeSong      = "song"
	MediaTypePodcast   = "podcast"
	MediaTypeAudiobook = "audiobook"
)

// MediaConnection is a per-(user, provider) account link. A dedicated table rather
// than piling provider columns onto User keeps User clean and makes "build each
// provider independently" literal. Credential fields are json:"-" (never serialised
// to API responses) and are stored encrypted with the Media.TokenKey at rest.
//
// ServerURL/RefreshToken/TokenExpiresAt/AccountID are nullable because providers
// differ: Plex needs ServerURL + AccountID (filters history to this user) and a
// long-lived token; Spotify needs RefreshToken + TokenExpiresAt and no ServerURL.
type MediaConnection struct {
	GormModel
	Enabled        bool       `json:"enabled" gorm:"not null; default: true"`
	UserID         uuid.UUID  `json:"" gorm:"type:varchar(100); not null; index"`
	User           User       `json:"user" gorm:"foreignKey:UserID; references:ID"`
	Provider       string     `json:"provider" gorm:"type:varchar(50); not null; index"`
	ServerURL      *string    `json:"server_url" gorm:"type:varchar(255); default: null"`
	AccessToken    *string    `json:"-" gorm:"type:longtext; default: null"`
	RefreshToken   *string    `json:"-" gorm:"type:longtext; default: null"`
	TokenExpiresAt *time.Time `json:"token_expires_at" gorm:"default: null"`
	AccountID      *string    `json:"account_id" gorm:"type:varchar(191); default: null"`
	LastSyncedAt   *time.Time `json:"last_synced_at" gorm:"default: null"`
}

// MediaPlayback is one row per played item — the timeline plus the source for
// cross-activity stats ("most listened", "fastest songs"). A queryable table, not
// a JSON blob on the session, because those stats are trivial over rows and
// painful over JSON. StartedAt/EndedAt are absolute timestamps and are the source
// of truth for matching against the activity window and for stream-based stats.
//
// The soundtrack attaches to the Exercise (session), not an Operation: the match
// window is a session window (one start + one duration), so a multi-operation
// session shares one soundtrack rather than duplicating it per operation.
type MediaPlayback struct {
	GormModel
	ExerciseID     uuid.UUID  `json:"" gorm:"type:varchar(100); not null; index"`
	Exercise       Exercise   `json:"exercise" gorm:"foreignKey:ExerciseID; references:ID"`
	Provider       string     `json:"provider" gorm:"type:varchar(50); not null"`
	MediaType      string     `json:"media_type" gorm:"type:varchar(50); not null; default: song"`
	Title          string     `json:"title" gorm:"type:varchar(255); not null"`
	Artist         *string    `json:"artist" gorm:"type:varchar(255); default: null"`
	Album          *string    `json:"album" gorm:"type:varchar(255); default: null"`
	ProviderItemID *string    `json:"provider_item_id" gorm:"type:varchar(191); default: null"`
	ArtworkURL     *string    `json:"artwork_url" gorm:"type:varchar(512); default: null"`
	StartedAt      time.Time  `json:"started_at" gorm:"not null"`
	EndedAt        *time.Time `json:"ended_at" gorm:"default: null"`
	// TrackLength is the full item length in seconds (repo convention: *time.Duration
	// and duration-ish ints hold a seconds count), display-only.
	TrackLength *int64 `json:"track_length" gorm:"default: null"`
}

// MediaConnectionObject is the enriched/safe read shape for a connection: identity
// and status without the credential fields, so it is safe to hand to the API.
type MediaConnectionObject struct {
	GormModel
	Enabled      bool       `json:"enabled"`
	User         uuid.UUID  `json:"user"`
	Provider     string     `json:"provider"`
	ServerURL    *string    `json:"server_url"`
	Connected    bool       `json:"connected"`
	LastSyncedAt *time.Time `json:"last_synced_at"`
}

// MediaPlaybackObject is the flattened read shape attached to OperationObject.
type MediaPlaybackObject struct {
	ID             uuid.UUID  `json:"id"`
	Provider       string     `json:"provider"`
	MediaType      string     `json:"media_type"`
	Title          string     `json:"title"`
	Artist         *string    `json:"artist"`
	Album          *string    `json:"album"`
	ProviderItemID *string    `json:"provider_item_id"`
	ArtworkURL     *string    `json:"artwork_url"`
	StartedAt      time.Time  `json:"started_at"`
	EndedAt        *time.Time `json:"ended_at"`
	TrackLength    *int64     `json:"track_length"`
}
