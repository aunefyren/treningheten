package models

// Spotify DTOs for the OAuth token exchange and the recently-played history.
// Only the fields Treningheten uses are modelled. See docs/media.md.

// SpotifyTokenResponse is the accounts.spotify.com/api/token reply for both the
// authorization-code exchange and the refresh-token grant. RefreshToken is only
// returned on the initial exchange (and sometimes on refresh); ExpiresIn is seconds.
type SpotifyTokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	Scope        string `json:"scope"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
}

// SpotifyRecentlyPlayed is the GET /v1/me/player/recently-played reply.
type SpotifyRecentlyPlayed struct {
	Items []SpotifyPlayHistory `json:"items"`
}

// SpotifyPlayHistory is one play. PlayedAt is the timestamp the track stopped
// playing (RFC3339), the same "scrobble = end of play" semantics as Plex's viewedAt.
type SpotifyPlayHistory struct {
	Track    SpotifyTrack `json:"track"`
	PlayedAt string       `json:"played_at"`
}

type SpotifyTrack struct {
	ID         string          `json:"id"`
	Name       string          `json:"name"`
	DurationMs int64           `json:"duration_ms"`
	Artists    []SpotifyArtist `json:"artists"`
	Album      SpotifyAlbum    `json:"album"`
}

type SpotifyArtist struct {
	Name string `json:"name"`
}

type SpotifyAlbum struct {
	Name   string         `json:"name"`
	Images []SpotifyImage `json:"images"`
}

type SpotifyImage struct {
	URL    string `json:"url"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

// SpotifyCallbackRequest carries the authorization code from the /oauth page after
// the user approves the Spotify consent screen.
type SpotifyCallbackRequest struct {
	Code string `json:"code"`
}
