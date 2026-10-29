package controllers

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/aunefyren/treningheten/auth"
	"github.com/aunefyren/treningheten/database"
	"github.com/aunefyren/treningheten/logger"
	"github.com/aunefyren/treningheten/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/thanhpk/randstr"
	"golang.org/x/crypto/bcrypt"
)

// OAuthClientRegistrationRequest is the RFC 7591 dynamic client registration body.
type OAuthClientRegistrationRequest struct {
	ClientName              string   `json:"client_name"`
	RedirectURIs            []string `json:"redirect_uris"`
	GrantTypes              []string `json:"grant_types"`
	ResponseTypes           []string `json:"response_types"`
	Scope                   string   `json:"scope"`
	TokenEndpointAuthMethod string   `json:"token_endpoint_auth_method"`
}

// OAuthRegister implements RFC 7591 Dynamic Client Registration.
func OAuthRegister(context *gin.Context) {
	// Registration is only offered to MCP clients; behave as if it does not exist otherwise.
	if !mcpEnabled() {
		context.JSON(http.StatusNotFound, gin.H{"error": "Dynamic client registration is disabled."})
		return
	}

	var request OAuthClientRegistrationRequest
	if err := context.ShouldBindJSON(&request); err != nil {
		oauthError(context, http.StatusBadRequest, "invalid_client_metadata", "invalid registration body")
		return
	}

	// Defaults.
	grantTypes := request.GrantTypes
	if len(grantTypes) == 0 {
		grantTypes = []string{"authorization_code", "refresh_token"}
	}
	responseTypes := request.ResponseTypes
	if len(responseTypes) == 0 {
		responseTypes = []string{"code"}
	}
	authMethod := request.TokenEndpointAuthMethod
	if authMethod == "" {
		authMethod = "none"
	}

	// authorization_code clients must register at least one redirect URI.
	needsRedirect := false
	for _, g := range grantTypes {
		if g == "authorization_code" {
			needsRedirect = true
		}
	}
	if needsRedirect && len(request.RedirectURIs) == 0 {
		oauthError(context, http.StatusBadRequest, "invalid_redirect_uri", "redirect_uris is required for the authorization_code grant")
		return
	}

	// Restrict requested scope to what the server supports.
	grantedScope := narrowScopeToSupported(request.Scope)

	clientName := strings.TrimSpace(request.ClientName)
	if clientName == "" {
		clientName = "Dynamically Registered Client"
	}

	client := models.OAuthClient{
		GormModel:               models.GormModel{ID: uuid.New()},
		ClientID:                uuid.NewString(),
		ClientName:              clientName,
		RedirectURIs:            strings.Join(request.RedirectURIs, " "),
		GrantTypes:              strings.Join(grantTypes, " "),
		ResponseTypes:           strings.Join(responseTypes, " "),
		Scope:                   grantedScope,
		TokenEndpointAuthMethod: authMethod,
		Public:                  authMethod == "none",
		FirstParty:              false,
	}

	// Confidential clients get a one-time secret.
	var plaintextSecret string
	if !client.Public {
		plaintextSecret = base64Secret()
		hash, err := bcrypt.GenerateFromPassword([]byte(plaintextSecret), 12)
		if err != nil {
			oauthError(context, http.StatusInternalServerError, "server_error", "failed to create client secret")
			return
		}
		hashStr := string(hash)
		client.ClientSecretHash = &hashStr
	}

	if err := database.CreateOAuthClient(&client); err != nil {
		logger.Log.Error("failed to register oauth client. error: " + err.Error())
		oauthError(context, http.StatusInternalServerError, "server_error", "failed to register client")
		return
	}

	context.Header("Cache-Control", "no-store")
	context.Header("Pragma", "no-cache")
	context.JSON(http.StatusCreated, models.OAuthClientRegistrationResponse{
		ClientID:                client.ClientID,
		ClientSecret:            plaintextSecret,
		ClientName:              client.ClientName,
		RedirectURIs:            request.RedirectURIs,
		GrantTypes:              grantTypes,
		ResponseTypes:           responseTypes,
		Scope:                   grantedScope,
		TokenEndpointAuthMethod: authMethod,
	})
}

// OAuthRevoke implements RFC 7009 token revocation. Only refresh tokens are
// revocable (access tokens are stateless JWTs); per the spec we respond 200
// regardless to avoid leaking token state.
func OAuthRevoke(context *gin.Context) {
	token := context.PostForm("token")
	if token != "" {
		if err := database.RevokeRefreshToken(auth.HashToken(token)); err != nil {
			logger.Log.Info("token revocation lookup failed. error: " + err.Error())
		}
	}
	context.Header("Cache-Control", "no-store")
	context.JSON(http.StatusOK, gin.H{})
}

// narrowScopeToSupported limits a requested scope string to AdvertisedScopes (never the
// reserved admin scope), defaulting to the general API scope when nothing valid is requested.
func narrowScopeToSupported(requested string) string {
	supported := map[string]bool{}
	for _, s := range models.AdvertisedScopes {
		supported[s] = true
	}
	var granted []string
	for _, s := range strings.Fields(requested) {
		if supported[s] {
			granted = append(granted, s)
		}
	}
	if len(granted) == 0 {
		return models.ScopeAPI
	}
	return strings.Join(granted, " ")
}

func base64Secret() string {
	return strings.ToLower(randstr.Hex(32))
}

// oauthClientPruneAge is how long a dynamically registered client may sit without a live
// refresh token before the daily job deletes it.
const oauthClientPruneAge = 7 * 24 * time.Hour

// PruneUnusedOAuthClients is the daily job that removes abandoned dynamic registrations.
func PruneUnusedOAuthClients() {
	removed, err := database.PruneUnusedOAuthClients(time.Now().Add(-oauthClientPruneAge))
	if err != nil {
		logger.Log.Error("failed to prune unused oauth clients. error: " + err.Error())
		return
	}
	if removed > 0 {
		logger.Log.Info("pruned unused oauth clients: " + strconv.FormatInt(removed, 10))
	}
}
