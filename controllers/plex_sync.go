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

// defaultActivityWindow is the last-resort session length when neither the session
// nor its operations/sets carry a duration — a real start time with an unknown
// length still captures a plausible (if generous) listening window.
const defaultActivityWindow = 2 * time.Hour

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

// resolveSessionWindow computes the absolute [start, end] listening window for a
// session and whether it is trustworthy enough to match against (ok). It is pure:
// fallbackSeconds is a session length derived from the operations/sets (0 when
// unknown), used only when the session carries no explicit Duration.
//
// ok is false when the session has no real clock time — either no Time at all, or a
// date-only midnight stamp (manual past-day logs write midnight, which is not a real
// start of day). We can't place a soundtrack within a day we don't have a time for,
// and a wrong soundtrack is worse than none, so the caller stamps the guard and
// stores nothing. Durations are a raw seconds count (repo convention).
func resolveSessionWindow(exercise models.Exercise, fallbackSeconds int64) (start time.Time, end time.Time, ok bool) {
	if exercise.Time == nil || isDateOnly(*exercise.Time) {
		return time.Time{}, time.Time{}, false
	}
	start = *exercise.Time

	seconds := int64(0)
	if exercise.Duration != nil && int64(*exercise.Duration) > 0 {
		seconds = int64(*exercise.Duration)
	} else if fallbackSeconds > 0 {
		seconds = fallbackSeconds
	}

	window := defaultActivityWindow
	if seconds > 0 {
		window = time.Duration(seconds) * time.Second
	}

	return start, start.Add(window), true
}

// isDateOnly reports whether a timestamp is exactly UTC midnight — the sentinel the
// day-date fallback uses when no clock time is known (see resolveSessionWindow). A
// genuine 00:00:00 workout is a rare, accepted false positive.
func isDateOnly(t time.Time) bool {
	u := t.UTC()
	return u.Hour() == 0 && u.Minute() == 0 && u.Second() == 0 && u.Nanosecond() == 0
}

// sessionFallbackSeconds derives a session length from its operations when the
// session itself has no Duration: first the sum of the operations' own durations
// (Strava carries these), else the sum of every set's logged Time (manual/Hevy).
// Returns 0 when nothing is logged, leaving the caller on the default window.
func sessionFallbackSeconds(exercise models.Exercise) int64 {
	exerciseObject, err := ConvertExerciseToExerciseObject(exercise)
	if err != nil {
		logger.Log.Warn("Failed to resolve session duration fallback. Error: " + err.Error())
		return 0
	}

	var opSeconds int64
	var setSeconds int64
	for _, operation := range exerciseObject.Operations {
		if operation.Duration != nil {
			opSeconds += int64(*operation.Duration)
		}
		for _, set := range operation.OperationSets {
			if set.Time != nil {
				setSeconds += int64(*set.Time)
			}
		}
	}

	if opSeconds > 0 {
		return opSeconds
	}
	return setSeconds
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

// PlexSyncExerciseForUser pulls the Plex listening history overlapping a session's
// window and stores it (delete-and-replace per provider). It always stamps
// Exercise.MediaRetrievedAt — even when nothing is found, no server is known, or the
// session has no trustworthy time — so the UI can tell "pulled, found nothing" from
// "never pulled".
func PlexSyncExerciseForUser(user models.User, exercise models.Exercise) error {
	connection, err := database.GetMediaConnectionForUserProvider(user.ID, models.MediaProviderPlex)
	if err != nil {
		return err
	}

	// No usable connection — record the attempt and move on.
	if connection == nil || connection.AccessToken == nil || connection.ServerURL == nil || *connection.ServerURL == "" {
		logger.Log.Trace("No usable Plex connection for media sync; stamping guard only.")
		return database.SetExerciseMediaRetrievedAt(exercise.ID, time.Now())
	}

	token, err := utilities.DecryptString(*connection.AccessToken, files.ConfigFile.Media.TokenKey)
	if err != nil {
		return errors.New("failed to decrypt Plex token: " + err.Error())
	}

	start, end, ok := resolveSessionWindow(exercise, sessionFallbackSeconds(exercise))
	if !ok {
		// No real clock time for the session (date-only/manual) — a soundtrack can't be
		// placed, so store nothing and just record the attempt.
		logger.Log.Trace("Session has no trustworthy time for media match; stamping guard only.")
		return database.SetExerciseMediaRetrievedAt(exercise.ID, time.Now())
	}

	accountID := ""
	if connection.AccountID != nil {
		accountID = *connection.AccountID
	}

	// Fail closed: without a resolved server-local account id we cannot scope history
	// to this user, and an unscoped pull would overlay other users' plays. Stamp the
	// guard and stop rather than leak (reconnect re-resolves the account).
	if accountID == "" {
		logger.Log.Warn("Plex media sync skipped: no server-local account id resolved (reconnect to fix). Stamping guard only.")
		return database.SetExerciseMediaRetrievedAt(exercise.ID, time.Now())
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

	if err := database.ReplaceMediaPlaybackForExerciseProvider(exercise.ID, models.MediaProviderPlex, playback); err != nil {
		return err
	}

	if err := database.SetExerciseMediaRetrievedAt(exercise.ID, time.Now()); err != nil {
		return err
	}

	now := time.Now()
	connection.LastSyncedAt = &now
	if _, err := database.UpdateMediaConnectionInDB(*connection); err != nil {
		logger.Log.Warn("Failed to update media connection last-synced time. Error: " + err.Error())
	}

	logger.Log.Info("Synced " + strconv.Itoa(len(playback)) + " Plex playback rows for session " + exercise.ID.String())
	return nil
}

// mediaSettleWindow is how long after a session's listening window closes before the
// one-time "settle" re-pull runs. It catches providers whose history lags (Spotify's
// recently-played), and is comfortably inside Spotify's ~24h window so the settle pull
// never lands past it. See docs/media.md.
const mediaSettleWindow = 1 * time.Hour

// mediaReconcileLookback bounds how far back the hourly reconcile scan reaches, so it
// doesn't re-walk all history every run. Older un-settled sessions are lost causes for
// short-window providers anyway (Spotify keeps ~24h).
const mediaReconcileLookback = 14 * 24 * time.Hour

// syncMediaProvidersForExercise pulls every enabled provider for the session,
// returning a per-provider warning list — one provider's failure never sinks the
// others (a Spotify allowlist 403 must not discard a successful Plex sync). Shared by
// the manual re-pull, the create-time trigger, and the reconcile cron.
func syncMediaProvidersForExercise(user models.User, exercise models.Exercise) []string {
	warnings := []string{}
	if plexEnabled() {
		if err := PlexSyncExerciseForUser(user, exercise); err != nil {
			logger.Log.Info("Failed to sync Plex media for session. Error: " + err.Error())
			warnings = append(warnings, "Plex: "+err.Error())
		}
	}
	if spotifyEnabled() {
		if err := SpotifySyncExerciseForUser(user, exercise); err != nil {
			logger.Log.Info("Failed to sync Spotify media for session. Error: " + err.Error())
			warnings = append(warnings, "Spotify: "+err.Error())
		}
	}
	if absEnabled() {
		if err := AudiobookshelfSyncExerciseForUser(user, exercise); err != nil {
			logger.Log.Info("Failed to sync Audiobookshelf media for session. Error: " + err.Error())
			warnings = append(warnings, "Audiobookshelf: "+err.Error())
		}
	}
	return warnings
}

// windowSettledBy reports whether the session's listening window has closed at least
// mediaSettleWindow ago — i.e. no lagging provider (Spotify) is still expected to
// reveal new plays, so a pull now is final. It is also true when the session has no
// trustworthy window (nothing to wait for), so such sessions are retired from the
// settle pass immediately rather than re-scanned every hour.
func windowSettledBy(exercise models.Exercise, now time.Time) bool {
	_, end, ok := resolveSessionWindow(exercise, sessionFallbackSeconds(exercise))
	if !ok {
		return true
	}
	return !end.Add(mediaSettleWindow).After(now)
}

// TriggerMediaSyncForExercise fires a media pull in the background after a session is
// synced/created, so the timeline appears on the next load (like Strava streams)
// without blocking the request. It is wired into the Strava and Hevy imports (both
// carry a fully-known time window at import); manual sessions instead wait for the
// reconcile cron. It is a no-op when the media feature is off or the user has no
// connected provider.
func TriggerMediaSyncForExercise(user models.User, exerciseID uuid.UUID) {
	if !mediaEnabled() {
		return
	}

	go func() {
		connections, err := database.GetMediaConnectionsForUser(user.ID)
		if err != nil || len(connections) == 0 {
			return
		}

		exercise, err := database.GetExerciseByIDAndUserID(exerciseID, user.ID)
		if err != nil {
			logger.Log.Warn("Background media sync could not load session. Error: " + err.Error())
			return
		} else if exercise == nil {
			return
		}

		// Fire once per session: the Strava cron re-syncs activities hourly, so skip
		// anything already pulled. The manual re-pull button bypasses this guard.
		if exercise.MediaRetrievedAt != nil {
			return
		}

		syncMediaProvidersForExercise(user, *exercise)

		// A backfilled import (window already closed) has no delayed provider data
		// still coming, so settle it now and spare it the reconcile cron's settle pass.
		// A fresh session stays un-settled so the cron does the one delayed re-pull.
		if windowSettledBy(*exercise, time.Now()) {
			if err := database.SetExerciseMediaSettled(exercise.ID, true); err != nil {
				logger.Log.Warn("Failed to mark session media settled. Error: " + err.Error())
			}
		}
	}()
}

// MediaReconcileForAllUsers is the hourly media job. It is a dedicated media task — not
// a side-effect of the Strava/Hevy syncs — so it owns the whole media lifecycle across
// every session type: it does the first pull for sessions no import triggered (manual,
// and any missed Strava/Hevy), and the one-time delayed "settle" re-pull that catches
// lagging providers (Spotify). It scans only recent sessions still owing work. See
// docs/media.md.
func MediaReconcileForAllUsers() {
	if !mediaEnabled() {
		return
	}

	userIDs, err := database.GetUserIDsWithMediaConnections()
	if err != nil {
		logger.Log.Error("Media reconcile failed to list users. Error: " + err.Error())
		return
	}

	since := time.Now().Add(-mediaReconcileLookback)
	for _, userID := range userIDs {
		user, err := database.GetUserInformation(userID)
		if err != nil {
			logger.Log.Warn("Media reconcile could not load user. Error: " + err.Error())
			continue
		}

		exercises, err := database.GetExercisesForMediaReconcile(userID, since)
		if err != nil {
			logger.Log.Warn("Media reconcile could not load sessions. Error: " + err.Error())
			continue
		}

		for _, exercise := range exercises {
			reconcileMediaForExercise(user, exercise)
		}
	}

	logger.Log.Info("Media reconcile task finished.")
}

// reconcileMediaForExercise runs the first-pull / settle state machine for one session
// (callers pre-filter to sessions still owing work). See docs/media.md.
func reconcileMediaForExercise(user models.User, exercise models.Exercise) {
	now := time.Now()

	switch {
	case exercise.MediaRetrievedAt == nil:
		// Never pulled — first pull. Covers manual sessions (no import trigger) and any
		// import that missed. Retire an already-closed/untrustworthy window immediately;
		// leave a fresh session un-settled for the settle pass on a later run.
		syncMediaProvidersForExercise(user, exercise)
		if windowSettledBy(exercise, now) {
			if err := database.SetExerciseMediaSettled(exercise.ID, true); err != nil {
				logger.Log.Warn("Failed to mark session media settled. Error: " + err.Error())
			}
		}
	case !exercise.MediaSettled && windowSettledBy(exercise, now):
		// Settle pass: the one delayed re-pull, now that the window has closed long
		// enough for a lagging provider to have recorded everything.
		syncMediaProvidersForExercise(user, exercise)
		if err := database.SetExerciseMediaSettled(exercise.ID, true); err != nil {
			logger.Log.Warn("Failed to mark session media settled. Error: " + err.Error())
		}
	}
	// else: pulled but window not yet closed — wait for a later run.
}
