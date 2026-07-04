package controllers

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/aunefyren/treningheten/database"
	"github.com/aunefyren/treningheten/files"
	"github.com/aunefyren/treningheten/logger"
	"github.com/aunefyren/treningheten/middlewares"
	"github.com/aunefyren/treningheten/models"
	"github.com/aunefyren/treningheten/utilities"

	"github.com/gin-gonic/gin"
)

// absListeningSessionsPageSize bounds the history page pulled per sync. Sessions are
// returned most-recent first, so one page comfortably covers any recent workout
// window (ABS history is durable, unlike Spotify's ~24h — no urgency to page deeper).
const absListeningSessionsPageSize = 100

// absEnabled reports whether Audiobookshelf is usable: the tenant media flag AND the
// provider flag must both be on. Unlike Plex/Spotify there are no app-level
// credentials to require — the per-user server URL + token is entered at connect time.
func absEnabled() bool {
	return mediaEnabled() && files.ConfigFile.Media.Audiobookshelf.Enabled
}

// requireABSEnabled aborts with 404 when Audiobookshelf (or the whole media feature)
// is disabled, so a disabled provider is indistinguishable from "no such route"
// (matching Plex/MCP).
func requireABSEnabled(context *gin.Context) bool {
	if !absEnabled() {
		context.JSON(http.StatusNotFound, gin.H{"error": "Audiobookshelf integration is not enabled."})
		context.Abort()
		return false
	}
	return true
}

// absRequest performs a bearer-authenticated GET against an Audiobookshelf server and
// returns the body + status. ABS is self-hosted behind the user's own TLS (normal
// certs, unlike Plex's plex.direct self-signed hosts), so default verification is used.
func absRequest(serverURL, path, token string) ([]byte, int, error) {
	rawURL := strings.TrimRight(serverURL, "/") + path
	req, err := http.NewRequest("GET", rawURL, nil)
	if err != nil {
		return nil, 0, errors.New("Audiobookshelf request generation threw error.")
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		logger.Log.Error("Audiobookshelf request threw error. Error: " + err.Error())
		return nil, 0, errors.New("Audiobookshelf request threw error.")
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, errors.New("Failed to read Audiobookshelf reply body.")
	}
	return body, resp.StatusCode, nil
}

// APIAudiobookshelfConnect validates a server URL + API token by calling GET /api/me
// and, on success, upserts the encrypted connection (storing the ABS user id for
// reference). History is pulled from the user-scoped /api/me endpoint, so no
// server-local account resolution is needed (unlike Plex).
func APIAudiobookshelfConnect(context *gin.Context) {
	if !requireABSEnabled(context) {
		return
	}

	userID, err := middlewares.GetAuthUsername(context.GetHeader("Authorization"))
	if err != nil {
		logger.Log.Info("Failed to get user ID. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user ID."})
		context.Abort()
		return
	}

	var request models.AudiobookshelfConnectRequest
	if err := context.ShouldBindJSON(&request); err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse request."})
		context.Abort()
		return
	}

	serverURL := strings.TrimRight(strings.TrimSpace(request.ServerURL), "/")
	token := strings.TrimSpace(request.Token)
	parsed, parseErr := url.Parse(serverURL)
	if parseErr != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		context.JSON(http.StatusBadRequest, gin.H{"error": "Enter a full server URL, e.g. https://abs.example.com"})
		context.Abort()
		return
	}
	if token == "" {
		context.JSON(http.StatusBadRequest, gin.H{"error": "Enter your Audiobookshelf API token."})
		context.Abort()
		return
	}

	body, status, err := absRequest(serverURL, "/api/me", token)
	if err != nil {
		context.JSON(http.StatusBadGateway, gin.H{"error": "Could not reach the Audiobookshelf server. Double-check the URL."})
		context.Abort()
		return
	}
	if status == http.StatusUnauthorized || status == http.StatusForbidden {
		context.JSON(http.StatusBadRequest, gin.H{"error": "Audiobookshelf rejected the token. Copy a fresh API token from your ABS account settings."})
		context.Abort()
		return
	}
	if status != http.StatusOK {
		logger.Log.Error("Audiobookshelf /api/me returned non-200. Status: " + strconv.Itoa(status))
		context.JSON(http.StatusBadGateway, gin.H{"error": "Audiobookshelf returned an unexpected response."})
		context.Abort()
		return
	}

	absUser := models.AudiobookshelfUser{}
	if err := json.Unmarshal(body, &absUser); err != nil || absUser.ID == "" {
		context.JSON(http.StatusBadGateway, gin.H{"error": "Failed to read the Audiobookshelf account."})
		context.Abort()
		return
	}

	accountID := absUser.ID
	connection, err := upsertMediaConnection(userID, models.MediaProviderAudiobookshelf, token, &serverURL, &accountID)
	if err != nil {
		logger.Log.Info("Failed to store Audiobookshelf connection. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to store Audiobookshelf connection."})
		context.Abort()
		return
	}

	object := ConvertMediaConnectionToObject(connection)
	context.JSON(http.StatusOK, gin.H{"message": "Audiobookshelf connected.", "connection": object})
}

// absClassifyMediaType maps an ABS session mediaType to Treningheten's vocabulary.
// ABS is the first provider that natively distinguishes the two spoken types, so this
// activates the typed rail nodes + "minutes listened" metric already in the frontend.
func absClassifyMediaType(absType string) string {
	switch strings.ToLower(strings.TrimSpace(absType)) {
	case "podcast":
		return models.MediaTypePodcast
	default:
		// "book" and anything else spoken read as an audiobook.
		return models.MediaTypeAudiobook
	}
}

// absFetchListeningSessions pulls the most recent listening sessions for the token's
// user (the /api/me endpoint is inherently user-scoped — no privacy filtering needed).
func absFetchListeningSessions(serverURL, token string) ([]models.AudiobookshelfListenSession, error) {
	path := "/api/me/listening-sessions?itemsPerPage=" + strconv.Itoa(absListeningSessionsPageSize) + "&page=0"
	body, status, err := absRequest(serverURL, path, token)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		logger.Log.Error("Audiobookshelf listening-sessions returned non-200. Status: " + strconv.Itoa(status))
		return nil, errors.New("Audiobookshelf history returned non-200 status.")
	}

	response := models.AudiobookshelfListeningSessionsResponse{}
	if err := json.Unmarshal(body, &response); err != nil {
		logger.Log.Error("Failed to parse Audiobookshelf history. Error: " + err.Error())
		return nil, errors.New("Failed to parse Audiobookshelf history.")
	}
	return response.Sessions, nil
}

// buildAudiobookshelfPlaybackForWindow maps ABS listening sessions into provider-
// neutral play events and defers window matching to the shared playbackForWindow. A
// session is coarser than a scrobble (one continuous listen), matched by its start
// time; TimeListening (seconds actually listened) is the rail span.
func buildAudiobookshelfPlaybackForWindow(sessions []models.AudiobookshelfListenSession, start, end time.Time) []models.MediaPlayback {
	events := []mediaPlayEvent{}

	for _, session := range sessions {
		if session.StartedAt <= 0 {
			continue
		}

		var lengthSeconds int64
		if session.TimeListening > 0 {
			lengthSeconds = int64(session.TimeListening)
		}

		events = append(events, mediaPlayEvent{
			mediaType:      absClassifyMediaType(session.MediaType),
			title:          session.DisplayTitle,
			artist:         session.DisplayAuthor,
			providerItemID: session.LibraryItemID,
			startedAt:      time.UnixMilli(session.StartedAt).UTC(),
			trackLengthSec: lengthSeconds,
		})
	}

	return playbackForWindow(events, start, end)
}

// AudiobookshelfSyncExerciseForUser pulls the ABS listening history overlapping a
// session's window and stores it (delete-and-replace per provider), stamping the pull
// guard regardless of outcome — mirroring PlexSyncExerciseForUser/SpotifySync.
func AudiobookshelfSyncExerciseForUser(user models.User, exercise models.Exercise) error {
	connection, err := database.GetMediaConnectionForUserProvider(user.ID, models.MediaProviderAudiobookshelf)
	if err != nil {
		return err
	}
	if connection == nil || connection.AccessToken == nil || connection.ServerURL == nil || *connection.ServerURL == "" {
		logger.Log.Trace("No usable Audiobookshelf connection for media sync; stamping guard only.")
		return database.SetExerciseMediaRetrievedAt(exercise.ID, time.Now())
	}

	token, err := utilities.DecryptString(*connection.AccessToken, files.ConfigFile.Media.TokenKey)
	if err != nil {
		return errors.New("failed to decrypt Audiobookshelf token: " + err.Error())
	}

	start, end, ok := resolveSessionWindow(exercise, sessionFallbackSeconds(exercise))
	if !ok {
		logger.Log.Trace("Session has no trustworthy time for media match; stamping guard only.")
		return database.SetExerciseMediaRetrievedAt(exercise.ID, time.Now())
	}

	sessions, err := absFetchListeningSessions(*connection.ServerURL, token)
	if err != nil {
		return err
	}

	playback := buildAudiobookshelfPlaybackForWindow(sessions, start, end)

	if err := database.ReplaceMediaPlaybackForExerciseProvider(exercise.ID, models.MediaProviderAudiobookshelf, playback); err != nil {
		return err
	}
	if err := database.SetExerciseMediaRetrievedAt(exercise.ID, time.Now()); err != nil {
		return err
	}

	now := time.Now()
	connection.LastSyncedAt = &now
	if _, err := database.UpdateMediaConnectionInDB(*connection); err != nil {
		logger.Log.Warn("Failed to update Audiobookshelf connection last-synced time. Error: " + err.Error())
	}

	logger.Log.Info("Synced " + strconv.Itoa(len(playback)) + " Audiobookshelf playback rows for session " + exercise.ID.String())
	return nil
}
