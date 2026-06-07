package controllers

import (
	"net/http"
	"strings"

	"github.com/aunefyren/treningheten/files"
	"github.com/aunefyren/treningheten/models"

	"github.com/gin-gonic/gin"
)

// baseURL returns the externally reachable base URL, preferring the configured
// external URL and falling back to the request's scheme + host.
func baseURL(context *gin.Context) string {
	if external := strings.TrimRight(files.ConfigFile.TreninghetenExternalURL, "/"); external != "" {
		return external
	}
	scheme := "http"
	if context.Request.TLS != nil || strings.EqualFold(context.GetHeader("X-Forwarded-Proto"), "https") {
		scheme = "https"
	}
	return scheme + "://" + context.Request.Host
}

// OAuthAuthorizationServerMetadata serves the RFC 8414 discovery document.
func OAuthAuthorizationServerMetadata(context *gin.Context) {
	base := baseURL(context)
	context.JSON(http.StatusOK, models.AuthorizationServerMetadata{
		Issuer:                            base,
		AuthorizationEndpoint:             base + "/authorize",
		TokenEndpoint:                     base + "/api/oauth/token",
		RegistrationEndpoint:              base + "/api/oauth/register",
		RevocationEndpoint:                base + "/api/oauth/revoke",
		ScopesSupported:                   models.SupportedScopes,
		ResponseTypesSupported:            []string{"code"},
		GrantTypesSupported:               []string{"authorization_code", "refresh_token", "password"},
		TokenEndpointAuthMethodsSupported: []string{"none", "client_secret_basic", "client_secret_post"},
		CodeChallengeMethodsSupported:     []string{"S256"},
	})
}

// OAuthProtectedResourceMetadata serves the RFC 9728 discovery document used by
// MCP clients to locate the authorization server.
func OAuthProtectedResourceMetadata(context *gin.Context) {
	base := baseURL(context)
	context.JSON(http.StatusOK, models.ProtectedResourceMetadata{
		Resource:               base,
		AuthorizationServers:   []string{base},
		ScopesSupported:        models.SupportedScopes,
		BearerMethodsSupported: []string{"header"},
	})
}
