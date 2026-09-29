package controllers

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/aunefyren/treningheten/database"
	"github.com/aunefyren/treningheten/files"
	"github.com/aunefyren/treningheten/models"
)

func withHevy(t *testing.T) {
	t.Helper()
	previous := files.ConfigFile
	files.ConfigFile.HevyEnabled = true
	key, _ := files.GenerateSecureKey(32)
	files.ConfigFile.HevyTokenKey = key
	t.Cleanup(func() {
		files.ConfigFile.HevyEnabled = previous.HevyEnabled
		files.ConfigFile.HevyTokenKey = previous.HevyTokenKey
	})
}

func hevyWorkout(id string, start time.Time, title string) models.HevyWorkout {
	weight, reps, duration := 60.0, 8.0, 600.0
	return models.HevyWorkout{
		ID: id, Title: title, Description: "Leg day notes", StartTime: start, EndTime: start.Add(45 * time.Minute),
		UpdatedAt: start, CreatedAt: start,
		Exercises: []models.HevyWorkoutExercise{
			{Index: 0, Title: "Squat (Barbell)", ExerciseTemplateID: "tmpl-squat", Notes: "deep", Sets: []models.HevyWorkoutSet{
				{Index: 0, Type: "warmup", WeightKg: &weight, Reps: &reps},
				{Index: 1, Type: "normal", WeightKg: &weight, Reps: &reps},
			}},
			{Index: 1, Title: "My Secret Move", ExerciseTemplateID: "tmpl-custom", Sets: []models.HevyWorkoutSet{
				{Index: 0, Type: "normal", DurationSeconds: &duration},
			}},
		},
	}
}

// fakeHevyAPI answers the Hevy endpoints the importer calls. Workouts and events are
// served as a single page.
func fakeHevyAPI(t *testing.T, workouts []models.HevyWorkout, events []models.HevyWorkoutEvent) {
	t.Helper()

	stubHevyAPI(t, func(writer http.ResponseWriter, request *http.Request) {
		if injectUpstreamFault(writer) {
			return
		}
		if request.Header.Get("api-key") == "rejected-key" {
			writer.WriteHeader(http.StatusUnauthorized)
			return
		}
		page := request.URL.Query().Get("page")
		writer.Header().Set("Content-Type", "application/json")
		switch {
		case request.URL.Path == "/user/info":
			_ = json.NewEncoder(writer).Encode(models.HevyUserInfoResponse{Data: models.HevyUserInfo{ID: "hevy-user", Name: "lifter", URL: "https://hevy.com/user/lifter"}})
		case request.URL.Path == "/exercise_templates":
			templates := []models.HevyExerciseTemplate{
				{ID: "tmpl-squat", Title: "Squat (Barbell)", Type: "weight_reps", PrimaryMuscleGroup: "quadriceps"},
				{ID: "tmpl-custom", Title: "My Secret Move", Type: "duration", IsCustom: true},
			}
			if page != "1" {
				templates = nil
			}
			_ = json.NewEncoder(writer).Encode(models.HevyExerciseTemplatesResponse{Page: 1, PageCount: 1, ExerciseTemplates: templates})
		case request.URL.Path == "/workouts":
			served := workouts
			if page != "1" {
				served = nil
			}
			_ = json.NewEncoder(writer).Encode(models.HevyWorkoutsResponse{Page: 1, PageCount: 1, Workouts: served})
		case request.URL.Path == "/workouts/events":
			served := events
			if page != "1" {
				served = nil
			}
			_ = json.NewEncoder(writer).Encode(models.HevyWorkoutEventsResponse{Page: 1, PageCount: 1, Events: served})
		default:
			t.Errorf("unexpected Hevy request %s", request.URL.String())
			writer.WriteHeader(http.StatusNotFound)
		}
	})
}

func TestHevyConnectionEndpoints(t *testing.T) {
	h := newAPIHarness(t)
	user, token := h.user("lifter@hevy.test", false)
	userPath := "/api/auth/users/" + user.ID.String()

	// Disabled: every Hevy endpoint refuses.
	h.expect(http.StatusBadRequest, "POST", userPath+"/hevy", token, models.UserHevyAPIKeyUpdateRequest{HevyAPIKey: "k"})
	h.expect(http.StatusBadRequest, "POST", userPath+"/hevy-sync", token, nil)

	withHevy(t)
	fakeHevyAPI(t, nil, nil)

	h.expect(http.StatusBadRequest, "POST", userPath+"/hevy", token, models.UserHevyAPIKeyUpdateRequest{HevyAPIKey: "  "})
	h.expect(http.StatusBadRequest, "POST", userPath+"/hevy", token, models.UserHevyAPIKeyUpdateRequest{HevyAPIKey: "rejected-key"})
	h.expect(http.StatusBadRequest, "POST", userPath+"/hevy-sync", token, nil)

	h.ok("POST", userPath+"/hevy", token, models.UserHevyAPIKeyUpdateRequest{HevyAPIKey: "good-key"})
	stored, _ := database.GetAllUserInformation(user.ID)
	if stored.HevyAPIKey == nil || strings.Contains(*stored.HevyAPIKey, "good-key") {
		t.Errorf("stored Hevy key = %v, want it present and encrypted", stored.HevyAPIKey)
	}
	if stored.HevyProfileURL == nil || *stored.HevyProfileURL != "https://hevy.com/user/lifter" {
		t.Errorf("profile url = %v", stored.HevyProfileURL)
	}

	h.ok("POST", userPath+"/hevy-sync", token, nil)
	h.ok("DELETE", userPath+"/hevy", token, nil)
	cleared, _ := database.GetAllUserInformation(user.ID)
	if cleared.HevyAPIKey != nil {
		t.Error("disconnecting left the Hevy key stored")
	}
}

func TestHevyBackfillAndEventsSync(t *testing.T) {
	h := newAPIHarness(t)
	withHevy(t)
	user, token := h.user("importer@hevy.test", false)

	encrypted, err := encryptHevyAPIKey("good-key")
	if err != nil {
		t.Fatal(err)
	}
	user.HevyAPIKey = &encrypted
	if _, err := database.UpdateUser(user); err != nil {
		t.Fatal(err)
	}

	first := hevyWorkout("workout-1", todayAt(3), "Legs")
	second := hevyWorkout("workout-2", todayAt(5), "More legs")
	fakeHevyAPI(t, []models.HevyWorkout{first, second}, nil)

	stored, _ := database.GetAllUserInformation(user.ID)
	if err := HevyBackfillForUser(stored); err != nil {
		t.Fatalf("backfill: %v", err)
	}
	countOperations := func() int {
		response := h.ok("GET", "/api/auth/operations", token, nil)
		return len(field(t, response, "operations").([]any))
	}
	if got := countOperations(); got != 4 {
		t.Fatalf("operations after backfill = %d, want 2 workouts × 2 exercises", got)
	}

	// Backfilling again replaces rather than duplicates.
	stored, _ = database.GetAllUserInformation(user.ID)
	if err := HevyBackfillForUser(stored); err != nil {
		t.Fatalf("second backfill: %v", err)
	}
	if got := countOperations(); got != 4 {
		t.Errorf("operations after re-backfill = %d, want still 4", got)
	}
	if stored.HevyLastSync == nil {
		t.Fatal("backfill did not record the sync baseline")
	}

	// The custom exercise's name survives even though it has no global action.
	operations := h.ok("GET", "/api/auth/operations", token, nil)
	sawCustom := false
	for _, operation := range field(t, operations, "operations").([]any) {
		if note, _ := operation.(map[string]any)["note"].(string); strings.Contains(note, "My Secret Move") {
			sawCustom = true
		}
	}
	if !sawCustom {
		t.Error("the custom Hevy exercise name was lost")
	}

	// Events: the first workout is edited down to one exercise, the second is deleted.
	edited := first
	edited.Title = "Legs (edited)"
	edited.Exercises = edited.Exercises[:1]
	fakeHevyAPI(t, nil, []models.HevyWorkoutEvent{
		{Type: "updated", Workout: &edited},
		{Type: "deleted", ID: "workout-2", DeletedAt: time.Now().Format(time.RFC3339)},
	})
	if err := HevyEventsSyncForUser(stored); err != nil {
		t.Fatalf("events sync: %v", err)
	}
	if got := countOperations(); got != 1 {
		t.Errorf("operations after events = %d, want 1 (edited workout's single exercise)", got)
	}

	HevyEventsSyncForAllUsers()

	// Without a key, or before the first backfill, the events sync is a no-op.
	if err := HevyEventsSyncForUser(models.User{}); err != nil {
		t.Errorf("no key: %v", err)
	}
	if err := HevyBackfillForUser(models.User{}); err == nil {
		t.Error("backfill without a key should fail")
	}
}

// A backfill runs in the background on the user row loaded when it started. It used to
// finish by saving that whole stale row, which wrote the Hevy key back if the user had
// disconnected meanwhile — resurrecting the connection.
func TestHevyBackfillDoesNotResurrectADisconnectedKey(t *testing.T) {
	h := newAPIHarness(t)
	withHevy(t)
	user, token := h.user("quitter@hevy.test", false)

	encrypted, _ := encryptHevyAPIKey("good-key")
	user.HevyAPIKey = &encrypted
	if _, err := database.UpdateUser(user); err != nil {
		t.Fatal(err)
	}
	fakeHevyAPI(t, []models.HevyWorkout{hevyWorkout("w", todayAt(3), "Legs")}, nil)

	staleCopy, _ := database.GetAllUserInformation(user.ID)
	h.ok("DELETE", "/api/auth/users/"+user.ID.String()+"/hevy", token, nil)

	if err := HevyBackfillForUser(staleCopy); err != nil {
		t.Fatalf("backfill: %v", err)
	}
	after, _ := database.GetAllUserInformation(user.ID)
	if after.HevyAPIKey != nil {
		t.Error("the finished backfill wrote the removed Hevy key back")
	}
	if after.HevyLastSync != nil {
		t.Error("a sync baseline was recorded for a disconnected user")
	}
}
