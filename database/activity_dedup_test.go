package database

import (
	"errors"
	"testing"
	"time"

	"github.com/aunefyren/treningheten/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// seedStravaExercise inserts an enabled, on session holding one operation per given Strava
// id (two ids make a combined session) at the given time.
func seedStravaExercise(t *testing.T, dayID uuid.UUID, at time.Time, stravaIDs ...string) models.Exercise {
	t.Helper()
	session := makeSession(t, dayID, at)
	for _, stravaID := range stravaIDs {
		op := makeOperation(t, session.ID, nil)
		seedOperationStravaSet(t, op.ID, stravaID)
	}
	return session
}

func exerciseIDs(exercises []models.Exercise) map[uuid.UUID]bool {
	ids := map[uuid.UUID]bool{}
	for _, exercise := range exercises {
		ids[exercise.ID] = true
	}
	return ids
}

func TestGetImportedExercises(t *testing.T) {
	newTestDB(t)

	user := makeTestUser(t, "imported@example.com", nil)
	other := makeTestUser(t, "imported-other@example.com", nil)
	day := makeDay(t, user.ID, time.Now())
	otherDay := makeDay(t, other.ID, time.Now())
	base := time.Date(2024, 6, 15, 12, 0, 0, 0, time.UTC)

	hevy := seedHevyExercise(t, day.ID, "hw-near", base.Add(5*time.Minute))
	strava := seedStravaExercise(t, day.ID, base.Add(-10*time.Minute), "1001")
	far := seedHevyExercise(t, day.ID, "hw-far", base.Add(3*time.Hour))
	manual := makeSession(t, day.ID, base)
	off := seedHevyExercise(t, day.ID, "hw-off", base)
	if err := Instance.Model(&off).Update("is_on", false).Error; err != nil {
		t.Fatal(err)
	}
	disabled := seedHevyExercise(t, day.ID, "hw-disabled", base)
	disableRow(t, &models.Exercise{}, disabled.ID)
	stranger := seedHevyExercise(t, otherDay.ID, "hw-stranger", base)

	t.Run("near time", func(t *testing.T) {
		got, err := GetImportedExercisesForUserNearTime(user.ID, base, 15*time.Minute)
		if err != nil {
			t.Fatalf("GetImportedExercisesForUserNearTime returned error: %v", err)
		}
		ids := exerciseIDs(got)
		if len(got) != 2 || !ids[hevy.ID] || !ids[strava.ID] {
			t.Errorf("got %d sessions, want only the nearby Hevy and Strava imports", len(got))
		}
		if got[0].ID != strava.ID {
			t.Errorf("sessions not ordered by time")
		}
	})

	t.Run("whole history", func(t *testing.T) {
		got, err := GetImportedExercisesForUser(user.ID)
		if err != nil {
			t.Fatalf("GetImportedExercisesForUser returned error: %v", err)
		}
		ids := exerciseIDs(got)
		if len(got) != 3 || !ids[hevy.ID] || !ids[strava.ID] || !ids[far.ID] {
			t.Errorf("got %d sessions, want the three imports", len(got))
		}
		for _, excluded := range []uuid.UUID{manual.ID, off.ID, disabled.ID, stranger.ID} {
			if ids[excluded] {
				t.Errorf("session %v should not be a duplicate candidate", excluded)
			}
		}
	})
}

func TestCountStravaActivitiesForExercises(t *testing.T) {
	newTestDB(t)

	user := makeTestUser(t, "stravacount@example.com", nil)
	day := makeDay(t, user.ID, time.Now())
	single := seedStravaExercise(t, day.ID, time.Now(), "2001")
	combined := seedStravaExercise(t, day.ID, time.Now(), "2002", "2003")
	manual := makeSession(t, day.ID, time.Now())

	// Pad past one chunk so the second IN-list query runs too.
	ids := []uuid.UUID{}
	for range stravaActivityCountChunk {
		ids = append(ids, uuid.New())
	}
	ids = append(ids, single.ID, combined.ID, manual.ID)

	counts, err := CountStravaActivitiesForExercises(ids)
	if err != nil {
		t.Fatalf("CountStravaActivitiesForExercises returned error: %v", err)
	}
	if counts[single.ID] != 1 || counts[combined.ID] != 2 {
		t.Errorf("counts = single %d, combined %d; want 1 and 2", counts[single.ID], counts[combined.ID])
	}
	if _, found := counts[manual.ID]; found {
		t.Errorf("a manual session should be absent from the counts")
	}

	empty, err := CountStravaActivitiesForExercises(nil)
	if err != nil || len(empty) != 0 {
		t.Errorf("empty input: got %v, %v; want an empty map", empty, err)
	}
}

func TestSupersededExercises(t *testing.T) {
	newTestDB(t)

	user := makeTestUser(t, "superseded@example.com", nil)
	day := makeDay(t, user.ID, time.Now())
	winner := seedHevyExercise(t, day.ID, "hw-winner", time.Now())
	nextWinner := seedHevyExercise(t, day.ID, "hw-next", time.Now())
	strava := seedStravaExercise(t, day.ID, time.Now(), "3001")
	hevy := seedHevyExercise(t, day.ID, "hw-loser", time.Now())

	for _, loser := range []uuid.UUID{strava.ID, hevy.ID} {
		if err := SetExerciseSupersededBy(loser, &winner.ID); err != nil {
			t.Fatalf("SetExerciseSupersededBy returned error: %v", err)
		}
	}

	if visible, _ := GetExerciseForUserWithStravaID(user.ID, "3001"); visible != nil {
		t.Errorf("a hidden duplicate is still found by the regular Strava lookup")
	}
	hiddenStrava, err := GetSupersededExerciseForUserWithStravaID(user.ID, "3001")
	if err != nil || hiddenStrava == nil || hiddenStrava.ID != strava.ID {
		t.Fatalf("GetSupersededExerciseForUserWithStravaID = %v, %v; want the hidden session", hiddenStrava, err)
	}
	if hiddenStrava.Enabled || hiddenStrava.SupersededByID == nil || *hiddenStrava.SupersededByID != winner.ID {
		t.Errorf("hidden session should be disabled and point at its winner")
	}
	hiddenHevy, err := GetSupersededExerciseForUserWithHevyWorkoutID(user.ID, "hw-loser")
	if err != nil || hiddenHevy == nil || hiddenHevy.ID != hevy.ID {
		t.Errorf("GetSupersededExerciseForUserWithHevyWorkoutID = %v, %v; want the hidden session", hiddenHevy, err)
	}

	all, err := GetSupersededExercisesForUser(user.ID)
	if err != nil || len(all) != 2 {
		t.Errorf("GetSupersededExercisesForUser = %d, %v; want 2", len(all), err)
	}

	if err := RepointSupersededExercises(winner.ID, nextWinner.ID); err != nil {
		t.Fatalf("RepointSupersededExercises returned error: %v", err)
	}
	if moved, _ := GetExercisesSupersededBy(winner.ID); len(moved) != 0 {
		t.Errorf("repointing left %d sessions behind the old winner", len(moved))
	}
	behindNext, err := GetExercisesSupersededBy(nextWinner.ID)
	if err != nil || len(behindNext) != 2 {
		t.Errorf("GetExercisesSupersededBy = %d, %v; want 2", len(behindNext), err)
	}

	if err := SetExerciseSupersededBy(strava.ID, nil); err != nil {
		t.Fatalf("restoring returned error: %v", err)
	}
	restored, err := GetExerciseForUserWithStravaID(user.ID, "3001")
	if err != nil || restored == nil || restored.SupersededByID != nil {
		t.Errorf("restored session = %v, %v; want it visible with no mark", restored, err)
	}
	if hidden, _ := GetSupersededExerciseForUserWithStravaID(user.ID, "3001"); hidden != nil {
		t.Errorf("a restored session is still reported hidden")
	}
	if hidden, _ := GetSupersededExerciseForUserWithHevyWorkoutID(user.ID, "nope"); hidden != nil {
		t.Errorf("an unknown Hevy workout is reported hidden")
	}
}

func TestMigrateStravaSkipHevyToSourcePriority(t *testing.T) {
	newTestDB(t) // runs Migrate() (and this backfill) once on empty data — a no-op

	skipper := makeTestUser(t, "skipper@example.com", nil)
	chooser := makeTestUser(t, "chooser@example.com", func(user *models.User) { user.ActivitySourcePriority = strPtr("strava,hevy") })
	keeper := makeTestUser(t, "keeper@example.com", nil)
	// Force the legacy flag on (explicit update sidesteps the default:false).
	if err := Instance.Model(&models.User{}).Where("id IN ?", []uuid.UUID{skipper.ID, chooser.ID}).Update("strava_skip_hevy_duplicates", true).Error; err != nil {
		t.Fatalf("failed to set strava_skip_hevy_duplicates: %v", err)
	}

	migrateStravaSkipHevyToSourcePriority()
	migrateStravaSkipHevyToSourcePriority() // self-limiting: a second run changes nothing

	cases := []struct {
		name string
		user models.User
		want *string
	}{
		{"flag on gets Hevy first", skipper, strPtr("hevy,strava")},
		{"an existing ranking is kept", chooser, strPtr("strava,hevy")},
		{"flag off stays unset", keeper, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			stored, err := GetAllUserInformation(tc.user.ID)
			if err != nil {
				t.Fatal(err)
			}
			if (stored.ActivitySourcePriority == nil) != (tc.want == nil) ||
				(tc.want != nil && *stored.ActivitySourcePriority != *tc.want) {
				t.Errorf("priority = %v, want %v", stored.ActivitySourcePriority, tc.want)
			}
			if stored.StravaSkipHevyDuplicates != nil && *stored.StravaSkipHevyDuplicates {
				t.Errorf("the legacy flag was not cleared")
			}
		})
	}
}

// failNthStatement makes the Nth query or update from now on fail (0 only counts), and
// returns a function reporting how many it saw.
func failNthStatement(t *testing.T, failAt int) func() int {
	t.Helper()
	seen := 0
	hook := func(tx *gorm.DB) {
		seen++
		if seen == failAt {
			_ = tx.AddError(errors.New("injected database fault"))
		}
	}
	callbacks := Instance.Callback()
	if err := callbacks.Query().Before("gorm:query").Register("fault:query", hook); err != nil {
		t.Fatal(err)
	}
	if err := callbacks.Update().Before("gorm:update").Register("fault:update", hook); err != nil {
		t.Fatal(err)
	}
	return func() int { return seen }
}

// TestMigrateStravaSkipHevyToSourcePriorityFaults fails each statement of the backfill in
// turn: it must log and stop, never panic, and never clear the legacy flag of a user whose
// ranking was not written.
func TestMigrateStravaSkipHevyToSourcePriorityFaults(t *testing.T) {
	seed := func(t *testing.T) models.User {
		newTestDB(t)
		user := makeTestUser(t, "faulty-skipper@example.com", nil)
		if err := Instance.Model(&models.User{}).Where("id = ?", user.ID).Update("strava_skip_hevy_duplicates", true).Error; err != nil {
			t.Fatal(err)
		}
		return user
	}

	seed(t)
	statements := failNthStatement(t, 0)
	migrateStravaSkipHevyToSourcePriority()
	total := statements()

	for failAt := 1; failAt <= total; failAt++ {
		user := seed(t)
		failNthStatement(t, failAt)
		migrateStravaSkipHevyToSourcePriority()

		stored := models.User{}
		if err := Instance.Where("id = ?", user.ID).First(&stored).Error; err != nil {
			t.Fatal(err)
		}
		if stored.ActivitySourcePriority == nil && stored.StravaSkipHevyDuplicates != nil && !*stored.StravaSkipHevyDuplicates {
			t.Errorf("statement %d failing cleared the flag without writing a ranking", failAt)
		}
	}
}
