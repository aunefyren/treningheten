package models

import (
	"time"

	"github.com/google/uuid"
)

// OAuth / API scopes. ScopeAPI is the legacy full read+write scope carried by
// access tokens; ScopeAPIRead / ScopeAPIWrite split that for Personal Access
// Tokens. ScopeAdmin gates admin endpoints and implies write.
const (
	ScopeAPI      = "api"       // full read+write API access (access tokens)
	ScopeAPIRead  = "api:read"  // read-only API access (GET/HEAD)
	ScopeAPIWrite = "api:write" // read+write API access
	ScopeAdmin    = "admin"     // admin endpoints
)

// SupportedScopes is advertised in discovery metadata.
var SupportedScopes = []string{ScopeAPI, ScopeAPIRead, ScopeAPIWrite, ScopeAdmin}

// FirstPartyClientID is the seeded OAuth client used by the Treningheten web app.
const FirstPartyClientID = "treningheten-web"

// OAuthClient is a registered OAuth 2.0 client (via Dynamic Client Registration
// or seeded for the first-party web app). Public clients (e.g. browser/MCP
// clients using PKCE) have no secret.
type OAuthClient struct {
	GormModel
	ClientID                string  `json:"client_id" gorm:"unique;not null;type:varchar(100)"`
	ClientSecretHash        *string `json:"-" gorm:"default:null"`
	ClientName              string  `json:"client_name" gorm:"not null"`
	RedirectURIs            string  `json:"redirect_uris" gorm:"not null"`               // space-delimited
	GrantTypes              string  `json:"grant_types" gorm:"not null"`                 // space-delimited
	ResponseTypes           string  `json:"response_types" gorm:"not null"`              // space-delimited
	Scope                   string  `json:"scope" gorm:"not null"`                       // space-delimited
	TokenEndpointAuthMethod string  `json:"token_endpoint_auth_method" gorm:"not null"`  // "none" | "client_secret_basic" | "client_secret_post"
	Public                  bool    `json:"-" gorm:"not null;default:true"`
	FirstParty              bool    `json:"-" gorm:"not null;default:false"`
}

// OAuthAuthorizationCode is a single-use, short-lived code issued by the
// authorization endpoint and exchanged at the token endpoint.
type OAuthAuthorizationCode struct {
	GormModel
	CodeHash            string     `json:"-" gorm:"unique;not null;type:varchar(100)"`
	ClientID            string     `json:"-" gorm:"not null;type:varchar(100)"`
	UserID              uuid.UUID  `json:"-" gorm:"not null;type:varchar(100)"`
	RedirectURI         string     `json:"-" gorm:"not null"`
	Scope               string     `json:"-" gorm:"not null"`
	CodeChallenge       string     `json:"-" gorm:"not null"`
	CodeChallengeMethod string     `json:"-" gorm:"not null"` // "S256"
	Resource            string     `json:"-" gorm:"default:null"`
	ExpiresAt           time.Time  `json:"-" gorm:"not null"`
	ConsumedAt          *time.Time `json:"-" gorm:"default:null"`
}

// OAuthRefreshToken is an opaque, hashed, revocable refresh token. Rotation is
// tracked via RotatedTo so that reuse of an already-rotated token can be
// detected and the whole chain revoked. This table is reused by Phase 2 PATs.
type OAuthRefreshToken struct {
	GormModel
	TokenHash  string     `json:"-" gorm:"unique;not null;type:varchar(100)"`
	UserID     uuid.UUID  `json:"-" gorm:"not null;type:varchar(100)"`
	ClientID   string     `json:"-" gorm:"not null;type:varchar(100)"`
	Scope      string     `json:"-" gorm:"not null"`
	ExpiresAt  time.Time  `json:"-" gorm:"not null"`
	RevokedAt  *time.Time `json:"-" gorm:"default:null"`
	RotatedTo  *uuid.UUID `json:"-" gorm:"type:varchar(100);default:null"`
	LastUsedAt *time.Time `json:"-" gorm:"default:null"`
}

// --- OAuth endpoint request/response DTOs ---

// OAuthTokenResponse is the RFC 6749 §5.1 successful token response.
type OAuthTokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"` // "Bearer"
	ExpiresIn    int    `json:"expires_in"` // seconds
	RefreshToken string `json:"refresh_token,omitempty"`
	Scope        string `json:"scope,omitempty"`
}

// OAuthError is the RFC 6749 §5.2 error response.
type OAuthError struct {
	ErrorCode        string `json:"error"`
	ErrorDescription string `json:"error_description,omitempty"`
	ErrorURI         string `json:"error_uri,omitempty"`
}

// OAuthClientRegistrationResponse is the RFC 7591 DCR response.
type OAuthClientRegistrationResponse struct {
	ClientID                string   `json:"client_id"`
	ClientSecret            string   `json:"client_secret,omitempty"`
	ClientName              string   `json:"client_name,omitempty"`
	RedirectURIs            []string `json:"redirect_uris"`
	GrantTypes              []string `json:"grant_types"`
	ResponseTypes           []string `json:"response_types"`
	Scope                   string   `json:"scope,omitempty"`
	TokenEndpointAuthMethod string   `json:"token_endpoint_auth_method"`
}

// AuthorizationServerMetadata is the RFC 8414 discovery document.
type AuthorizationServerMetadata struct {
	Issuer                            string   `json:"issuer"`
	AuthorizationEndpoint             string   `json:"authorization_endpoint"`
	TokenEndpoint                     string   `json:"token_endpoint"`
	RegistrationEndpoint              string   `json:"registration_endpoint"`
	RevocationEndpoint                string   `json:"revocation_endpoint"`
	ScopesSupported                   []string `json:"scopes_supported"`
	ResponseTypesSupported            []string `json:"response_types_supported"`
	GrantTypesSupported               []string `json:"grant_types_supported"`
	TokenEndpointAuthMethodsSupported []string `json:"token_endpoint_auth_methods_supported"`
	CodeChallengeMethodsSupported     []string `json:"code_challenge_methods_supported"`
}

// ProtectedResourceMetadata is the RFC 9728 discovery document used by MCP
// clients to locate the authorization server.
type ProtectedResourceMetadata struct {
	Resource             string   `json:"resource"`
	AuthorizationServers []string `json:"authorization_servers"`
	ScopesSupported      []string `json:"scopes_supported,omitempty"`
	BearerMethodsSupported []string `json:"bearer_methods_supported,omitempty"`
}
