package database

import (
	"testing"
	"time"

	"github.com/aunefyren/treningheten/models"

	"github.com/google/uuid"
)

func TestGetUserIDsWithMediaConnectionsAndUpdate(t *testing.T) {
	newTestDB(t)

	user := makeTestUser(t, "mediaconn@example.com", nil)
	conn := models.MediaConnection{
		GormModel: models.GormModel{ID: uuid.New()},
		Enabled:   true,
		UserID:    user.ID,
		Provider:  models.MediaProviderPlex,
		AccountID: strPtr("acc-1"),
	}
	if _, err := CreateMediaConnectionInDB(conn); err != nil {
		t.Fatalf("CreateMediaConnectionInDB returned error: %v", err)
	}
	// A second connection for the same user must not double-count the user id.
	conn2 := models.MediaConnection{
		GormModel: models.GormModel{ID: uuid.New()},
		Enabled:   true,
		UserID:    user.ID,
		Provider:  models.MediaProviderSpotify,
	}
	if _, err := CreateMediaConnectionInDB(conn2); err != nil {
		t.Fatalf("CreateMediaConnectionInDB (spotify) returned error: %v", err)
	}

	ids, err := GetUserIDsWithMediaConnections()
	if err != nil {
		t.Fatalf("GetUserIDsWithMediaConnections returned error: %v", err)
	}
	if len(ids) != 1 || ids[0] != user.ID {
		t.Errorf("got %d distinct user ids, want 1", len(ids))
	}

	// UpdateMediaConnectionInDB (Save) persists a field change.
	conn.AccountID = strPtr("acc-2")
	if _, err := UpdateMediaConnectionInDB(conn); err != nil {
		t.Fatalf("UpdateMediaConnectionInDB returned error: %v", err)
	}
	reloaded, err := GetMediaConnectionForUserProvider(user.ID, models.MediaProviderPlex)
	if err != nil || reloaded == nil {
		t.Fatalf("failed to reload connection: err=%v", err)
	}
	if reloaded.AccountID == nil || *reloaded.AccountID != "acc-2" {
		t.Errorf("account id not persisted: %v", reloaded.AccountID)
	}
}

func TestGetAllExercisesForMediaSync(t *testing.T) {
	newTestDB(t)

	user := makeTestUser(t, "mediasync@example.com", nil)
	day := makeDay(t, user.ID, time.Now())

	// Two enabled sessions (regardless of media pull state) → both returned, time-ordered.
	makeSession(t, day.ID, time.Date(2024, 6, 15, 8, 0, 0, 0, time.UTC))
	makeSession(t, day.ID, time.Date(2024, 6, 15, 10, 0, 0, 0, time.UTC))

	// A disabled session must be excluded.
	seedExercise(t, day.ID, false, true)

	exercises, err := GetAllExercisesForMediaSync(user.ID)
	if err != nil {
		t.Fatalf("GetAllExercisesForMediaSync returned error: %v", err)
	}
	if len(exercises) != 2 {
		t.Errorf("got %d sessions, want 2 (enabled only)", len(exercises))
	}
}
