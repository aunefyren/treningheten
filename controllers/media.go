package controllers

import (
	"net/http"

	"github.com/aunefyren/treningheten/database"
	"github.com/aunefyren/treningheten/files"
	"github.com/aunefyren/treningheten/logger"
	"github.com/aunefyren/treningheten/middlewares"
	"github.com/aunefyren/treningheten/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// mediaEnabled reports the tenant-wide media feature flag.
func mediaEnabled() bool {
	return files.ConfigFile.Media.Enabled
}

// plexEnabled reports whether Plex is usable: the tenant flag AND the provider flag
// must both be on (see docs/media.md).
func plexEnabled() bool {
	return mediaEnabled() && files.ConfigFile.Media.Plex.Enabled
}

// requirePlexEnabled aborts the request with 404 when Plex (or the whole media
// feature) is disabled, so a disabled provider is indistinguishable from "no such
// route" — matching how the MCP server behaves when turned off.
func requirePlexEnabled(context *gin.Context) bool {
	if !plexEnabled() {
		context.JSON(http.StatusNotFound, gin.H{"error": "Plex integration is not enabled."})
		context.Abort()
		return false
	}
	return true
}

// APIGetMediaConnections lists the requesting user's media connections as safe
// objects (no credentials).
func APIGetMediaConnections(context *gin.Context) {
	if !mediaEnabled() {
		context.JSON(http.StatusNotFound, gin.H{"error": "Media integration is not enabled."})
		context.Abort()
		return
	}

	userID, err := middlewares.GetAuthUsername(context.GetHeader("Authorization"))
	if err != nil {
		logger.Log.Info("Failed to get user ID. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user ID."})
		context.Abort()
		return
	}

	connections, err := database.GetMediaConnectionsForUser(userID)
	if err != nil {
		logger.Log.Info("Failed to get media connections. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get media connections."})
		context.Abort()
		return
	}

	objects := []models.MediaConnectionObject{}
	for _, connection := range connections {
		objects = append(objects, ConvertMediaConnectionToObject(connection))
	}

	context.JSON(http.StatusOK, gin.H{"message": "Media connections retrieved.", "connections": objects})
}

// APIDeleteMediaConnection disconnects a provider for the requesting user. The
// already-overlaid playback rows are left intact (historical facts, not credentials).
func APIDeleteMediaConnection(context *gin.Context) {
	if !mediaEnabled() {
		context.JSON(http.StatusNotFound, gin.H{"error": "Media integration is not enabled."})
		context.Abort()
		return
	}

	userID, err := middlewares.GetAuthUsername(context.GetHeader("Authorization"))
	if err != nil {
		logger.Log.Info("Failed to get user ID. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user ID."})
		context.Abort()
		return
	}

	provider := context.Param("provider")
	if provider != models.MediaProviderPlex && provider != models.MediaProviderSpotify {
		context.JSON(http.StatusBadRequest, gin.H{"error": "Unknown media provider."})
		context.Abort()
		return
	}

	if err := database.DeleteMediaConnectionForUserProvider(userID, provider); err != nil {
		logger.Log.Info("Failed to delete media connection. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to disconnect provider."})
		context.Abort()
		return
	}

	context.JSON(http.StatusOK, gin.H{"message": "Provider disconnected."})
}

// APISyncMediaForOperation is the manual per-activity re-pull. It runs synchronously
// (the user is waiting on a button), re-pulls every connected provider for the
// operation, and returns the refreshed operation object with its timeline.
func APISyncMediaForOperation(context *gin.Context) {
	if !mediaEnabled() {
		context.JSON(http.StatusNotFound, gin.H{"error": "Media integration is not enabled."})
		context.Abort()
		return
	}

	userID, err := middlewares.GetAuthUsername(context.GetHeader("Authorization"))
	if err != nil {
		logger.Log.Info("Failed to get user ID. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user ID."})
		context.Abort()
		return
	}

	operationID, err := uuid.Parse(context.Param("operation_id"))
	if err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": "Invalid operation id."})
		context.Abort()
		return
	}

	operation, err := database.GetOperationByIDAndUserID(operationID, userID)
	if err != nil {
		logger.Log.Info("Failed to get operation. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to get operation."})
		context.Abort()
		return
	}

	user, err := database.GetUserInformation(userID)
	if err != nil {
		logger.Log.Info("Failed to get user. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user."})
		context.Abort()
		return
	}

	if plexEnabled() {
		if err := PlexSyncOperationForUser(user, operation); err != nil {
			logger.Log.Info("Failed to sync Plex media for operation. Error: " + err.Error())
			context.JSON(http.StatusBadGateway, gin.H{"error": "Failed to sync media from Plex."})
			context.Abort()
			return
		}
	}
	if spotifyEnabled() {
		if err := SpotifySyncOperationForUser(user, operation); err != nil {
			logger.Log.Info("Failed to sync Spotify media for operation. Error: " + err.Error())
			context.JSON(http.StatusBadGateway, gin.H{"error": "Failed to sync media from Spotify."})
			context.Abort()
			return
		}
	}

	operation, err = database.GetOperationByIDAndUserID(operationID, userID)
	if err != nil {
		logger.Log.Info("Failed to reload operation. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to reload operation."})
		context.Abort()
		return
	}

	operationObject, err := ConvertOperationToOperationObject(operation)
	if err != nil {
		logger.Log.Info("Failed to convert operation. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to convert operation."})
		context.Abort()
		return
	}

	context.JSON(http.StatusOK, gin.H{"message": "Media synced.", "operation": operationObject})
}

// ConvertMediaPlaybackToObjects flattens stored MediaPlayback rows into the read
// shape attached to OperationObject. It is a pure mapping (no DB work) and drops
// the credential-free playback rows straight through; there are no sensitive
// fields on MediaPlayback, but the Object shape keeps the read layer consistent
// with the rest of the Convert*Object conventions.
func ConvertMediaPlaybackToObjects(playback []models.MediaPlayback) []models.MediaPlaybackObject {
	objects := []models.MediaPlaybackObject{}

	for _, item := range playback {
		objects = append(objects, models.MediaPlaybackObject{
			ID:             item.ID,
			Provider:       item.Provider,
			MediaType:      item.MediaType,
			Title:          item.Title,
			Artist:         item.Artist,
			Album:          item.Album,
			ProviderItemID: item.ProviderItemID,
			ArtworkURL:     item.ArtworkURL,
			StartedAt:      item.StartedAt,
			EndedAt:        item.EndedAt,
			TrackLength:    item.TrackLength,
		})
	}

	return objects
}

// ConvertMediaConnectionToObject builds the safe read shape for a connection,
// stripping the credential fields. Connected reflects whether a usable token is
// stored.
func ConvertMediaConnectionToObject(connection models.MediaConnection) models.MediaConnectionObject {
	return models.MediaConnectionObject{
		GormModel:    connection.GormModel,
		Enabled:      connection.Enabled,
		User:         connection.UserID,
		Provider:     connection.Provider,
		ServerURL:    connection.ServerURL,
		Connected:    connection.AccessToken != nil && *connection.AccessToken != "",
		LastSyncedAt: connection.LastSyncedAt,
	}
}
