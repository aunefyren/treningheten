package controllers

import (
	"net/http"
	"testing"
	"time"

	"github.com/aunefyren/treningheten/database"
	"github.com/aunefyren/treningheten/models"

	"github.com/google/uuid"
)

// nextMonday returns the Monday at least `weeks` weeks from now, as a date.
func nextMonday(weeks int) time.Time {
	day := time.Now().AddDate(0, 0, 7*weeks)
	for day.Weekday() != time.Monday {
		day = day.AddDate(0, 0, 1)
	}
	return time.Date(day.Year(), day.Month(), day.Day(), 12, 0, 0, 0, time.UTC)
}

// seedPrize inserts a prize and returns its ID.
func seedPrize(t *testing.T, name string) uuid.UUID {
	t.Helper()
	prize := models.Prize{Name: name, Quantity: 1}
	prize.ID = uuid.New()
	if err := database.Instance.Create(&prize).Error; err != nil {
		t.Fatalf("failed to seed prize: %v", err)
	}
	return prize.ID
}

// seedOngoingSeason inserts a season that started `weeksAgo` weeks ago (on a Monday) and
// runs for `lengthWeeks`, open for joining at any time. The admin API refuses seasons
// that start in the past, so tests that need a running season seed it directly.
func seedOngoingSeason(t *testing.T, name string, weeksAgo int, lengthWeeks int) models.Season {
	t.Helper()

	now := time.Now()
	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)
	for start.Weekday() != time.Monday {
		start = start.AddDate(0, 0, -1)
	}
	start = start.AddDate(0, 0, -7*weeksAgo)
	end := start.AddDate(0, 0, 7*lengthWeeks-1)
	end = time.Date(end.Year(), end.Month(), end.Day(), 23, 59, 59, 0, time.Local)

	joinAnytime := true
	season := models.Season{
		Name:        name,
		Description: "A season for tests",
		Start:       start,
		End:         end,
		PrizeID:     seedPrize(t, name+" prize"),
		Sickleave:   2,
		JoinAnytime: &joinAnytime,
		Enabled:     true,
	}
	season.ID = uuid.New()
	if err := database.Instance.Create(&season).Error; err != nil {
		t.Fatalf("failed to seed season: %v", err)
	}
	return season
}

func TestSeasonAdminCreationValidation(t *testing.T) {
	h := newAPIHarness(t)
	_, adminToken := h.user("admin@season.test", true)
	_, userToken := h.user("user@season.test", false)

	h.expect(http.StatusOK, "POST", "/api/admin/prizes", adminToken, models.PrizeCreationRequest{Name: "Pizza", Quantity: 2})
	prizes := h.expect(http.StatusOK, "GET", "/api/admin/prizes", adminToken, nil)
	prizeList, _ := field(t, prizes, "prizes").([]any)
	if len(prizeList) != 1 {
		t.Fatalf("prizes = %v, want exactly the one created", prizeList)
	}
	prizeID := uuid.MustParse(idOf(t, prizeList[0]))

	start := nextMonday(1)
	valid := models.SeasonCreationRequest{
		Name: "Spring season", Description: "desc", Start: start, End: start.AddDate(0, 0, 13),
		Prize: prizeID, Sickleave: 1, TimeZone: "UTC",
	}

	// Non-admins never reach the handler.
	h.expect(http.StatusForbidden, "POST", "/api/admin/seasons", userToken, valid)
	// No token at all is a 401 challenge.
	h.expect(http.StatusUnauthorized, "POST", "/api/admin/seasons", "", valid)

	invalid := map[string]models.SeasonCreationRequest{}
	shortName := valid
	shortName.Name = "abc"
	invalid["short name"] = shortName
	notMonday := valid
	notMonday.Start = start.AddDate(0, 0, 1)
	invalid["start not a Monday"] = notMonday
	notSunday := valid
	notSunday.End = start.AddDate(0, 0, 12)
	invalid["end not a Sunday"] = notSunday
	sickleave := valid
	sickleave.Sickleave = 100
	invalid["sick leave out of range"] = sickleave
	unknownPrize := valid
	unknownPrize.Prize = uuid.New()
	invalid["unknown prize"] = unknownPrize
	past := valid
	past.Start = nextMonday(-3)
	invalid["start in the past"] = past

	for name, request := range invalid {
		if code := h.do("POST", "/api/admin/seasons", adminToken, request).Code; code != http.StatusBadRequest {
			t.Errorf("%s: status = %d, want 400", name, code)
		}
	}

	if code := h.do("POST", "/api/admin/seasons", adminToken, "{not json").Code; code != http.StatusBadRequest {
		t.Errorf("malformed body: status = %d, want 400", code)
	}

	h.expect(http.StatusCreated, "POST", "/api/admin/seasons", adminToken, valid)
	// Names are unique.
	h.expect(http.StatusBadRequest, "POST", "/api/admin/seasons", adminToken, valid)

	// A future season: joinable, listed, and the goal can be withdrawn before it starts.
	seasons := h.expect(http.StatusOK, "GET", "/api/auth/seasons", userToken, nil)
	seasonList, _ := field(t, seasons, "seasons").([]any)
	if len(seasonList) != 1 {
		t.Fatalf("seasons = %d, want 1", len(seasonList))
	}
	seasonID := idOf(t, seasonList[0])

	h.expect(http.StatusOK, "GET", "/api/auth/seasons/"+seasonID, userToken, nil)
	h.expect(http.StatusBadRequest, "GET", "/api/auth/seasons/not-a-uuid", userToken, nil)

	h.expect(http.StatusBadRequest, "POST", "/api/auth/goals", userToken, models.GoalCreationRequest{ExerciseInterval: 0, SeasonID: uuid.MustParse(seasonID)})
	h.expect(http.StatusBadRequest, "POST", "/api/auth/goals", userToken, models.GoalCreationRequest{ExerciseInterval: 22, SeasonID: uuid.MustParse(seasonID)})
	h.expect(http.StatusCreated, "POST", "/api/auth/goals", userToken, models.GoalCreationRequest{ExerciseInterval: 3, Competing: true, SeasonID: uuid.MustParse(seasonID)})
	h.expect(http.StatusBadRequest, "POST", "/api/auth/goals", userToken, models.GoalCreationRequest{ExerciseInterval: 3, SeasonID: uuid.MustParse(seasonID)})

	goals := h.expect(http.StatusCreated, "GET", "/api/auth/goals", userToken, nil)
	goalList, _ := field(t, goals, "goals").([]any)
	if len(goalList) != 1 {
		t.Fatalf("goals = %d, want 1", len(goalList))
	}
	if left := field(t, goalList[0], "sickleave_left"); left != float64(1) {
		t.Errorf("sickleave_left = %v, want 1 (the season's allowance)", left)
	}
	goalID := idOf(t, goalList[0])

	_, strangerToken := h.user("stranger@season.test", false)
	h.expect(http.StatusUnauthorized, "DELETE", "/api/auth/goals/"+goalID, strangerToken, nil)
	h.expect(http.StatusBadRequest, "DELETE", "/api/auth/goals/nope", userToken, nil)
	h.expect(http.StatusCreated, "DELETE", "/api/auth/goals/"+goalID, userToken, nil)
}

func TestOngoingSeasonReads(t *testing.T) {
	h := newAPIHarness(t)
	user, token := h.user("runner@season.test", false)
	_, otherToken := h.user("rival@season.test", false)
	season := seedOngoingSeason(t, "Ongoing season", 2, 6)

	h.expect(http.StatusCreated, "POST", "/api/auth/goals", token, models.GoalCreationRequest{ExerciseInterval: 2, Competing: true, SeasonID: season.ID})
	h.expect(http.StatusCreated, "POST", "/api/auth/goals", otherToken, models.GoalCreationRequest{ExerciseInterval: 1, Competing: true, SeasonID: season.ID})

	// Two sessions last week and one this week for the runner.
	lastWeek := time.Now().AddDate(0, 0, -7)
	for lastWeek.Weekday() != time.Wednesday {
		lastWeek = lastWeek.AddDate(0, 0, -1)
	}
	seedExerciseDayWithExercises(t, user.ID, lastWeek, 2)

	seasonPath := "/api/auth/seasons/" + season.ID.String()
	h.ok("GET", "/api/auth/seasons/get-on-going", token, nil)
	h.ok("GET", seasonPath+"/leaderboard", token, nil)
	weeks := h.ok("GET", seasonPath+"/weeks", token, nil)
	if _, found := weeks["leaderboard"]; !found && len(weeks) < 2 {
		t.Errorf("weeks response looks empty: %v", weeks)
	}
	h.ok("GET", seasonPath+"/weeks-personal", token, nil)
	h.ok("GET", seasonPath+"/activities", token, nil)

	unknown := "/api/auth/seasons/" + uuid.NewString()
	for _, path := range []string{unknown + "/leaderboard", unknown + "/weeks", unknown + "/weeks-personal"} {
		if code := h.do("GET", path, token, nil).Code; code < 400 {
			t.Errorf("GET %s: status = %d, want an error for an unknown season", path, code)
		}
	}

	// Sick leave: registering it for the current week uses one of the season's two.
	h.expect(http.StatusOK, "POST", "/api/auth/sickleave/"+season.ID.String(), token, nil)
	h.expect(http.StatusBadRequest, "POST", "/api/auth/sickleave/"+season.ID.String(), token, nil)
	h.expect(http.StatusBadRequest, "POST", "/api/auth/sickleave/not-a-uuid", token, nil)

	// A goal can't be withdrawn once the season has started.
	goals := h.expect(http.StatusCreated, "GET", "/api/auth/goals", token, nil)
	goalID := idOf(t, field(t, goals, "goals").([]any)[0])
	h.expect(http.StatusBadRequest, "DELETE", "/api/auth/goals/"+goalID, token, nil)
}

// Goal.Competing and Exercise.IsOn are tagged default:true, so GORM dropped a false value
// from the INSERT: a member who joined as non-competing was stored as competing (and could
// be handed debts), and a session created "off" was stored as on.
func TestFalseFlagsSurviveCreation(t *testing.T) {
	h := newAPIHarness(t)
	_, token := h.user("casual@flags.test", false)
	season := seedOngoingSeason(t, "Flags season", 1, 4)

	h.expect(http.StatusCreated, "POST", "/api/auth/goals", token, models.GoalCreationRequest{ExerciseInterval: 2, Competing: false, SeasonID: season.ID})
	goals := h.expect(http.StatusCreated, "GET", "/api/auth/goals", token, nil)
	if competing := field(t, field(t, goals, "goals").([]any)[0], "competing"); competing != false {
		t.Errorf("competing = %v, want false as requested", competing)
	}

	day := h.ok("GET", "/api/auth/exercise-days/week?today=true", token, nil)
	created := h.expect(http.StatusCreated, "POST", "/api/auth/exercises", token, models.ExerciseCreationRequest{
		ExerciseDayID: uuid.MustParse(idOf(t, day, "exercise")), IsOn: false,
	})
	if isOn := field(t, created, "exercise", "is_on"); isOn != false {
		t.Errorf("is_on = %v, want false as requested", isOn)
	}
	exerciseID := idOf(t, created, "exercise")
	var stored models.Exercise
	database.Instance.Where("id = ?", exerciseID).First(&stored)
	if stored.IsOn {
		t.Error("the stored session is on")
	}
}
