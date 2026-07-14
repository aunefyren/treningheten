package controllers

import (
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
