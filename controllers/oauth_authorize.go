package controllers

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/aunefyren/treningheten/auth"
	"github.com/aunefyren/treningheten/database"
	"github.com/aunefyren/treningheten/logger"
	"github.com/aunefyren/treningheten/middlewares"
	"github.com/aunefyren/treningheten/models"

	"github.com/gin-gonic/gin"
	"github.com/thanhpk/randstr"
)

const authorizationCodeTTL = 10 * time.Minute

// validateRedirectURI requires an exact match against a registered redirect URI.
func validateRedirectURI(client models.OAuthClient, redirectURI string) bool {
	for _, registered := range strings.Fields(client.RedirectURIs) {
		if registered == redirectURI {
			return true
		}
	}
	return false
}

// narrowScope returns the requested scope limited to what the client is allowed,
// defaulting to the client's full scope when none is requested.
func narrowScope(client models.OAuthClient, requested string) string {
	allowed := map[string]bool{}
	for _, s := range strings.Fields(client.Scope) {
		allowed[s] = true
	}
	if strings.TrimSpace(requested) == "" {
		return client.Scope
	}
	var granted []string
	for _, s := range strings.Fields(requested) {
		if allowed[s] {
			granted = append(granted, s)
		}
	}
	return strings.Join(granted, " ")
}

// OAuthAuthorizeInfo validates an authorization request and returns details for
// the consent screen. It never redirects; redirect-based error reporting is the
// job of the decision endpoint once the redirect URI is trusted.
func OAuthAuthorizeInfo(context *gin.Context) {
	clientID := context.Query("client_id")
	redirectURI := context.Query("redirect_uri")
	responseType := context.Query("response_type")
	codeChallenge := context.Query("code_challenge")
	codeChallengeMethod := context.Query("code_challenge_method")

	client, err := database.GetOAuthClientByClientID(clientID)
	if err != nil {
		oauthError(context, http.StatusBadRequest, "invalid_request", "unknown client_id")
		return
	}
	if !validateRedirectURI(client, redirectURI) {
		oauthError(context, http.StatusBadRequest, "invalid_request", "redirect_uri is not registered for this client")
		return
	}
	if responseType != "code" {
		oauthError(context, http.StatusBadRequest, "unsupported_response_type", "only response_type=code is supported")
		return
	}
	if codeChallenge == "" || codeChallengeMethod != "S256" {
		oauthError(context, http.StatusBadRequest, "invalid_request", "PKCE with code_challenge_method=S256 is required")
		return
	}

	context.JSON(http.StatusOK, gin.H{
		"client_name":  client.ClientName,
		"client_id":    client.ClientID,
		"redirect_uri": redirectURI,
		"scope":        narrowScope(client, context.Query("scope")),
	})
}

// OAuthAuthorizeDecision records the resource owner's approval/denial and issues
// an authorization code (on approval). Requires an authenticated user.
func OAuthAuthorizeDecision(context *gin.Context) {
	userID, err := middlewares.GetAuthUsername(context.GetHeader("Authorization"))
	if err != nil {
		oauthError(context, http.StatusUnauthorized, "access_denied", "user not authenticated")
		return
	}

	clientID := context.PostForm("client_id")
	redirectURI := context.PostForm("redirect_uri")
	state := context.PostForm("state")
	requestedScope := context.PostForm("scope")
	codeChallenge := context.PostForm("code_challenge")
	codeChallengeMethod := context.PostForm("code_challenge_method")
	resource := context.PostForm("resource")
	approve := context.PostForm("approve") == "true"

	client, err := database.GetOAuthClientByClientID(clientID)
	if err != nil {
		oauthError(context, http.StatusBadRequest, "invalid_request", "unknown client_id")
		return
	}
	// Only redirect to a registered URI; otherwise this would be an open redirect.
	if !validateRedirectURI(client, redirectURI) {
		oauthError(context, http.StatusBadRequest, "invalid_request", "redirect_uri is not registered for this client")
		return
	}
	if codeChallenge == "" || codeChallengeMethod != "S256" {
		oauthError(context, http.StatusBadRequest, "invalid_request", "PKCE with code_challenge_method=S256 is required")
		return
	}

	if !approve {
		context.JSON(http.StatusOK, gin.H{"redirect": buildRedirect(redirectURI, map[string]string{
			"error": "access_denied",
			"state": state,
		})})
		return
	}

	scope := narrowScope(client, requestedScope)

	// Generate a single-use authorization code.
	code := generateOpaqueAuthCode()
	stored := models.OAuthAuthorizationCode{
		CodeHash:            auth.HashToken(code),
		ClientID:            client.ClientID,
		UserID:              userID,
		RedirectURI:         redirectURI,
		Scope:               scope,
		CodeChallenge:       codeChallenge,
		CodeChallengeMethod: codeChallengeMethod,
		Resource:            resource,
		ExpiresAt:           time.Now().Add(authorizationCodeTTL),
	}
	if err := database.CreateAuthorizationCode(&stored); err != nil {
		logger.Log.Error("failed to store authorization code. error: " + err.Error())
		oauthError(context, http.StatusInternalServerError, "server_error", "failed to issue authorization code")
		return
	}

	context.JSON(http.StatusOK, gin.H{"redirect": buildRedirect(redirectURI, map[string]string{
		"code":  code,
		"state": state,
	})})
}

// oauthTokenAuthorizationCodeGrant exchanges an authorization code (+PKCE
// verifier) for tokens.
func oauthTokenAuthorizationCodeGrant(context *gin.Context) {
	client, err := resolveClient(context)
	if err != nil {
		oauthError(context, http.StatusUnauthorized, "invalid_client", "unknown or unauthorized client")
		return
	}

	code := context.PostForm("code")
	redirectURI := context.PostForm("redirect_uri")
	codeVerifier := context.PostForm("code_verifier")
	if code == "" || codeVerifier == "" {
		oauthError(context, http.StatusBadRequest, "invalid_request", "code and code_verifier are required")
		return
	}

	stored, err := database.GetAuthorizationCodeByHash(auth.HashToken(code))
	if err != nil {
		oauthError(context, http.StatusBadRequest, "invalid_grant", "invalid authorization code")
		return
	}
	if stored.ClientID != client.ClientID {
		oauthError(context, http.StatusBadRequest, "invalid_grant", "authorization code was issued to a different client")
		return
	}
	if stored.RedirectURI != redirectURI {
		oauthError(context, http.StatusBadRequest, "invalid_grant", "redirect_uri mismatch")
		return
	}
	if stored.ConsumedAt != nil || stored.ExpiresAt.Before(time.Now()) {
		oauthError(context, http.StatusBadRequest, "invalid_grant", "authorization code expired or already used")
		return
	}
	if !verifyPKCE(stored.CodeChallenge, codeVerifier) {
		oauthError(context, http.StatusBadRequest, "invalid_grant", "PKCE verification failed")
		return
	}

	// Enforce single-use atomically before issuing tokens.
	if err := database.ConsumeAuthorizationCode(stored.ID); err != nil {
		oauthError(context, http.StatusBadRequest, "invalid_grant", "authorization code already used")
		return
	}

	userObject, err := database.GetUserInformation(stored.UserID)
	if err != nil {
		oauthError(context, http.StatusInternalServerError, "server_error", "failed to load user")
		return
	}
	admin := userObject.Admin != nil && *userObject.Admin && strings.Contains(stored.Scope, models.ScopeAdmin)

	tokenSet, err := auth.IssueTokenSet(stored.UserID, admin, stored.Scope, client.ClientID)
	if err != nil {
		oauthError(context, http.StatusInternalServerError, "server_error", "failed to issue tokens")
		return
	}

	writeTokenResponse(context, tokenSet)
}

// verifyPKCE checks an S256 code_challenge against the presented verifier.
func verifyPKCE(codeChallenge string, codeVerifier string) bool {
	sum := sha256.Sum256([]byte(codeVerifier))
	expected := base64.RawURLEncoding.EncodeToString(sum[:])
	return subtle.ConstantTimeCompare([]byte(expected), []byte(codeChallenge)) == 1
}

func generateOpaqueAuthCode() string {
	return base64.RawURLEncoding.EncodeToString([]byte(randstr.String(48)))
}

// buildRedirect appends query parameters to a redirect URI.
func buildRedirect(redirectURI string, params map[string]string) string {
	u, err := url.Parse(redirectURI)
	if err != nil {
		return redirectURI
	}
	q := u.Query()
	for k, v := range params {
		if v != "" {
			q.Set(k, v)
		}
	}
	u.RawQuery = q.Encode()
	return u.String()
}
