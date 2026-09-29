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

// seedSeason inserts a season with explicit dates and join policy.
func seedSeason(t *testing.T, name string, start, end time.Time, joinAnytime bool) models.Season {
	t.Helper()
	season := models.Season{Name: name, Description: "d", Start: start, End: end, PrizeID: seedPrize(t, name+" prize"), JoinAnytime: &joinAnytime, Enabled: true}
	season.ID = uuid.New()
	if err := database.Instance.Create(&season).Error; err != nil {
		t.Fatal(err)
	}
	return season
}

func seasonNames(t *testing.T, response map[string]any) string {
	t.Helper()
	names := []string{}
	for _, season := range field(t, response, "seasons").([]any) {
		names = append(names, field(t, season, "name").(string))
	}
	return strings.Join(names, ",")
}

func TestSeasonListsAndSundayReminders(t *testing.T) {
	h := newAPIHarness(t)
	server := withControllerSMTP(t)
	user, token := h.user("lists@season.test", false)
	_, quietToken := h.user("quiet@season.test", false)

	now := time.Now()
	future := seedSeason(t, "Future open", now.AddDate(0, 0, 14), now.AddDate(0, 0, 60), false)
	joinedFuture := seedSeason(t, "Future joined", now.AddDate(0, 0, 14), now.AddDate(0, 0, 60), false)
	ongoingOpen := seedSeason(t, "Ongoing open", now.AddDate(0, 0, -14), now.AddDate(0, 0, 30), true)
	seedSeason(t, "Ongoing closed", now.AddDate(0, 0, -14), now.AddDate(0, 0, 30), false)
	seedSeason(t, "Finished", now.AddDate(0, 0, -90), now.AddDate(0, 0, -30), true)
	seedGoal(t, user.ID, joinedFuture, 2)
	seedGoal(t, user.ID, ongoingOpen, 2)
	_ = future

	potential := seasonNames(t, h.ok("GET", "/api/auth/seasons?potential=true", token, nil))
	if potential != "Future open" {
		t.Errorf("potential seasons = %q, want only the future one not yet joined", potential)
	}
	countdown := seasonNames(t, h.ok("GET", "/api/auth/seasons?countdown=true", token, nil))
	if countdown != "Future joined" {
		t.Errorf("countdown seasons = %q, want only the joined future one", countdown)
	}
	// Someone who joined nothing may still join the open ongoing season.
	if other := seasonNames(t, h.ok("GET", "/api/auth/seasons?potential=true", quietToken, nil)); !strings.Contains(other, "Ongoing open") || strings.Contains(other, "Ongoing closed") {
		t.Errorf("potential seasons for a newcomer = %q", other)
	}
	if all := seasonNames(t, h.ok("GET", "/api/auth/seasons", token, nil)); strings.Count(all, ",") != 4 {
		t.Errorf("all seasons = %q, want all five", all)
	}

	// Sunday reminders go to members with the alert on — and only them.
	h.ok("PATCH", "/api/auth/users/"+user.ID.String(), token, models.UserPartialUpdateRequest{SundayAlert: boolPtr(true)})
	SendSundayReminders()
	messages := server.Messages()
	if len(messages) != 1 || messages[0].To[0] != "lists@season.test" {
		t.Errorf("reminders = %+v, want exactly one to the opted-in member", messages)
	}
}

func TestSharedFeedsRespectConsent(t *testing.T) {
	h := newAPIHarness(t)
	alice, aliceToken := h.user("alice@feed.test", false)
	bob, bobToken := h.user("bob@feed.test", false)
	_, strangerToken := h.user("stranger@feed.test", false)

	season := seedOngoingSeason(t, "Feed season", 1, 4)
	seedGoal(t, alice.ID, season, 2)
	seedGoal(t, bob.ID, season, 2)

	session := createTodaySession(t, h, bobToken)
	h.expect(http.StatusCreated, "POST", "/api/auth/operations", bobToken, models.OperationCreationRequest{
		ExerciseID: uuid.MustParse(session), Type: "moving", WeightUnit: "kg", DistanceUnit: "km",
	})

	feedHasBob := func(token string) bool {
		feed := h.ok("GET", "/api/auth/activities/shared", token, nil)
		return strings.Contains(mustJSON(t, feed), bob.ID.String())
	}
	if !feedHasBob(aliceToken) {
		t.Error("a season peer doesn't see a shared session")
	}
	if feedHasBob(strangerToken) {
		t.Error("someone who never shared a season sees the session")
	}

	h.ok("GET", "/api/auth/users/"+bob.ID.String()+"/activities", aliceToken, nil)
	h.ok("GET", "/api/auth/users/"+bob.ID.String()+"/statistics", aliceToken, nil)
	h.ok("GET", "/api/auth/seasons/"+season.ID.String()+"/activities", aliceToken, nil)

	// Bob withdraws consent: the peer feed drops him.
	database.Instance.Model(&models.User{}).Where("id = ?", bob.ID).Update("share_activities", false)
	if feedHasBob(aliceToken) {
		t.Error("the feed still shows a user who stopped sharing")
	}
}

func TestExerciseDaysByGoalAndWeekReduction(t *testing.T) {
	h := newAPIHarness(t)
	user, token := h.user("weeks@days.test", false)
	season := seedOngoingSeason(t, "Days season", 1, 4)
	goal := seedGoal(t, user.ID, season, 3)

	h.expect(http.StatusCreated, "POST", "/api/auth/exercises/week", token, models.WeekCreationRequest{Days: currentWeekDays(3), TimeZone: "UTC"})
	// Registering fewer sessions for the same days turns the extra ones off.
	h.expect(http.StatusCreated, "POST", "/api/auth/exercises/week", token, models.WeekCreationRequest{Days: currentWeekDays(1), TimeZone: "UTC"})

	var onCount int64
	database.Instance.Model(&models.Exercise{}).
		Joins("JOIN exercise_days ON exercise_days.id = exercises.exercise_day_id").
		Where("exercise_days.user_id = ? AND exercises.is_on = ?", user.ID, true).Count(&onCount)
	days := 0
	for _, day := range currentWeekDays(1) {
		if day.ExerciseInterval > 0 {
			days++
		}
	}
	if int(onCount) != days {
		t.Errorf("sessions on after reducing = %d, want one per past day (%d)", onCount, days)
	}

	byGoal := h.ok("GET", "/api/auth/exercise-days?goal="+goal.ID.String(), token, nil)
	if byGoal["exercise"] == nil {
		t.Errorf("exercise days by goal = %v", byGoal)
	}
	h.expect(http.StatusBadRequest, "GET", "/api/auth/exercise-days?goal=nope", token, nil)
	h.ok("GET", "/api/auth/exercise-days", token, nil)
}
