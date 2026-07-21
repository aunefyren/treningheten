package database

import (
	"testing"

	"github.com/aunefyren/treningheten/models"

	"github.com/google/uuid"
)

func TestRegisterUserDefaultsObservedMaxHR(t *testing.T) {
	newTestDB(t)

	user := makeTestUser(t, "default@example.com", nil)
	if user.ObservedMaxHeartrate == nil || *user.ObservedMaxHeartrate != 0 {
		t.Fatalf("new user observed max = %v, want a concrete 0", user.ObservedMaxHeartrate)
	}
}

func TestBumpObservedMaxHeartrate(t *testing.T) {
	newTestDB(t)
	user := makeTestUser(t, "bump@example.com", nil)

	get := func() *int {
		var u models.User
		if err := Instance.First(&u, "id = ?", user.ID).Error; err != nil {
			t.Fatalf("failed to reload user: %v", err)
		}
		return u.ObservedMaxHeartrate
	}

	// Raises from the default 0.
	if err := BumpObservedMaxHeartrate(user.ID, 180); err != nil {
		t.Fatalf("bump 180: %v", err)
	}
	if v := get(); v == nil || *v != 180 {
		t.Fatalf("after bump 180, observed = %v", v)
	}

	// A lower candidate is ignored.
	if err := BumpObservedMaxHeartrate(user.ID, 150); err != nil {
		t.Fatalf("bump 150: %v", err)
	}
	if v := get(); *v != 180 {
		t.Fatalf("after lower bump, observed = %v, want 180 unchanged", *v)
	}

	// A higher candidate raises it.
	if err := BumpObservedMaxHeartrate(user.ID, 192); err != nil {
		t.Fatalf("bump 192: %v", err)
	}
	if v := get(); *v != 192 {
		t.Fatalf("after bump 192, observed = %v", *v)
	}

	// Zero/negative candidates are no-ops.
	if err := BumpObservedMaxHeartrate(user.ID, 0); err != nil {
		t.Fatalf("bump 0: %v", err)
	}
	if v := get(); *v != 192 {
		t.Fatalf("after bump 0, observed = %v, want 192 unchanged", *v)
	}
}

func TestBackfillObservedMaxHeartrate(t *testing.T) {
	newTestDB(t)

	// A legacy user (observed max NULL) with a stored stream peaking at 188 bpm, and a
	// second legacy user with no activities at all.
	withStreams := makeTestUser(t, "legacy-hr@example.com", nil)
	noActivity := makeTestUser(t, "legacy-none@example.com", nil)
	nullify(t, withStreams.ID)
	nullify(t, noActivity.ID)

	seedStreamActivity(t, withStreams.ID, []int{120, 188, 300, 170}) // 300 is a dropout, ignored

	backfillObservedMaxHeartrate()

	if v := reloadObserved(t, withStreams.ID); v == nil || *v != 188 {
		t.Fatalf("user with streams: observed = %v, want 188", v)
	}
	// The activity-less user is marked processed (0), not left NULL.
	if v := reloadObserved(t, noActivity.ID); v == nil || *v != 0 {
		t.Fatalf("user without activity: observed = %v, want 0", v)
	}

	// Idempotent: a second run doesn't lower or disturb the computed value.
	backfillObservedMaxHeartrate()
	if v := reloadObserved(t, withStreams.ID); v == nil || *v != 188 {
		t.Fatalf("after second backfill: observed = %v, want 188", v)
	}
}

// nullify forces a user's observed max back to NULL, simulating a legacy row.
func nullify(t *testing.T, userID uuid.UUID) {
	t.Helper()
	if err := Instance.Exec("UPDATE users SET observed_max_heartrate = NULL WHERE id = ?", userID).Error; err != nil {
		t.Fatalf("failed to null observed max: %v", err)
	}
}

func reloadObserved(t *testing.T, userID uuid.UUID) *int {
	t.Helper()
	var u models.User
	if err := Instance.First(&u, "id = ?", userID).Error; err != nil {
		t.Fatalf("failed to reload user: %v", err)
	}
	return u.ObservedMaxHeartrate
}

// seedStreamActivity builds the day → exercise → operation → set-with-streams chain that
// links a heart-rate stream to a user, so the backfill's join can find it.
func seedStreamActivity(t *testing.T, userID uuid.UUID, hr []int) {
	t.Helper()

	day := models.ExerciseDay{UserID: &userID}
	day.ID = uuid.New()
	if err := Instance.Create(&day).Error; err != nil {
		t.Fatalf("create exercise day: %v", err)
	}

	exercise := models.Exercise{ExerciseDayID: day.ID}
	exercise.ID = uuid.New()
	if err := Instance.Create(&exercise).Error; err != nil {
		t.Fatalf("create exercise: %v", err)
	}

	operation := models.Operation{ExerciseID: exercise.ID, Type: "moving"}
	operation.ID = uuid.New()
	if err := Instance.Create(&operation).Error; err != nil {
		t.Fatalf("create operation: %v", err)
	}

	set := models.OperationSet{
		OperationID: operation.ID,
		StravaStreams: &models.StravaStreamsJSON{StravaActivityStreams: models.StravaActivityStreams{
			Heartrate: &models.StravaStream[int]{Data: hr},
		}},
	}
	set.ID = uuid.New()
	if err := Instance.Create(&set).Error; err != nil {
		t.Fatalf("create operation set: %v", err)
	}
}
