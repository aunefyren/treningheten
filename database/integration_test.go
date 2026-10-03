package database

import (
	"testing"
	"time"

	"github.com/aunefyren/treningheten/models"

	"github.com/google/uuid"
)

func TestIntegrationStatusLifecycle(t *testing.T) {
	newTestDB(t)

	user := makeTestUser(t, "health@example.com", nil)
	other := makeTestUser(t, "health-other@example.com", nil)

	missing, err := GetIntegrationStatus(user.ID, models.IntegrationProviderPlex)
	if err != nil || missing != nil {
		t.Fatalf("GetIntegrationStatus on a healthy connection = %v, %v; want nil, nil", missing, err)
	}

	since := time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)
	saved, err := SaveIntegrationStatus(models.IntegrationStatus{
		UserID: user.ID, Provider: models.IntegrationProviderPlex, Status: models.IntegrationStatusAuthFailed, FailingSince: &since,
	})
	if err != nil {
		t.Fatalf("SaveIntegrationStatus returned error: %v", err)
	}
	if saved.ID == uuid.Nil {
		t.Fatalf("SaveIntegrationStatus did not assign an id")
	}

	// Saving the loaded row again updates it in place rather than inserting a duplicate.
	notified := since.Add(time.Hour)
	saved.NotifiedAt = &notified
	if _, err := SaveIntegrationStatus(saved); err != nil {
		t.Fatalf("SaveIntegrationStatus (update) returned error: %v", err)
	}
	var count int64
	Instance.Model(&models.IntegrationStatus{}).Where("user_id = ?", user.ID).Count(&count)
	if count != 1 {
		t.Fatalf("rows for the user = %d, want 1", count)
	}

	found, err := GetIntegrationStatus(user.ID, models.IntegrationProviderPlex)
	if err != nil || found == nil {
		t.Fatalf("GetIntegrationStatus = %v, %v", found, err)
	}
	if found.Status != models.IntegrationStatusAuthFailed || found.FailingSince == nil || !found.FailingSince.Equal(since) || found.NotifiedAt == nil {
		t.Errorf("reloaded status = %+v", found)
	}

	// Another user's row is not this user's.
	if row, _ := GetIntegrationStatus(other.ID, models.IntegrationProviderPlex); row != nil {
		t.Errorf("another user sees the row: %+v", row)
	}

	if err := DeleteIntegrationStatus(user.ID, models.IntegrationProviderPlex); err != nil {
		t.Fatalf("DeleteIntegrationStatus returned error: %v", err)
	}
	if row, _ := GetIntegrationStatus(user.ID, models.IntegrationProviderPlex); row != nil {
		t.Errorf("row survived the delete: %+v", row)
	}

	// The delete is hard, so the unique (user, provider) slot is free for the next breakage.
	if _, err := SaveIntegrationStatus(models.IntegrationStatus{
		UserID: user.ID, Provider: models.IntegrationProviderPlex, Status: models.IntegrationStatusOK, FailingSince: &since,
	}); err != nil {
		t.Errorf("SaveIntegrationStatus after delete returned error: %v", err)
	}
}

func TestGetMediaConnectionsForProvider(t *testing.T) {
	newTestDB(t)

	first := makeTestUser(t, "plex-a@example.com", nil)
	second := makeTestUser(t, "plex-b@example.com", nil)
	for _, connection := range []models.MediaConnection{
		{Enabled: true, UserID: first.ID, Provider: models.MediaProviderPlex},
		{Enabled: true, UserID: second.ID, Provider: models.MediaProviderPlex},
		{Enabled: true, UserID: second.ID, Provider: models.MediaProviderSpotify},
	} {
		connection.ID = uuid.New()
		if _, err := CreateMediaConnectionInDB(connection); err != nil {
			t.Fatal(err)
		}
	}
	// A disconnected (soft-deleted) connection is not checked.
	if err := DeleteMediaConnectionForUserProvider(second.ID, models.MediaProviderPlex); err != nil {
		t.Fatal(err)
	}

	connections, err := GetMediaConnectionsForProvider(models.MediaProviderPlex)
	if err != nil {
		t.Fatalf("GetMediaConnectionsForProvider returned error: %v", err)
	}
	if len(connections) != 1 || connections[0].UserID != first.ID {
		t.Errorf("got %d plex connections, want only the first user's", len(connections))
	}
}

func TestGetExercisesForMediaBackfill(t *testing.T) {
	newTestDB(t)

	user := makeTestUser(t, "backfill@example.com", nil)
	other := makeTestUser(t, "backfill-other@example.com", nil)
	day := makeDay(t, user.ID, time.Now())
	otherDay := makeDay(t, other.ID, time.Now())

	recent := makeSession(t, day.ID, time.Now().Add(-time.Hour))
	old := makeSession(t, day.ID, time.Now().Add(-48*time.Hour))
	if err := Instance.Model(&old).Update("created_at", time.Now().Add(-30*24*time.Hour)).Error; err != nil {
		t.Fatal(err)
	}
	makeSession(t, otherDay.ID, time.Now())
	seedExercise(t, day.ID, false, true)

	exercises, err := GetExercisesForMediaBackfill(user.ID, time.Now().Add(-7*24*time.Hour))
	if err != nil {
		t.Fatalf("GetExercisesForMediaBackfill returned error: %v", err)
	}
	if len(exercises) != 1 || exercises[0].ID != recent.ID {
		t.Errorf("got %d sessions, want only the recent enabled one of this user", len(exercises))
	}
}
