package controllers

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/aunefyren/treningheten/database"
	"github.com/aunefyren/treningheten/models"

	"github.com/google/uuid"
)

// seedDaySession inserts a day on date with one session, returning both ids.
func seedDaySession(t *testing.T, userID uuid.UUID, date time.Time, isOn bool) (uuid.UUID, uuid.UUID) {
	t.Helper()
	day := models.ExerciseDay{Date: date, Enabled: true, UserID: &userID}
	day.ID = uuid.New()
	if err := database.Instance.Omit("User", "Goal").Create(&day).Error; err != nil {
		t.Fatal(err)
	}
	session := models.Exercise{Enabled: true, IsOn: true, ExerciseDayID: day.ID}
	session.ID = uuid.New()
	if err := database.Instance.Omit("ExerciseDay").Create(&session).Error; err != nil {
		t.Fatal(err)
	}
	if !isOn {
		database.Instance.Model(&models.Exercise{}).Where("id = ?", session.ID).Update("is_on", false)
	}
	return day.ID, session.ID
}

// Sessions in a finished week are frozen: they can't be switched on or off, re-counted,
// or added to; the current week follows the day rules.
func TestSessionAndWeekRules(t *testing.T) {
	h := newAPIHarness(t)
	user, token := h.user("rules@validation.test", false)

	lastWeek := wednesdayOfWeeksAgo(1)
	pastDay, pastSession := seedDaySession(t, user.ID, lastWeek, true)
	_, pastOffSession := seedDaySession(t, user.ID, lastWeek.AddDate(0, 0, 1), false)
	pastPath := "/api/auth/exercises/" + pastSession.String()

	h.expect(http.StatusBadRequest, "PUT", pastPath, token, models.ExerciseUpdateRequest{IsOn: false})
	h.expect(http.StatusBadRequest, "PUT", "/api/auth/exercises/"+pastOffSession.String(), token, models.ExerciseUpdateRequest{IsOn: true})
	h.expect(http.StatusBadRequest, "PUT", pastPath, token, models.ExerciseUpdateRequest{IsOn: true, CountsTowardGoal: boolPtr(false)})
	h.expect(http.StatusBadRequest, "POST", "/api/auth/exercises", token, models.ExerciseCreationRequest{ExerciseDayID: pastDay, IsOn: true})

	// A session's time must fall on its own day, and parse.
	sessionID := createTodaySession(t, h, token)
	sessionPath := "/api/auth/exercises/" + sessionID
	h.expect(http.StatusBadRequest, "PUT", sessionPath, token, models.ExerciseUpdateRequest{IsOn: true, Time: "half past nine"})
	h.expect(http.StatusBadRequest, "PUT", sessionPath, token, models.ExerciseUpdateRequest{IsOn: true, Time: time.Now().AddDate(0, 0, -2).Format(time.RFC3339)})
	h.expect(http.StatusBadRequest, "PUT", sessionPath, token, models.ExerciseUpdateRequest{IsOn: true, Note: strings.Repeat("n", 300)})
	h.expect(http.StatusNotFound, "PUT", "/api/auth/exercises/"+uuid.NewString(), token, models.ExerciseUpdateRequest{IsOn: true})

	// Three sessions a day at most, by creation or by switching one back on.
	day := h.ok("GET", "/api/auth/exercise-days/week?today=true", token, nil)
	dayID := uuid.MustParse(idOf(t, day, "exercise"))
	second := h.expect(http.StatusCreated, "POST", "/api/auth/exercises", token, models.ExerciseCreationRequest{ExerciseDayID: dayID, IsOn: true})
	h.expect(http.StatusCreated, "POST", "/api/auth/exercises", token, models.ExerciseCreationRequest{ExerciseDayID: dayID, IsOn: true})
	h.expect(http.StatusBadRequest, "POST", "/api/auth/exercises", token, models.ExerciseCreationRequest{ExerciseDayID: dayID, IsOn: true})
	secondPath := "/api/auth/exercises/" + idOf(t, second, "exercise")
	h.ok("PUT", secondPath, token, models.ExerciseUpdateRequest{IsOn: false})
	h.ok("PUT", secondPath, token, models.ExerciseUpdateRequest{IsOn: true})

	// Week registration rules.
	days := currentWeekDays(1)
	long := append([]models.ExerciseDayCreationRequest{}, days...)
	long[0].Note = strings.Repeat("x", 300)
	h.expect(http.StatusBadRequest, "POST", "/api/auth/exercises/week", token, models.WeekCreationRequest{Days: long, TimeZone: "UTC"})
	future := append([]models.ExerciseDayCreationRequest{}, days...)
	future[6].ExerciseInterval = 1
	if future[6].Date.After(time.Now()) {
		h.expect(http.StatusBadRequest, "POST", "/api/auth/exercises/week", token, models.WeekCreationRequest{Days: future, TimeZone: "UTC"})
	}
}

func TestOperationCreationValidation(t *testing.T) {
	h := newAPIHarness(t)
	_, token := h.user("opvalid@validation.test", false)
	sessionID := uuid.MustParse(createTodaySession(t, h, token))
	equipment := "anvil"
	unknownAction := uuid.New()

	for name, request := range map[string]models.OperationCreationRequest{
		"type":          {ExerciseID: sessionID, Type: "flying", WeightUnit: "kg", DistanceUnit: "km"},
		"distance unit": {ExerciseID: sessionID, Type: "moving", WeightUnit: "kg", DistanceUnit: "parsecs"},
		"weight unit":   {ExerciseID: sessionID, Type: "lifting", WeightUnit: "stone", DistanceUnit: "km"},
		"equipment":     {ExerciseID: sessionID, Type: "lifting", WeightUnit: "kg", DistanceUnit: "km", Equipment: &equipment},
		"action":        {ExerciseID: sessionID, Action: &unknownAction, Type: "lifting", WeightUnit: "kg", DistanceUnit: "km"},
	} {
		if code := h.do("POST", "/api/auth/operations", token, request).Code; code != http.StatusBadRequest && code != http.StatusNotFound {
			t.Errorf("invalid %s: status = %d, want a client error", name, code)
		}
	}
	barbells := "barbells"
	h.expect(http.StatusCreated, "POST", "/api/auth/operations", token, models.OperationCreationRequest{ExerciseID: sessionID, Type: "lifting", WeightUnit: "lb", DistanceUnit: "miles", Equipment: &barbells})
}
