package controllers

import (
	"net/http"
	"testing"
	"time"

	"github.com/aunefyren/treningheten/database"
	"github.com/aunefyren/treningheten/models"

	"github.com/google/uuid"
)

// seedHevySession inserts an exercise day for the user with one on Hevy-imported session at
// the given time, and returns the session.
func seedHevySession(t *testing.T, userID uuid.UUID, at time.Time, workoutID string) models.Exercise {
	t.Helper()
	session := seedSharedActivity(t, userID, at, nil, "", true)
	if err := database.Instance.Model(&session).Update("hevy_workout_id", workoutID).Error; err != nil {
		t.Fatalf("failed to set hevy_workout_id: %v", err)
	}
	session.HevyWorkoutID = &workoutID
	return session
}

// seedCombinedStravaSession inserts a session holding two Strava activities, as
// APIStravaCombine leaves it.
func seedCombinedStravaSession(t *testing.T, userID uuid.UUID, at time.Time) models.Exercise {
	t.Helper()
	session := seedSharedActivity(t, userID, at, nil, "combined-1", true)
	operation := models.Operation{ExerciseID: session.ID, Enabled: true}
	operation.ID = uuid.New()
	if err := database.Instance.Omit("Exercise", "Action", "Gear").Create(&operation).Error; err != nil {
		t.Fatalf("failed to seed operation: %v", err)
	}
	stravaID := "combined-2"
	set := models.OperationSet{OperationID: operation.ID, Enabled: true, StravaID: &stravaID}
	set.ID = uuid.New()
	if err := database.Instance.Omit("Operation").Create(&set).Error; err != nil {
		t.Fatalf("failed to seed operation set: %v", err)
	}
	return session
}

// userWithPriority creates a user with the given source ranking (nil leaves it unset).
func userWithPriority(t *testing.T, email string, priority *string) models.User {
	t.Helper()
	user := createTestUser(t, email, "Dedup")
	if priority != nil {
		if err := database.Instance.Model(&models.User{}).Where("id = ?", user.ID).Update("activity_source_priority", *priority).Error; err != nil {
			t.Fatalf("failed to set activity_source_priority: %v", err)
		}
		user.ActivitySourcePriority = priority
	}
	return user
}

// loadExercise reads a session straight from the table, hidden or not.
func loadExercise(t *testing.T, id uuid.UUID) models.Exercise {
	t.Helper()
	exercise := models.Exercise{}
	if err := database.Instance.Where("id = ?", id).First(&exercise).Error; err != nil {
		t.Fatalf("failed to load exercise %v: %v", id, err)
	}
	return exercise
}

// assertHiddenBehind checks a session is hidden behind winnerID, or visible when winnerID
// is nil.
func assertHiddenBehind(t *testing.T, label string, id uuid.UUID, winnerID *uuid.UUID) {
	t.Helper()
	exercise := loadExercise(t, id)
	if winnerID == nil {
		if !exercise.Enabled || exercise.SupersededByID != nil {
			t.Errorf("%s: want visible, got enabled=%v superseded_by=%v", label, exercise.Enabled, exercise.SupersededByID)
		}
		return
	}
	if exercise.Enabled || exercise.SupersededByID == nil || *exercise.SupersededByID != *winnerID {
		t.Errorf("%s: want hidden behind %v, got enabled=%v superseded_by=%v", label, *winnerID, exercise.Enabled, exercise.SupersededByID)
	}
}

func TestParseActivitySourcePriority(t *testing.T) {
	cases := []struct {
		name string
		raw  *string
		want []string
	}{
		{"unset", nil, []string{}},
		{"empty", stringPtr(""), []string{}},
		{"ranked", stringPtr("hevy,strava"), []string{"hevy", "strava"}},
		{"untidy", stringPtr(" Strava ,, HEVY "), []string{"strava", "hevy"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := parseActivitySourcePriority(tc.raw)
			if len(got) != len(tc.want) {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
			for index := range got {
				if got[index] != tc.want[index] {
					t.Errorf("got %v, want %v", got, tc.want)
				}
			}
		})
	}
}

func TestNormaliseActivitySourcePriority(t *testing.T) {
	cases := []struct {
		name    string
		raw     string
		want    string
		wantErr bool
	}{
		{"off", "", "", false},
		{"default", "hevy,strava", "hevy,strava", false},
		{"tidied", " Strava , hevy ", "strava,hevy", false},
		{"single source", "strava", "strava", false},
		{"unknown source", "hevy,garmin", "", true},
		{"repeated source", "hevy,strava,hevy", "", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := normaliseActivitySourcePriority(tc.raw)
			if (err != nil) != tc.wantErr {
				t.Fatalf("error = %v, wantErr %v", err, tc.wantErr)
			}
			if got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
	if defaultActivitySourcePriority() != "hevy,strava" {
		t.Errorf("default ranking = %q, want Hevy first", defaultActivitySourcePriority())
	}
}

func TestResolveImportedDuplicates(t *testing.T) {
	base := time.Date(2024, 6, 15, 12, 0, 0, 0, time.UTC)

	cases := []struct {
		name         string
		priority     *string
		incoming     string // "hevy" or "strava"
		existing     string // "hevy", "strava", "manual", "combined" or "hevy-off"
		offset       time.Duration
		wantIncoming bool // incoming hidden behind existing
		wantExisting bool // existing hidden behind incoming
	}{
		{"Hevy first hides an incoming Strava copy", stringPtr("hevy,strava"), "strava", "hevy", 5 * time.Minute, true, false},
		{"Hevy first: an incoming Hevy workout hides the Strava copy", stringPtr("hevy,strava"), "hevy", "strava", -5 * time.Minute, false, true},
		{"Strava first hides an incoming Hevy copy", stringPtr("strava,hevy"), "hevy", "strava", 0, true, false},
		{"Strava first: an incoming Strava activity hides the Hevy copy", stringPtr("strava,hevy"), "strava", "hevy", 0, false, true},
		{"off hides nothing", stringPtr(""), "strava", "hevy", 0, false, false},
		{"unset hides nothing", nil, "strava", "hevy", 0, false, false},
		{"a single ranked source hides nothing", stringPtr("hevy"), "strava", "hevy", 0, false, false},
		{"an unranked incoming source hides nothing", stringPtr("hevy,garmin"), "strava", "hevy", 0, false, false},
		{"outside the window", stringPtr("hevy,strava"), "strava", "hevy", duplicateActivityWindow + time.Minute, false, false},
		{"a manual session is never a duplicate", stringPtr("hevy,strava"), "strava", "manual", 0, false, false},
		{"a combined session is never a duplicate", stringPtr("hevy,strava"), "hevy", "combined", 0, false, false},
		{"the same source is never a duplicate", stringPtr("hevy,strava"), "hevy", "hevy", 0, false, false},
		{"a switched-off session is ignored", stringPtr("hevy,strava"), "strava", "hevy-off", 0, false, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			newControllerTestDB(t)
			user := userWithPriority(t, "resolve@example.com", tc.priority)

			var existing models.Exercise
			switch tc.existing {
			case "hevy":
				existing = seedHevySession(t, user.ID, base.Add(tc.offset), "hw-existing")
			case "hevy-off":
				existing = seedHevySession(t, user.ID, base.Add(tc.offset), "hw-existing")
				if err := database.SetExerciseIsOn(existing.ID, false); err != nil {
					t.Fatal(err)
				}
			case "strava":
				existing = seedSharedActivity(t, user.ID, base.Add(tc.offset), nil, "9001", true)
			case "manual":
				existing = seedSharedActivity(t, user.ID, base.Add(tc.offset), nil, "", true)
			case "combined":
				existing = seedCombinedStravaSession(t, user.ID, base.Add(tc.offset))
			}

			var incoming models.Exercise
			if tc.incoming == "hevy" {
				incoming = seedHevySession(t, user.ID, base, "hw-incoming")
			} else {
				incoming = seedSharedActivity(t, user.ID, base, nil, "9002", true)
			}

			hidden, err := resolveImportedDuplicates(user, incoming)
			if err != nil {
				t.Fatalf("resolveImportedDuplicates returned error: %v", err)
			}
			if hidden != tc.wantIncoming {
				t.Errorf("reported hidden = %v, want %v", hidden, tc.wantIncoming)
			}

			incomingWinner, existingWinner := (*uuid.UUID)(nil), (*uuid.UUID)(nil)
			if tc.wantIncoming {
				incomingWinner = &existing.ID
			}
			if tc.wantExisting {
				existingWinner = &incoming.ID
			}
			assertHiddenBehind(t, "incoming", incoming.ID, incomingWinner)
			if tc.existing != "hevy-off" {
				assertHiddenBehind(t, "existing", existing.ID, existingWinner)
			}
		})
	}

	t.Run("a session with no time is skipped", func(t *testing.T) {
		newControllerTestDB(t)
		user := userWithPriority(t, "untimed@example.com", stringPtr("hevy,strava"))
		hidden, err := resolveImportedDuplicates(user, models.Exercise{})
		if err != nil || hidden {
			t.Errorf("got %v, %v; want a no-op", hidden, err)
		}
	})
}

func TestHideDuplicateExerciseFlattensChains(t *testing.T) {
	newControllerTestDB(t)
	user := createTestUser(t, "chain@example.com", "Chain")
	now := time.Now()
	first := seedHevySession(t, user.ID, now, "hw-1")
	second := seedSharedActivity(t, user.ID, now, nil, "9101", true)
	third := seedHevySession(t, user.ID, now, "hw-3")

	if err := hideDuplicateExercise(first.ID, second.ID); err != nil {
		t.Fatal(err)
	}
	if err := hideDuplicateExercise(second.ID, third.ID); err != nil {
		t.Fatal(err)
	}
	assertHiddenBehind(t, "first", first.ID, &third.ID)
	assertHiddenBehind(t, "second", second.ID, &third.ID)
}

func TestReconcileDuplicateActivitiesForUser(t *testing.T) {
	newControllerTestDB(t)
	user := userWithPriority(t, "reconcile@example.com", nil)
	base := time.Date(2024, 6, 15, 8, 0, 0, 0, time.UTC)

	morningHevy := seedHevySession(t, user.ID, base, "hw-morning")
	morningStrava := seedSharedActivity(t, user.ID, base.Add(3*time.Minute), nil, "9201", true)
	eveningHevy := seedHevySession(t, user.ID, base.Add(10*time.Hour), "hw-evening")
	eveningStrava := seedSharedActivity(t, user.ID, base.Add(10*time.Hour-time.Minute), nil, "9202", true)
	loneRun := seedSharedActivity(t, user.ID, base.Add(5*time.Hour), nil, "9203", true)

	steps := []struct {
		name     string
		priority *string
		hidden   map[uuid.UUID]*uuid.UUID
	}{
		{"turning it on applies to past activities", stringPtr("hevy,strava"), map[uuid.UUID]*uuid.UUID{
			morningStrava.ID: &morningHevy.ID, eveningStrava.ID: &eveningHevy.ID,
		}},
		{"flipping the ranking swaps which copy shows", stringPtr("strava,hevy"), map[uuid.UUID]*uuid.UUID{
			morningHevy.ID: &morningStrava.ID, eveningHevy.ID: &eveningStrava.ID,
		}},
		{"running it again changes nothing", stringPtr("strava,hevy"), map[uuid.UUID]*uuid.UUID{
			morningHevy.ID: &morningStrava.ID, eveningHevy.ID: &eveningStrava.ID,
		}},
		{"turning it off restores everything", stringPtr(""), map[uuid.UUID]*uuid.UUID{}},
	}
	all := []uuid.UUID{morningHevy.ID, morningStrava.ID, eveningHevy.ID, eveningStrava.ID, loneRun.ID}
	for _, step := range steps {
		t.Run(step.name, func(t *testing.T) {
			user.ActivitySourcePriority = step.priority
			if err := reconcileDuplicateActivitiesForUser(user); err != nil {
				t.Fatalf("reconcileDuplicateActivitiesForUser returned error: %v", err)
			}
			for _, id := range all {
				assertHiddenBehind(t, id.String(), id, step.hidden[id])
			}
		})
	}
}

// TestDuplicateDetectionThroughSyncs drives the real Strava and Hevy imports: the Strava
// copy of a Hevy workout is hidden, stays hidden through a re-sync, and comes back when the
// Hevy workout is deleted.
func TestDuplicateDetectionThroughSyncs(t *testing.T) {
	h := newAPIHarness(t)
	fake := withStrava(t)
	fake.activities = []models.StravaGetActivitiesRequestReply{stravaActivity(3101, "WeightTraining", todayAt(60), nil, false)}

	user, token := h.user("synced@dedup.test", false)
	if err := database.Instance.Model(&models.User{}).Where("id = ?", user.ID).Update("activity_source_priority", "hevy,strava").Error; err != nil {
		t.Fatal(err)
	}
	h.ok("POST", "/api/auth/users/"+user.ID.String()+"/strava", token, models.UserStravaCodeUpdateRequest{StravaCode: "one-time-code"})
	stravaSession, err := database.GetExerciseForUserWithStravaID(user.ID, "3101")
	if err != nil || stravaSession == nil {
		t.Fatalf("Strava activity not imported: %v", err)
	}

	stored, err := database.GetAllUserInformation(user.ID)
	if err != nil {
		t.Fatal(err)
	}
	templates := map[string]models.HevyExerciseTemplate{
		"tmpl-squat":  {ID: "tmpl-squat", Title: "Squat (Barbell)", Type: "weight_reps", PrimaryMuscleGroup: "quadriceps"},
		"tmpl-custom": {ID: "tmpl-custom", Title: "My Secret Move", Type: "duration", IsCustom: true},
	}
	workout := hevyWorkout("hevy-dedup", todayAt(65), "Legs")
	if err := HevySyncWorkoutForUser(stored, workout, templates); err != nil {
		t.Fatalf("Hevy import: %v", err)
	}
	hevySession, err := database.GetExerciseForUserWithHevyWorkoutID(user.ID, workout.ID)
	if err != nil || hevySession == nil {
		t.Fatalf("Hevy workout not imported: %v", err)
	}
	assertHiddenBehind(t, "Strava copy", stravaSession.ID, &hevySession.ID)

	if err := StravaSyncWeekForUser(stored, time.Now()); err != nil {
		t.Fatalf("Strava re-sync: %v", err)
	}
	if again, _ := database.GetExerciseForUserWithStravaID(user.ID, "3101"); again != nil {
		t.Errorf("the re-sync imported the hidden Strava activity again")
	}
	assertHiddenBehind(t, "Strava copy after re-sync", stravaSession.ID, &hevySession.ID)

	if err := hevyDeleteWorkoutForUser(stored, workout.ID); err != nil {
		t.Fatalf("Hevy delete: %v", err)
	}
	assertHiddenBehind(t, "Strava copy after the Hevy workout was deleted", stravaSession.ID, nil)
}

// TestHiddenHevyWorkoutThroughSyncs covers a Hevy workout losing to Strava: a re-import
// leaves it hidden, and deleting it in Hevy drops the mark so it can never be restored.
func TestHiddenHevyWorkoutThroughSyncs(t *testing.T) {
	newControllerTestDB(t)
	database.SeedActions()
	user := userWithPriority(t, "hidden@hevy.test", stringPtr("strava,hevy"))
	stravaSession := seedSharedActivity(t, user.ID, todayAt(120).UTC(), nil, "9301", true)

	templates := map[string]models.HevyExerciseTemplate{
		"tmpl-squat":  {ID: "tmpl-squat", Title: "Squat (Barbell)", Type: "weight_reps", PrimaryMuscleGroup: "quadriceps"},
		"tmpl-custom": {ID: "tmpl-custom", Title: "My Secret Move", Type: "duration", IsCustom: true},
	}
	workout := hevyWorkout("hevy-hidden", todayAt(122), "Legs")
	for range 2 {
		if err := HevySyncWorkoutForUser(user, workout, templates); err != nil {
			t.Fatalf("Hevy import: %v", err)
		}
	}
	hidden, err := database.GetSupersededExerciseForUserWithHevyWorkoutID(user.ID, workout.ID)
	if err != nil || hidden == nil {
		t.Fatalf("Hevy workout not hidden: %v", err)
	}
	var sessions int64
	database.Instance.Model(&models.Exercise{}).Where("hevy_workout_id = ?", workout.ID).Count(&sessions)
	if sessions != 1 {
		t.Errorf("a re-import of a hidden workout created %d sessions, want 1", sessions)
	}
	assertHiddenBehind(t, "Hevy copy", hidden.ID, &stravaSession.ID)

	if err := hevyDeleteWorkoutForUser(user, workout.ID); err != nil {
		t.Fatalf("Hevy delete: %v", err)
	}
	deleted := loadExercise(t, hidden.ID)
	if deleted.Enabled || deleted.SupersededByID != nil {
		t.Errorf("deleted hidden workout: enabled=%v superseded_by=%v; want disabled with no mark", deleted.Enabled, deleted.SupersededByID)
	}
	if err := hevyDeleteWorkoutForUser(user, "never-imported"); err != nil {
		t.Errorf("deleting an unknown workout: %v", err)
	}
}

func TestPartialUpdateActivitySourcePriority(t *testing.T) {
	h := newAPIHarness(t)
	user, token := h.user("ranking@dedup.test", false)
	userPath := "/api/auth/users/" + user.ID.String()

	base := time.Date(2024, 6, 15, 8, 0, 0, 0, time.UTC)
	hevySession := seedHevySession(t, user.ID, base, "hw-api")
	stravaSession := seedSharedActivity(t, user.ID, base, nil, "9401", true)

	for _, invalid := range []string{"garmin", "hevy,hevy"} {
		h.expect(http.StatusBadRequest, "PATCH", userPath, token, map[string]any{"activity_source_priority": invalid})
	}

	h.ok("PATCH", userPath, token, map[string]any{"activity_source_priority": " Strava , hevy "})
	stored, err := database.GetAllUserInformation(user.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.ActivitySourcePriority == nil || *stored.ActivitySourcePriority != "strava,hevy" {
		t.Fatalf("stored priority = %v, want strava,hevy", stored.ActivitySourcePriority)
	}

	// The reconcile runs in the background; wait for it to land.
	deadline := time.Now().Add(5 * time.Second)
	for loadExercise(t, hevySession.ID).Enabled && time.Now().Before(deadline) {
		time.Sleep(20 * time.Millisecond)
	}
	assertHiddenBehind(t, "Hevy copy", hevySession.ID, &stravaSession.ID)

	// Other updates leave the ranking alone.
	h.ok("PATCH", userPath, token, map[string]any{"sunday_alert": true})
	if again, _ := database.GetAllUserInformation(user.ID); again.ActivitySourcePriority == nil || *again.ActivitySourcePriority != "strava,hevy" {
		t.Errorf("an unrelated update changed the ranking to %v", again.ActivitySourcePriority)
	}
}

// TestHevyConnectSetsDefaultSourcePriority covers the first-connect default, and that a
// ranking the user already chose — including off — is kept.
func TestHevyConnectSetsDefaultSourcePriority(t *testing.T) {
	cases := []struct {
		name     string
		priority *string
		want     string
	}{
		{"unset gets Hevy first", nil, "hevy,strava"},
		{"off is kept", stringPtr(""), ""},
		{"a chosen ranking is kept", stringPtr("strava,hevy"), "strava,hevy"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := newAPIHarness(t)
			withHevy(t)
			fakeHevyAPI(t, nil, nil)
			user, token := h.user("connect@dedup.test", false)
			if tc.priority != nil {
				if err := database.Instance.Model(&models.User{}).Where("id = ?", user.ID).Update("activity_source_priority", *tc.priority).Error; err != nil {
					t.Fatal(err)
				}
			}

			h.ok("POST", "/api/auth/users/"+user.ID.String()+"/hevy", token, models.UserHevyAPIKeyUpdateRequest{HevyAPIKey: "good-key"})
			stored, err := database.GetAllUserInformation(user.ID)
			if err != nil {
				t.Fatal(err)
			}
			if stored.ActivitySourcePriority == nil || *stored.ActivitySourcePriority != tc.want {
				t.Errorf("priority = %v, want %q", stored.ActivitySourcePriority, tc.want)
			}
		})
	}
}

// TestStravaCombineRestoresHiddenDuplicates covers that a combined session takes no part in
// duplicate detection: the Hevy workout one of its parts was hiding comes back.
func TestStravaCombineRestoresHiddenDuplicates(t *testing.T) {
	h := newAPIHarness(t)
	fake := withStrava(t)
	fake.activities = []models.StravaGetActivitiesRequestReply{
		stravaActivity(3201, "Run", todayAt(60), nil, false),
		stravaActivity(3202, "Run", todayAt(180), nil, false),
	}

	user, token := h.user("combine@dedup.test", false)
	if err := database.Instance.Model(&models.User{}).Where("id = ?", user.ID).Update("activity_source_priority", "strava,hevy").Error; err != nil {
		t.Fatal(err)
	}
	h.ok("POST", "/api/auth/users/"+user.ID.String()+"/strava", token, models.UserStravaCodeUpdateRequest{StravaCode: "one-time-code"})

	stored, err := database.GetAllUserInformation(user.ID)
	if err != nil {
		t.Fatal(err)
	}
	templates := map[string]models.HevyExerciseTemplate{
		"tmpl-squat":  {ID: "tmpl-squat", Title: "Squat (Barbell)", Type: "weight_reps", PrimaryMuscleGroup: "quadriceps"},
		"tmpl-custom": {ID: "tmpl-custom", Title: "My Secret Move", Type: "duration", IsCustom: true},
	}
	workout := hevyWorkout("hevy-combine", todayAt(61), "Legs")
	if err := HevySyncWorkoutForUser(stored, workout, templates); err != nil {
		t.Fatalf("Hevy import: %v", err)
	}
	hidden, err := database.GetSupersededExerciseForUserWithHevyWorkoutID(user.ID, workout.ID)
	if err != nil || hidden == nil {
		t.Fatalf("Hevy workout not hidden behind the Strava run: %v", err)
	}

	h.expect(http.StatusCreated, "POST", "/api/auth/exercises/strava-combine", token, []string{"3201", "3202"})
	assertHiddenBehind(t, "Hevy copy after combining", hidden.ID, nil)
}

// TestDuplicateDetectionReportsDatabaseFaults sweeps every database operation of each
// duplicate-detection entry point, failing one at a time on a fresh database, and checks the
// failure is reported rather than swallowed.
func TestDuplicateDetectionReportsDatabaseFaults(t *testing.T) {
	base := time.Date(2024, 6, 15, 8, 0, 0, 0, time.UTC)

	scenarios := []struct {
		name  string
		setup func(t *testing.T) func() error
	}{
		{"resolve, incoming wins", func(t *testing.T) func() error {
			user := userWithPriority(t, "fault-wins@example.com", stringPtr("hevy,strava"))
			seedSharedActivity(t, user.ID, base, nil, "9501", true)
			incoming := seedHevySession(t, user.ID, base, "hw-fault")
			return func() error { _, err := resolveImportedDuplicates(user, incoming); return err }
		}},
		{"resolve, incoming loses", func(t *testing.T) func() error {
			user := userWithPriority(t, "fault-loses@example.com", stringPtr("hevy,strava"))
			seedHevySession(t, user.ID, base, "hw-fault")
			incoming := seedSharedActivity(t, user.ID, base, nil, "9502", true)
			return func() error { _, err := resolveImportedDuplicates(user, incoming); return err }
		}},
		{"restore", func(t *testing.T) func() error {
			user := userWithPriority(t, "fault-restore@example.com", stringPtr("hevy,strava"))
			winner := seedHevySession(t, user.ID, base, "hw-fault")
			loser := seedSharedActivity(t, user.ID, base, nil, "9503", true)
			if err := hideDuplicateExercise(loser.ID, winner.ID); err != nil {
				t.Fatal(err)
			}
			return func() error { return restoreDuplicatesOf(user, winner.ID) }
		}},
		{"reconcile", func(t *testing.T) func() error {
			user := userWithPriority(t, "fault-reconcile@example.com", stringPtr("strava,hevy"))
			hevy := seedHevySession(t, user.ID, base, "hw-fault")
			strava := seedSharedActivity(t, user.ID, base, nil, "9504", true)
			if err := hideDuplicateExercise(strava.ID, hevy.ID); err != nil {
				t.Fatal(err)
			}
			return func() error { return reconcileDuplicateActivitiesForUser(user) }
		}},
		{"forget a deleted hidden Hevy workout", func(t *testing.T) func() error {
			user := userWithPriority(t, "fault-forget@example.com", stringPtr("strava,hevy"))
			strava := seedSharedActivity(t, user.ID, base, nil, "9505", true)
			hevy := seedHevySession(t, user.ID, base, "hw-fault")
			if err := hideDuplicateExercise(hevy.ID, strava.ID); err != nil {
				t.Fatal(err)
			}
			return func() error { return hevyForgetHiddenWorkoutForUser(user, "hw-fault") }
		}},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			newControllerTestDB(t)
			injector := installFaultInjector(t, database.Instance)
			call := scenario.setup(t)
			injector.arm(0)
			if err := call(); err != nil {
				t.Fatalf("clean run failed: %v", err)
			}
			operations := injector.disarm()

			for failAt := 1; failAt <= operations; failAt++ {
				newControllerTestDB(t)
				injector := installFaultInjector(t, database.Instance)
				call := scenario.setup(t)
				injector.arm(failAt)
				err := call()
				injector.disarm()
				if err == nil {
					t.Errorf("operation %d of %d failed, but no error was reported", failAt, operations)
				}
			}
		})
	}
}
