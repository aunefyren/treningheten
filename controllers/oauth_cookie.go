package controllers

import (
	"net/http"

	"github.com/aunefyren/treningheten/auth"
	"github.com/aunefyren/treningheten/files"
	"github.com/aunefyren/treningheten/middlewares"
	"github.com/aunefyren/treningheten/models"

	"github.com/gin-gonic/gin"
)

const (
	// refreshCookieName holds the first-party web client's refresh token. It is distinct
	// from the legacy JS cookie "treningheten_refresh", which the server ignores.
	refreshCookieName = "treningheten_rt"
	// refreshCookiePath keeps the cookie off every request except the OAuth endpoints.
	refreshCookiePath = "/api/oauth"
)

// refreshCookieSecure follows the same rule as the HSTS header: Secure only when the
// configured external URL is https, so plain-HTTP installs keep working.
func refreshCookieSecure() bool {
	return middlewares.IsHTTPSURL(files.ConfigFile.TreninghetenExternalURL)
}

// setRefreshCookie moves a first-party refresh token into an HttpOnly, SameSite=Strict
// cookie and returns the token set without it, ready to be sent as the response body.
// Every first-party token issuer goes through it.
func setRefreshCookie(context *gin.Context, tokenSet models.OAuthTokenResponse) models.OAuthTokenResponse {
	if tokenSet.RefreshToken != "" {
		context.SetSameSite(http.SameSiteStrictMode)
		context.SetCookie(refreshCookieName, tokenSet.RefreshToken, int(auth.RefreshTokenTTL.Seconds()), refreshCookiePath, "", refreshCookieSecure(), true)
	}
	tokenSet.RefreshToken = ""
	return tokenSet
}

// clearRefreshCookie deletes the first-party refresh cookie.
func clearRefreshCookie(context *gin.Context) {
	context.SetSameSite(http.SameSiteStrictMode)
	context.SetCookie(refreshCookieName, "", -1, refreshCookiePath, "", refreshCookieSecure(), true)
}
