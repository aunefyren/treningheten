package controllers

import (
	"errors"
	"strconv"
	"time"

	"github.com/aunefyren/treningheten/database"
	"github.com/aunefyren/treningheten/logger"
	"github.com/aunefyren/treningheten/models"

	"github.com/google/uuid"
)

// Integration health for the media providers (Plex, Spotify, Audiobookshelf): the daily
// check and the recovery backfill. See docs/integration-health.md.

// spotifyHistoryWindow is how far back Spotify's recently-played reaches. A backfill
// further back than this would only ever find nothing.
const spotifyHistoryWindow = 24 * time.Hour

// mediaProviderEnabled reports whether a media provider is switched on.
func mediaProviderEnabled(provider string) bool {
	switch provider {
	case models.MediaProviderPlex:
		return plexEnabled()
	case models.MediaProviderSpotify:
		return spotifyEnabled()
	case models.MediaProviderAudiobookshelf:
		return absEnabled()
	default:
		return false
	}
}

// mediaProviderSync is the per-session pull for a media provider. A switch rather than a
// map var: the syncs record their health, which can start a recovery that calls back
// here, and a package-level map of them would be an initialization cycle.
func mediaProviderSync(provider string) func(models.User, models.Exercise) error {
	switch provider {
	case models.MediaProviderPlex:
		return PlexSyncExerciseForUser
	case models.MediaProviderSpotify:
		return SpotifySyncExerciseForUser
	case models.MediaProviderAudiobookshelf:
		return AudiobookshelfSyncExerciseForUser
	default:
		return nil
	}
}

// mediaProviderCheck makes one cheap authenticated call for a connection. Errors that
// are health signals are tagged by the provider clients.
func mediaProviderCheck(provider string, connection *models.MediaConnection) error {
	switch provider {
	case models.MediaProviderPlex:
		return plexCheckConnection(connection)
	case models.MediaProviderSpotify:
		return spotifyCheckConnection(connection)
	case models.MediaProviderAudiobookshelf:
		return absCheckConnection(connection)
	default:
		return errors.New("Unknown media provider.")
	}
}

// checkMediaConnectionForUser checks one user's connection to a media provider and
// records the outcome. No connection means nothing to check.
func checkMediaConnectionForUser(userID uuid.UUID, provider string) error {
	connection, err := database.GetMediaConnectionForUserProvider(userID, provider)
	if err != nil {
		return err
	} else if connection == nil || connection.AccessToken == nil {
		return nil
	}

	err = mediaProviderCheck(provider, connection)
	recordIntegrationOutcome(userID, provider, err)
	return err
}

// MediaHealthCheckForProvider checks every connection to one media provider; see
// IntegrationHealthCheckForAllUsers.
func MediaHealthCheckForProvider(provider string) {
	connections, err := database.GetMediaConnectionsForProvider(provider)
	if err != nil {
		logger.Log.Error("Media health check failed to list " + provider + " connections. Error: " + err.Error())
		return
	}

	for _, connection := range connections {
		if err := checkMediaConnectionForUser(connection.UserID, provider); err != nil {
			logger.Log.Info("Media health check failed for " + provider + " user " + connection.UserID.String() + ". Error: " + err.Error())
		}
	}
}

// mediaBackfillSince re-pulls a media provider's soundtrack for every session created
// since its connection started failing. The session-level pull guards (MediaRetrievedAt,
// MediaSettled) can't be trusted for that gap: another provider's success stamps the
// session as pulled even when this one failed. A day of slack covers a session created
// just before the first failure whose pull came after it.
//
// If the provider fails again part-way, the backfill stops and the gap's start is kept,
// so the next recovery picks up where this one left off.
func mediaBackfillSince(user models.User, provider string, since time.Time) {
	sync := mediaProviderSync(provider)
	if sync == nil {
		return
	}

	from := since.Add(-24 * time.Hour)
	if provider == models.MediaProviderSpotify {
		// Spotify can't answer for anything older than its history window.
		if limit := time.Now().Add(-spotifyHistoryWindow); from.Before(limit) {
			from = limit
		}
	}

	exercises, err := database.GetExercisesForMediaBackfill(user.ID, from)
	if err != nil {
		logger.Log.Warn("Media backfill could not load sessions. Error: " + err.Error())
		return
	}

	synced := 0
	for _, exercise := range exercises {
		if err := sync(user, exercise); err != nil {
			if integrationErrorStatus(err) != "" {
				recordIntegrationFailureSince(user.ID, provider, err, since)
				logger.Log.Warn("Media backfill for " + provider + " stopped after " + strconv.Itoa(synced) + " sessions. Error: " + err.Error())
				return
			}
			logger.Log.Warn("Media backfill for " + provider + " skipped session " + exercise.ID.String() + ". Error: " + err.Error())
			continue
		}
		synced++
	}

	logger.Log.Info("Media backfill re-pulled " + strconv.Itoa(synced) + " " + provider + " sessions for user " + user.ID.String() + ".")
}
