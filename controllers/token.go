package controllers

import (
	"net/http"

	"github.com/aunefyren/treningheten/files"
	"github.com/aunefyren/treningheten/logger"
	"github.com/aunefyren/treningheten/middlewares"

	"github.com/gin-gonic/gin"
)

// ValidateToken returns session metadata for the current access token. Token
// renewal is handled by the OAuth refresh-token grant, not here.
func ValidateToken(context *gin.Context) {
	claims, err := middlewares.GetTokenClaims(context.GetHeader("Authorization"))
	if err != nil {
		logger.Log.Info("Failed to validate session. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Invalid session. Please log in again."})
		context.Abort()
		return
	}

	// Get public VAPID key
	VAPIDSettings, err := GetVAPIDSettings()
	if err != nil {
		logger.Log.Info("Failed to get public VAPID key from config. Error: " + err.Error())
	}

	context.JSON(http.StatusOK, gin.H{
		"message":              "Valid session!",
		"data":                 claims,
		"vapid_public_key":     VAPIDSettings.VAPIDPublicKey,
		"strava_enabled":       files.ConfigFile.StravaEnabled,
		"strava_client_id":     files.ConfigFile.StravaClientID,
		"strava_redirect_uri":  files.ConfigFile.StravaRedirectURI,
		"hevy_enabled":         files.ConfigFile.HevyEnabled,
		"media_enabled":        files.ConfigFile.Media.Enabled,
		"plex_enabled":         files.ConfigFile.Media.Enabled && files.ConfigFile.Media.Plex.Enabled,
		"spotify_enabled":      files.ConfigFile.Media.Enabled && files.ConfigFile.Media.Spotify.Enabled,
		"spotify_client_id":    files.ConfigFile.Media.Spotify.ClientID,
		"spotify_redirect_uri": files.ConfigFile.Media.Spotify.RedirectURI,
	})
}
