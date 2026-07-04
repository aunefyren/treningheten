package models

type ConfigStruct struct {
	Timezone                string         `json:"timezone"`
	PrivateKey              string         `json:"private_key"`
	DBUsername              string         `json:"db_username"`
	DBPassword              string         `json:"db_password"`
	DBName                  string         `json:"db_name"`
	DBIP                    string         `json:"db_ip"`
	DBPort                  int            `json:"db_port"`
	DBSSL                   bool           `json:"db_ssl"`
	DBLocation              string         `json:"db_location"`
	DBType                  string         `json:"db_type"`
	TreninghetenPort        int            `json:"treningheten_port"`
	TreninghetenName        string         `json:"treningheten_name"`
	TreninghetenDescription string         `json:"treningheten_description"`
	TreninghetenExternalURL string         `json:"treningheten_external_url"`
	TreninghetenVersion     string         `json:"treningheten_version"`
	TreninghetenEnvironment string         `json:"treningheten_environment"`
	TreninghetenTestEmail   string         `json:"treningheten_test_email"`
	TreninghetenLogLevel    string         `json:"treningheten_log_level"`
	SMTPEnabled             bool           `json:"smtp_enabled"`
	SMTPHost                string         `json:"smtp_host"`
	SMTPPort                int            `json:"smtp_port"`
	SMTPUsername            string         `json:"smtp_username"`
	SMTPPassword            string         `json:"smtp_password"`
	SMTPFrom                string         `json:"smtp_from"`
	VAPIDPublicKey          string         `json:"vapid_publickey"`
	VAPIDSecretKey          string         `json:"vapid_secretkey"`
	VAPIDContact            string         `json:"vapid_contact"`
	StravaEnabled           bool           `json:"strava_enabled"`
	StravaClientID          string         `json:"strava_client_id"`
	StravaClientSecret      string         `json:"strava_client_secret"`
	StravaRedirectURI       string         `json:"strava_redirect_uri"`
	StravaTokenKey          string         `json:"strava_token_key"`
	HevyEnabled             bool           `json:"hevy_enabled"`
	HevyTokenKey            string         `json:"hevy_token_key"`
	MCPEnabled              *bool          `json:"mcp_enabled"`
	Ollama                  OllamaSettings `json:"ollama"`
	Media                   MediaSettings  `json:"media"`
}

// MediaSettings gates the media/audio integration. Enabled is the tenant-wide
// feature flag for the whole feature; each provider is gated independently by its
// own MediaProviderSettings.Enabled (mirroring how Strava/Hevy are gated). TokenKey
// is the AES-256-GCM key used to encrypt stored provider credentials at rest, and
// is generated automatically on first run (see files/config.go).
type MediaSettings struct {
	Enabled        bool                   `json:"enabled"`
	TokenKey       string                 `json:"token_key"`
	Plex           PlexSettings           `json:"plex"`
	Spotify        SpotifySettings        `json:"spotify"`
	Audiobookshelf AudiobookshelfSettings `json:"audiobookshelf"`
}

// AudiobookshelfSettings is the Audiobookshelf provider gate. Enabled is the
// per-provider flag (a provider is usable only when both Media.Enabled and this are
// true). Unlike Plex/Spotify it needs no app-level credentials — the connection is a
// self-hosted server URL + per-user API token entered by the user, so this struct is
// just the on/off switch.
type AudiobookshelfSettings struct {
	Enabled bool `json:"enabled"`
}

// SpotifySettings is the Spotify provider gate. Unlike Plex (a self-hosted PIN
// flow), Spotify uses the OAuth authorization-code flow against a registered app,
// so it needs ClientID/ClientSecret (from developer.spotify.com) and a RedirectURI
// that is whitelisted in that app and points at this install's /oauth page.
type SpotifySettings struct {
	Enabled      bool   `json:"enabled"`
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	RedirectURI  string `json:"redirect_uri"`
}

// PlexSettings is the Plex provider gate. Enabled is the per-provider flag (a
// provider is usable only when both Media.Enabled and this are true).
// ClientIdentifier is the stable per-install Plex client id (X-Plex-Client-Identifier),
// auto-generated on first run; Plex ties authorized devices to it, so it must stay
// constant across the PIN create/poll flow and over the install's lifetime.
type PlexSettings struct {
	Enabled          bool   `json:"enabled"`
	ClientIdentifier string `json:"client_identifier"`
}

type OllamaSettings struct {
	Enabled bool   `json:"enabled"`
	URL     string `json:"url"`
	Model   string `json:"model"`
	APIKey  string `json:"api_key"`
}

type VAPIDSettings struct {
	VAPIDPublicKey string `json:"vapid_publickey"`
	VAPIDSecretKey string `json:"vapid_secretkey"`
	VAPIDContact   string `json:"vapid_contact"`
}
