package controllers

import (
	"net/http"
	"testing"
	"time"

	"github.com/aunefyren/treningheten/models"

	"github.com/google/uuid"
)

func float64Ptr(value float64) *float64 { return &value }
func int64Ptr(value int64) *int64       { return &value }
func stringPtr(value string) *string    { return &value }
func boolPtr(value bool) *bool          { return &value }

// currentWeekDays returns this ISO week's Monday..Sunday as request days, with the given
// exercise interval on every day up to and including today (future days must be 0).
func currentWeekDays(interval int) []models.ExerciseDayCreationRequest {
	now := time.Now()
	monday := time.Date(now.Year(), now.Month(), now.Day(), 12, 0, 0, 0, time.Local)
	for monday.Weekday() != time.Monday {
		monday = monday.AddDate(0, 0, -1)
	}

	days := make([]models.ExerciseDayCreationRequest, 0, 7)
	for i := 0; i < 7; i++ {
		date := monday.AddDate(0, 0, i)
		dayInterval := interval
		if date.After(now) {
			dayInterval = 0
		}
		days = append(days, models.ExerciseDayCreationRequest{Date: date, Note: "day note", ExerciseInterval: dayInterval})
	}
	return days
}

// createTodaySession opens today's exercise day and adds one session to it, returning the
// session (exercise) id.
func createTodaySession(t *testing.T, h *apiHarness, token string) string {
	t.Helper()

	day := h.ok("GET", "/api/auth/exercise-days/week?today=true", token, nil)
	dayID := idOf(t, day, "exercise")

	created := h.expect(http.StatusCreated, "POST", "/api/auth/exercises", token, models.ExerciseCreationRequest{
		ExerciseDayID: uuid.MustParse(dayID), Note: "session note", IsOn: true, Duration: int64Ptr(3600),
	})
	return idOf(t, created, "exercise")
}

func TestWorkoutLoggingFlow(t *testing.T) {
	h := newAPIHarness(t)
	user, token := h.user("lifter@workout.test", false)
	_, strangerToken := h.user("stranger@workout.test", false)

	// Day lookups by weekday and their validation.
	h.ok("GET", "/api/auth/exercise-days/week?weekDay=3", token, nil)
	h.expect(http.StatusBadRequest, "GET", "/api/auth/exercise-days/week", token, nil)
	h.expect(http.StatusBadRequest, "GET", "/api/auth/exercise-days/week?weekDay=x", token, nil)
	h.expect(http.StatusBadRequest, "GET", "/api/auth/exercise-days/week?weekDay=9", token, nil)

	sessionID := createTodaySession(t, h, token)

	// Session update: time, goal flag and privacy.
	h.ok("PUT", "/api/auth/exercises/"+sessionID, token, models.ExerciseUpdateRequest{
		Note: "updated", IsOn: true, Duration: int64Ptr(1800), Time: time.Now().Format(time.RFC3339),
		CountsTowardGoal: boolPtr(true), Private: boolPtr(false),
	})
	if code := h.do("PUT", "/api/auth/exercises/"+sessionID, strangerToken, models.ExerciseUpdateRequest{Note: "hijack"}).Code; code < 400 {
		t.Errorf("stranger updating a session: status = %d, want an error", code)
	}
	h.expect(http.StatusBadRequest, "PUT", "/api/auth/exercises/not-a-uuid", token, models.ExerciseUpdateRequest{})

	// Pick a seeded action and log a lifting activity with two sets.
	actions := h.ok("GET", "/api/auth/actions", token, nil)
	actionList, _ := field(t, actions, "actions").([]any)
	if len(actionList) == 0 {
		t.Fatal("no seeded actions returned")
	}
	actionID := uuid.MustParse(idOf(t, actionList[0]))
	actionName, _ := field(t, actionList[0], "name").(string)
	h.ok("GET", "/api/auth/actions?experienced=true", token, nil)

	operation := h.expect(http.StatusCreated, "POST", "/api/auth/operations", token, models.OperationCreationRequest{
		ExerciseID: uuid.MustParse(sessionID), Action: &actionID, Type: "lifting", WeightUnit: "kg", DistanceUnit: "km",
	})
	operationID := idOf(t, operation, "operation")
	if got := field(t, operation, "operation", "action", "id"); got != actionID.String() {
		t.Errorf("created operation action = %v, want %s", got, actionID)
	}
	if got, want := field(t, operation, "operation", "type"), field(t, actionList[0], "type"); got != want {
		t.Errorf("created operation type = %v, want the action's type %v", got, want)
	}

	set := h.expect(http.StatusCreated, "POST", "/api/auth/operation-sets", token, models.OperationSetCreationRequest{
		OperationID: uuid.MustParse(operationID), Repetitions: float64Ptr(10), Weight: float64Ptr(60),
	})
	setID := idOf(t, set, "operation_set")
	h.expect(http.StatusCreated, "POST", "/api/auth/operation-sets", token, models.OperationSetCreationRequest{
		OperationID: uuid.MustParse(operationID), Repetitions: float64Ptr(8), Weight: float64Ptr(70), Time: int64Ptr(90),
	})
	if code := h.do("POST", "/api/auth/operation-sets", strangerToken, models.OperationSetCreationRequest{OperationID: uuid.MustParse(operationID)}).Code; code < 400 {
		t.Errorf("stranger adding a set: status = %d, want an error", code)
	}

	h.ok("PUT", "/api/auth/operation-sets/"+setID, token, models.OperationSetUpdateRequest{
		Repetitions: float64Ptr(12), Weight: float64Ptr(62.5), Distance: float64Ptr(0), MovingTime: int64Ptr(30),
	})
	if code := h.do("PUT", "/api/auth/operation-sets/"+setID, strangerToken, models.OperationSetUpdateRequest{Repetitions: float64Ptr(1)}).Code; code < 400 {
		t.Errorf("stranger updating a set: status = %d, want an error", code)
	}

	sets := h.ok("GET", "/api/auth/operation-sets?operation_id="+operationID, token, nil)
	if got := len(field(t, sets, "operation_sets").([]any)); got != 2 {
		t.Errorf("operation sets = %d, want 2", got)
	}
	h.expect(http.StatusBadRequest, "GET", "/api/auth/operation-sets?operation_id=nope", token, nil)

	// Operation update: tags and description.
	updated := h.ok("PUT", "/api/auth/operations/"+operationID, token, models.OperationUpdateRequest{
		Action: actionName, Type: "lifting", WeightUnit: "lb", DistanceUnit: "miles", Equipment: "barbells",
		Tags: &[]string{"race"}, Description: stringPtr("heavy day"),
	})
	if unit := field(t, updated, "operation", "weight_unit"); unit != "lb" {
		t.Errorf("weight_unit = %v, want lb", unit)
	}
	for name, request := range map[string]models.OperationUpdateRequest{
		"unknown tag":       {Type: "lifting", WeightUnit: "kg", DistanceUnit: "km", Tags: &[]string{"not-a-tag"}},
		"unknown action":    {Action: "No such exercise", Type: "lifting", WeightUnit: "kg", DistanceUnit: "km"},
		"unknown type":      {Type: "flying", WeightUnit: "kg", DistanceUnit: "km"},
		"unknown equipment": {Type: "lifting", WeightUnit: "kg", DistanceUnit: "km", Equipment: "anvil"},
		"unknown unit":      {Type: "lifting", WeightUnit: "stone", DistanceUnit: "km"},
	} {
		if code := h.do("PUT", "/api/auth/operations/"+operationID, token, request).Code; code != http.StatusBadRequest {
			t.Errorf("%s: status = %d, want 400", name, code)
		}
	}

	fetched := h.ok("GET", "/api/auth/operations/"+operationID, token, nil)
	if description := field(t, fetched, "operation", "description"); description != "heavy day" {
		t.Errorf("description = %v, want the updated one", description)
	}
	if code := h.do("GET", "/api/auth/operations/"+operationID, strangerToken, nil).Code; code < 400 {
		t.Errorf("stranger reading an operation: status = %d, want an error", code)
	}
	h.ok("GET", "/api/auth/operations", token, nil)

	// Reads built on top of the logged data.
	now := time.Now()
	statsPath := "/api/auth/actions/" + actionID.String() + "/statistics"
	h.ok("GET", statsPath+"?start="+now.AddDate(0, -1, 0).UTC().Format(time.RFC3339)+"&end="+now.AddDate(0, 0, 1).UTC().Format(time.RFC3339), token, nil)
	h.expect(http.StatusBadRequest, "GET", statsPath, token, nil)
	h.expect(http.StatusBadRequest, "GET", statsPath+"?start=yesterday&end=today", token, nil)
	h.expect(http.StatusBadRequest, "GET", "/api/auth/actions/nope/statistics", token, nil)

	h.ok("GET", "/api/auth/activities", token, nil)
	h.ok("GET", "/api/auth/activities?q=heavy&sort=date&order=asc&limit=5&offset=0", token, nil)
	h.ok("GET", "/api/auth/users/"+user.ID.String()+"/activities", token, nil)
	h.ok("GET", "/api/auth/users/"+user.ID.String()+"/statistics", token, nil)
	h.ok("GET", "/api/auth/exercises/week", token, nil)
	h.ok("GET", "/api/auth/exercise-days?year="+now.Format("2006"), token, nil)

	day := h.ok("GET", "/api/auth/exercise-days/week?today=true", token, nil)
	dayID := idOf(t, day, "exercise")
	h.expect(http.StatusCreated, "GET", "/api/auth/exercise-days/"+dayID, token, nil)
	h.expect(http.StatusCreated, "POST", "/api/auth/exercise-days/"+dayID, token, models.ExerciseDayUpdateRequest{Note: "good day"})

	// Tear down: a set, then the operation.
	h.ok("DELETE", "/api/auth/operation-sets/"+setID, token, nil)
	if code := h.do("DELETE", "/api/auth/operations/"+operationID, strangerToken, nil).Code; code < 400 {
		t.Errorf("stranger deleting an operation: status = %d, want an error", code)
	}
	h.ok("DELETE", "/api/auth/operations/"+operationID, token, nil)
	if code := h.do("GET", "/api/auth/operations/"+operationID, token, nil).Code; code < 400 {
		t.Errorf("reading a deleted operation: status = %d, want an error", code)
	}
}

func TestMovingActivityAndGear(t *testing.T) {
	h := newAPIHarness(t)
	_, token := h.user("runner@gear.test", false)
	_, strangerToken := h.user("stranger@gear.test", false)

	sessionID := createTodaySession(t, h, token)
	operation := h.expect(http.StatusCreated, "POST", "/api/auth/operations", token, models.OperationCreationRequest{
		ExerciseID: uuid.MustParse(sessionID), Type: "moving", WeightUnit: "kg", DistanceUnit: "km",
	})
	operationID := idOf(t, operation, "operation")
	h.expect(http.StatusCreated, "POST", "/api/auth/operation-sets", token, models.OperationSetCreationRequest{
		OperationID: uuid.MustParse(operationID), Distance: float64Ptr(10.5), Time: int64Ptr(3000),
	})

	// Gear CRUD with validation.
	h.expect(http.StatusBadRequest, "POST", "/api/auth/gear", token, models.GearCreationRequest{Name: "", Type: "shoe"})
	h.expect(http.StatusBadRequest, "POST", "/api/auth/gear", token, models.GearCreationRequest{Name: "Boat", Type: "boat"})
	shoe := h.expect(http.StatusCreated, "POST", "/api/auth/gear", token, models.GearCreationRequest{
		Name: "Racer", Type: "shoe", Brand: stringPtr("Brand"), Model: stringPtr("Model 1"), Nickname: stringPtr("fast"),
	})
	shoeID := idOf(t, shoe, "gear")

	h.ok("PUT", "/api/auth/exercises/"+sessionID+"/gear", token, map[string]any{"gear_id": shoeID})
	if code := h.do("PUT", "/api/auth/exercises/"+sessionID+"/gear", strangerToken, map[string]any{"gear_id": shoeID}).Code; code < 400 {
		t.Errorf("stranger setting gear: status = %d, want an error", code)
	}

	gear := h.ok("GET", "/api/auth/gear", token, nil)
	gearList := field(t, gear, "gear").([]any)
	if len(gearList) != 1 {
		t.Fatalf("gear = %d, want 1", len(gearList))
	}
	if distance := field(t, gearList[0], "distance"); distance != 10.5 {
		t.Errorf("gear distance = %v, want 10.5 summed from the linked set", distance)
	}

	h.ok("PUT", "/api/auth/gear/"+shoeID, token, models.GearUpdateRequest{Name: stringPtr("Racer 2"), Retired: boolPtr(true), IsPrimary: boolPtr(true)})
	h.expect(http.StatusBadRequest, "PUT", "/api/auth/gear/"+shoeID, token, models.GearUpdateRequest{Type: stringPtr("boat")})
	if code := h.do("PUT", "/api/auth/gear/"+shoeID, strangerToken, models.GearUpdateRequest{Name: stringPtr("mine")}).Code; code < 400 {
		t.Errorf("stranger updating gear: status = %d, want an error", code)
	}

	// The operation carries the gear and can have it cleared.
	h.ok("PUT", "/api/auth/operations/"+operationID, token, models.OperationUpdateRequest{Type: "moving", WeightUnit: "kg", DistanceUnit: "km", GearID: stringPtr("")})
	h.ok("PUT", "/api/auth/operations/"+operationID, token, models.OperationUpdateRequest{Type: "moving", WeightUnit: "kg", DistanceUnit: "km", GearID: stringPtr(shoeID)})

	if code := h.do("DELETE", "/api/auth/gear/"+shoeID, strangerToken, nil).Code; code < 400 {
		t.Errorf("stranger deleting gear: status = %d, want an error", code)
	}
	h.ok("DELETE", "/api/auth/gear/"+shoeID, token, nil)
	h.expect(http.StatusBadRequest, "DELETE", "/api/auth/gear/nope", token, nil)
}

func TestRegisterWeek(t *testing.T) {
	h := newAPIHarness(t)
	_, token := h.user("weekly@workout.test", false)

	days := currentWeekDays(1)
	h.expect(http.StatusCreated, "POST", "/api/auth/exercises/week", token, models.WeekCreationRequest{Days: days, TimeZone: "UTC"})
	week := h.ok("GET", "/api/auth/exercises/week", token, nil)
	if got := len(field(t, week, "week", "days").([]any)); got != 7 {
		t.Errorf("week days = %d, want 7", got)
	}
	// Re-registering the same week updates it in place.
	h.expect(http.StatusCreated, "POST", "/api/auth/exercises/week", token, models.WeekCreationRequest{Days: currentWeekDays(2), TimeZone: "UTC"})

	invalid := map[string]models.WeekCreationRequest{
		"six days":       {Days: days[:6], TimeZone: "UTC"},
		"wrong order":    {Days: append([]models.ExerciseDayCreationRequest{days[1], days[0]}, days[2:]...), TimeZone: "UTC"},
		"four a day":     {Days: currentWeekDays(4), TimeZone: "UTC"},
		"last week":      {Days: shiftDays(days, -7), TimeZone: "UTC"},
		"duplicate days": {Days: append(append([]models.ExerciseDayCreationRequest{}, days[:6]...), days[5]), TimeZone: "UTC"},
	}
	for name, request := range invalid {
		if code := h.do("POST", "/api/auth/exercises/week", token, request).Code; code != http.StatusBadRequest {
			t.Errorf("%s: status = %d, want 400", name, code)
		}
	}
	if code := h.do("POST", "/api/auth/exercises/week", token, models.WeekCreationRequest{Days: days, TimeZone: "Nowhere/Zone"}).Code; code != http.StatusInternalServerError {
		t.Errorf("bad time zone: status = %d, want 500", code)
	}
}

func shiftDays(days []models.ExerciseDayCreationRequest, offset int) []models.ExerciseDayCreationRequest {
	shifted := make([]models.ExerciseDayCreationRequest, len(days))
	for i, day := range days {
		day.Date = day.Date.AddDate(0, 0, offset)
		shifted[i] = day
	}
	return shifted
}
