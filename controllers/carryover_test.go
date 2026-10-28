package controllers

import (
	"encoding/json"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/aunefyren/treningheten/database"
	"github.com/aunefyren/treningheten/models"

	"github.com/google/uuid"
)

// --- The rule ----------------------------------------------------------------

func TestNextCarryBank(t *testing.T) {
	twoForOneWeek := carryOverSettings{max: 2, weeks: 1}
	twoForTwoWeeks := carryOverSettings{max: 2, weeks: 2}

	tests := []struct {
		name       string
		bank       carryBank
		workouts   int
		goal       int
		optedIn    bool
		sickLeave  bool
		week       int
		settings   carryOverSettings
		wantUsed   int
		wantEarned int
		wantBank   carryBank
	}{
		{"extra workouts are saved", carryBank{}, 4, 3, false, false, 0, twoForOneWeek, 0, 1, carryBank{1}},
		{"saving stops at the cap", carryBank{}, 7, 3, false, false, 0, twoForOneWeek, 0, 2, carryBank{1, 1}},
		{"the cap counts what is already held", carryBank{1}, 6, 3, false, false, 1, twoForOneWeek, 0, 1, carryBank{1, 2}},
		{"a full bank saves nothing more", carryBank{1, 1}, 5, 3, false, false, 1, twoForOneWeek, 0, 0, carryBank{1, 1}},
		{"meeting the goal exactly saves nothing", carryBank{}, 3, 3, false, false, 0, twoForOneWeek, 0, 0, carryBank{}},
		{"a saved workout expires after its weeks", carryBank{1}, 0, 3, true, false, 2, twoForOneWeek, 0, 0, carryBank{}},
		{"a saved workout lasts as many weeks as the season says", carryBank{2}, 2, 3, true, false, 2, twoForTwoWeeks, 1, 0, carryBank{}},
		{"oldest are used first", carryBank{2, 3}, 2, 3, true, false, 2, twoForTwoWeeks, 1, 0, carryBank{3}},
		{"a top-up covers a partly done week", carryBank{3, 3}, 2, 3, true, false, 2, twoForTwoWeeks, 1, 0, carryBank{3}},
		{"they cover the whole shortfall or nothing", carryBank{3}, 1, 3, true, false, 2, twoForTwoWeeks, 0, 0, carryBank{3}},
		{"an empty week can be covered whole", carryBank{3, 3}, 0, 2, true, false, 2, twoForTwoWeeks, 2, 0, carryBank{}},
		{"nothing is spent without opting in", carryBank{3, 3}, 2, 3, false, false, 2, twoForTwoWeeks, 0, 0, carryBank{3, 3}},
		{"a met week never spends, even opted in", carryBank{3}, 3, 3, true, false, 2, twoForTwoWeeks, 0, 0, carryBank{3}},
		{"an opted-in week beyond its goal still saves", carryBank{3}, 5, 3, true, false, 2, twoForTwoWeeks, 0, 1, carryBank{3, 4}},
		{"sick leave neither spends", carryBank{3}, 2, 3, true, true, 2, twoForTwoWeeks, 0, 0, carryBank{3}},
		{"sick leave neither earns", carryBank{}, 6, 3, false, true, 2, twoForTwoWeeks, 0, 0, carryBank{}},
		{"saved workouts keep ageing through sick leave", carryBank{1}, 0, 3, false, true, 2, twoForOneWeek, 0, 0, carryBank{}},
		{"off when the season holds none", carryBank{}, 6, 3, false, false, 0, carryOverSettings{max: 0, weeks: 1}, 0, 0, carryBank{}},
		{"off without a lifetime", carryBank{}, 6, 3, false, false, 0, carryOverSettings{max: 2, weeks: 0}, 0, 0, carryBank{}},
		{"a goal of zero never saves", carryBank{}, 3, 0, false, false, 0, twoForOneWeek, 0, 0, carryBank{}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			before := append(carryBank{}, tc.bank...)
			used, earned, bank := nextCarryBank(tc.bank, tc.workouts, tc.goal, tc.optedIn, tc.sickLeave, tc.week, tc.settings)
			if used != tc.wantUsed || earned != tc.wantEarned {
				t.Errorf("used %d, earned %d; want used %d, earned %d", used, earned, tc.wantUsed, tc.wantEarned)
			}
			if !reflect.DeepEqual(bank, tc.wantBank) {
				t.Errorf("bank = %v, want %v", bank, tc.wantBank)
			}
			if !reflect.DeepEqual(tc.bank, before) {
				t.Errorf("the bank passed in was changed: %v, was %v", tc.bank, before)
			}
		})
	}
}

func TestSavedWorkoutsRule(t *testing.T) {
	tests := map[string]struct {
		max, weeks int
		want       string
	}{
		"off":                 {0, 0, ""},
		"off with a lifetime": {0, 3, ""},
		"one for a week":      {1, 1, "1 extra workout can be saved, for 1 week."},
		"several for a week":  {2, 1, "Up to 2 extra workouts can be saved, each for 1 week."},
		"several for weeks":   {3, 4, "Up to 3 extra workouts can be saved, each for 4 weeks."},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			if got := savedWorkoutsRule(tc.max, tc.weeks); got != tc.want {
				t.Errorf("savedWorkoutsRule(%d, %d) = %q, want %q", tc.max, tc.weeks, got, tc.want)
			}
		})
	}
}

func TestSundayReminderSavedWorkoutsNote(t *testing.T) {
	tests := map[string]struct {
		result models.UserWeekResults
		want   string
	}{
		"short and holding enough": {models.UserWeekResults{WeekCompletion: 2.0 / 3, CarriedIn: 2}, "You're 1 workout short of your goal, and you have 2 saved workouts"},
		"already made up":          {models.UserWeekResults{WeekCompletion: 2.0 / 3, CarriedIn: 2, CarryOptIn: true, CarryUsed: 1}, "Your saved workouts make up the rest"},
		"not enough saved":         {models.UserWeekResults{WeekCompletion: 0, CarriedIn: 2}, ""},
		"goal met":                 {models.UserWeekResults{WeekCompletion: 1, CarriedIn: 2}, ""},
		"sick leave":               {models.UserWeekResults{WeekCompletion: 2.0 / 3, CarriedIn: 2, SickLeave: true}, ""},
		"nothing saved":            {models.UserWeekResults{WeekCompletion: 2.0 / 3}, ""},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := sundayReminderSavedWorkoutsNote(tc.result, 3)
			if (tc.want == "") != (got == "") || !strings.HasPrefix(got, tc.want) {
				t.Errorf("note = %q, want it to start with %q", got, tc.want)
			}
		})
	}
}

// --- The season walk ---------------------------------------------------------

// carryOverSeason is a four-week season starting 2024-01-01 (a Monday) with saved workouts on.
func carryOverSeason(goals ...models.GoalObject) models.SeasonObject {
	return models.SeasonObject{
		Start:          time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		End:            time.Date(2024, 1, 28, 23, 59, 59, 0, time.UTC),
		Goals:          goals,
		CarryOverMax:   2,
		CarryOverWeeks: 2,
	}
}

func seedCarryOverUse(t *testing.T, goalID uuid.UUID, date time.Time) {
	t.Helper()
	use := models.CarryOverUse{Enabled: true, GoalID: goalID, Date: date}
	use.ID = uuid.New()
	if err := database.CreateCarryOverUseInDB(use); err != nil {
		t.Fatalf("failed to seed the use of saved workouts: %v", err)
	}
}

// resultsByWeek returns one goal's results oldest week first.
func resultsByWeek(t *testing.T, weeks []models.WeekResults, goalID uuid.UUID) []models.UserWeekResults {
	t.Helper()
	results := []models.UserWeekResults{}
	for i := len(weeks) - 1; i >= 0; i-- {
		result := userWeekResult(weeks[i], goalID)
		if result == nil {
			t.Fatalf("no result for goal in week %d", weeks[i].WeekNumber)
		}
		results = append(results, *result)
	}
	return results
}

func TestSeasonWalkSavedWeekCountsAsMet(t *testing.T) {
	newControllerTestDB(t)

	saver := makeGoalObject(uuid.New(), uuid.New(), 3, time.Date(2023, 12, 1, 0, 0, 0, 0, time.UTC))
	season := carryOverSeason(saver)

	seedExerciseDayWithExercises(t, saver.User.ID, wednesdayOfWeek(season.Start, 0), 3)
	seedExerciseDayWithExercises(t, saver.User.ID, wednesdayOfWeek(season.Start, 0).AddDate(0, 0, 1), 2) // week 1: 5/3, saves 2
	seedExerciseDayWithExercises(t, saver.User.ID, wednesdayOfWeek(season.Start, 1), 2)                  // week 2: 2/3, opted in
	seedCarryOverUse(t, saver.ID, wednesdayOfWeek(season.Start, 1))
	seedExerciseDayWithExercises(t, saver.User.ID, wednesdayOfWeek(season.Start, 2), 1) // week 3: 1/3, opted in, 1 left: not enough
	seedCarryOverUse(t, saver.ID, wednesdayOfWeek(season.Start, 2))
	seedExerciseDayWithExercises(t, saver.User.ID, wednesdayOfWeek(season.Start, 3), 3) // week 4: met

	weeks, err := RetrieveWeekResultsFromSeasonWithinTimeframe(season.Start, season.End, season)
	if err != nil {
		t.Fatal(err)
	}
	results := resultsByWeek(t, weeks, saver.ID)

	type want struct {
		carriedIn, used, earned, streakIn int
		met, saved                        bool
	}
	wants := []want{
		{0, 0, 2, 0, true, false},
		{2, 1, 0, 1, true, true},   // made up with one saved workout: met, streak grows
		{1, 0, 0, 2, false, false}, // one left can't cover two missing: not spent, streak breaks
		{0, 0, 0, 0, true, false},  // earned in week 1 and lasting two weeks, it ran out after week 3
	}
	for i, w := range wants {
		r := results[i]
		if r.CarriedIn != w.carriedIn || r.CarryUsed != w.used || r.CarryEarned != w.earned || r.CurrentStreak != w.streakIn {
			t.Errorf("week %d: carried in %d, used %d, earned %d, streak in %d; want %+v", i+1, r.CarriedIn, r.CarryUsed, r.CarryEarned, r.CurrentStreak, w)
		}
		if r.GoalMet() != w.met || r.SavedByCarryOver() != w.saved {
			t.Errorf("week %d: met %v, saved %v; want %+v", i+1, r.GoalMet(), r.SavedByCarryOver(), w)
		}
		if len(r.CarryExpiries) != r.CarriedIn {
			t.Errorf("week %d: %d use-by dates for %d saved workouts", i+1, len(r.CarryExpiries), r.CarriedIn)
		}
	}

	// Earned in week 1, lasting two weeks: usable to the Sunday of week 3.
	if got, want := results[1].CarryExpiries[0], season.Start.AddDate(0, 0, 20); !got.Equal(want) {
		t.Errorf("use-by = %v, want %v", got, want)
	}
	// The saved week is judged met everywhere the outcome rules are read.
	if !results[1].WonWeek() || results[1].OwesSpin() || results[1].WheelTickets() != 2 {
		t.Errorf("saved week: won %v, owes %v, tickets %d", results[1].WonWeek(), results[1].OwesSpin(), results[1].WheelTickets())
	}
	if !results[1].CarryOptIn || !results[2].CarryOptIn || results[0].CarryOptIn {
		t.Error("opt-ins are not reported on the weeks that have them")
	}
}

func TestSeasonWalkSickLeaveTakesPrecedence(t *testing.T) {
	newControllerTestDB(t)

	goal := makeGoalObject(uuid.New(), uuid.New(), 3, time.Date(2023, 12, 1, 0, 0, 0, 0, time.UTC))
	season := carryOverSeason(goal)

	seedExerciseDayWithExercises(t, goal.User.ID, wednesdayOfWeek(season.Start, 0), 3)
	seedExerciseDayWithExercises(t, goal.User.ID, wednesdayOfWeek(season.Start, 0).AddDate(0, 0, 1), 1) // saves 1
	seedExerciseDayWithExercises(t, goal.User.ID, wednesdayOfWeek(season.Start, 1), 2)
	seedCarryOverUse(t, goal.ID, wednesdayOfWeek(season.Start, 1))
	seedSickleave(t, goal.ID, wednesdayOfWeek(season.Start, 1), true)

	weeks, err := RetrieveWeekResultsFromSeasonWithinTimeframe(season.Start, season.Start.AddDate(0, 0, 13), season)
	if err != nil {
		t.Fatal(err)
	}
	results := resultsByWeek(t, weeks, goal.ID)
	sick := results[1]
	if !sick.SickLeave || sick.CarryUsed != 0 || sick.SavedByCarryOver() {
		t.Errorf("sick-leave week: sick %v, used %d", sick.SickLeave, sick.CarryUsed)
	}
	if sick.CarriedIn != 1 {
		t.Errorf("sick-leave week: carried in %d, want 1 (still held)", sick.CarriedIn)
	}
}

func TestSeasonWalkKeepsEachSeasonsSavedWorkoutsApart(t *testing.T) {
	newControllerTestDB(t)

	userID := uuid.New()
	joined := time.Date(2023, 12, 1, 0, 0, 0, 0, time.UTC)
	easy := makeGoalObject(userID, uuid.New(), 2, joined) // 3 workouts: saves 1
	hard := makeGoalObject(userID, uuid.New(), 3, joined) // 3 workouts: saves nothing
	off := makeGoalObject(userID, uuid.New(), 2, joined)  // feature off: saves nothing

	easySeason := carryOverSeason(easy)
	hardSeason := carryOverSeason(hard)
	offSeason := carryOverSeason(off)
	offSeason.CarryOverMax, offSeason.CarryOverWeeks = 0, 0

	seedExerciseDayWithExercises(t, userID, wednesdayOfWeek(easySeason.Start, 0), 3)
	seedExerciseDayWithExercises(t, userID, wednesdayOfWeek(easySeason.Start, 1), 1)
	for _, goal := range []models.GoalObject{easy, hard, off} {
		seedCarryOverUse(t, goal.ID, wednesdayOfWeek(easySeason.Start, 1))
	}

	end := easySeason.Start.AddDate(0, 0, 13)
	cases := []struct {
		name   string
		season models.SeasonObject
		goal   models.GoalObject
		saved  int
		met    bool
	}{
		{"goal of two", easySeason, easy, 1, true},
		{"goal of three", hardSeason, hard, 0, false},
		{"feature off", offSeason, off, 0, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			weeks, err := RetrieveWeekResultsFromSeasonWithinTimeframe(tc.season.Start, end, tc.season)
			if err != nil {
				t.Fatal(err)
			}
			week2 := resultsByWeek(t, weeks, tc.goal.ID)[1]
			if week2.CarriedIn != tc.saved || week2.GoalMet() != tc.met {
				t.Errorf("week 2: carried in %d, met %v; want %d, %v", week2.CarriedIn, week2.GoalMet(), tc.saved, tc.met)
			}
			if tc.season.CarryOverMax == 0 && (week2.CarryOptIn || len(week2.CarryExpiries) != 0) {
				t.Error("a season without saved workouts reports them")
			}
		})
	}
}

// TestSavedWeekWinsTheWheelAndOwesNothing runs the Monday processing on a week one member
// made up with a saved workout while a rival missed: the saver is a winner on the wheel, the
// rival owes the spin, and the saver gets no debt.
func TestSavedWeekWinsTheWheelAndOwesNothing(t *testing.T) {
	h := newAPIHarness(t)
	withControllerSMTP(t)
	saver, saverToken := h.user("saver@carry.test", false)
	rival, _ := h.user("rival@carry.test", false)

	season := seedOngoingSeason(t, "Saved workouts season", 3, 6)
	database.Instance.Model(&models.Season{}).Where("id = ?", season.ID).Updates(map[string]any{"carry_over_max": 2, "carry_over_weeks": 2})
	season.CarryOverMax, season.CarryOverWeeks = 2, 2
	saverGoal := seedGoal(t, saver.ID, season, 3)
	seedGoal(t, rival.ID, season, 3)

	seedExerciseDayWithExercises(t, saver.ID, wednesdayOfWeeksAgo(2), 3)
	seedExerciseDayWithExercises(t, saver.ID, wednesdayOfWeeksAgo(2).AddDate(0, 0, 1), 1) // saves 1
	seedExerciseDayWithExercises(t, rival.ID, wednesdayOfWeeksAgo(2), 3)
	seedExerciseDayWithExercises(t, saver.ID, wednesdayOfWeeksAgo(1), 2) // 2/3, made up
	seedCarryOverUse(t, saverGoal.ID, wednesdayOfWeeksAgo(1))
	seedExerciseDayWithExercises(t, rival.ID, wednesdayOfWeeksAgo(1), 1) // missed

	week, err := GenerateDebtForWeek(wednesdayOfWeeksAgo(1), season, nil)
	if err != nil {
		t.Fatal(err)
	}
	saverResult := userWeekResult(week, saverGoal.ID)
	if saverResult == nil || !saverResult.SavedByCarryOver() || !saverResult.WonWeek() || saverResult.WheelTickets() != 2 {
		t.Fatalf("saver's week = %+v, want won with a saved workout and two tickets", saverResult)
	}

	debts := h.ok("GET", "/api/auth/debts", saverToken, nil)
	body := mustJSON(t, debts)
	if !strings.Contains(body, rival.ID.String()) {
		t.Errorf("the rival who missed owes no spin: %s", body)
	}
	if _, found, _ := database.GetDebtForWeekForUserInSeasonID(wednesdayOfWeeksAgo(1), saver.ID, season.ID); found {
		t.Error("the saver got a debt for a week saved workouts made up")
	}
}

func TestDayHeaderAgreesWithTheLeaderboardOnSavedWorkouts(t *testing.T) {
	h := newAPIHarness(t)
	user, token := h.user("day@carry.test", false)

	season := seedOngoingSeason(t, "Saved workouts day season", 2, 6)
	database.Instance.Model(&models.Season{}).Where("id = ?", season.ID).Updates(map[string]any{"carry_over_max": 2, "carry_over_weeks": 1})
	goal := seedGoal(t, user.ID, season, 2)

	seedExerciseDayWithExercises(t, user.ID, wednesdayOfWeeksAgo(1), 3) // saves 1
	seedExerciseDayWithExercises(t, user.ID, wednesdayOfWeeksAgo(0), 1) // 1/2 this week
	seedCarryOverUse(t, goal.ID, wednesdayOfWeeksAgo(0))

	day, err := database.GetExerciseDayByDateAndUserID(user.ID, wednesdayOfWeeksAgo(0))
	if err != nil || day == nil {
		t.Fatalf("seeded day not found: %v", err)
	}
	header := h.ok("GET", "/api/auth/exercise-days/"+day.ID.String()+"/week", token, nil)
	seasons := field(t, field(t, header, "week"), "seasons").([]any)
	if len(seasons) != 1 {
		t.Fatalf("day header seasons = %v", seasons)
	}
	entry := seasons[0]
	if field(t, entry, "goal_met") != true || field(t, entry, "carry_used") != float64(1) || field(t, entry, "workouts") != float64(1) {
		t.Errorf("day header = %v, want 1 workout made up with 1 saved workout", entry)
	}

	board := h.ok("GET", "/api/auth/seasons/"+season.ID.String()+"/weeks", token, nil)
	thisWeek := field(t, board, "leaderboard").([]any)[0]
	result := field(t, thisWeek, "users").([]any)[0]
	if field(t, result, "carry_used") != float64(1) || field(t, result, "carried_in") != float64(1) || field(t, result, "carry_opt_in") != true {
		t.Errorf("leaderboard this week = %v, want 1 saved workout held and used", result)
	}
}

// --- The API -----------------------------------------------------------------

func TestUseSavedWorkoutsFlow(t *testing.T) {
	h := newAPIHarness(t)
	user, token := h.user("optin@carry.test", false)
	_, strangerToken := h.user("stranger@carry.test", false)

	season := seedOngoingSeason(t, "Opt-in season", 2, 6)
	database.Instance.Model(&models.Season{}).Where("id = ?", season.ID).Updates(map[string]any{"carry_over_max": 2, "carry_over_weeks": 2})
	path := "/api/auth/carryover/" + season.ID.String()

	// Joined, but nothing saved yet.
	goal := seedGoal(t, user.ID, season, 2)
	h.expect(http.StatusBadRequest, "POST", path, token, nil)

	// A week beyond the goal saves one; now it can be turned on, once.
	seedExerciseDayWithExercises(t, user.ID, wednesdayOfWeeksAgo(1), 3)
	h.ok("POST", path, token, nil)
	h.expect(http.StatusBadRequest, "POST", path, token, nil)

	board := h.ok("GET", "/api/auth/seasons/"+season.ID.String()+"/weeks", token, nil)
	result := field(t, field(t, board, "leaderboard").([]any)[0], "users").([]any)[0]
	if field(t, result, "carry_opt_in") != true {
		t.Errorf("this week's result doesn't show the opt-in: %v", result)
	}

	// Undo while the week runs, then nothing left to undo.
	h.ok("DELETE", path, token, nil)
	h.expect(http.StatusBadRequest, "DELETE", path, token, nil)
	if use, _ := database.GetCarryOverUseForGoalWithinWeek(time.Now(), goal.ID); use != nil {
		t.Error("the opt-in is still on after undoing it")
	}

	// Someone outside the season, a bad ID and an unknown season.
	h.expect(http.StatusBadRequest, "POST", path, strangerToken, nil)
	h.expect(http.StatusBadRequest, "POST", "/api/auth/carryover/not-a-uuid", token, nil)
	h.expect(http.StatusBadRequest, "POST", "/api/auth/carryover/"+uuid.NewString(), token, nil)

	// Sick leave this week rules it out.
	seedSickleave(t, goal.ID, time.Now(), false)
	h.ok("POST", "/api/auth/sickleave/"+season.ID.String(), token, nil)
	h.expect(http.StatusBadRequest, "POST", path, token, nil)
}

func TestUseSavedWorkoutsOnlyForTheRunningWeek(t *testing.T) {
	h := newAPIHarness(t)
	user, token := h.user("past@carry.test", false)

	// An ended season: its weeks are settled.
	ended := seedOngoingSeason(t, "Ended saved season", 8, 4)
	database.Instance.Model(&models.Season{}).Where("id = ?", ended.ID).Updates(map[string]any{"carry_over_max": 2, "carry_over_weeks": 2})
	seedGoal(t, user.ID, ended, 1)
	h.expect(http.StatusBadRequest, "POST", "/api/auth/carryover/"+ended.ID.String(), token, nil)
	h.expect(http.StatusBadRequest, "DELETE", "/api/auth/carryover/"+ended.ID.String(), token, nil)

	// A running season: last week's opt-in can't be undone from this week.
	running := seedOngoingSeason(t, "Running saved season", 3, 6)
	database.Instance.Model(&models.Season{}).Where("id = ?", running.ID).Updates(map[string]any{"carry_over_max": 2, "carry_over_weeks": 2})
	goal := seedGoal(t, user.ID, running, 1)
	seedCarryOverUse(t, goal.ID, wednesdayOfWeeksAgo(1))
	h.expect(http.StatusBadRequest, "DELETE", "/api/auth/carryover/"+running.ID.String(), token, nil)
	if use, _ := database.GetCarryOverUseForGoalWithinWeek(wednesdayOfWeeksAgo(1), goal.ID); use == nil {
		t.Error("undoing this week touched last week's opt-in")
	}

	// A season without saved workouts refuses it outright.
	plain := seedOngoingSeason(t, "Plain season", 1, 4)
	seedGoal(t, user.ID, plain, 1)
	h.expect(http.StatusBadRequest, "POST", "/api/auth/carryover/"+plain.ID.String(), token, nil)
}

func TestSeasonCreationValidatesSavedWorkouts(t *testing.T) {
	h := newAPIHarness(t)
	_, adminToken := h.user("admin@carry.test", true)
	prizeID := seedPrize(t, "Saved prize")
	start := nextMonday(2)

	request := func(name string, max int, weeks int) models.SeasonCreationRequest {
		return models.SeasonCreationRequest{
			Name: name, Description: "d", Start: start, End: start.AddDate(0, 0, 27),
			Prize: prizeID, CarryOverMax: max, CarryOverWeeks: weeks,
		}
	}

	invalid := map[string]models.SeasonCreationRequest{
		"negative count":        request("Negative saved", -1, 1),
		"count above the bound": request("Too many saved", maxCarryOverMax+1, 1),
		"on without a lifetime": request("No lifetime", 2, 0),
		"lifetime too long":     request("Too long saved", 2, maxCarryOverWeeks+1),
	}
	for name, body := range invalid {
		t.Run(name, func(t *testing.T) {
			h.expect(http.StatusBadRequest, "POST", "/api/admin/seasons", adminToken, body)
		})
	}

	h.expect(http.StatusCreated, "POST", "/api/admin/seasons", adminToken, request("Saved season on", 2, 1))
	h.expect(http.StatusCreated, "POST", "/api/admin/seasons", adminToken, request("Saved season off", 0, 5))

	var on, off models.Season
	database.Instance.Where("name = ?", "Saved season on").First(&on)
	database.Instance.Where("name = ?", "Saved season off").First(&off)
	if on.CarryOverMax != 2 || on.CarryOverWeeks != 1 {
		t.Errorf("season on = %d / %d, want 2 / 1", on.CarryOverMax, on.CarryOverWeeks)
	}
	if off.CarryOverMax != 0 || off.CarryOverWeeks != 0 {
		t.Errorf("season off = %d / %d, want 0 / 0 (a lifetime means nothing while off)", off.CarryOverMax, off.CarryOverWeeks)
	}
}

// --- What the AI greeting, MCP and the Sunday reminder say -------------------

func TestOllamaPayloadSavedWorkouts(t *testing.T) {
	on := models.SeasonObject{CarryOverMax: 2, CarryOverWeeks: 1}
	useBy := time.Date(2026, 11, 8, 0, 0, 0, 0, time.UTC)

	// Two of three done, one saved workout makes up the rest: met, nothing remaining.
	madeUp := models.UserWeekResults{WeekCompletion: 2.0 / 3, CarriedIn: 1, CarryExpiries: []time.Time{useBy}, CarryOptIn: true, CarryUsed: 1, Competing: true}
	payload := ollamaSeasonPayloadForWeek("Autumn", 3, 0, 2, madeUp)
	addSavedWorkoutsToPayload(&payload, on, madeUp)
	if payload.WeeklyGoalMet == nil || !*payload.WeeklyGoalMet || payload.WorkoutsRemaining == nil || *payload.WorkoutsRemaining != 0 {
		t.Errorf("a made-up week reads as unmet: %+v", payload)
	}
	body, _ := json.Marshal(payload)
	for _, want := range []string{`"saved_workouts":1`, `"saved_workouts_use_by":"Sunday 8 November"`, `"using_saved_workouts_this_week":true`, `"saved_workouts_filling_this_weeks_gap":1`, `"saved_workouts_rule":"Up to 2`} {
		if !strings.Contains(string(body), want) {
			t.Errorf("payload lacks %s: %s", want, body)
		}
	}

	// A season without the feature says nothing about it.
	off := ollamaSeasonPayloadForWeek("Spring", 3, 0, 2, models.UserWeekResults{WeekCompletion: 2.0 / 3})
	addSavedWorkoutsToPayload(&off, models.SeasonObject{}, models.UserWeekResults{WeekCompletion: 2.0 / 3})
	if body, _ := json.Marshal(off); strings.Contains(string(body), "saved") {
		t.Errorf("a season without saved workouts mentions them: %s", body)
	}
}

func TestMCPSeasonReportsSavedWorkouts(t *testing.T) {
	newControllerTestDB(t)
	user := createTestUser(t, "mcp@carry.test", "Mia")

	season := seedOngoingSeason(t, "MCP saved season", 3, 6)
	database.Instance.Model(&models.Season{}).Where("id = ?", season.ID).Updates(map[string]any{"carry_over_max": 2, "carry_over_weeks": 2})
	goal := seedGoal(t, user.ID, season, 2)

	seedExerciseDayWithExercises(t, user.ID, wednesdayOfWeeksAgo(2), 3) // saves 1
	seedExerciseDayWithExercises(t, user.ID, wednesdayOfWeeksAgo(1), 1) // made up with it
	seedCarryOverUse(t, goal.ID, wednesdayOfWeeksAgo(1))

	got, err := assembleSeason(user.ID, season.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.SavedWorkoutsRule == "" || got.WeeksMadeUpBySaved == nil || *got.WeeksMadeUpBySaved != 1 {
		t.Errorf("season = %+v, want the rule and one week made up", got)
	}
	if got.SavedWorkouts == nil || *got.SavedWorkouts != 0 {
		t.Errorf("saved workouts now = %v, want 0 (the one saved was used)", got.SavedWorkouts)
	}

	plain := seedOngoingSeason(t, "MCP plain season", 1, 4)
	seedGoal(t, user.ID, plain, 2)
	if got, _ := assembleSeason(user.ID, plain.ID); got.SavedWorkoutsRule != "" || got.SavedWorkouts != nil {
		t.Errorf("a season without saved workouts reports them: %+v", got)
	}
}

func TestSundayReminderMentionsSavedWorkouts(t *testing.T) {
	h := newAPIHarness(t)
	server := withControllerSMTP(t)
	user, token := h.user("reminder@carry.test", false)

	season := seedOngoingSeason(t, "Reminder saved season", 2, 6)
	database.Instance.Model(&models.Season{}).Where("id = ?", season.ID).Updates(map[string]any{"carry_over_max": 2, "carry_over_weeks": 2})
	seedGoal(t, user.ID, season, 1)
	seedExerciseDayWithExercises(t, user.ID, wednesdayOfWeeksAgo(1), 3) // saves 2; nothing this week

	h.ok("PATCH", "/api/auth/users/"+user.ID.String(), token, models.UserPartialUpdateRequest{SundayAlert: boolPtr(true)})
	SendSundayReminders()

	messages := server.Messages()
	if len(messages) != 1 {
		t.Fatalf("reminders = %d, want 1", len(messages))
	}
	data := strings.ReplaceAll(messages[0].Data, "=\r\n", "")
	if !strings.Contains(data, "Use saved workouts") || !strings.Contains(data, season.ID.String()) {
		t.Errorf("reminder doesn't point to the saved workouts:\n%s", data)
	}
}
