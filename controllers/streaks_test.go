package controllers

import (
	"testing"
	"time"

	"github.com/aunefyren/treningheten/models"
)

// TestComputePersonalStreaksCountsTowardGoalGate covers that a session only keeps the
// personal streak alive when it is enabled, on and flagged to count toward the goal —
// the same rule the goal-counting database queries enforce (see exerciseCountsTowardGoal).
func TestComputePersonalStreaksCountsTowardGoalGate(t *testing.T) {
	today := time.Now().UTC()

	day := func(exercises ...models.ExerciseObject) models.ExerciseDayObject {
		return models.ExerciseDayObject{Date: today, Exercises: exercises}
	}
	ex := func(enabled, isOn, counts bool) models.ExerciseObject {
		return models.ExerciseObject{Enabled: enabled, IsOn: isOn, CountsTowardGoal: counts}
	}

	tests := []struct {
		name        string
		days        []models.ExerciseDayObject
		wantDayBest int
	}{
		{"counting session keeps the day active", []models.ExerciseDayObject{day(ex(true, true, true))}, 1},
		{"non-counting session leaves the day inactive", []models.ExerciseDayObject{day(ex(true, true, false))}, 0},
		{"builder-deleted session leaves the day inactive", []models.ExerciseDayObject{day(ex(true, false, true))}, 0},
		{"disabled session leaves the day inactive", []models.ExerciseDayObject{day(ex(false, true, true))}, 0},
		{"mixed: one counting session is enough", []models.ExerciseDayObject{day(ex(true, true, false), ex(true, true, true))}, 1},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := computePersonalStreaks(tc.days)
			if got.DayBest != tc.wantDayBest {
				t.Errorf("DayBest = %d, want %d", got.DayBest, tc.wantDayBest)
			}
		})
	}
}

// TestApplyCountsTowardGoalUpdate covers the nil-means-no-change rule the update handler
// relies on so note/time/is_on edits don't silently zero the goal-counting flag.
func TestApplyCountsTowardGoalUpdate(t *testing.T) {
	tr := true
	fa := false
	tests := []struct {
		name      string
		current   bool
		requested *bool
		want      bool
	}{
		{"nil keeps current true", true, nil, true},
		{"nil keeps current false", false, nil, false},
		{"explicit true sets true", false, &tr, true},
		{"explicit false sets false", true, &fa, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := applyCountsTowardGoalUpdate(tc.current, tc.requested); got != tc.want {
				t.Errorf("applyCountsTowardGoalUpdate(%v, %v) = %v, want %v", tc.current, tc.requested, got, tc.want)
			}
		})
	}
}

// TestCountsTowardGoalChangeBlocked covers the freeze: the goal-counting flag may only be
// changed while the session's week is current, and only when the change is real.
func TestCountsTowardGoalChangeBlocked(t *testing.T) {
	tr := true
	fa := false
	tests := []struct {
		name          string
		current       bool
		requested     *bool
		weekIsCurrent bool
		want          bool
	}{
		{"real change, past week → blocked", true, &fa, false, true},
		{"real change, current week → allowed", true, &fa, true, false},
		{"no-op change, past week → allowed", true, &tr, false, false},
		{"omitted flag, past week → allowed", true, nil, false, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := countsTowardGoalChangeBlocked(tc.current, tc.requested, tc.weekIsCurrent); got != tc.want {
				t.Errorf("countsTowardGoalChangeBlocked(%v, %v, %v) = %v, want %v", tc.current, tc.requested, tc.weekIsCurrent, got, tc.want)
			}
		})
	}
}
