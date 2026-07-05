package controllers

import (
	"testing"
	"time"

	"github.com/aunefyren/treningheten/database"
	"github.com/aunefyren/treningheten/files"
	"github.com/aunefyren/treningheten/models"

	"github.com/google/uuid"
)

// seedSessionWithSoundtrack seeds an enabled day → exercise → operation owned by the
// user, plus the given MediaPlayback rows attached to the exercise, and returns the
// operation (activity) id. Associations are omitted so rows can be created by id
// without building the full parent graph (FK enforcement is off in the harness).
func seedSessionWithSoundtrack(t *testing.T, userID uuid.UUID, retrievedAt *time.Time, plays []models.MediaPlayback) uuid.UUID {
	t.Helper()

	day := models.ExerciseDay{Date: time.Now(), Enabled: true, UserID: &userID}
	day.ID = uuid.New()
	if err := database.Instance.Omit("User", "Goal").Create(&day).Error; err != nil {
		t.Fatalf("seed day: %v", err)
	}

	exercise := models.Exercise{Enabled: true, IsOn: true, ExerciseDayID: day.ID, MediaRetrievedAt: retrievedAt}
	exercise.ID = uuid.New()
	if err := database.Instance.Omit("ExerciseDay").Create(&exercise).Error; err != nil {
		t.Fatalf("seed exercise: %v", err)
	}

	operation := models.Operation{Enabled: true, ExerciseID: exercise.ID}
	operation.ID = uuid.New()
	if err := database.Instance.Omit("Exercise", "Action").Create(&operation).Error; err != nil {
		t.Fatalf("seed operation: %v", err)
	}

	for i := range plays {
		plays[i].ID = uuid.New()
		plays[i].ExerciseID = exercise.ID
		if err := database.Instance.Omit("Exercise").Create(&plays[i]).Error; err != nil {
			t.Fatalf("seed playback: %v", err)
		}
	}

	return operation.ID
}

func TestAssembleWorkoutSoundtrack(t *testing.T) {
	newControllerTestDB(t)

	prevMedia := files.ConfigFile.Media.Enabled
	files.ConfigFile.Media.Enabled = true
	t.Cleanup(func() { files.ConfigFile.Media.Enabled = prevMedia })

	user := createTestUser(t, "soundtrack@example.com", "Sound")

	base := time.Date(2026, 7, 5, 10, 0, 0, 0, time.UTC)
	retrieved := base.Add(2 * time.Hour)
	artist := "Artist"
	length := int64(210)

	// Seed two plays out of chronological order to prove the tool sorts by start.
	later := base.Add(4 * time.Minute)
	activityID := seedSessionWithSoundtrack(t, user.ID, &retrieved, []models.MediaPlayback{
		{Provider: models.MediaProviderSpotify, MediaType: models.MediaTypeSong, Title: "Second", Artist: &artist, StartedAt: later, TrackLength: &length},
		{Provider: models.MediaProviderPlex, MediaType: models.MediaTypeSong, Title: "First", StartedAt: base},
	})

	out, err := assembleWorkoutSoundtrack(user.ID, activityID)
	if err != nil {
		t.Fatalf("assembleWorkoutSoundtrack: %v", err)
	}
	if !out.HasSoundtrack {
		t.Fatal("expected HasSoundtrack=true")
	}
	if len(out.Tracks) != 2 {
		t.Fatalf("expected 2 tracks, got %d", len(out.Tracks))
	}
	if out.Tracks[0].Title != "First" || out.Tracks[1].Title != "Second" {
		t.Fatalf("expected play order First,Second; got %s,%s", out.Tracks[0].Title, out.Tracks[1].Title)
	}
	if out.Tracks[1].Provider != models.MediaProviderSpotify || out.Tracks[1].Artist == nil || *out.Tracks[1].Artist != artist {
		t.Fatalf("field mapping wrong for second track: %+v", out.Tracks[1])
	}
	if out.Tracks[1].TrackLengthSeconds == nil || *out.Tracks[1].TrackLengthSeconds != length {
		t.Fatalf("expected track length %d, got %v", length, out.Tracks[1].TrackLengthSeconds)
	}
	if out.RetrievedAt == nil || !out.RetrievedAt.Equal(retrieved) {
		t.Fatalf("expected RetrievedAt %v, got %v", retrieved, out.RetrievedAt)
	}
}

func TestAssembleWorkoutSoundtrackEmptyAndDisabled(t *testing.T) {
	newControllerTestDB(t)

	prevMedia := files.ConfigFile.Media.Enabled
	t.Cleanup(func() { files.ConfigFile.Media.Enabled = prevMedia })

	user := createTestUser(t, "empty@example.com", "Empty")

	// Enabled but no plays matched → soft "no soundtrack", not an error.
	files.ConfigFile.Media.Enabled = true
	activityID := seedSessionWithSoundtrack(t, user.ID, nil, nil)

	out, err := assembleWorkoutSoundtrack(user.ID, activityID)
	if err != nil {
		t.Fatalf("assembleWorkoutSoundtrack (empty): %v", err)
	}
	if out.HasSoundtrack || len(out.Tracks) != 0 || out.Message == "" {
		t.Fatalf("expected empty soft response, got %+v", out)
	}
	if exerciseHasSoundtrack(uuid.New()) {
		t.Fatal("exerciseHasSoundtrack should be false for a session with no plays")
	}

	// Disabled → soft response without touching the DB; presence check is false.
	files.ConfigFile.Media.Enabled = false
	out, err = assembleWorkoutSoundtrack(user.ID, activityID)
	if err != nil {
		t.Fatalf("assembleWorkoutSoundtrack (disabled): %v", err)
	}
	if out.HasSoundtrack || out.Message == "" {
		t.Fatalf("expected disabled soft response, got %+v", out)
	}
}
