package database

import (
	"testing"
	"time"

	"github.com/aunefyren/treningheten/models"

	"github.com/google/uuid"
)

// makeMediaPlayback builds a minimal valid MediaPlayback row for a song. Playback
// is keyed to the session (exercise), not the operation.
func makeMediaPlayback(exerciseID uuid.UUID, provider, title string, startedAt time.Time) models.MediaPlayback {
	pb := models.MediaPlayback{
		ExerciseID: exerciseID,
		Provider:   provider,
		MediaType:  models.MediaTypeSong,
		Title:      title,
		StartedAt:  startedAt,
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

func TestReplaceMediaPlaybackForExerciseProvider(t *testing.T) {
	newTestDB(t)
	user := makeTestUser(t, "playback@example.com", nil)
	dayID := seedDay(t, user.ID, time.Now(), true)
	exerciseID := seedExerciseWithID(t, dayID)

	base := time.Date(2026, 6, 27, 10, 0, 0, 0, time.UTC)

	// First pull: two Plex tracks plus one Spotify track (different provider).
	plexFirst := []models.MediaPlayback{
		makeMediaPlayback(exerciseID, models.MediaProviderPlex, "Song A", base),
		makeMediaPlayback(exerciseID, models.MediaProviderPlex, "Song B", base.Add(3*time.Minute)),
	}
	if err := ReplaceMediaPlaybackForExerciseProvider(exerciseID, models.MediaProviderPlex, plexFirst); err != nil {
		t.Fatalf("first plex replace failed: %v", err)
	}
	spotify := []models.MediaPlayback{
		makeMediaPlayback(exerciseID, models.MediaProviderSpotify, "Spotify Song", base.Add(time.Minute)),
	}
	if err := ReplaceMediaPlaybackForExerciseProvider(exerciseID, models.MediaProviderSpotify, spotify); err != nil {
		t.Fatalf("spotify replace failed: %v", err)
	}

	all, err := GetMediaPlaybackForExercise(exerciseID)
	if err != nil {
		t.Fatalf("get playback failed: %v", err)
	}
	if len(all) != 3 {
		t.Fatalf("expected 3 rows across providers, got %d", len(all))
	}

	// Re-pull Plex with a single (different) track. Delete-and-replace must wipe the
	// old two Plex rows but leave the Spotify row untouched.
	plexSecond := []models.MediaPlayback{
		makeMediaPlayback(exerciseID, models.MediaProviderPlex, "Song C", base.Add(5*time.Minute)),
	}
	if err := ReplaceMediaPlaybackForExerciseProvider(exerciseID, models.MediaProviderPlex, plexSecond); err != nil {
		t.Fatalf("second plex replace failed: %v", err)
	}

	all, err = GetMediaPlaybackForExercise(exerciseID)
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

	// Non-destructive empty guard: an empty pull must NOT delete existing rows — it
	// means "nothing new / outside my window" (e.g. a Spotify pull past its ~24h
	// window), not "authoritatively zero". Both providers' rows survive untouched.
	if err := ReplaceMediaPlaybackForExerciseProvider(exerciseID, models.MediaProviderPlex, nil); err != nil {
		t.Fatalf("empty replace failed: %v", err)
	}
	all, _ = GetMediaPlaybackForExercise(exerciseID)
	if len(all) != 2 {
		t.Fatalf("empty pull must preserve existing rows, got %d", len(all))
	}
}

func TestSetExerciseMediaRetrievedAt(t *testing.T) {
	newTestDB(t)
	user := makeTestUser(t, "guard@example.com", nil)
	dayID := seedDay(t, user.ID, time.Now(), true)
	exerciseID := seedExerciseWithID(t, dayID)

	stamp := time.Date(2026, 6, 27, 12, 0, 0, 0, time.UTC)
	if err := SetExerciseMediaRetrievedAt(exerciseID, stamp); err != nil {
		t.Fatalf("stamp failed: %v", err)
	}

	reloaded := models.Exercise{}
	if err := Instance.Where("id = ?", exerciseID).First(&reloaded).Error; err != nil {
		t.Fatalf("reload failed: %v", err)
	}
	if reloaded.MediaRetrievedAt == nil {
		t.Fatalf("expected media_retrieved_at to be set")
	}
	if !reloaded.MediaRetrievedAt.Equal(stamp) {
		t.Errorf("got %v, want %v", reloaded.MediaRetrievedAt, stamp)
	}
}

func TestSetExerciseMediaSettled(t *testing.T) {
	newTestDB(t)
	user := makeTestUser(t, "settled@example.com", nil)
	dayID := seedDay(t, user.ID, time.Now(), true)
	exerciseID := seedExerciseWithID(t, dayID)

	// Defaults to false on a freshly created session.
	reloaded := models.Exercise{}
	if err := Instance.Where("id = ?", exerciseID).First(&reloaded).Error; err != nil {
		t.Fatalf("reload failed: %v", err)
	}
	if reloaded.MediaSettled {
		t.Fatalf("expected media_settled to default to false")
	}

	if err := SetExerciseMediaSettled(exerciseID, true); err != nil {
		t.Fatalf("settle failed: %v", err)
	}
	if err := Instance.Where("id = ?", exerciseID).First(&reloaded).Error; err != nil {
		t.Fatalf("reload failed: %v", err)
	}
	if !reloaded.MediaSettled {
		t.Errorf("expected media_settled to be true after set")
	}
}

func TestGetExercisesForMediaReconcile(t *testing.T) {
	newTestDB(t)
	user := makeTestUser(t, "reconcile@example.com", nil)
	dayID := seedDay(t, user.ID, time.Now(), true)

	// Never-pulled and pulled-but-not-settled sessions are candidates.
	neverPulled := seedExerciseWithID(t, dayID)
	pulledUnsettled := seedExerciseWithID(t, dayID)
	if err := SetExerciseMediaRetrievedAt(pulledUnsettled, time.Now()); err != nil {
		t.Fatalf("stamp failed: %v", err)
	}

	// A settled session is excluded.
	settled := seedExerciseWithID(t, dayID)
	if err := SetExerciseMediaRetrievedAt(settled, time.Now()); err != nil {
		t.Fatalf("stamp failed: %v", err)
	}
	if err := SetExerciseMediaSettled(settled, true); err != nil {
		t.Fatalf("settle failed: %v", err)
	}

	got, err := GetExercisesForMediaReconcile(user.ID, time.Now().Add(-24*time.Hour))
	if err != nil {
		t.Fatalf("reconcile query failed: %v", err)
	}

	ids := map[uuid.UUID]bool{}
	for _, e := range got {
		ids[e.ID] = true
	}
	if !ids[neverPulled] || !ids[pulledUnsettled] {
		t.Errorf("expected never-pulled and pulled-unsettled sessions, got %v", ids)
	}
	if ids[settled] {
		t.Errorf("settled session must be excluded from reconcile scan")
	}

	// A lookback that predates every session's creation returns nothing.
	none, err := GetExercisesForMediaReconcile(user.ID, time.Now().Add(1*time.Hour))
	if err != nil {
		t.Fatalf("reconcile query failed: %v", err)
	}
	if len(none) != 0 {
		t.Errorf("expected no sessions outside the lookback, got %d", len(none))
	}
}
