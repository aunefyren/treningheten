package models

type ServerInfoReply struct {
	Timezone            string `json:"timezone"`
	TreninghetenVersion string `json:"treningheten_version"`

	Name        string `json:"name"`
	Description string `json:"description"`
	Environment string `json:"environment"`
	ExternalURL string `json:"external_url"`
	LogLevel    string `json:"log_level"`
	Port        int    `json:"port"`

	Database ServerInfoDatabase `json:"database"`
	SMTP     ServerInfoSMTP     `json:"smtp"`
	Strava   ServerInfoStrava   `json:"strava"`
	Hevy     ServerInfoHevy     `json:"hevy"`
	AI       ServerInfoAI       `json:"ai"`
	Push     ServerInfoPush     `json:"push"`
}

type ServerInfoDatabase struct {
	Type     string `json:"type"`
	Host     string `json:"host,omitempty"`
	Port     int    `json:"port,omitempty"`
	Name     string `json:"name,omitempty"`
	Location string `json:"location,omitempty"`
	SSL      bool   `json:"ssl"`
}

type ServerInfoSMTP struct {
	Enabled bool   `json:"enabled"`
	Host    string `json:"host,omitempty"`
	Port    int    `json:"port,omitempty"`
	From    string `json:"from,omitempty"`
}

type ServerInfoStrava struct {
	Enabled     bool   `json:"enabled"`
	Configured  bool   `json:"configured"`
	RedirectURI string `json:"redirect_uri,omitempty"`
}

type ServerInfoHevy struct {
	Enabled bool `json:"enabled"`
}

type ServerInfoAI struct {
	Enabled   bool   `json:"enabled"`
	URL       string `json:"url,omitempty"`
	Model     string `json:"model,omitempty"`
	APIKeySet bool   `json:"api_key_set"`
}

type ServerInfoPush struct {
	Configured bool   `json:"configured"`
	Contact    string `json:"contact,omitempty"`
}
