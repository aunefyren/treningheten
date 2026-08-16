package controllers

import (
	"testing"
	"time"

	"github.com/aunefyren/treningheten/database"
	"github.com/aunefyren/treningheten/models"

	"github.com/google/uuid"
)

// seedSharedActivity inserts one exercise day for the user at the given time with a single
// session on it, optionally carrying an action and a Strava id, and returns the session.
func seedSharedActivity(t *testing.T, userID uuid.UUID, at time.Time, actionID *uuid.UUID, stravaID string, isOn bool) models.Exercise {
	t.Helper()

	day := models.ExerciseDay{Date: at, Enabled: true, UserID: &userID}
	day.ID = uuid.New()
	if err := database.Instance.Omit("User", "Goal").Create(&day).Error; err != nil {
		t.Fatalf("failed to seed exercise day: %v", err)
	}

	sessionTime := at
	exercise := models.Exercise{ExerciseDayID: day.ID, Enabled: true, IsOn: isOn, Time: &sessionTime}
	exercise.ID = uuid.New()
	if err := database.Instance.Omit("ExerciseDay").Create(&exercise).Error; err != nil {
		t.Fatalf("failed to seed exercise: %v", err)
	}
	// IsOn carries a default:true tag, so a false must be written explicitly.
	if err := database.Instance.Model(&models.Exercise{}).Where("id = ?", exercise.ID).Update("is_on", isOn).Error; err != nil {
		t.Fatalf("failed to set is_on: %v", err)
	}

	operation := models.Operation{ExerciseID: exercise.ID, Enabled: true, ActionID: actionID}
	operation.ID = uuid.New()
	if err := database.Instance.Omit("Exercise", "Action", "Gear").Create(&operation).Error; err != nil {
		t.Fatalf("failed to seed operation: %v", err)
	}

	if stravaID != "" {
		set := models.OperationSet{OperationID: operation.ID, Enabled: true, StravaID: &stravaID}
		set.ID = uuid.New()
		if err := database.Instance.Omit("Operation").Create(&set).Error; err != nil {
			t.Fatalf("failed to seed operation set: %v", err)
		}
	}

	return exercise
}

// TestBuildActivitiesFromExerciseDays covers the front-page activity shape shared by the
// season feed and the peer feed: newest first, switched-off sessions excluded, and an
// action resolved for every activity.
func TestBuildActivitiesFromExerciseDays(t *testing.T) {
	newControllerTestDB(t)
	database.SeedActions()

	user := createTestUser(t, "sharedfeed@example.com", "Shared")
	run, err := database.GetActionByStravaName("Run")
	if err != nil || run == nil {
		t.Fatalf("failed to resolve the Run action: %v", err)
	}

	base := time.Date(2026, 8, 5, 8, 0, 0, 0, time.UTC)
	seedSharedActivity(t, user.ID, base, &run.ID, "", true)
	seedSharedActivity(t, user.ID, base.Add(2*time.Hour), &run.ID, "", true)
	seedSharedActivity(t, user.ID, base.Add(time.Hour), nil, "", true) // no action
	seedSharedActivity(t, user.ID, base.Add(3*time.Hour), &run.ID, "", false)

	days := []models.ExerciseDay{}
	if err := database.Instance.Where("user_id = ?", user.ID).Find(&days).Error; err != nil {
		t.Fatalf("failed to load exercise days: %v", err)
	}

	activities, err := buildActivitiesFromExerciseDays(days)
	if err != nil {
		t.Fatalf("buildActivitiesFromExerciseDays returned error: %v", err)
	}

	if len(activities) != 3 {
		t.Fatalf("got %d activities, want 3 (the switched-off session excluded)", len(activities))
	}
	// Newest first.
	for i := 1; i < len(activities); i++ {
		if activities[i].Time.After(activities[i-1].Time) {
			t.Fatalf("activities are not newest-first: %v then %v", activities[i-1].Time, activities[i].Time)
		}
	}
	// Every activity carries at least one action — a session whose operations resolve none
	// falls back to the general "Workout" action rather than rendering actionless.
	for _, activity := range activities {
		if len(activity.Actions) == 0 {
			t.Errorf("activity %v has no actions", activity.ExerciseID)
		}
		if activity.User.ID != user.ID {
			t.Errorf("activity carries user %v, want %v", activity.User.ID, user.ID)
		}
	}

	if activities := mustBuild(t, nil); len(activities) != 0 {
		t.Errorf("no exercise days produced %d activities, want none", len(activities))
	}
}

// TestBuildActivitiesFromExerciseDaysExcludesPrivate covers the privacy rule: a session
// marked private is left out of the feed entirely — including the owner's own view, since
// this builder backs every social surface and applies one rule for all viewers.
func TestBuildActivitiesFromExerciseDaysExcludesPrivate(t *testing.T) {
	newControllerTestDB(t)
	database.SeedActions()

	user := createTestUser(t, "privatefeed@example.com", "Private")
	run, err := database.GetActionByStravaName("Run")
	if err != nil || run == nil {
		t.Fatalf("failed to resolve the Run action: %v", err)
	}

	base := time.Date(2026, 8, 5, 8, 0, 0, 0, time.UTC)
	visible := seedSharedActivity(t, user.ID, base, &run.ID, "", true)
	hidden := seedSharedActivity(t, user.ID, base.Add(time.Hour), &run.ID, "", true)
	if err := database.Instance.Model(&models.Exercise{}).Where("id = ?", hidden.ID).Update("private", true).Error; err != nil {
		t.Fatalf("failed to mark session private: %v", err)
	}

	days := []models.ExerciseDay{}
	if err := database.Instance.Where("user_id = ?", user.ID).Find(&days).Error; err != nil {
		t.Fatalf("failed to load exercise days: %v", err)
	}

	activities := mustBuild(t, days)
	if len(activities) != 1 {
		t.Fatalf("got %d activities, want 1 (the private session excluded)", len(activities))
	}
	if activities[0].ExerciseID != visible.ID {
		t.Errorf("feed carries session %v, want the non-private %v", activities[0].ExerciseID, visible.ID)
	}
}

// mustBuild is a small wrapper so an empty-input assertion stays readable.
func mustBuild(t *testing.T, days []models.ExerciseDay) []models.Activity {
	t.Helper()
	activities, err := buildActivitiesFromExerciseDays(days)
	if err != nil {
		t.Fatalf("buildActivitiesFromExerciseDays returned error: %v", err)
	}
	return activities
}

// TestBuildActivitiesFromExerciseDaysStravaPrivacy covers that Strava links follow the
// session owner's StravaPublic setting (which defaults to on). The feed is now visible to
// every past season-mate, so opting out must withhold the ids from all of them.
func TestBuildActivitiesFromExerciseDaysStravaPrivacy(t *testing.T) {
	newControllerTestDB(t)
	database.SeedActions()

	private := createTestUser(t, "stravaprivate@example.com", "Private")
	at := time.Date(2026, 8, 5, 8, 0, 0, 0, time.UTC)
	seedSharedActivity(t, private.ID, at, nil, "9999", true)

	loadDays := func(userID uuid.UUID) []models.ExerciseDay {
		days := []models.ExerciseDay{}
		if err := database.Instance.Where("user_id = ?", userID).Find(&days).Error; err != nil {
			t.Fatalf("failed to load exercise days: %v", err)
		}
		return days
	}

	// StravaPublic defaults to on, so the ids travel with the activity.
	activities := mustBuild(t, loadDays(private.ID))
	if len(activities) != 1 {
		t.Fatalf("got %d activities, want 1", len(activities))
	}
	if len(activities[0].StravaIDs) != 1 || activities[0].StravaIDs[0] != "9999" {
		t.Fatalf("Strava ids = %v, want the session's id for a public user", activities[0].StravaIDs)
	}

	// Opting out withholds them — the feed now reaches every past season-mate, so this is
	// the switch standing between a private user's Strava profile and all of them.
	if err := database.Instance.Model(&models.User{}).Where("id = ?", private.ID).Update("strava_public", false).Error; err != nil {
		t.Fatalf("failed to clear strava_public: %v", err)
	}
	activities = mustBuild(t, loadDays(private.ID))
	if len(activities[0].StravaIDs) != 0 {
		t.Errorf("a non-public user's Strava ids leaked into the feed: %v", activities[0].StravaIDs)
	}
}
