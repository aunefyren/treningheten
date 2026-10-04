package controllers

import (
	"encoding/json"
	"errors"
	"io"
	"math"
	"net/http"
	"sort"
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

// absListeningSessionsPageSize is the history page size requested per call. Sessions
// come back most-recent first, so a recent workout sits on the first page — but ABS
// opens a fresh session on every device switch or unclean close, so an older one can
// be several pages back (see absFetchListeningSessions).
const absListeningSessionsPageSize = 100

// absMaxListeningSessionPages caps how far back one sync pages, so a session months
// old (or a server ignoring the paging parameters) can't turn one sync into an
// unbounded crawl of the user's history.
const absMaxListeningSessionPages = 20

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
// The server URL is user-supplied, so the connection goes through the media destination
// policy (see media_dial.go) — a blocked address fails to dial.
func absRequest(serverURL, path, token string) ([]byte, int, error) {
	rawURL := strings.TrimRight(serverURL, "/") + path
	req, err := http.NewRequest("GET", rawURL, nil)
	if err != nil {
		return nil, 0, errors.New("Audiobookshelf request generation threw error.")
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := mediaHTTPClient(20*time.Second, nil).Do(req)
	if err != nil {
		logger.Log.Error("Audiobookshelf request threw error. Error: " + err.Error())
		return nil, 0, integrationUnavailableError("Audiobookshelf request threw error.")
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

	token := strings.TrimSpace(request.Token)
	serverURL, urlErr := validateMediaServerURL(request.ServerURL)
	if urlErr != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": urlErr.Error()})
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

	// A reconnect replaces a dead token: clear the broken status and re-pull the gap.
	resumeIntegrationAfterReconnect(userID, models.IntegrationProviderAudiobookshelf)

	object := ConvertMediaConnectionToObject(connection)
	context.JSON(http.StatusOK, gin.H{"message": "Audiobookshelf connected.", "connection": object})
}

// absStatusError maps an ABS reply status onto an integration error: 401/403 is a
// rejected token (revoked, or a newer ABS expiring it), anything else that isn't a 200
// means the server isn't answering properly. nil for a 200.
func absStatusError(what string, statusCode int) error {
	switch {
	case statusCode == http.StatusOK:
		return nil
	case statusCode == http.StatusUnauthorized || statusCode == http.StatusForbidden:
		logger.Log.Error(what + " rejected the Audiobookshelf token. Status: " + strconv.Itoa(statusCode))
		return integrationAuthError(what + " rejected the Audiobookshelf token.")
	default:
		logger.Log.Error(what + " returned non-200. Status: " + strconv.Itoa(statusCode))
		return integrationUnavailableError(what + " returned non-200 status.")
	}
}

// absCheckConnection is Audiobookshelf's health check: one call to /api/me with the
// stored token.
func absCheckConnection(connection *models.MediaConnection) error {
	if connection.ServerURL == nil || *connection.ServerURL == "" {
		return nil
	}

	token, err := utilities.DecryptString(*connection.AccessToken, files.ConfigFile.Media.TokenKey)
	if err != nil {
		return errors.New("Failed to decrypt Audiobookshelf token. Error: " + err.Error())
	}

	_, status, err := absRequest(*connection.ServerURL, "/api/me", token)
	if err != nil {
		return err
	}
	return absStatusError("Audiobookshelf health check", status)
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

// absFetchListeningSessions pulls the token user's listening sessions back to `since`
// (the /api/me endpoint is inherently user-scoped — no privacy filtering needed).
// Sessions come most-recently-active first, so it pages until a page reaches a session
// last active before `since` — nothing further back can overlap the window — or the
// history runs out, or absMaxListeningSessionPages is hit. Stopping after the first
// page instead made a re-pull of an older workout silently match nothing, and the
// non-destructive empty guard then kept its stale rows.
func absFetchListeningSessions(serverURL, token string, since time.Time) ([]models.AudiobookshelfListenSession, error) {
	sessions := []models.AudiobookshelfListenSession{}

	for page := 0; page < absMaxListeningSessionPages; page++ {
		response, err := absFetchListeningSessionsPage(serverURL, token, page)
		if err != nil {
			return nil, err
		}
		sessions = append(sessions, response.Sessions...)

		if !absHasOlderListeningSessions(response, page, since) {
			break
		}
	}

	return sessions, nil
}

// absFetchListeningSessionsPage fetches one page of the listening-session history.
func absFetchListeningSessionsPage(serverURL, token string, page int) (models.AudiobookshelfListeningSessionsResponse, error) {
	response := models.AudiobookshelfListeningSessionsResponse{}

	path := "/api/me/listening-sessions?itemsPerPage=" + strconv.Itoa(absListeningSessionsPageSize) + "&page=" + strconv.Itoa(page)
	body, status, err := absRequest(serverURL, path, token)
	if err != nil {
		return response, err
	}
	if err := absStatusError("Audiobookshelf history", status); err != nil {
		return response, err
	}

	if err := json.Unmarshal(body, &response); err != nil {
		logger.Log.Error("Failed to parse Audiobookshelf history. Error: " + err.Error())
		return response, integrationUnavailableError("Failed to parse Audiobookshelf history.")
	}
	return response, nil
}

// absHasOlderListeningSessions reports whether the page after `page` is worth
// fetching: the page was full, the server doesn't say it was the last one, and every
// session on it was still active at or after `since`. A short page means the history
// ran out even when the server leaves numPages unset.
func absHasOlderListeningSessions(response models.AudiobookshelfListeningSessionsResponse, page int, since time.Time) bool {
	if len(response.Sessions) < absListeningSessionsPageSize {
		return false
	}
	if response.NumPages > 0 && page+1 >= response.NumPages {
		return false
	}

	sinceMs := since.UnixMilli()
	for _, session := range response.Sessions {
		lastActive := session.UpdatedAt
		if session.StartedAt > lastActive {
			lastActive = session.StartedAt
		}
		if lastActive < sinceMs {
			return false
		}
	}
	return true
}

// absContinuousSlack is how much longer than its listened time a session may span and
// still count as one continuous listen, so its UpdatedAt is the real end.
const absContinuousSlack = 2 * time.Minute

// absResumeHandoff is how soon after a session's UpdatedAt the next session of the same
// item must start for the old one to count as closed on resume (see absCoverageEnd).
const absResumeHandoff = time.Minute

// buildAudiobookshelfPlaybackForWindow maps ABS listening sessions into provider-
// neutral play events and defers window matching to the shared playbackForWindow. A
// session is coarser than a scrobble (one continuous listen, or one with pauses in it);
// absCoverageEnd decides which stretch of [startedAt, updatedAt] was listening, and the
// playback positions let a listen resumed in a fresh session merge with the one before.
func buildAudiobookshelfPlaybackForWindow(sessions []models.AudiobookshelfListenSession, start, end time.Time) []models.MediaPlayback {
	ordered := make([]models.AudiobookshelfListenSession, 0, len(sessions))
	for _, session := range sessions {
		if session.StartedAt > 0 {
			ordered = append(ordered, session)
		}
	}
	sort.SliceStable(ordered, func(i, j int) bool { return ordered[i].StartedAt < ordered[j].StartedAt })

	// The session that follows each one on the same item, for the closed-on-resume check.
	nextOfItem := make([]*models.AudiobookshelfListenSession, len(ordered))
	lastOfItem := map[string]int{}
	for i, session := range ordered {
		item := absSessionItemID(session)
		if previous, ok := lastOfItem[item]; ok {
			nextOfItem[previous] = &ordered[i]
		}
		lastOfItem[item] = i
	}

	events := []mediaPlayEvent{}
	for i, session := range ordered {
		listened := absListenedSeconds(session)

		events = append(events, mediaPlayEvent{
			mediaType: absClassifyMediaType(session.MediaType),
			title:     session.DisplayTitle,
			artist:    session.DisplayAuthor,
			// For a podcast the library item is the *show*, so the episode id is the
			// identity; books have no episode and are identified by the item itself.
			// Without this every episode of a series shares one id and the overlap
			// merge would collapse unrelated episodes into each other.
			providerItemID:    absSessionItemID(session),
			providerParentID:  session.LibraryItemID,
			providerSessionID: session.ID,
			startedAt:         time.UnixMilli(session.StartedAt).UTC(),
			coverageEnd:       absCoverageEnd(session, listened, nextOfItem[i]),
			trackLengthSec:    listened,
			// ABS reports time actually listened, not the item's length — so merged
			// sessions on one episode add up rather than taking the longest.
			trackLengthIsListened: true,
			hasPosition:           session.StartTime > 0 || session.CurrentTime > 0,
			positionStart:         session.StartTime,
			positionEnd:           session.CurrentTime,
		})
	}

	return playbackForWindow(events, start, end)
}

// absSessionItemID is the most specific thing a session played: the episode, else the
// library item (a book).
func absSessionItemID(session models.AudiobookshelfListenSession) string {
	return firstNonEmpty(session.EpisodeID, session.LibraryItemID)
}

// absListenedSeconds is the time a session actually listened: TimeListening, or — when
// the client sent it as null — how far the playback position moved. Either way it is
// capped at the session's wall-clock span, since no one listens longer than that.
func absListenedSeconds(session models.AudiobookshelfListenSession) int64 {
	listened := session.TimeListening
	if listened <= 0 {
		listened = session.CurrentTime - session.StartTime
	}
	if listened <= 0 {
		return 0
	}
	if session.UpdatedAt > session.StartedAt {
		listened = math.Min(listened, float64(session.UpdatedAt-session.StartedAt)/1000)
	}
	return int64(math.Round(listened))
}

// absCoverageEnd is the end of the stretch of a session that was listening. UpdatedAt is
// only when the session was last touched, so it is the end just when the session was one
// continuous listen (its span barely exceeds the listened time). Otherwise the session
// holds a pause, and where it fell is only known in one case: some clients (AudioBooth)
// close a paused session when playback resumes, so when the next session of the item
// starts right after UpdatedAt the pause was at the end and listening was
// [startedAt, startedAt + listened]. Taking UpdatedAt there put a podcast paused the
// night before into the next morning's workout. In any other case the pause could be
// anywhere (the ABS iOS app keeps it inside the session), so UpdatedAt stays the end and
// the listened time is spread over the span. Zero = unknown.
func absCoverageEnd(session models.AudiobookshelfListenSession, listenedSec int64, next *models.AudiobookshelfListenSession) time.Time {
	startedAt := time.UnixMilli(session.StartedAt).UTC()
	listenedEnd := startedAt.Add(time.Duration(listenedSec) * time.Second)

	if session.UpdatedAt <= session.StartedAt {
		if listenedSec > 0 {
			return listenedEnd
		}
		return time.Time{}
	}
	updatedAt := time.UnixMilli(session.UpdatedAt).UTC()

	if listenedSec > 0 && updatedAt.Sub(listenedEnd) > absContinuousSlack && next != nil {
		handoff := time.UnixMilli(next.StartedAt).Sub(updatedAt)
		if handoff >= -absResumeHandoff && handoff <= absResumeHandoff {
			return listenedEnd
		}
	}
	return updatedAt
}

// MatchAudiobookshelfSessions exposes the ABS window matcher to developer tooling
// (scripts/mediaprobe), so real history can be replayed through exactly the logic a
// sync runs without a database or a stored connection.
func MatchAudiobookshelfSessions(sessions []models.AudiobookshelfListenSession, start, end time.Time) []models.MediaPlayback {
	return buildAudiobookshelfPlaybackForWindow(sessions, start, end)
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

	sessions, err := absFetchListeningSessions(*connection.ServerURL, token, start.Add(-mediaMatchGrace))
	recordIntegrationOutcome(user.ID, models.IntegrationProviderAudiobookshelf, err)
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
