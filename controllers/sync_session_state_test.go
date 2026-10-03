package controllers

import (
	"testing"
	"time"

	"github.com/aunefyren/treningheten/database"
	"github.com/aunefyren/treningheten/models"
)

// TestStravaResyncKeepsADeletedSessionDeleted covers that on/off is the user's decision: a
// session deleted in the builder (is_on=false) must stay deleted through later syncs rather
// than being switched back on by the hourly run, while a fresh import still arrives on.
func TestStravaResyncKeepsADeletedSessionDeleted(t *testing.T) {
	h := newAPIHarness(t)
	fake := withStrava(t)
	fake.activities = []models.StravaGetActivitiesRequestReply{stravaActivity(3001, "Run", todayAt(1), nil, false)}

	user, token := h.user("deleted@strava.test", false)
	h.ok("POST", "/api/auth/users/"+user.ID.String()+"/strava", token, models.UserStravaCodeUpdateRequest{StravaCode: "one-time-code"})

	imported, err := database.GetExerciseForUserWithStravaID(user.ID, "3001")
	if err != nil || imported == nil {
		t.Fatalf("imported session not found: %v", err)
	}
	if !imported.IsOn {
		t.Fatal("a fresh Strava import arrived switched off")
	}

	if err := database.SetExerciseIsOn(imported.ID, false); err != nil {
		t.Fatalf("SetExerciseIsOn returned error: %v", err)
	}

	stored, err := database.GetAllUserInformation(user.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := StravaSyncWeekForUser(stored, time.Now()); err != nil {
		t.Fatalf("resync: %v", err)
	}

	resynced, err := database.GetExerciseForUserWithStravaID(user.ID, "3001")
	if err != nil || resynced == nil {
		t.Fatalf("session lost on resync: %v", err)
	}
	if resynced.ID != imported.ID {
		t.Errorf("resync created a new session instead of refreshing the deleted one")
	}
	if resynced.IsOn {
		t.Error("a re-sync switched a builder-deleted session back on")
	}
}

// TestHevyResyncKeepsADeletedSessionDeleted is the Hevy counterpart: a re-import of the same
// workout rebuilds its contents but must leave a builder delete in place.
func TestHevyResyncKeepsADeletedSessionDeleted(t *testing.T) {
	newControllerTestDB(t)
	database.SeedActions()
	user := createTestUser(t, "deleted@hevy.test", "Hevy")

	templates := map[string]models.HevyExerciseTemplate{
		"tmpl-squat":  {ID: "tmpl-squat", Title: "Squat (Barbell)", Type: "weight_reps", PrimaryMuscleGroup: "quadriceps"},
		"tmpl-custom": {ID: "tmpl-custom", Title: "My Secret Move", Type: "duration", IsCustom: true},
	}
	workout := hevyWorkout("hevy-deleted", todayAt(60), "Legs")

	if err := HevySyncWorkoutForUser(user, workout, templates); err != nil {
		t.Fatalf("first import: %v", err)
	}
	imported, err := database.GetExerciseForUserWithHevyWorkoutID(user.ID, workout.ID)
	if err != nil || imported == nil {
		t.Fatalf("imported session not found: %v", err)
	}
	if !imported.IsOn {
		t.Fatal("a fresh Hevy import arrived switched off")
	}

	if err := database.SetExerciseIsOn(imported.ID, false); err != nil {
		t.Fatalf("SetExerciseIsOn returned error: %v", err)
	}

	workout.Title = "Legs, edited in Hevy"
	if err := HevySyncWorkoutForUser(user, workout, templates); err != nil {
		t.Fatalf("re-import: %v", err)
	}

	resynced, err := database.GetExerciseForUserWithHevyWorkoutID(user.ID, workout.ID)
	if err != nil || resynced == nil {
		t.Fatalf("session lost on re-import: %v", err)
	}
	if resynced.IsOn {
		t.Error("a re-import switched a builder-deleted session back on")
	}
	if resynced.Note != workout.Title {
		t.Errorf("the re-import did not refresh the session: note = %q", resynced.Note)
	}
}

// TestStravaTokenRefreshDoesNotRevertAConcurrentProfileChange covers that rotating the stored
// refresh token writes only the credential: the sync authorizes with a copy of the user loaded
// before it started, and saving that whole copy used to undo edits made in the meantime.
func TestStravaTokenRefreshDoesNotRevertAConcurrentProfileChange(t *testing.T) {
	h := newAPIHarness(t)
	withStrava(t)

	user, token := h.user("stale@strava.test", false)
	h.ok("POST", "/api/auth/users/"+user.ID.String()+"/strava", token, models.UserStravaCodeUpdateRequest{StravaCode: "one-time-code"})

	staleCopy, err := database.GetAllUserInformation(user.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := database.Instance.Model(&models.User{}).Where("id = ?", user.ID).Update("first_name", "Renamed").Error; err != nil {
		t.Fatal(err)
	}

	if _, err := StravaGetAuthorizationForUser(staleCopy); err != nil {
		t.Fatalf("StravaGetAuthorizationForUser returned error: %v", err)
	}

	stored, err := database.GetAllUserInformation(user.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.FirstName != "Renamed" {
		t.Errorf("first name = %q, the token refresh saved a stale user row", stored.FirstName)
	}
	if stored.StravaID == nil || *stored.StravaID != "4242" {
		t.Errorf("strava id = %v, want 4242 kept from the first connection", stored.StravaID)
	}
}
