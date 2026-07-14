package database

import (
	"testing"
	"time"

	"github.com/aunefyren/treningheten/models"

	"github.com/google/uuid"
)

// seedHevyExercise inserts an enabled, on Hevy-imported exercise (carrying a Hevy workout id)
// at the given time and returns it.
func seedHevyExercise(t *testing.T, dayID uuid.UUID, hevyID string, at time.Time) models.Exercise {
	t.Helper()
	ex := models.Exercise{ExerciseDayID: dayID, Enabled: true, IsOn: true, Time: &at, HevyWorkoutID: &hevyID}
	ex.ID = uuid.New()
	insertRow(t, &ex)
	return ex
}

func TestCreateAndGetExerciseByExerciseDayID(t *testing.T) {
	newTestDB(t)

	user := makeTestUser(t, "exbyday@example.com", nil)
	day := makeDay(t, user.ID, time.Now())

	ex := models.Exercise{ExerciseDayID: day.ID, Enabled: true, IsOn: true}
	ex.ID = uuid.New()
	if err := CreateExerciseForExerciseDayInDatabase(ex); err != nil {
		t.Fatalf("CreateExerciseForExerciseDayInDatabase returned error: %v", err)
	}

	// A disabled exercise on the same day must be excluded.
	seedExercise(t, day.ID, false, true)

	exercises, err := GetExerciseByExerciseDayID(day.ID)
	if err != nil {
		t.Fatalf("GetExerciseByExerciseDayID returned error: %v", err)
	}
	if len(exercises) != 1 {
		t.Errorf("got %d exercises, want 1 (enabled only)", len(exercises))
	}
}

func TestGetOnExerciseCountsForExerciseDayIDs(t *testing.T) {
	newTestDB(t)

	user := makeTestUser(t, "oncounts@example.com", nil)
	day1 := makeDay(t, user.ID, time.Now())
	day2 := makeDay(t, user.ID, time.Now())

	// day1: 2 on + 1 off → 2.
	seedExercise(t, day1.ID, true, true)
	seedExercise(t, day1.ID, true, true)
	seedExercise(t, day1.ID, true, false)
	// day2: 1 on.
	seedExercise(t, day2.ID, true, true)

	counts, err := GetOnExerciseCountsForExerciseDayIDs([]uuid.UUID{day1.ID, day2.ID})
	if err != nil {
		t.Fatalf("GetOnExerciseCountsForExerciseDayIDs returned error: %v", err)
	}
	if counts[day1.ID] != 2 {
		t.Errorf("day1 count: got %d, want 2", counts[day1.ID])
	}
	if counts[day2.ID] != 1 {
		t.Errorf("day2 count: got %d, want 1", counts[day2.ID])
	}

	empty, err := GetOnExerciseCountsForExerciseDayIDs(nil)
	if err != nil {
		t.Fatalf("GetOnExerciseCountsForExerciseDayIDs(nil) returned error: %v", err)
	}
	if len(empty) != 0 {
		t.Errorf("no ids: got %d entries, want 0", len(empty))
	}
}

func TestTurnExerciseOnOff(t *testing.T) {
	newTestDB(t)

	user := makeTestUser(t, "turnonoff@example.com", nil)
	day := makeDay(t, user.ID, time.Now())
	session := makeSession(t, day.ID, time.Now())

	if err := UpdateExerciseByTurningOffByExerciseID(session.ID); err != nil {
		t.Fatalf("UpdateExerciseByTurningOffByExerciseID returned error: %v", err)
	}
	// GetExerciseByIDAndUserID requires is_on = 1, so an off exercise is not returned.
	off, err := GetExerciseByIDAndUserID(session.ID, user.ID)
	if err != nil {
		t.Fatalf("GetExerciseByIDAndUserID returned error: %v", err)
	}
	// On no match the query leaves the destination untouched, so callers get an empty (zero-ID) record.
	if off != nil && off.ID != uuid.Nil {
		t.Errorf("expected empty record for an off exercise, got %v", off.ID)
	}

	if err := UpdateExerciseByTurningOnByExerciseID(session.ID); err != nil {
		t.Fatalf("UpdateExerciseByTurningOnByExerciseID returned error: %v", err)
	}
	on, err := GetExerciseByIDAndUserID(session.ID, user.ID)
	if err != nil {
		t.Fatalf("GetExerciseByIDAndUserID returned error: %v", err)
	}
	if on == nil || on.ID != session.ID {
		t.Errorf("expected the exercise back on, got %v", on)
	}

	// Unknown id → error.
	if err := UpdateExerciseByTurningOnByExerciseID(uuid.New()); err == nil {
		t.Errorf("expected error turning on an unknown exercise")
	}
}

func TestGetExerciseByIDAndUserVariants(t *testing.T) {
	newTestDB(t)

	user := makeTestUser(t, "exbyiduser@example.com", nil)
	stranger := makeTestUser(t, "exbyidstranger@example.com", nil)
	day := makeDay(t, user.ID, time.Now())
	off := makeSessionOff(t, day.ID, time.Now())

	// GetExerciseByIDAndUserID filters is_on, so an off exercise is nil...
	byID, err := GetExerciseByIDAndUserID(off.ID, user.ID)
	if err != nil {
		t.Fatalf("GetExerciseByIDAndUserID returned error: %v", err)
	}
	if byID != nil && byID.ID != uuid.Nil {
		t.Errorf("expected empty record for off exercise from GetExerciseByIDAndUserID, got %v", byID.ID)
	}

	// ...but GetAllExerciseByIDAndUserID does not filter is_on, so it is found.
	all, err := GetAllExerciseByIDAndUserID(off.ID, user.ID)
	if err != nil {
		t.Fatalf("GetAllExerciseByIDAndUserID returned error: %v", err)
	}
	if all.ID != off.ID {
		t.Errorf("expected to find off exercise, got %v", all.ID)
	}

	// Scoped to the owner: another user gets an error.
	if _, err := GetAllExerciseByIDAndUserID(off.ID, stranger.ID); err == nil {
		t.Errorf("expected error reading another user's exercise")
	}
}

func TestCreateAndUpdateExerciseInDB(t *testing.T) {
	newTestDB(t)

	user := makeTestUser(t, "exupdate@example.com", nil)
	day := makeDay(t, user.ID, time.Now())

	ex := models.Exercise{ExerciseDayID: day.ID, Enabled: true, IsOn: true, Note: "before"}
	ex.ID = uuid.New()
	created, err := CreateExerciseInDB(ex)
	if err != nil {
		t.Fatalf("CreateExerciseInDB returned error: %v", err)
	}

	created.Note = "after"
	updated, err := UpdateExerciseInDB(created)
	if err != nil {
		t.Fatalf("UpdateExerciseInDB returned error: %v", err)
	}
	if updated.Note != "after" {
		t.Errorf("note: got %q, want %q", updated.Note, "after")
	}
}

func TestGetExerciseForUserWithStravaAndHevyID(t *testing.T) {
	newTestDB(t)

	user := makeTestUser(t, "exstravahevy@example.com", nil)
	day := makeDay(t, user.ID, time.Now())

	// Strava-sourced session: an operation with a Strava-tagged set.
	stravaSession := makeSession(t, day.ID, time.Now())
	op := makeOperation(t, stravaSession.ID, nil)
	seedOperationStravaSet(t, op.ID, "556677")

	foundStrava, err := GetExerciseForUserWithStravaID(user.ID, "556677")
	if err != nil {
		t.Fatalf("GetExerciseForUserWithStravaID returned error: %v", err)
	}
	if foundStrava == nil || foundStrava.ID != stravaSession.ID {
		t.Errorf("expected to find session by strava id")
	}

	// Hevy-imported session.
	hevySession := seedHevyExercise(t, day.ID, "hw-123", time.Now())
	foundHevy, err := GetExerciseForUserWithHevyWorkoutID(user.ID, "hw-123")
	if err != nil {
		t.Fatalf("GetExerciseForUserWithHevyWorkoutID returned error: %v", err)
	}
	if foundHevy == nil || foundHevy.ID != hevySession.ID {
		t.Errorf("expected to find session by hevy workout id")
	}

	// Unknown ids → nil.
	if got, _ := GetExerciseForUserWithHevyWorkoutID(user.ID, "nope"); got != nil {
		t.Errorf("expected nil for unknown hevy id, got %v", got)
	}
}

func TestGetExercisesForUserNearTime(t *testing.T) {
	newTestDB(t)

	user := makeTestUser(t, "neartime@example.com", nil)
	day := makeDay(t, user.ID, time.Now())
	base := time.Date(2024, 6, 15, 12, 0, 0, 0, time.UTC)
	window := time.Hour

	// A Hevy session near base, and a Strava session near base.
	hevySession := seedHevyExercise(t, day.ID, "hw-near", base.Add(10*time.Minute))
	stravaSession := makeSession(t, day.ID, base.Add(-15*time.Minute))
	op := makeOperation(t, stravaSession.ID, nil)
	seedOperationStravaSet(t, op.ID, "998877")

	nearHevy, err := GetHevyExerciseForUserNearTime(user.ID, base, window)
	if err != nil {
		t.Fatalf("GetHevyExerciseForUserNearTime returned error: %v", err)
	}
	if nearHevy == nil || nearHevy.ID != hevySession.ID {
		t.Errorf("expected to find hevy session near time")
	}

	nearStrava, err := GetStravaExerciseForUserNearTime(user.ID, base, window)
	if err != nil {
		t.Fatalf("GetStravaExerciseForUserNearTime returned error: %v", err)
	}
	if nearStrava == nil || nearStrava.ID != stravaSession.ID {
		t.Errorf("expected to find strava session near time")
	}

	// Far away in time → no match.
	far := base.Add(5 * time.Hour)
	if got, _ := GetHevyExerciseForUserNearTime(user.ID, far, window); got != nil {
		t.Errorf("expected no hevy session far from window, got %v", got)
	}
}

func TestGetAllExerciseDaysWithExerciseByUserID(t *testing.T) {
	newTestDB(t)

	user := makeTestUser(t, "dayswith@example.com", nil)

	// Day with an on exercise → included.
	onDay := makeDay(t, user.ID, time.Now())
	seedExercise(t, onDay.ID, true, true)

	// Day with only an off exercise → excluded.
	offDay := makeDay(t, user.ID, time.Now())
	seedExercise(t, offDay.ID, true, false)

	days, err := GetAllExerciseDaysWithExerciseByUserID(user.ID)
	if err != nil {
		t.Fatalf("GetAllExerciseDaysWithExerciseByUserID returned error: %v", err)
	}
	if len(days) != 1 || days[0].ID != onDay.ID {
		t.Errorf("got %d days, want 1 (only the day with an on exercise)", len(days))
	}
}

func TestGetStravaExercisesByUserID(t *testing.T) {
	newTestDB(t)

	user := makeTestUser(t, "stravaex@example.com", nil)
	day := makeDay(t, user.ID, time.Now())

	// Session with a Strava-tagged set → included.
	stravaSession := makeSession(t, day.ID, time.Now())
	op := makeOperation(t, stravaSession.ID, nil)
	seedOperationStravaSet(t, op.ID, "424242")

	// Session without any Strava set → excluded.
	plain := makeSession(t, day.ID, time.Now())
	plainOp := makeOperation(t, plain.ID, nil)
	makeSet(t, plainOp.ID, f64Ptr(5), nil, nil, nil)

	exercises, err := GetStravaExercisesByUserID(user.ID)
	if err != nil {
		t.Fatalf("GetStravaExercisesByUserID returned error: %v", err)
	}
	if len(exercises) != 1 || exercises[0].ID != stravaSession.ID {
		t.Errorf("got %d strava exercises, want 1", len(exercises))
	}
}
