package database

import (
	"time"

	"github.com/aunefyren/treningheten/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// GetMediaConnectionsForUser returns every enabled media connection for a user,
// one per connected provider.
func GetMediaConnectionsForUser(userID uuid.UUID) (connections []models.MediaConnection, err error) {
	connections = []models.MediaConnection{}

	record := Instance.Where("`media_connections`.enabled = ?", 1).
		Where("`media_connections`.user_id = ?", userID).
		Order("`media_connections`.provider asc").
		Find(&connections)

	if record.Error != nil {
		return connections, record.Error
	}

	return
}

// GetUserIDsWithMediaConnections returns the distinct user ids that have at least one
// enabled media connection — the candidate set the reconcile cron scans, so it skips
// users with no connected provider entirely.
func GetUserIDsWithMediaConnections() ([]uuid.UUID, error) {
	userIDs := []uuid.UUID{}

	record := Instance.Model(&models.MediaConnection{}).
		Where("`media_connections`.enabled = ?", 1).
		Distinct().
		Pluck("user_id", &userIDs)

	if record.Error != nil {
		return nil, record.Error
	}

	return userIDs, nil
}

// GetMediaConnectionForUserProvider fetches the single connection for a (user,
// provider) pair, or nil when none exists.
func GetMediaConnectionForUserProvider(userID uuid.UUID, provider string) (connection *models.MediaConnection, err error) {
	found := models.MediaConnection{}
	record := Instance.Where("`media_connections`.enabled = ?", 1).
		Where("`media_connections`.user_id = ?", userID).
		Where("`media_connections`.provider = ?", provider).
		Find(&found)

	if record.Error != nil {
		return nil, record.Error
	} else if record.RowsAffected == 0 {
		return nil, nil
	}

	return &found, nil
}

func CreateMediaConnectionInDB(connection models.MediaConnection) (models.MediaConnection, error) {
	record := Instance.Create(&connection)
	if record.Error != nil {
		return connection, record.Error
	}
	return connection, nil
}

func UpdateMediaConnectionInDB(connection models.MediaConnection) (models.MediaConnection, error) {
	record := Instance.Save(&connection)
	if record.Error != nil {
		return connection, record.Error
	}
	return connection, nil
}

// DeleteMediaConnectionForUserProvider soft-deletes the connection for a (user,
// provider) pair. The user's already-overlaid MediaPlayback rows are left intact
// (they are historical facts about past activities, not live credentials).
func DeleteMediaConnectionForUserProvider(userID uuid.UUID, provider string) error {
	record := Instance.Where("`media_connections`.user_id = ?", userID).
		Where("`media_connections`.provider = ?", provider).
		Delete(&models.MediaConnection{})
	return record.Error
}

// GetMediaPlaybackForExercise returns the listening timeline for a session,
// ordered chronologically (the natural timeline order).
func GetMediaPlaybackForExercise(exerciseID uuid.UUID) (playback []models.MediaPlayback, err error) {
	playback = []models.MediaPlayback{}

	record := Instance.Where("`media_playbacks`.exercise_id = ?", exerciseID).
		Order("`media_playbacks`.started_at asc").
		Find(&playback)

	if record.Error != nil {
		return playback, record.Error
	}

	return
}

// ReplaceMediaPlaybackForExerciseProvider is the idempotent pull primitive:
// delete-and-replace per (session, provider). A non-empty pull wipes that session's
// rows for the provider and reinserts, which side-steps a fragile compound de-dupe
// key (a song can legitimately repeat within one session) and makes re-pull "just
// work" — the same spirit as Strava sync. Other providers' rows are untouched.
//
// Non-destructive empty guard: an **empty** pull is a no-op — it never deletes
// existing rows. These providers only ever *add* to history (they don't retract
// plays), so an empty result means "nothing new / outside my window" (e.g. a Spotify
// pull past its ~24h window), not "authoritatively zero". Preserving the prior pull's
// rows stops a stale/out-of-window pull from erasing a good soundtrack. First-pull-empty
// still stores nothing (there's nothing to preserve). See docs/media.md.
func ReplaceMediaPlaybackForExerciseProvider(exerciseID uuid.UUID, provider string, playback []models.MediaPlayback) error {
	if len(playback) == 0 {
		return nil
	}

	return Instance.Transaction(func(tx *gorm.DB) error {
		del := tx.Unscoped().
			Where("exercise_id = ?", exerciseID).
			Where("provider = ?", provider).
			Delete(&models.MediaPlayback{})
		if del.Error != nil {
			return del.Error
		}

		for i := range playback {
			if playback[i].ID == uuid.Nil {
				playback[i].ID = uuid.New()
			}
			playback[i].ExerciseID = exerciseID
			playback[i].Provider = provider
		}

		create := tx.Create(&playback)
		return create.Error
	})
}

// SetExerciseMediaRetrievedAt stamps the per-session pull guard so the UI can
// distinguish "pulled, found nothing" from "never pulled".
func SetExerciseMediaRetrievedAt(exerciseID uuid.UUID, at time.Time) error {
	record := Instance.Model(&models.Exercise{}).
		Where("`exercises`.id = ?", exerciseID).
		Update("media_retrieved_at", at)
	return record.Error
}

// SetExerciseMediaSettled flips the one-time settle guard so the reconcile cron
// leaves the session alone from then on (see docs/media.md).
func SetExerciseMediaSettled(exerciseID uuid.UUID, settled bool) error {
	record := Instance.Model(&models.Exercise{}).
		Where("`exercises`.id = ?", exerciseID).
		Update("media_settled", settled)
	return record.Error
}

// GetExercisesForMediaReconcile returns the caller's recently-created sessions that
// still owe media work — never pulled (media_retrieved_at IS NULL) or pulled but not
// yet settled (media_settled = 0) — bounded to a recent lookback so the hourly scan
// doesn't re-walk all history. The reconcile job runs the first-pull / settle state
// machine over these (see docs/media.md).
//
// since is bound in its own (local) location on purpose: the sqlite backend stores
// timestamps as local-offset RFC3339 text and compares lexically, so binding UTC would
// misalign the ordering; MySQL/Postgres compare instants regardless of location.
// GetAllExercisesForMediaSync returns every enabled session for a user, regardless of
// its media pull state — used by the admin re-sync endpoint to force a fresh match over
// a user's whole history (unlike GetExercisesForMediaReconcile, which only returns
// sessions still owing work within the reconcile lookback).
func GetAllExercisesForMediaSync(userID uuid.UUID) ([]models.Exercise, error) {
	var exercises []models.Exercise

	record := Instance.
		Where("`exercises`.enabled = ?", 1).
		Joins("JOIN `exercise_days` on `exercises`.exercise_day_id = `exercise_days`.id").
		Where("`exercise_days`.enabled = ?", 1).
		Where("`exercise_days`.user_id = ?", userID).
		Order("`exercises`.`time` ASC").
		Find(&exercises)

	if record.Error != nil {
		return nil, record.Error
	}

	return exercises, nil
}

func GetExercisesForMediaReconcile(userID uuid.UUID, since time.Time) ([]models.Exercise, error) {
	var exercises []models.Exercise

	record := Instance.
		Where("`exercises`.enabled = ?", 1).
		Where("`exercises`.created_at >= ?", since).
		Where("(`exercises`.media_retrieved_at IS NULL OR `exercises`.media_settled = ?)", 0).
		Joins("JOIN `exercise_days` on `exercises`.exercise_day_id = `exercise_days`.id").
		Where("`exercise_days`.enabled = ?", 1).
		Where("`exercise_days`.user_id = ?", userID).
		Order("`exercises`.`time` ASC").
		Find(&exercises)

	if record.Error != nil {
		return nil, record.Error
	}

	return exercises, nil
}
