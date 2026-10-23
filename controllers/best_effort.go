package controllers

import (
	"github.com/aunefyren/treningheten/database"
	"github.com/aunefyren/treningheten/models"

	"github.com/google/uuid"
)

// refreshBestEfforts recomputes an activity's best efforts from its stored stream. The tracked
// distances and the glitch cap depend on the activity type, so this runs when the type
// changes; the Strava sync and the stream backfill cover everything else. An activity
// without a stream ends up with no best efforts.
func refreshBestEfforts(operationID uuid.UUID, userID uuid.UUID) error {
	streams, stravaName, err := database.GetOperationStreamsForUser(operationID, userID)
	if err != nil {
		return err
	}
	return database.ReplaceBestEffortsForOperation(operationID, userID, models.ComputeBestEfforts(streams, stravaName))
}

// sameUUIDPointer reports whether two optional IDs are both unset or equal.
func sameUUIDPointer(a *uuid.UUID, b *uuid.UUID) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}
