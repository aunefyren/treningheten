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

// GetMediaPlaybackForOperation returns the listening timeline for an operation,
// ordered chronologically (the natural timeline order).
func GetMediaPlaybackForOperation(operationID uuid.UUID) (playback []models.MediaPlayback, err error) {
	playback = []models.MediaPlayback{}

	record := Instance.Where("`media_playbacks`.operation_id = ?", operationID).
		Order("`media_playbacks`.started_at asc").
		Find(&playback)

	if record.Error != nil {
		return playback, record.Error
	}

	return
}

// ReplaceMediaPlaybackForOperationProvider is the idempotent pull primitive:
// delete-and-replace per (operation, provider). Every pull wipes that operation's
// rows for the provider and reinserts, which side-steps a fragile compound de-dupe
// key (a song can legitimately repeat within one session) and makes re-pull "just
// work" — the same spirit as Strava sync. Other providers' rows are untouched.
func ReplaceMediaPlaybackForOperationProvider(operationID uuid.UUID, provider string, playback []models.MediaPlayback) error {
	return Instance.Transaction(func(tx *gorm.DB) error {
		del := tx.Unscoped().
			Where("operation_id = ?", operationID).
			Where("provider = ?", provider).
			Delete(&models.MediaPlayback{})
		if del.Error != nil {
			return del.Error
		}

		if len(playback) == 0 {
			return nil
		}

		for i := range playback {
			if playback[i].ID == uuid.Nil {
				playback[i].ID = uuid.New()
			}
			playback[i].OperationID = operationID
			playback[i].Provider = provider
		}

		create := tx.Create(&playback)
		return create.Error
	})
}

// SetOperationMediaRetrievedAt stamps the per-activity pull guard so the UI can
// distinguish "pulled, found nothing" from "never pulled".
func SetOperationMediaRetrievedAt(operationID uuid.UUID, at time.Time) error {
	record := Instance.Model(&models.Operation{}).
		Where("`operations`.id = ?", operationID).
		Update("media_retrieved_at", at)
	return record.Error
}
