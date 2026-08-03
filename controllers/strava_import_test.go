package controllers

import (
	"testing"
	"time"

	"github.com/aunefyren/treningheten/database"
	"github.com/aunefyren/treningheten/models"

	"github.com/google/uuid"
)

// excludeActivityType flags one activity type as not counting toward the user's goal, by
// resolving it the way the settings screen does — through its Strava name.
func excludeActivityType(t *testing.T, userID uuid.UUID, stravaName string) models.Action {
	t.Helper()

	action, err := database.GetActionByStravaName(stravaName)
	if err != nil {
		t.Fatalf("GetActionByStravaName(%q) returned error: %v", stravaName, err)
	}
	if action == nil {
		t.Fatalf("no seeded action for Strava name %q", stravaName)
	}
	if err := database.UpsertActivityGoalSettingInDB(userID, action.ID, false); err != nil {
		t.Fatalf("failed to store activity goal setting: %v", err)
	}
	return *action
}

// TestStravaActivityCountsTowardGoal covers the per-activity-type opt-out applied when a
// Strava activity is imported: an excluded sport type must not count, everything else must,
// and an unresolvable type fails open (silently dropping a real workout from the goal is
// worse than honouring a missed opt-out).
func TestStravaActivityCountsTowardGoal(t *testing.T) {
	newControllerTestDB(t)
	database.SeedActions()

	user := createTestUser(t, "stravacounts@example.com", "Strava")
	excludeActivityType(t, user.ID, "Walk")

	tests := []struct {
		name      string
		sportType string
		want      bool
	}{
		{"excluded type does not count", "Walk", false},
		{"other types still count", "Run", true},
		{"unknown sport type fails open", "Kitesurf", true},
		{"empty sport type fails open", "", true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := stravaActivityCountsTowardGoal(user.ID, test.sportType); got != test.want {
				t.Errorf("stravaActivityCountsTowardGoal(%q) = %v, want %v", test.sportType, got, test.want)
			}
		})
	}

	// The opt-out is per user: another user with no settings is unaffected by this one's.
	other := createTestUser(t, "stravacounts-other@example.com", "Other")
	if !stravaActivityCountsTowardGoal(other.ID, "Walk") {
		t.Errorf("one user's exclusion leaked onto another user")
	}
}

// TestStravaImportPersistsGoalCountingOptOut is the regression test for an excluded Strava
// activity still counting toward the weekly goal in production. It walks the exact write the
// importer performs for a fresh session — build the Exercise, Save it, then persist the
// goal-counting decision — and asserts the stored row really reads false.
//
// Setting the field on the struct is not enough: it carries a `default:true` tag, so GORM
// omits the false zero value from the INSERT and the column default wins. That is what
// SetExerciseCountsTowardGoal exists to prevent, and this test fails without it.
func TestStravaImportPersistsGoalCountingOptOut(t *testing.T) {
	newControllerTestDB(t)
	database.SeedActions()

	user := createTestUser(t, "stravapersist@example.com", "Strava")
	excludeActivityType(t, user.ID, "Walk")

	now := time.Now()
	day := models.ExerciseDay{Date: now, Enabled: true, UserID: &user.ID}
	day.ID = uuid.New()
	if err := database.CreateExerciseDayInDB(day); err != nil {
		t.Fatalf("failed to create exercise day: %v", err)
	}

	countsTowardGoal := stravaActivityCountsTowardGoal(user.ID, "Walk")
	if countsTowardGoal {
		t.Fatalf("the walk was expected to be excluded before it is even stored")
	}

	exercise := models.Exercise{ExerciseDayID: day.ID, Enabled: true, IsOn: true, Time: &now}
	exercise.ID = uuid.New()
	exercise.CountsTowardGoal = countsTowardGoal

	stored, err := database.UpdateExerciseInDB(exercise)
	if err != nil {
		t.Fatalf("UpdateExerciseInDB returned error: %v", err)
	}
	if err := database.SetExerciseCountsTowardGoal(stored.ID, countsTowardGoal); err != nil {
		t.Fatalf("SetExerciseCountsTowardGoal returned error: %v", err)
	}

	reloaded, err := database.GetExerciseByIDAndUserID(stored.ID, user.ID)
	if err != nil {
		t.Fatalf("GetExerciseByIDAndUserID returned error: %v", err)
	}
	if reloaded == nil {
		t.Fatalf("imported session not found")
	}
	if reloaded.CountsTowardGoal {
		t.Errorf("an excluded walk was stored as counting toward the goal")
	}

	// And the session must be excluded from the goal-counting read path, not merely flagged.
	object, err := ConvertExerciseToExerciseObject(*reloaded)
	if err != nil {
		t.Fatalf("ConvertExerciseToExerciseObject returned error: %v", err)
	}
	if exerciseCountsTowardGoal(object) {
		t.Errorf("an excluded walk still counts toward the goal on the read path")
	}
}

// TestStravaImportGoalCountingSurvivesResync covers that the opt-out is snapshotted at
// import only: a re-sync of an existing session must not overwrite a manual builder toggle
// in either direction.
func TestStravaImportGoalCountingSurvivesResync(t *testing.T) {
	newControllerTestDB(t)
	database.SeedActions()

	user := createTestUser(t, "stravaresync@example.com", "Strava")
	excludeActivityType(t, user.ID, "Walk")

	now := time.Now()
	day := models.ExerciseDay{Date: now, Enabled: true, UserID: &user.ID}
	day.ID = uuid.New()
	if err := database.CreateExerciseDayInDB(day); err != nil {
		t.Fatalf("failed to create exercise day: %v", err)
	}

	// An imported walk the user has since decided *should* count.
	exercise := models.Exercise{ExerciseDayID: day.ID, Enabled: true, IsOn: true, Time: &now}
	exercise.ID = uuid.New()
	if _, err := database.UpdateExerciseInDB(exercise); err != nil {
		t.Fatalf("UpdateExerciseInDB returned error: %v", err)
	}
	if err := database.SetExerciseCountsTowardGoal(exercise.ID, true); err != nil {
		t.Fatalf("SetExerciseCountsTowardGoal returned error: %v", err)
	}

	// Re-sync: the importer refreshes the session but, because it is not new, leaves the
	// goal-counting flag alone. Mirroring StravaSyncActivityForUser, the flag it saves is
	// the one already stored.
	existing, err := database.GetExerciseByIDAndUserID(exercise.ID, user.ID)
	if err != nil || existing == nil {
		t.Fatalf("failed to reload session: %v", err)
	}
	existing.Note = "refreshed by a re-sync"
	if _, err := database.UpdateExerciseInDB(*existing); err != nil {
		t.Fatalf("UpdateExerciseInDB returned error: %v", err)
	}

	reloaded, err := database.GetExerciseByIDAndUserID(exercise.ID, user.ID)
	if err != nil || reloaded == nil {
		t.Fatalf("failed to reload session: %v", err)
	}
	if !reloaded.CountsTowardGoal {
		t.Errorf("a re-sync reset the user's manual goal-counting toggle")
	}
	if reloaded.Note != "refreshed by a re-sync" {
		t.Errorf("the re-sync did not update the session: note = %q", reloaded.Note)
	}
}
