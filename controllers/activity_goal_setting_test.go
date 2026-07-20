package controllers

import (
	"testing"

	"github.com/aunefyren/treningheten/models"

	"github.com/google/uuid"
)

// TestCountsTowardGoalForActions covers the import default rule: a session counts unless every
// one of its activity types is flagged off (and an empty type list always counts).
func TestCountsTowardGoalForActions(t *testing.T) {
	off := uuid.New()  // flagged "doesn't count"
	on := uuid.New()   // no setting → counts
	off2 := uuid.New() // also flagged off
	offActions := map[uuid.UUID]bool{off: true, off2: true}

	tests := []struct {
		name      string
		actionIDs []uuid.UUID
		want      bool
	}{
		{"no actions counts", nil, true},
		{"single off action does not count", []uuid.UUID{off}, false},
		{"single on action counts", []uuid.UUID{on}, true},
		{"mixed: one on action is enough", []uuid.UUID{off, on}, true},
		{"all off does not count", []uuid.UUID{off, off2}, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := countsTowardGoalForActions(tc.actionIDs, offActions); got != tc.want {
				t.Errorf("countsTowardGoalForActions(%v) = %v, want %v", tc.actionIDs, got, tc.want)
			}
		})
	}
}

// TestMergeActivityGoalSettings covers the settings-screen view: every activity type is listed,
// annotated with the stored preference, defaulting to counting when no row exists.
func TestMergeActivityGoalSettings(t *testing.T) {
	run := models.Action{Name: "Run", Type: "cardio", HasLogo: true}
	run.ID = uuid.New()
	walk := models.Action{Name: "Walk", Type: "cardio"}
	walk.ID = uuid.New()
	actions := []models.Action{run, walk}

	// User flagged Walk off; Run has no row → defaults to counting.
	walkSetting := models.UserActivityGoalSetting{ActionID: walk.ID, CountsTowardGoal: false}
	objects := mergeActivityGoalSettings(actions, []models.UserActivityGoalSetting{walkSetting})

	if len(objects) != 2 {
		t.Fatalf("got %d objects, want 2", len(objects))
	}
	byID := map[uuid.UUID]models.ActivityGoalSettingObject{}
	for _, object := range objects {
		byID[object.ActionID] = object
	}
	if !byID[run.ID].CountsTowardGoal {
		t.Errorf("Run should default to counting")
	}
	if byID[walk.ID].CountsTowardGoal {
		t.Errorf("Walk should reflect the stored off setting")
	}
	if byID[run.ID].ActionName != "Run" || !byID[run.ID].ActionHasLogo {
		t.Errorf("Run action fields not carried through: %+v", byID[run.ID])
	}
}
