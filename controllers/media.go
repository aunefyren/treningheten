package controllers

import (
	"net/http"
	"strings"

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
	if provider != models.MediaProviderPlex && provider != models.MediaProviderSpotify && provider != models.MediaProviderAudiobookshelf {
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

// APISyncMediaForExercise is the manual per-session re-pull. It runs synchronously
// (the user is waiting on a button), re-pulls every connected provider for the
// session, and returns the refreshed exercise object with its timeline.
func APISyncMediaForExercise(context *gin.Context) {
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

	exerciseID, err := uuid.Parse(context.Param("exercise_id"))
	if err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": "Invalid exercise id."})
		context.Abort()
		return
	}

	exercise, err := database.GetExerciseByIDAndUserID(exerciseID, userID)
	if err != nil {
		logger.Log.Info("Failed to get exercise. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to get exercise."})
		context.Abort()
		return
	} else if exercise == nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": "Exercise not found."})
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

	// Pull every enabled provider (shared helper isolates one provider's failure from
	// the others). The manual button intentionally bypasses the settle guard and does
	// not flip MediaSettled — an early manual pull shouldn't consume the automatic
	// settle pass that might catch later data (see docs/media.md).
	warnings := syncMediaProvidersForExercise(user, *exercise)

	exercise, err = database.GetExerciseByIDAndUserID(exerciseID, userID)
	if err != nil {
		logger.Log.Info("Failed to reload exercise. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to reload exercise."})
		context.Abort()
		return
	}

	exerciseObject, err := ConvertExerciseToExerciseObject(*exercise)
	if err != nil {
		logger.Log.Info("Failed to convert exercise. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to convert exercise."})
		context.Abort()
		return
	}

	response := gin.H{"message": "Media synced.", "exercise": exerciseObject}
	if len(warnings) > 0 {
		response["warning"] = strings.Join(warnings, " · ")
	}
	context.JSON(http.StatusOK, response)
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
