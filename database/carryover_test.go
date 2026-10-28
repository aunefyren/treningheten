package database

import (
	"testing"
	"time"

	"github.com/aunefyren/treningheten/models"

	"github.com/google/uuid"
)

func TestCarryOverUsesByWeek(t *testing.T) {
	newTestDB(t)

	goalID := uuid.New()
	monday := time.Date(2026, 10, 26, 0, 0, 0, 0, time.Local) // a Monday, stored at midnight
	nextMonday := monday.AddDate(0, 0, 7)
	for _, date := range []time.Time{monday, nextMonday} {
		use := models.CarryOverUse{Enabled: true, GoalID: goalID, Date: date}
		use.ID = uuid.New()
		if err := CreateCarryOverUseInDB(use); err != nil {
			t.Fatalf("CreateCarryOverUseInDB returned error: %v", err)
		}
	}

	// The week is found from any of its days, Monday's midnight included.
	for _, day := range []time.Time{monday, monday.AddDate(0, 0, 6).Add(23 * time.Hour)} {
		use, err := GetCarryOverUseForGoalWithinWeek(day, goalID)
		if err != nil || use == nil || !use.Date.Equal(monday) {
			t.Errorf("GetCarryOverUseForGoalWithinWeek(%v) = %v, %v; want the Monday row", day, use, err)
		}
	}
	if use, err := GetCarryOverUseForGoalWithinWeek(monday, uuid.New()); err != nil || use != nil {
		t.Errorf("another goal's week = %v, %v; want nil", use, err)
	}

	uses, err := GetCarryOverUsesForGoalIDsBetweenDates([]uuid.UUID{goalID}, monday, nextMonday.AddDate(0, 0, 6))
	if err != nil || len(uses) != 2 {
		t.Fatalf("GetCarryOverUsesForGoalIDsBetweenDates = %d rows, %v; want 2", len(uses), err)
	}

	// Undoing one week leaves the other.
	disabled, err := DisableCarryOverUsesForGoalWithinWeek(monday.AddDate(0, 0, 3), goalID)
	if err != nil || disabled != 1 {
		t.Fatalf("DisableCarryOverUsesForGoalWithinWeek = %d, %v; want 1", disabled, err)
	}
	if use, _ := GetCarryOverUseForGoalWithinWeek(monday, goalID); use != nil {
		t.Error("the undone week is still on")
	}
	if use, _ := GetCarryOverUseForGoalWithinWeek(nextMonday, goalID); use == nil {
		t.Error("undoing one week turned off the next")
	}
	if disabled, _ := DisableCarryOverUsesForGoalWithinWeek(monday, goalID); disabled != 0 {
		t.Errorf("undoing twice disabled %d rows", disabled)
	}
}
