package controllers

import (
	"math"
	"testing"
	"time"

	"github.com/aunefyren/treningheten/models"

	"github.com/google/uuid"
)

// These are characterization tests: they pin the *current* behaviour of the season
// week-result logic (RetrieveWeekResultsFromSeasonWithinTimeframe / GetWeekResultForGoal)
// so a later batch/cache refactor can be proven equivalent. They are not a statement
// that every behaviour here is ideal — notably CurrentStreak is reported as the streak
// going *into* a completed week (it increments afterwards), which the assertions below
// encode as-is.

func floatEquals(a, b float64) bool {
	return math.Abs(a-b) < 1e-9
}

// makeGoalObject builds the minimal GoalObject the week-result logic reads: the user ID,
// goal ID, weekly interval, and CreatedAt (which drives FullWeekParticipation).
func makeGoalObject(userID, seasonID uuid.UUID, interval int, joinDate time.Time) models.GoalObject {
	goal := models.GoalObject{
		SeasonID:         seasonID,
		ExerciseInterval: interval,
		Competing:        true,
	}
	goal.ID = uuid.New()
	goal.User.ID = userID
	goal.CreatedAt = joinDate
	return goal
}

// wednesdayOfWeek returns the Wednesday of the Nth week (0-based) of a Monday-aligned
// season. Seeding mid-week keeps rows clear of the Monday/Sunday range boundaries.
func wednesdayOfWeek(seasonStart time.Time, weekIndex int) time.Time {
	return seasonStart.AddDate(0, 0, 7*weekIndex+2)
}

func TestRetrieveWeekResultsStreaksAndCompletion(t *testing.T) {
	newControllerTestDB(t)

	// 2024-01-01 is a Monday (ISO 2024-W01), so each 7-day step lands on a Monday and
	// the Monday→Sunday window for each step is a single ISO week.
	seasonStart := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	seasonEnd := time.Date(2024, 1, 28, 0, 0, 0, 0, time.UTC) // Sunday ending the 4th week

	userID := uuid.New()
	seasonID := uuid.New()

	// Joined well before the season → FullWeekParticipation is true for every week.
	goal := makeGoalObject(userID, seasonID, 3, time.Date(2023, 12, 1, 0, 0, 0, 0, time.UTC))

	season := models.SeasonObject{
		Start: seasonStart,
		End:   seasonEnd,
		Goals: []models.GoalObject{goal},
	}

	seedExerciseDayWithExercises(t, userID, wednesdayOfWeek(seasonStart, 0), 3) // week 1: 3/3 complete
	seedExerciseDayWithExercises(t, userID, wednesdayOfWeek(seasonStart, 1), 3) // week 2: complete
	seedExerciseDayWithExercises(t, userID, wednesdayOfWeek(seasonStart, 2), 1) // week 3: 1/3 incomplete
	seedExerciseDayWithExercises(t, userID, wednesdayOfWeek(seasonStart, 3), 3) // week 4: complete

	results, err := RetrieveWeekResultsFromSeasonWithinTimeframe(seasonStart, seasonEnd, season)
	if err != nil {
		t.Fatalf("RetrieveWeekResultsFromSeasonWithinTimeframe returned error: %v", err)
	}
	if len(results) != 4 {
		t.Fatalf("expected 4 weeks, got %d", len(results))
	}

	// Results come back newest-first. Streaks reflect the accumulation across weeks:
	// w1 complete (streak in: 0), w2 complete (1), w3 incomplete (2, then resets),
	// w4 complete (0 again).
	want := []struct {
		weekNumber int
		completion float64
		streak     int
	}{
		{4, 1.0, 0},
		{3, 1.0 / 3.0, 2},
		{2, 1.0, 1},
		{1, 1.0, 0},
	}

	for i, w := range want {
		wr := results[i]
		if wr.WeekNumber != w.weekNumber {
			t.Errorf("results[%d]: week number = %d, want %d", i, wr.WeekNumber, w.weekNumber)
		}
		if len(wr.UserWeekResults) != 1 {
			t.Fatalf("results[%d]: expected 1 user result, got %d", i, len(wr.UserWeekResults))
		}
		u := wr.UserWeekResults[0]
		if !floatEquals(u.WeekCompletion, w.completion) {
			t.Errorf("results[%d] (week %d): completion = %v, want %v", i, w.weekNumber, u.WeekCompletion, w.completion)
		}
		if u.CurrentStreak != w.streak {
			t.Errorf("results[%d] (week %d): current streak = %d, want %d", i, w.weekNumber, u.CurrentStreak, w.streak)
		}
		if !u.FullWeekParticipation {
			t.Errorf("results[%d] (week %d): expected full week participation", i, w.weekNumber)
		}
		if u.SickLeave {
			t.Errorf("results[%d] (week %d): unexpected sick leave", i, w.weekNumber)
		}
	}
}

func TestRetrieveWeekResultsSickLeavePreservesStreak(t *testing.T) {
	newControllerTestDB(t)

	seasonStart := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	seasonEnd := time.Date(2024, 1, 14, 0, 0, 0, 0, time.UTC) // Sunday ending the 2nd week

	userID := uuid.New()
	seasonID := uuid.New()
	goal := makeGoalObject(userID, seasonID, 3, time.Date(2023, 12, 1, 0, 0, 0, 0, time.UTC))

	season := models.SeasonObject{
		Start: seasonStart,
		End:   seasonEnd,
		Goals: []models.GoalObject{goal},
	}

	seedExerciseDayWithExercises(t, userID, wednesdayOfWeek(seasonStart, 0), 3) // week 1: complete → streak builds
	// Week 2: no exercises, but the goal used sick leave that week.
	seedSickleave(t, goal.ID, wednesdayOfWeek(seasonStart, 1), true)

	results, err := RetrieveWeekResultsFromSeasonWithinTimeframe(seasonStart, seasonEnd, season)
	if err != nil {
		t.Fatalf("RetrieveWeekResultsFromSeasonWithinTimeframe returned error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 weeks, got %d", len(results))
	}

	// Newest first: results[0] = week 2 (sick leave), results[1] = week 1 (complete).
	week2 := results[0].UserWeekResults[0]
	if !week2.SickLeave {
		t.Errorf("week 2: expected SickLeave true")
	}
	if week2.CurrentStreak != 1 {
		t.Errorf("week 2: streak = %d, want 1 (preserved through sick leave)", week2.CurrentStreak)
	}

	week1 := results[1].UserWeekResults[0]
	if !floatEquals(week1.WeekCompletion, 1.0) {
		t.Errorf("week 1: completion = %v, want 1.0", week1.WeekCompletion)
	}
	if week1.SickLeave {
		t.Errorf("week 1: unexpected sick leave")
	}
}

func TestRetrieveWeekResultsSeasonStartsMidWeek(t *testing.T) {
	newControllerTestDB(t)

	// Season starts on a Wednesday. The first week's exercise window still runs from that
	// week's Monday, so an exercise logged *before* the start date but within week 1 must
	// count. This pins the batched fetch window back to the Monday of the season's first week.
	seasonStart := time.Date(2024, 1, 3, 0, 0, 0, 0, time.UTC) // Wednesday, ISO 2024-W01
	seasonEnd := time.Date(2024, 1, 14, 0, 0, 0, 0, time.UTC)  // Sunday ending the 2nd week

	userID := uuid.New()
	seasonID := uuid.New()
	goal := makeGoalObject(userID, seasonID, 1, time.Date(2023, 12, 1, 0, 0, 0, 0, time.UTC))

	season := models.SeasonObject{
		Start: seasonStart,
		End:   seasonEnd,
		Goals: []models.GoalObject{goal},
	}

	// Tuesday 2024-01-02 is in week 1's window but before the season's start date (Wed 01-03).
	seedExerciseDayWithExercises(t, userID, time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC), 1)

	results, err := RetrieveWeekResultsFromSeasonWithinTimeframe(seasonStart, seasonEnd, season)
	if err != nil {
		t.Fatalf("RetrieveWeekResultsFromSeasonWithinTimeframe returned error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 weeks, got %d", len(results))
	}

	// Newest first: results[0] = week 2 (empty), results[1] = week 1 (the Monday exercise).
	week1 := results[1]
	if week1.WeekNumber != 1 {
		t.Errorf("week 1: week number = %d, want 1", week1.WeekNumber)
	}
	if !floatEquals(week1.UserWeekResults[0].WeekCompletion, 1.0) {
		t.Errorf("week 1: completion = %v, want 1.0 (Monday-before-start exercise should count)", week1.UserWeekResults[0].WeekCompletion)
	}
}

func TestRetrieveWeekResultsFullWeekParticipation(t *testing.T) {
	newControllerTestDB(t)

	seasonStart := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	seasonEnd := time.Date(2024, 1, 21, 0, 0, 0, 0, time.UTC) // Sunday ending the 3rd week

	userID := uuid.New()
	seasonID := uuid.New()

	// Joined in week 2 (Monday 2024-01-08): weeks 1 and 2 are not full participation,
	// week 3 onward is.
	goal := makeGoalObject(userID, seasonID, 3, time.Date(2024, 1, 8, 0, 0, 0, 0, time.UTC))

	season := models.SeasonObject{
		Start: seasonStart,
		End:   seasonEnd,
		Goals: []models.GoalObject{goal},
	}

	results, err := RetrieveWeekResultsFromSeasonWithinTimeframe(seasonStart, seasonEnd, season)
	if err != nil {
		t.Fatalf("RetrieveWeekResultsFromSeasonWithinTimeframe returned error: %v", err)
	}
	if len(results) != 3 {
		t.Fatalf("expected 3 weeks, got %d", len(results))
	}

	// Newest first: week 3, week 2, week 1.
	want := []struct {
		weekNumber int
		fullWeek   bool
	}{
		{3, true},
		{2, false},
		{1, false},
	}

	for i, w := range want {
		wr := results[i]
		if wr.WeekNumber != w.weekNumber {
			t.Errorf("results[%d]: week number = %d, want %d", i, wr.WeekNumber, w.weekNumber)
		}
		if got := wr.UserWeekResults[0].FullWeekParticipation; got != w.fullWeek {
			t.Errorf("results[%d] (week %d): full week participation = %v, want %v", i, w.weekNumber, got, w.fullWeek)
		}
	}
}
