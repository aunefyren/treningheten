package controllers

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/aunefyren/treningheten/database"
	"github.com/aunefyren/treningheten/files"
	"github.com/aunefyren/treningheten/logger"
	"github.com/aunefyren/treningheten/models"
	"github.com/aunefyren/treningheten/utilities"

	"github.com/google/uuid"
)

// defaultActivityWindow is used when an activity has no recorded duration, so a
// timeless manual log still captures a plausible listening window.
const defaultActivityWindow = 3 * time.Hour

// isPlexAudioListen reports whether a Plex history item is actual audio listening.
// Only "track" qualifies — music, audiobooks (Plex stores them in music libraries),
// and audio podcasts all surface as tracks. Video plays ("episode", "movie",
// "clip") are watching, not listening, and are excluded from the overlay.
func isPlexAudioListen(plexType string) bool {
	return strings.EqualFold(plexType, "track")
}

// classifyPlexMediaType maps an audio Plex item to Treningheten's media vocabulary.
// Everything audio currently reads as a song; distinguishing audiobooks/podcasts
// needs the item's library section/agent (a later refinement — see docs/media.md).
func classifyPlexMediaType(plexType string) string {
	return models.MediaTypeSong
}

// activityWindowForExercise resolves the absolute [start, end] window of an
// activity from its exercise, mirroring ConvertExerciseToExerciseObject's Time
// fallbacks. Duration is stored as a raw seconds count (repo convention).
func activityWindowForExercise(exercise models.Exercise) (start time.Time, end time.Time) {
	if exercise.Time != nil {
		start = *exercise.Time
	} else if day, err := database.GetExerciseDayByID(exercise.ExerciseDayID); err == nil && day != nil {
		start = day.Date
	} else {
		start = time.Now()
	}

	window := defaultActivityWindow
	if exercise.Duration != nil && int64(*exercise.Duration) > 0 {
		window = time.Duration(int64(*exercise.Duration)) * time.Second
	}

	return start, start.Add(window)
}

// buildPlexPlaybackForWindow maps Plex history items into provider-neutral play
// events (audio only — TV/movie plays are watching, not listening) and defers the
// window matching + EndedAt clamp to the shared playbackForWindow.
//
// Note: account scoping is handled at fetch time (the server history request), not
// here — the server uses server-local account ids that don't match the plex.tv
// global id, so filtering by it client-side would wrongly drop everything.
func buildPlexPlaybackForWindow(items []models.PlexHistoryMetadata, start, end time.Time) []models.MediaPlayback {
	events := []mediaPlayEvent{}

	for _, item := range items {
		if item.ViewedAt <= 0 || !isPlexAudioListen(item.Type) {
			continue
		}

		var lengthSeconds int64
		if item.Duration > 0 {
			lengthSeconds = item.Duration / 1000
		}

		events = append(events, mediaPlayEvent{
			mediaType:      classifyPlexMediaType(item.Type),
			title:          item.Title,
			artist:         item.GrandparentTitle,
			album:          item.ParentTitle,
			providerItemID: item.RatingKey,
			startedAt:      time.Unix(item.ViewedAt, 0).UTC(),
			trackLengthSec: lengthSeconds,
		})
	}

	return playbackForWindow(events, start, end)
}

// plexFetchHistory queries the PMS server history within the window, scoped to the
// given server-local accountID. History is ALWAYS scoped — an empty accountID is
// rejected by the caller rather than fetched unscoped, so other users' plays can
// never leak onto an activity.
func plexFetchHistory(serverURL, token, accountID string, start, end time.Time) ([]models.PlexHistoryMetadata, error) {
	base := strings.TrimRight(serverURL, "/") + "/status/sessions/history/all"

	v := url.Values{}
	v.Set("sort", "viewedAt:asc")
	if accountID != "" {
		v.Set("accountID", accountID)
	}
	// Pull a slightly padded window; the pure matcher does the precise filtering.
	v.Set("viewedAt>", strconv.FormatInt(start.Add(-mediaMatchGrace).Unix(), 10))
	v.Set("viewedAt<", strconv.FormatInt(end.Add(mediaMatchGrace).Unix(), 10))

	req, err := http.NewRequest("GET", base+"?"+v.Encode(), nil)
	if err != nil {
		logger.Log.Error("Plex history request generation threw error. Error: " + err.Error())
		return nil, errors.New("Plex history request generation threw error.")
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Plex-Token", token)
	req.Header.Set("X-Plex-Client-Identifier", files.ConfigFile.Media.Plex.ClientIdentifier)

	// Plex serves a self-signed cert on plex.direct hostnames, so skip verification
	// (the trust model the official clients use for these addresses). The stored
	// ServerURL was probed for reachability at connect time.
	client := &http.Client{
		Timeout:   15 * time.Second,
		Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}},
	}
	resp, err := client.Do(req)
	if err != nil {
		logger.Log.Error("Plex history request threw error. Error: " + err.Error())
		return nil, errors.New("Plex history request threw error.")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		logger.Log.Error("Plex history returned non-200. Status: " + strconv.Itoa(resp.StatusCode))
		return nil, errors.New("Plex history returned non-200 status.")
	}

	history := models.PlexHistoryResponse{}
	if err := json.NewDecoder(resp.Body).Decode(&history); err != nil {
		logger.Log.Error("Failed to parse Plex history. Error: " + err.Error())
		return nil, errors.New("Failed to parse Plex history.")
	}

	return history.MediaContainer.Metadata, nil
}

// PlexSyncOperationForUser pulls the Plex listening history overlapping one
// operation's activity window and stores it (delete-and-replace per provider). It
// always stamps Operation.MediaRetrievedAt — even when nothing is found or no
// server is known — so the UI can tell "pulled, found nothing" from "never pulled".
func PlexSyncOperationForUser(user models.User, operation models.Operation) error {
	connection, err := database.GetMediaConnectionForUserProvider(user.ID, models.MediaProviderPlex)
	if err != nil {
		return err
	}

	// No usable connection — record the attempt and move on.
	if connection == nil || connection.AccessToken == nil || connection.ServerURL == nil || *connection.ServerURL == "" {
		logger.Log.Trace("No usable Plex connection for media sync; stamping guard only.")
		return database.SetOperationMediaRetrievedAt(operation.ID, time.Now())
	}

	token, err := utilities.DecryptString(*connection.AccessToken, files.ConfigFile.Media.TokenKey)
	if err != nil {
		return errors.New("failed to decrypt Plex token: " + err.Error())
	}

	exercise, err := database.GetExerciseByIDAndUserID(operation.ExerciseID, user.ID)
	if err != nil {
		return err
	} else if exercise == nil {
		return errors.New("exercise not found for media sync")
	}

	start, end := activityWindowForExercise(*exercise)

	accountID := ""
	if connection.AccountID != nil {
		accountID = *connection.AccountID
	}

	// Fail closed: without a resolved server-local account id we cannot scope history
	// to this user, and an unscoped pull would overlay other users' plays. Stamp the
	// guard and stop rather than leak (reconnect re-resolves the account).
	if accountID == "" {
		logger.Log.Warn("Plex media sync skipped: no server-local account id resolved (reconnect to fix). Stamping guard only.")
		return database.SetOperationMediaRetrievedAt(operation.ID, time.Now())
	}

	items, err := plexFetchHistory(*connection.ServerURL, token, accountID, start, end)
	if err != nil {
		return err
	}

	logger.Log.Info(fmt.Sprintf("Plex history: fetched %d items for window %s..%s (account %q)", len(items), start.UTC().Format(time.RFC3339), end.UTC().Format(time.RFC3339), accountID))
	if len(items) > 0 {
		first := time.Unix(items[0].ViewedAt, 0).UTC()
		last := time.Unix(items[len(items)-1].ViewedAt, 0).UTC()
		logger.Log.Info(fmt.Sprintf("Plex history: first item %q viewedAt %s, last item viewedAt %s", items[0].Title, first.Format(time.RFC3339), last.Format(time.RFC3339)))
	}

	playback := buildPlexPlaybackForWindow(items, start, end)

	if err := database.ReplaceMediaPlaybackForOperationProvider(operation.ID, models.MediaProviderPlex, playback); err != nil {
		return err
	}

	if err := database.SetOperationMediaRetrievedAt(operation.ID, time.Now()); err != nil {
		return err
	}

	now := time.Now()
	connection.LastSyncedAt = &now
	if _, err := database.UpdateMediaConnectionInDB(*connection); err != nil {
		logger.Log.Warn("Failed to update media connection last-synced time. Error: " + err.Error())
	}

	logger.Log.Info("Synced " + strconv.Itoa(len(playback)) + " Plex playback rows for operation " + operation.ID.String())
	return nil
}

// TriggerMediaSyncForOperation fires a media pull in the background after an
// operation is created, so the timeline appears on the next load (like Strava
// streams) without blocking the create request. It is a no-op when the media
// feature is off or the user has no connected provider.
func TriggerMediaSyncForOperation(user models.User, operationID uuid.UUID) {
	if !mediaEnabled() {
		return
	}

	go func() {
		connections, err := database.GetMediaConnectionsForUser(user.ID)
		if err != nil || len(connections) == 0 {
			return
		}

		operation, err := database.GetOperationByIDAndUserID(operationID, user.ID)
		if err != nil {
			logger.Log.Warn("Background media sync could not load operation. Error: " + err.Error())
			return
		}

		// Fire once per activity: the Strava cron re-syncs activities hourly, so skip
		// anything already pulled. The manual re-pull button bypasses this guard.
		if operation.MediaRetrievedAt != nil {
			return
		}

		if plexEnabled() {
			if err := PlexSyncOperationForUser(user, operation); err != nil {
				logger.Log.Warn("Background Plex media sync failed. Error: " + err.Error())
			}
		}
		if spotifyEnabled() {
			if err := SpotifySyncOperationForUser(user, operation); err != nil {
				logger.Log.Warn("Background Spotify media sync failed. Error: " + err.Error())
			}
		}
	}()
}
