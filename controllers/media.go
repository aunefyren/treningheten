package controllers

import (
	"net/http"
	"net/url"
	"slices"
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

// APISyncMediaForUsers is the admin bulk media re-sync. It mirrors the Strava
// sync-activities-for-users endpoint: an optional user_ids filter (empty = every user
// with a media connection) and an optional exercise_ids filter (empty = all of each
// user's sessions). It runs in the background and force re-pulls every enabled provider
// for each matched session (bypassing the per-session pull guard), so a stale or
// newly-connected provider's history is re-matched over the whole history.
func APISyncMediaForUsers(context *gin.Context) {
	if !mediaEnabled() {
		context.JSON(http.StatusBadRequest, gin.H{"error": "Media integration is not enabled."})
		context.Abort()
		return
	}

	var syncRequest models.MediaSyncForUsersRequest
	if err := context.ShouldBindJSON(&syncRequest); err != nil {
		logger.Log.Error("failed to parse request. error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "failed to parse request"})
		context.Abort()
		return
	}

	usersToSync := []models.User{}
	if len(syncRequest.UserIDs) > 0 {
		for _, userID := range syncRequest.UserIDs {
			parsedID, err := uuid.Parse(userID)
			if err != nil {
				context.JSON(http.StatusBadRequest, gin.H{"error": "failed to parse user ID"})
				context.Abort()
				return
			}

			user, err := database.GetAllUserInformation(parsedID)
			if err != nil {
				context.JSON(http.StatusNotFound, gin.H{"error": "failed to find user by ID"})
				context.Abort()
				return
			}

			connections, err := database.GetMediaConnectionsForUser(user.ID)
			if err != nil {
				context.JSON(http.StatusInternalServerError, gin.H{"error": "failed to check media connections for " + user.ID.String()})
				context.Abort()
				return
			}

			if user.Enabled && len(connections) > 0 {
				usersToSync = append(usersToSync, user)
			} else {
				context.JSON(http.StatusBadRequest, gin.H{"error": "user has no media connection to sync " + user.ID.String()})
				context.Abort()
				return
			}
		}
	} else {
		userIDs, err := database.GetUserIDsWithMediaConnections()
		if err != nil {
			logger.Log.Info("Failed to list users with media connections. Error: " + err.Error())
			context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list users with media connections."})
			context.Abort()
			return
		}

		for _, userID := range userIDs {
			user, err := database.GetAllUserInformation(userID)
			if err != nil {
				logger.Log.Warn("Media sync could not load user. Error: " + err.Error())
				continue
			}
			if user.Enabled {
				usersToSync = append(usersToSync, user)
			}
		}
	}

	go MediaSyncForUsers(usersToSync, syncRequest.ExerciseIDs)

	context.JSON(http.StatusAccepted, gin.H{"message": "Media sync started!"})
}

// MediaSyncForUsers force re-pulls every enabled provider for each matched session of
// each user. An empty exerciseIDs re-syncs all of a user's sessions; otherwise only the
// listed ones. It calls syncMediaProvidersForExercise directly, which re-pulls
// regardless of the MediaRetrievedAt guard (the delete-and-replace primitive is
// idempotent and no-ops on an empty pull, so re-running is safe).
func MediaSyncForUsers(usersToSync []models.User, exerciseIDs []string) {
	for _, user := range usersToSync {
		exercises, err := database.GetAllExercisesForMediaSync(user.ID)
		if err != nil {
			logger.Log.Info("Failed to get user sessions for media sync. Error: " + err.Error())
			continue
		}

		synced := 0
		for _, exercise := range exercises {
			if len(exerciseIDs) > 0 && !slices.Contains(exerciseIDs, exercise.ID.String()) {
				continue
			}
			syncMediaProvidersForExercise(user, exercise)
			synced++
		}

		logger.Log.Infof("media re-synced %d sessions for user '%s %s'", synced, user.FirstName, user.LastName)
	}

	logger.Log.Infof("Media sync finished for %d users", len(usersToSync))
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
			ArtworkURL:     resolveMediaArtworkURL(item.Provider, item.ArtworkURL),
			StartedAt:      item.StartedAt,
			EndedAt:        item.EndedAt,
			TrackLength:    item.TrackLength,
		})
	}

	return objects
}

// resolveMediaArtworkURL turns a stored artwork reference into a URL the browser can
// load. Spotify (and other public-URL providers) pass through unchanged; a Plex row
// stores a PMS-relative thumb path (which needs the server token to fetch), so it is
// rewritten to the authenticated artwork proxy rather than exposing the credential.
func resolveMediaArtworkURL(provider string, artwork *string) *string {
	if artwork == nil || *artwork == "" {
		return artwork
	}
	if provider == models.MediaProviderPlex && strings.HasPrefix(*artwork, "/library/") {
		proxied := "/api/auth/media/plex/artwork?path=" + url.QueryEscape(*artwork)
		return &proxied
	}
	return artwork
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
