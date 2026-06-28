package database

import (
	"testing"
	"time"

	"github.com/aunefyren/treningheten/models"

	"github.com/google/uuid"
)

// makeMediaPlayback builds a minimal valid MediaPlayback row for a song.
func makeMediaPlayback(operationID uuid.UUID, provider, title string, startedAt time.Time) models.MediaPlayback {
	pb := models.MediaPlayback{
		OperationID: operationID,
		Provider:    provider,
		MediaType:   models.MediaTypeSong,
		Title:       title,
		StartedAt:   startedAt,
	}
	pb.ID = uuid.New()
	return pb
}

func TestMediaConnectionLifecycle(t *testing.T) {
	newTestDB(t)
	user := makeTestUser(t, "media@example.com", nil)

	// No connection yet.
	none, err := GetMediaConnectionForUserProvider(user.ID, models.MediaProviderPlex)
	if err != nil {
		t.Fatalf("lookup returned error: %v", err)
	}
	if none != nil {
		t.Fatalf("expected no connection, got %v", none)
	}

	token := "secret-token"
	server := "https://plex.example.com"
	conn := models.MediaConnection{
		GormModel:   models.GormModel{ID: uuid.New()},
		Enabled:     true,
		UserID:      user.ID,
		Provider:    models.MediaProviderPlex,
		ServerURL:   &server,
		AccessToken: &token,
		AccountID:   strPtr("42"),
	}
	if _, err := CreateMediaConnectionInDB(conn); err != nil {
		t.Fatalf("failed to create connection: %v", err)
	}

	got, err := GetMediaConnectionForUserProvider(user.ID, models.MediaProviderPlex)
	if err != nil || got == nil {
		t.Fatalf("failed to reload connection: %v (got %v)", err, got)
	}
	if got.AccessToken == nil || *got.AccessToken != token {
		t.Errorf("token not persisted: %v", got.AccessToken)
	}

	// Lookup is scoped per provider — a different provider must not match.
	if other, _ := GetMediaConnectionForUserProvider(user.ID, models.MediaProviderSpotify); other != nil {
		t.Errorf("expected no spotify connection, got %v", other)
	}

	// Listing returns the connection.
	list, err := GetMediaConnectionsForUser(user.ID)
	if err != nil {
		t.Fatalf("list returned error: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 connection, got %d", len(list))
	}

	// Deleting removes it.
	if err := DeleteMediaConnectionForUserProvider(user.ID, models.MediaProviderPlex); err != nil {
		t.Fatalf("delete returned error: %v", err)
	}
	if gone, _ := GetMediaConnectionForUserProvider(user.ID, models.MediaProviderPlex); gone != nil {
		t.Errorf("expected connection gone after delete, got %v", gone)
	}
}

func TestReplaceMediaPlaybackForOperationProvider(t *testing.T) {
	newTestDB(t)
	user := makeTestUser(t, "playback@example.com", nil)
	dayID := seedDay(t, user.ID, time.Now(), true)
	exerciseID := seedExerciseWithID(t, dayID)
	operationID := seedOperationWithGear(t, exerciseID, nil)

	base := time.Date(2026, 6, 27, 10, 0, 0, 0, time.UTC)

	// First pull: two Plex tracks plus one Spotify track (different provider).
	plexFirst := []models.MediaPlayback{
		makeMediaPlayback(operationID, models.MediaProviderPlex, "Song A", base),
		makeMediaPlayback(operationID, models.MediaProviderPlex, "Song B", base.Add(3*time.Minute)),
	}
	if err := ReplaceMediaPlaybackForOperationProvider(operationID, models.MediaProviderPlex, plexFirst); err != nil {
		t.Fatalf("first plex replace failed: %v", err)
	}
	spotify := []models.MediaPlayback{
		makeMediaPlayback(operationID, models.MediaProviderSpotify, "Spotify Song", base.Add(time.Minute)),
	}
	if err := ReplaceMediaPlaybackForOperationProvider(operationID, models.MediaProviderSpotify, spotify); err != nil {
		t.Fatalf("spotify replace failed: %v", err)
	}

	all, err := GetMediaPlaybackForOperation(operationID)
	if err != nil {
		t.Fatalf("get playback failed: %v", err)
	}
	if len(all) != 3 {
		t.Fatalf("expected 3 rows across providers, got %d", len(all))
	}

	// Re-pull Plex with a single (different) track. Delete-and-replace must wipe the
	// old two Plex rows but leave the Spotify row untouched.
	plexSecond := []models.MediaPlayback{
		makeMediaPlayback(operationID, models.MediaProviderPlex, "Song C", base.Add(5*time.Minute)),
	}
	if err := ReplaceMediaPlaybackForOperationProvider(operationID, models.MediaProviderPlex, plexSecond); err != nil {
		t.Fatalf("second plex replace failed: %v", err)
	}

	all, err = GetMediaPlaybackForOperation(operationID)
	if err != nil {
		t.Fatalf("get playback failed: %v", err)
	}
	if len(all) != 2 {
		t.Fatalf("expected 2 rows after re-pull, got %d", len(all))
	}

	var plexTitles, spotifyCount = []string{}, 0
	for _, pb := range all {
		switch pb.Provider {
		case models.MediaProviderPlex:
			plexTitles = append(plexTitles, pb.Title)
		case models.MediaProviderSpotify:
			spotifyCount++
		}
	}
	if len(plexTitles) != 1 || plexTitles[0] != "Song C" {
		t.Errorf("expected only Song C for plex, got %v", plexTitles)
	}
	if spotifyCount != 1 {
		t.Errorf("spotify row should survive a plex re-pull, got %d", spotifyCount)
	}

	// Results are ordered by started_at ascending (timeline order).
	if !all[0].StartedAt.Before(all[1].StartedAt) {
		t.Errorf("expected ascending started_at order, got %v then %v", all[0].StartedAt, all[1].StartedAt)
	}

	// Replacing with an empty slice clears the provider's rows.
	if err := ReplaceMediaPlaybackForOperationProvider(operationID, models.MediaProviderPlex, nil); err != nil {
		t.Fatalf("empty replace failed: %v", err)
	}
	all, _ = GetMediaPlaybackForOperation(operationID)
	if len(all) != 1 {
		t.Fatalf("expected only the spotify row left, got %d", len(all))
	}
}

func TestSetOperationMediaRetrievedAt(t *testing.T) {
	newTestDB(t)
	user := makeTestUser(t, "guard@example.com", nil)
	dayID := seedDay(t, user.ID, time.Now(), true)
	exerciseID := seedExerciseWithID(t, dayID)
	operationID := seedOperationWithGear(t, exerciseID, nil)

	stamp := time.Date(2026, 6, 27, 12, 0, 0, 0, time.UTC)
	if err := SetOperationMediaRetrievedAt(operationID, stamp); err != nil {
		t.Fatalf("stamp failed: %v", err)
	}

	reloaded := models.Operation{}
	if err := Instance.Where("id = ?", operationID).First(&reloaded).Error; err != nil {
		t.Fatalf("reload failed: %v", err)
	}
	if reloaded.MediaRetrievedAt == nil {
		t.Fatalf("expected media_retrieved_at to be set")
	}
	if !reloaded.MediaRetrievedAt.Equal(stamp) {
		t.Errorf("got %v, want %v", reloaded.MediaRetrievedAt, stamp)
	}
}
