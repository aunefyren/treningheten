package controllers

import (
	"net/http"
	"strings"

	"github.com/aunefyren/treningheten/auth"
	"github.com/aunefyren/treningheten/database"
	"github.com/aunefyren/treningheten/logger"
	"github.com/aunefyren/treningheten/models"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

// oauthError writes an RFC 6749 §5.2 error response.
func oauthError(context *gin.Context, status int, code string, description string) {
	context.Header("Cache-Control", "no-store")
	context.Header("Pragma", "no-cache")
	context.JSON(status, models.OAuthError{ErrorCode: code, ErrorDescription: description})
	context.Abort()
}

// resolveClient identifies and authenticates the calling client from the
// request (form body or HTTP Basic). A public client needs no secret.
func resolveClient(context *gin.Context) (models.OAuthClient, error) {
	clientID := context.PostForm("client_id")
	clientSecret := context.PostForm("client_secret")

	// HTTP Basic credentials take precedence (client_secret_basic).
	if basicID, basicSecret, ok := context.Request.BasicAuth(); ok {
		clientID = basicID
		clientSecret = basicSecret
	}

	if clientID == "" {
		return models.OAuthClient{}, errInvalidClient
	}

	client, err := database.GetOAuthClientByClientID(clientID)
	if err != nil {
		return models.OAuthClient{}, errInvalidClient
	}

	// Confidential clients must present a valid secret.
	if !client.Public {
		if client.ClientSecretHash == nil {
			return models.OAuthClient{}, errInvalidClient
		}
		if bcrypt.CompareHashAndPassword([]byte(*client.ClientSecretHash), []byte(clientSecret)) != nil {
			return models.OAuthClient{}, errInvalidClient
		}
	}

	return client, nil
}

var errInvalidClient = &clientError{}

type clientError struct{}

func (e *clientError) Error() string { return "invalid_client" }

// OAuthToken is the RFC 6749 token endpoint. Accepts form-encoded requests.
func OAuthToken(context *gin.Context) {
	grantType := context.PostForm("grant_type")

	switch grantType {
	case "password":
		oauthTokenPasswordGrant(context)
	case "refresh_token":
		oauthTokenRefreshGrant(context)
	case "authorization_code":
		oauthTokenAuthorizationCodeGrant(context)
	case "":
		oauthError(context, http.StatusBadRequest, "invalid_request", "grant_type is required")
	default:
		oauthError(context, http.StatusBadRequest, "unsupported_grant_type", "unsupported grant_type: "+grantType)
	}
}

// oauthTokenPasswordGrant handles the first-party resource-owner password grant.
func oauthTokenPasswordGrant(context *gin.Context) {
	client, err := resolveClient(context)
	if err != nil {
		oauthError(context, http.StatusUnauthorized, "invalid_client", "unknown or unauthorized client")
		return
	}

	// ROPC is only permitted for the trusted first-party client.
	if !client.FirstParty {
		oauthError(context, http.StatusBadRequest, "unauthorized_client", "client may not use the password grant")
		return
	}

	username := strings.TrimSpace(strings.ToLower(context.PostForm("username")))
	password := context.PostForm("password")
	if username == "" || password == "" {
		oauthError(context, http.StatusBadRequest, "invalid_request", "username and password are required")
		return
	}

	user, err := database.GetAllUserInformationByEmail(username)
	if err != nil {
		oauthError(context, http.StatusBadRequest, "invalid_grant", "invalid credentials")
		return
	}
	if user.CheckPassword(password) != nil {
		oauthError(context, http.StatusBadRequest, "invalid_grant", "invalid credentials")
		return
	}

	admin := user.Admin != nil && *user.Admin
	tokenSet, err := auth.IssueTokenSet(user.ID, admin, auth.ScopeForUser(admin), client.ClientID)
	if err != nil {
		logger.Log.Error("failed to issue token set. error: " + err.Error())
		oauthError(context, http.StatusInternalServerError, "server_error", "failed to issue tokens")
		return
	}

	writeTokenResponse(context, tokenSet)
}

// oauthTokenRefreshGrant handles the refresh_token grant with rotation.
func oauthTokenRefreshGrant(context *gin.Context) {
	client, err := resolveClient(context)
	if err != nil {
		oauthError(context, http.StatusUnauthorized, "invalid_client", "unknown or unauthorized client")
		return
	}

	refreshToken := context.PostForm("refresh_token")
	if refreshToken == "" {
		oauthError(context, http.StatusBadRequest, "invalid_request", "refresh_token is required")
		return
	}

	tokenSet, err := auth.RefreshTokenSet(refreshToken, client.ClientID)
	if err != nil {
		oauthError(context, http.StatusBadRequest, "invalid_grant", err.Error())
		return
	}

	writeTokenResponse(context, tokenSet)
}

// oauthTokenAuthorizationCodeGrant is implemented alongside the authorization
// endpoint (see oauth_authorize.go).

func writeTokenResponse(context *gin.Context, tokenSet models.OAuthTokenResponse) {
	context.Header("Cache-Control", "no-store")
	context.Header("Pragma", "no-cache")
	context.JSON(http.StatusOK, tokenSet)
}
