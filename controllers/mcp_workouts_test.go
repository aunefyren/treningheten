package controllers

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/aunefyren/treningheten/database"
	"github.com/aunefyren/treningheten/files"
	"github.com/aunefyren/treningheten/models"

	"github.com/google/uuid"
)

// seedBareSessionWithSoundtrack seeds a session with NO operations — what the front page's "+"
// button creates — plus playback rows, and returns the session (exercise) id. A session like
// this has no activity id to address, so the workout id is the only way to reach its
// soundtrack.
func seedBareSessionWithSoundtrack(t *testing.T, userID uuid.UUID, plays []models.MediaPlayback) uuid.UUID {
	t.Helper()

	day := models.ExerciseDay{Date: time.Now(), Enabled: true, UserID: &userID}
	day.ID = uuid.New()
	if err := database.Instance.Omit("User", "Goal").Create(&day).Error; err != nil {
		t.Fatalf("seed day: %v", err)
	}

	exercise := models.Exercise{Enabled: true, IsOn: true, ExerciseDayID: day.ID}
	exercise.ID = uuid.New()
	if err := database.Instance.Omit("ExerciseDay").Create(&exercise).Error; err != nil {
		t.Fatalf("seed exercise: %v", err)
	}

	for i := range plays {
		plays[i].ID = uuid.New()
		plays[i].ExerciseID = exercise.ID
		if err := database.Instance.Omit("Exercise").Create(&plays[i]).Error; err != nil {
			t.Fatalf("seed playback: %v", err)
		}
	}

	return exercise.ID
}

func TestAssembleWorkoutSoundtrackAcceptsWorkoutID(t *testing.T) {
	newControllerTestDB(t)

	prevMedia := files.ConfigFile.Media.Enabled
	files.ConfigFile.Media.Enabled = true
	t.Cleanup(func() { files.ConfigFile.Media.Enabled = prevMedia })

	user := createTestUser(t, "workout-soundtrack@example.com", "Sound")
	base := time.Date(2026, 7, 5, 10, 0, 0, 0, time.UTC)

	workoutID := seedBareSessionWithSoundtrack(t, user.ID, []models.MediaPlayback{
		{Provider: models.MediaProviderPlex, MediaType: models.MediaTypeSong, Title: "Only track", StartedAt: base},
	})

	out, err := assembleWorkoutSoundtrack(user.ID, workoutID)
	if err != nil {
		t.Fatalf("assembleWorkoutSoundtrack by workout id: %v", err)
	}
	if !out.HasSoundtrack || len(out.Tracks) != 1 {
		t.Fatalf("workout id should resolve the session's soundtrack, got %+v", out)
	}
	if out.Tracks[0].Title != "Only track" {
		t.Errorf("track title: %q", out.Tracks[0].Title)
	}
}

func TestAssembleWorkoutSoundtrackRejectsOtherUsersWorkout(t *testing.T) {
	newControllerTestDB(t)

	prevMedia := files.ConfigFile.Media.Enabled
	files.ConfigFile.Media.Enabled = true
	t.Cleanup(func() { files.ConfigFile.Media.Enabled = prevMedia })

	owner := createTestUser(t, "owner-soundtrack@example.com", "Owner")
	other := createTestUser(t, "other-soundtrack@example.com", "Other")
	base := time.Date(2026, 7, 5, 10, 0, 0, 0, time.UTC)

	workoutID := seedBareSessionWithSoundtrack(t, owner.ID, []models.MediaPlayback{
		{Provider: models.MediaProviderPlex, MediaType: models.MediaTypeSong, Title: "Private listen", StartedAt: base},
	})

	// Accepting a second id kind must not widen what a caller can reach.
	if _, err := assembleWorkoutSoundtrack(other.ID, workoutID); err == nil {
		t.Fatal("another user's workout id must not resolve")
	}
}

func TestExplainActivityIDMiss(t *testing.T) {
	newControllerTestDB(t)

	user := createTestUser(t, "idmiss@example.com", "Miss")
	workoutID := seedBareSessionWithSoundtrack(t, user.ID, nil)
	original := errors.New("Record not found.")

	// A workout id passed where an activity id belongs: say so, so the caller can correct it.
	err := explainActivityIDMiss(user.ID, workoutID, original)
	if err == nil || !strings.Contains(err.Error(), "workout id") {
		t.Fatalf("expected a workout-id explanation, got %v", err)
	}

	// An id that is neither keeps the original error.
	if err := explainActivityIDMiss(user.ID, uuid.New(), original); !errors.Is(err, original) {
		t.Errorf("unknown id should keep the original error, got %v", err)
	}
}

func TestSessionFeedItemToWorkout(t *testing.T) {
	hevyID := "hevy-123"
	note := "felt good"
	session := models.SessionFeedItem{
		ExerciseID:       uuid.New(),
		Date:             time.Date(2026, 3, 2, 0, 0, 0, 0, time.UTC),
		Note:             "leg day",
		Distance:         5,
		DistanceUnit:     "km",
		DurationSeconds:  3600,
		TopWeight:        90,
		WeightUnit:       "kg",
		SetCount:         4,
		HevyWorkoutID:    &hevyID,
		CountsTowardGoal: true,
		ActivityCount:    2,
		Activities: []models.ActivityFeedItem{
			{OperationID: uuid.New(), ActionName: "Run", Distance: 5, DistanceUnit: "km", Note: &note},
			{OperationID: uuid.New(), ActionName: "Squat", TopWeight: 90, WeightUnit: "kg"},
		},
	}

	got := sessionFeedItemToWorkout(session)
	if got.ID != session.ExerciseID.String() {
		t.Errorf("workout id should be the session id, got %q", got.ID)
	}
	// Strava wins over Hevy; without a Strava set this one is Hevy-sourced.
	if got.Source != "hevy" {
		t.Errorf("source: got %q, want hevy", got.Source)
	}
	if len(got.Activities) != 2 || got.Activities[0].Action != "Run" {
		t.Fatalf("activities not carried through: %+v", got.Activities)
	}
	if got.DistanceUnit != "km" || got.WeightUnit != "kg" {
		t.Errorf("units should be kept when the metric is present: %+v", got)
	}

	// A workout with no activities is normal, not an error — and its arrays must be empty
	// rather than nil so the JSON reads as [] instead of null.
	bare := sessionFeedItemToWorkout(models.SessionFeedItem{ExerciseID: uuid.New(), DurationSeconds: 1800})
	if bare.Source != "manual" {
		t.Errorf("a bare session is manual, got %q", bare.Source)
	}
	if bare.Activities == nil || len(bare.Activities) != 0 {
		t.Errorf("activities should be an empty slice, got %v", bare.Activities)
	}
	if bare.DistanceUnit != "" || bare.WeightUnit != "" {
		t.Errorf("units should be dropped for zero metrics: %+v", bare)
	}
	if bare.DurationSeconds != 1800 {
		t.Errorf("duration should survive: %d", bare.DurationSeconds)
	}
}

func TestAssembleWorkoutSearchReturnsSessionsWithoutActivities(t *testing.T) {
	newControllerTestDB(t)

	user := createTestUser(t, "workoutsearch@example.com", "Search")
	seedBareSessionWithSoundtrack(t, user.ID, nil)

	workouts, total, hasMore, err := assembleWorkoutSearch(user.ID, models.ActivityFeedFilter{
		Sort: "date", Order: "desc", Limit: 20,
	})
	if err != nil {
		t.Fatalf("assembleWorkoutSearch: %v", err)
	}
	if total != 1 || len(workouts) != 1 {
		t.Fatalf("a session logged without activities must be searchable: total %d / len %d", total, len(workouts))
	}
	if hasMore {
		t.Errorf("has_more should be false on a single-page result")
	}
	if workouts[0].ActivityCount != 0 {
		t.Errorf("activity count: got %d, want 0", workouts[0].ActivityCount)
	}
}
