package controllers

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/aunefyren/treningheten/models"
)

// f64 returns a pointer to a float64 literal for building test set data.
func f64(v float64) *float64 { return &v }

// secondsDuration wraps a raw seconds count in a time.Duration, matching how the
// app stores durations as a raw seconds count in an int64 (not nanoseconds).
func secondsDuration(seconds int64) *int64 {
	d := seconds
	return &d
}

// workoutDay builds an ExerciseDayObject with a single enabled, on exercise that
// carries the given operations, for use in latest-workout tests.
func workoutDay(date time.Time, note string, ops []models.OperationObject) models.ExerciseDayObject {
	return models.ExerciseDayObject{
		Date: date,
		Note: note,
		Exercises: []models.ExerciseObject{
			{Enabled: true, IsOn: true, Operations: ops},
		},
	}
}

func TestBuildLatestWorkoutRollsUpRecentDay(t *testing.T) {
	now := time.Date(2026, 6, 17, 12, 0, 0, 0, time.UTC)
	yesterday := now.AddDate(0, 0, -1)

	run := models.OperationObject{
		Action:       &models.Action{Name: "Running"},
		DistanceUnit: "km",
		Duration:     secondsDuration(1800), // 30 minutes
		OperationSets: []models.OperationSetObject{
			{Distance: f64(5.0)},
		},
	}
	// A second activity on the same day, distance in metres to exercise km conversion.
	walk := models.OperationObject{
		Action:       &models.Action{Name: "Walking"},
		DistanceUnit: "m",
		OperationSets: []models.OperationSetObject{
			{Distance: f64(2000)}, // 2 km
		},
	}

	days := []models.ExerciseDayObject{
		workoutDay(yesterday, "Felt strong", []models.OperationObject{run, walk}),
		// An older day that should be ignored once a newer one exists.
		workoutDay(now.AddDate(0, 0, -10), "old", []models.OperationObject{run}),
	}

	latest := buildLatestWorkout(days, now)
	if latest == nil {
		t.Fatal("expected a latest workout block, got nil")
	}
	if latest.DaysAgo != 1 {
		t.Errorf("DaysAgo = %d, want 1", latest.DaysAgo)
	}
	if got, want := latest.TotalDistanceKm, 7.0; got != want {
		t.Errorf("TotalDistanceKm = %v, want %v", got, want)
	}
	if latest.DurationMinutes != 30 {
		t.Errorf("DurationMinutes = %d, want 30", latest.DurationMinutes)
	}
	if len(latest.Activities) != 2 || latest.Activities[0] != "Running" || latest.Activities[1] != "Walking" {
		t.Errorf("Activities = %v, want [Running Walking]", latest.Activities)
	}
	if latest.Note != "Felt strong" {
		t.Errorf("Note = %q, want %q", latest.Note, "Felt strong")
	}
}

func TestBuildLatestWorkoutOmitsStaleDay(t *testing.T) {
	now := time.Date(2026, 6, 17, 12, 0, 0, 0, time.UTC)
	run := models.OperationObject{Action: &models.Action{Name: "Running"}}

	// Most recent workout is 4 days ago, beyond latestWorkoutMaxAgeDays (3).
	days := []models.ExerciseDayObject{
		workoutDay(now.AddDate(0, 0, -4), "", []models.OperationObject{run}),
	}

	if got := buildLatestWorkout(days, now); got != nil {
		t.Errorf("expected nil for stale workout, got %+v", got)
	}
}

func TestBuildLatestWorkoutIgnoresDaysWithoutActiveExercise(t *testing.T) {
	now := time.Date(2026, 6, 17, 12, 0, 0, 0, time.UTC)

	// A recent day whose only exercise is toggled off must not count.
	off := models.ExerciseDayObject{
		Date: now.AddDate(0, 0, -1),
		Exercises: []models.ExerciseObject{
			{Enabled: true, IsOn: false},
		},
	}

	if got := buildLatestWorkout([]models.ExerciseDayObject{off}, now); got != nil {
		t.Errorf("expected nil when no active exercise, got %+v", got)
	}
}

func TestOllamaSeasonPayloadForWeek(t *testing.T) {
	boolOf := func(p *bool) any {
		if p == nil {
			return nil
		}
		return *p
	}
	intOf := func(p *int) any {
		if p == nil {
			return nil
		}
		return *p
	}

	tests := []struct {
		name          string
		workouts      int
		result        models.UserWeekResults
		wantMet       any
		wantRemaining any
		wantEntries   int
		wantSick      bool
	}{
		{
			name:          "competing, behind on the goal",
			workouts:      1,
			result:        models.UserWeekResults{WeekCompletion: 1.0 / 3, CurrentStreak: 4, Competing: true},
			wantMet:       false,
			wantRemaining: 2,
			wantEntries:   5,
		},
		{
			name:          "goal met, remaining floors at zero",
			workouts:      5,
			result:        models.UserWeekResults{WeekCompletion: 1.5, CurrentStreak: 2},
			wantMet:       true,
			wantRemaining: 0,
		},
		{
			// The week is paused: no goal figures and no prize entries for the model to push.
			name:     "sick leave drops goal figures and prize entries",
			workouts: 1,
			result:   models.UserWeekResults{WeekCompletion: 1.0 / 3, CurrentStreak: 4, Competing: true, SickLeave: true},
			wantSick: true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := ollamaSeasonPayloadForWeek("Autumn", 3, 2, tc.workouts, tc.result)
			if got.Name != "Autumn" || got.WeeklyGoalWorkouts != 3 || got.SickLeaveWeeksLeft != 2 {
				t.Fatalf("identity fields not carried: %+v", got)
			}
			if got.CurrentStreakWeeks != tc.result.CurrentStreak || got.Competing != tc.result.Competing {
				t.Fatalf("streak/competing = %d/%v, want %d/%v", got.CurrentStreakWeeks, got.Competing, tc.result.CurrentStreak, tc.result.Competing)
			}
			if got.OnSickLeaveThisWeek != tc.wantSick {
				t.Fatalf("on sick leave = %v, want %v", got.OnSickLeaveThisWeek, tc.wantSick)
			}
			if boolOf(got.WeeklyGoalMet) != tc.wantMet || intOf(got.WorkoutsRemaining) != tc.wantRemaining {
				t.Fatalf("met/remaining = %v/%v, want %v/%v", boolOf(got.WeeklyGoalMet), intOf(got.WorkoutsRemaining), tc.wantMet, tc.wantRemaining)
			}
			if got.PrizeEntriesIfYouMeetGoal != tc.wantEntries {
				t.Fatalf("prize entries = %d, want %d", got.PrizeEntriesIfYouMeetGoal, tc.wantEntries)
			}
		})
	}
}

// TestOllamaSeasonPayloadSickLeaveJSON checks what the model actually reads: a sick-leave
// season has no goal, remaining or prize keys at all, while a normal week keeps a false/zero.
func TestOllamaSeasonPayloadSickLeaveJSON(t *testing.T) {
	sick, _ := json.Marshal(ollamaSeasonPayloadForWeek("Autumn", 3, 0, 0, models.UserWeekResults{SickLeave: true, Competing: true}))
	for _, key := range []string{"weekly_goal_met", "workouts_remaining_to_meet_goal", "prize_entries_to_win_if_a_rival_fails"} {
		if strings.Contains(string(sick), key) {
			t.Fatalf("sick-leave payload should omit %q: %s", key, sick)
		}
	}
	if !strings.Contains(string(sick), `"on_sick_leave_this_week":true`) || !strings.Contains(string(sick), `"sick_leave_weeks_left":0`) {
		t.Fatalf("sick-leave payload missing its flags: %s", sick)
	}

	normal, _ := json.Marshal(ollamaSeasonPayloadForWeek("Autumn", 3, 1, 3, models.UserWeekResults{WeekCompletion: 1}))
	if !strings.Contains(string(normal), `"weekly_goal_met":true`) || !strings.Contains(string(normal), `"workouts_remaining_to_meet_goal":0`) {
		t.Fatalf("normal payload should keep goal figures even at zero: %s", normal)
	}
}
