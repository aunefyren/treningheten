package models

// Plex DTOs for the plex.tv PIN auth flow and resource discovery. Only the fields
// Treningheten uses are modelled. See docs/media.md.

// PlexPin is the plex.tv/api/v2/pins resource. AuthToken is null until the user
// authorizes the PIN in their browser, then holds the account token.
type PlexPin struct {
	ID        int64   `json:"id"`
	Code      string  `json:"code"`
	AuthToken *string `json:"authToken"`
}

// PlexAccount is the plex.tv/api/v2/user resource for the authorized token. ID is
// stored as MediaConnection.AccountID to filter server history to this user.
type PlexAccount struct {
	ID       int64  `json:"id"`
	UUID     string `json:"uuid"`
	Username string `json:"username"`
	Email    string `json:"email"`
}

// PlexServerAccount is one account on a Plex Media Server (from {server}/accounts).
// Its ID is the SERVER-LOCAL account id used in history rows (the owner is usually
// 1) — distinct from the plex.tv global account id returned by plex.tv/api/v2/user.
type PlexServerAccount struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

// PlexServerAccountsResponse is the {server}/accounts reply (Accept: json).
type PlexServerAccountsResponse struct {
	MediaContainer struct {
		Account []PlexServerAccount `json:"Account"`
	} `json:"MediaContainer"`
}

// PlexConnection is one reachable address for a Plex resource (server). URI is the
// fully-qualified connection URL (e.g. the https plex.direct address).
type PlexConnection struct {
	Protocol string `json:"protocol"`
	Address  string `json:"address"`
	Port     int    `json:"port"`
	URI      string `json:"uri"`
	Local    bool   `json:"local"`
	Relay    bool   `json:"relay"`
}

// PlexResource is one device/server from plex.tv/api/v2/resources. A Plex Media
// Server has "server" in Provides.
type PlexResource struct {
	Name             string           `json:"name"`
	Product          string           `json:"product"`
	Provides         string           `json:"provides"`
	ClientIdentifier string           `json:"clientIdentifier"`
	Owned            bool             `json:"owned"`
	Connections      []PlexConnection `json:"connections"`
}

// PlexPinResponse is what the connect endpoint hands the frontend so it can open
// the Plex auth window and then poll for completion.
type PlexPinResponse struct {
	PinID   int64  `json:"pin_id"`
	Code    string `json:"code"`
	AuthURL string `json:"auth_url"`
}

// PlexPinCheckResponse is the poll result: Authorized flips true once the user has
// approved the PIN, at which point Connection carries the safe connection object.
type PlexPinCheckResponse struct {
	Authorized bool                   `json:"authorized"`
	Connection *MediaConnectionObject `json:"connection"`
}

// PlexServerURLRequest overrides the auto-discovered server URL — needed when Plex
// is fronted by a reverse proxy (e.g. Cloudflare on :443), which Plex's resources
// API does not advertise.
type PlexServerURLRequest struct {
	ServerURL string `json:"server_url"`
}

// PlexHistoryResponse is the PMS /status/sessions/history/all reply (Accept: json).
type PlexHistoryResponse struct {
	MediaContainer struct {
		Size     int                   `json:"size"`
		Metadata []PlexHistoryMetadata `json:"Metadata"`
	} `json:"MediaContainer"`
}

// PlexHistoryMetadata is one played item from the server history. ViewedAt is the
// scrobble timestamp (unix seconds); Duration is the full item length in
// milliseconds (not always present in history payloads). The grandparent/parent
// titles are generic across media types: artist/show/author and album/season/series.
type PlexHistoryMetadata struct {
	RatingKey        string `json:"ratingKey"`
	Key              string `json:"key"`
	Title            string `json:"title"`
	GrandparentTitle string `json:"grandparentTitle"`
	ParentTitle      string `json:"parentTitle"`
	Type             string `json:"type"`
	Thumb            string `json:"thumb"`
	ViewedAt         int64  `json:"viewedAt"`
	AccountID        int64  `json:"accountID"`
	Duration         int64  `json:"duration"`
}
