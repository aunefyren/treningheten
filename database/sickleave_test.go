package database

import (
	"testing"
	"time"

	"github.com/aunefyren/treningheten/models"

	"github.com/google/uuid"
)

func TestCreateAndGetUnusedSickleave(t *testing.T) {
	newTestDB(t)

	goalID := uuid.New()
	sickleave := models.Sickleave{GoalID: goalID, Enabled: true, Used: false}
	sickleave.ID = uuid.New()
	if err := CreateSickleave(sickleave); err != nil {
		t.Fatalf("CreateSickleave returned error: %v", err)
	}

	unused, ok, err := GetUnusedSickleaveForGoalWithinWeek(goalID)
	if err != nil {
		t.Fatalf("GetUnusedSickleaveForGoalWithinWeek returned error: %v", err)
	}
	if !ok || len(unused) != 1 {
		t.Errorf("expected 1 unused sick leave, got ok=%v len=%d", ok, len(unused))
	}

	// A goal with no sick leave → not found.
	_, ok, err = GetUnusedSickleaveForGoalWithinWeek(uuid.New())
	if err != nil {
		t.Fatalf("GetUnusedSickleaveForGoalWithinWeek(none) returned error: %v", err)
	}
	if ok {
		t.Errorf("expected not found for goal without sick leave")
	}
}

func TestSetSickleaveToUsedAndGetUsedWithinWeek(t *testing.T) {
	newTestDB(t)

	goalID := uuid.New()
	sickleave := models.Sickleave{GoalID: goalID, Enabled: true, Used: false}
	sickleave.ID = uuid.New()
	if err := CreateSickleave(sickleave); err != nil {
		t.Fatalf("CreateSickleave returned error: %v", err)
	}

	// Marking it used also stamps its date to today.
	if err := SetSickleaveToUsedByID(sickleave.ID); err != nil {
		t.Fatalf("SetSickleaveToUsedByID returned error: %v", err)
	}

	// It is now used, so it drops out of the unused listing...
	_, ok, err := GetUnusedSickleaveForGoalWithinWeek(goalID)
	if err != nil {
		t.Fatalf("GetUnusedSickleaveForGoalWithinWeek returned error: %v", err)
	}
	if ok {
		t.Errorf("expected no unused sick leave after it was used")
	}

	// ...and shows up as used for this week.
	used, err := GetUsedSickleaveForGoalWithinWeek(time.Now(), goalID)
	if err != nil {
		t.Fatalf("GetUsedSickleaveForGoalWithinWeek returned error: %v", err)
	}
	if used == nil || used.ID != sickleave.ID {
		t.Errorf("expected to find the used sick leave for this week, got %v", used)
	}

	// Setting an unknown sick leave to used affects no rows and errors.
	if err := SetSickleaveToUsedByID(uuid.New()); err == nil {
		t.Errorf("expected error setting unknown sick leave to used")
	}
}
