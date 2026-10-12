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

func TestStravaGoalCountingAfterActionChange(t *testing.T) {
	walk, hike, run := uuid.New(), uuid.New(), uuid.New()
	offActions := map[uuid.UUID]bool{walk: true}

	tests := []struct {
		name        string
		change      stravaGoalCountingChange
		wantCounts  bool
		wantChanged bool
	}{
		{
			name:       "walk edited into a hike starts counting",
			change:     stravaGoalCountingChange{stored: false, previousActionID: walk, actionID: hike, offActions: offActions, stravaActivityCount: 1, weekIsCurrent: true},
			wantCounts: true, wantChanged: true,
		},
		{
			name:       "run edited into a walk stops counting",
			change:     stravaGoalCountingChange{stored: true, previousActionID: run, actionID: walk, offActions: offActions, stravaActivityCount: 1, weekIsCurrent: true},
			wantCounts: false, wantChanged: true,
		},
		{
			name:       "manual toggle wins",
			change:     stravaGoalCountingChange{stored: true, previousActionID: walk, actionID: run, offActions: offActions, stravaActivityCount: 1, weekIsCurrent: true},
			wantCounts: true, wantChanged: false,
		},
		{
			name:       "unchanged action",
			change:     stravaGoalCountingChange{stored: false, previousActionID: walk, actionID: walk, offActions: offActions, stravaActivityCount: 1, weekIsCurrent: true},
			wantCounts: false, wantChanged: false,
		},
		{
			name:       "combined session left alone",
			change:     stravaGoalCountingChange{stored: false, previousActionID: walk, actionID: hike, offActions: offActions, stravaActivityCount: 2, weekIsCurrent: true},
			wantCounts: false, wantChanged: false,
		},
		{
			name:       "past week frozen",
			change:     stravaGoalCountingChange{stored: false, previousActionID: walk, actionID: hike, offActions: offActions, stravaActivityCount: 1, weekIsCurrent: false},
			wantCounts: false, wantChanged: false,
		},
		{
			name:       "both types count, nothing to change",
			change:     stravaGoalCountingChange{stored: true, previousActionID: run, actionID: hike, offActions: offActions, stravaActivityCount: 1, weekIsCurrent: true},
			wantCounts: true, wantChanged: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			counts, changed := stravaGoalCountingAfterActionChange(test.change)
			if counts != test.wantCounts || changed != test.wantChanged {
				t.Errorf("stravaGoalCountingAfterActionChange() = (%v, %v), want (%v, %v)", counts, changed, test.wantCounts, test.wantChanged)
			}
		})
	}
}

func TestIsCurrentISOWeek(t *testing.T) {
	now := time.Date(2026, 10, 14, 12, 0, 0, 0, time.UTC) // Wednesday
	tests := []struct {
		name string
		date time.Time
		want bool
	}{
		{"same day", now, true},
		{"monday of the week", time.Date(2026, 10, 12, 0, 0, 0, 0, time.UTC), true},
		{"sunday of the week", time.Date(2026, 10, 18, 23, 0, 0, 0, time.UTC), true},
		{"previous sunday", time.Date(2026, 10, 11, 12, 0, 0, 0, time.UTC), false},
		{"same week a year earlier", now.AddDate(-1, 0, 0), false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := isCurrentISOWeek(test.date, now); got != test.want {
				t.Errorf("isCurrentISOWeek(%s) = %v, want %v", test.date, got, test.want)
			}
		})
	}
}

// TestReevaluateStravaSessionGoalCounting walks the production case through the database: an
// imported walk excluded from the goal is edited into a hike on Strava, and the re-sync must
// make the session count. A sport type change on a session the user toggled by hand is left
// alone, as are lookups with no previous action.
func TestReevaluateStravaSessionGoalCounting(t *testing.T) {
	newControllerTestDB(t)
	database.SeedActions()

	user := createTestUser(t, "stravareevaluate@example.com", "Strava")
	walk := excludeActivityType(t, user.ID, "Walk")
	hike, err := database.GetActionByStravaName("Hike")
	if err != nil || hike == nil {
		t.Fatalf("no seeded Hike action: %v", err)
	}

	now := time.Now()
	day := models.ExerciseDay{Date: now, Enabled: true, UserID: &user.ID}
	day.ID = uuid.New()
	if err := database.CreateExerciseDayInDB(day); err != nil {
		t.Fatalf("failed to create exercise day: %v", err)
	}

	seedSession := func(countsTowardGoal bool) models.Exercise {
		t.Helper()
		exercise := models.Exercise{ExerciseDayID: day.ID, Enabled: true, IsOn: true, Time: &now}
		exercise.ID = uuid.New()
		stored, err := database.UpdateExerciseInDB(exercise)
		if err != nil {
			t.Fatalf("UpdateExerciseInDB returned error: %v", err)
		}
		if err := database.SetExerciseCountsTowardGoal(stored.ID, countsTowardGoal); err != nil {
			t.Fatalf("SetExerciseCountsTowardGoal returned error: %v", err)
		}
		stored.CountsTowardGoal = countsTowardGoal
		return stored
	}
	storedFlag := func(id uuid.UUID) bool {
		t.Helper()
		reloaded, err := database.GetExerciseByIDAndUserID(id, user.ID)
		if err != nil || reloaded == nil {
			t.Fatalf("failed to reload session: %v", err)
		}
		return reloaded.CountsTowardGoal
	}

	t.Run("walk edited into a hike starts counting", func(t *testing.T) {
		session := seedSession(false)
		reevaluateStravaSessionGoalCounting(user.ID, session, &walk.ID, &hike.ID, now)
		if !storedFlag(session.ID) {
			t.Errorf("the hike still does not count toward the goal")
		}
	})

	t.Run("manual toggle survives", func(t *testing.T) {
		session := seedSession(true)
		run, err := database.GetActionByStravaName("Run")
		if err != nil || run == nil {
			t.Fatalf("no seeded Run action: %v", err)
		}
		// Previously a walk the user chose to count; now a run — still counts, untouched.
		reevaluateStravaSessionGoalCounting(user.ID, session, &walk.ID, &run.ID, now)
		if !storedFlag(session.ID) {
			t.Errorf("a manually counted session was reset")
		}
	})

	t.Run("no previous action is a no-op", func(t *testing.T) {
		session := seedSession(false)
		reevaluateStravaSessionGoalCounting(user.ID, session, nil, &hike.ID, now)
		if storedFlag(session.ID) {
			t.Errorf("a session with no previous action was re-evaluated")
		}
	})

	t.Run("past week is frozen", func(t *testing.T) {
		session := seedSession(false)
		reevaluateStravaSessionGoalCounting(user.ID, session, &walk.ID, &hike.ID, now.AddDate(0, 0, -7))
		if storedFlag(session.ID) {
			t.Errorf("a session in a settled week was re-evaluated")
		}
	})
}

// TestReevaluateStravaSessionGoalCountingFaults fails each database operation in turn: the
// re-evaluation is best effort, so every failure must leave the stored flag untouched.
func TestReevaluateStravaSessionGoalCountingFaults(t *testing.T) {
	newControllerTestDB(t)
	database.SeedActions()

	user := createTestUser(t, "stravareevaluatefaults@example.com", "Strava")
	walk := excludeActivityType(t, user.ID, "Walk")
	hike, err := database.GetActionByStravaName("Hike")
	if err != nil || hike == nil {
		t.Fatalf("no seeded Hike action: %v", err)
	}

	now := time.Now()
	day := models.ExerciseDay{Date: now, Enabled: true, UserID: &user.ID}
	day.ID = uuid.New()
	if err := database.CreateExerciseDayInDB(day); err != nil {
		t.Fatalf("failed to create exercise day: %v", err)
	}
	session := models.Exercise{ExerciseDayID: day.ID, Enabled: true, IsOn: true, Time: &now}
	session.ID = uuid.New()
	stored, err := database.UpdateExerciseInDB(session)
	if err != nil {
		t.Fatalf("UpdateExerciseInDB returned error: %v", err)
	}
	if err := database.SetExerciseCountsTowardGoal(stored.ID, false); err != nil {
		t.Fatalf("SetExerciseCountsTowardGoal returned error: %v", err)
	}
	stored.CountsTowardGoal = false

	injector := installFaultInjector(t, database.Instance)
	injector.arm(0)
	reevaluateStravaSessionGoalCounting(user.ID, stored, &walk.ID, &hike.ID, now)
	operations := injector.disarm()
	if err := database.SetExerciseCountsTowardGoal(stored.ID, false); err != nil {
		t.Fatalf("failed to reset flag: %v", err)
	}

	for failAt := 1; failAt <= operations; failAt++ {
		injector.arm(failAt)
		reevaluateStravaSessionGoalCounting(user.ID, stored, &walk.ID, &hike.ID, now)
		injector.disarm()

		reloaded, err := database.GetExerciseByIDAndUserID(stored.ID, user.ID)
		if err != nil || reloaded == nil {
			t.Fatalf("failed to reload session: %v", err)
		}
		if reloaded.CountsTowardGoal {
			t.Errorf("fault at operation %d of %d still changed the flag", failAt, operations)
		}
	}
}
