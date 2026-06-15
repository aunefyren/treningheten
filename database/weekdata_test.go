package database

import (
	"testing"
	"time"

	"github.com/aunefyren/treningheten/models"

	"github.com/google/uuid"
)

// seedDay inserts an exercise day (associations omitted) and returns its ID. The enabled
// column is set via an explicit Update because GORM omits zero-valued bools that carry a
// `default: true` tag on Create, which would otherwise flip false back to true.
func seedDay(t *testing.T, userID uuid.UUID, date time.Time, enabled bool) uuid.UUID {
	t.Helper()
	day := models.ExerciseDay{Date: date, Enabled: enabled, UserID: &userID}
	day.ID = uuid.New()
	if err := Instance.Omit("User", "Goal").Create(&day).Error; err != nil {
		t.Fatalf("failed to seed exercise day: %v", err)
	}
	if err := Instance.Model(&models.ExerciseDay{}).Where("id = ?", day.ID).Update("enabled", enabled).Error; err != nil {
		t.Fatalf("failed to set exercise day enabled: %v", err)
	}
	return day.ID
}

// seedExercise inserts a single exercise under a day (association omitted). enabled/is_on
// are set via an explicit Update for the same default-tag reason as seedDay.
func seedExercise(t *testing.T, dayID uuid.UUID, enabled, isOn bool) {
	t.Helper()
	exercise := models.Exercise{Enabled: enabled, IsOn: isOn, ExerciseDayID: dayID}
	exercise.ID = uuid.New()
	if err := Instance.Omit("ExerciseDay").Create(&exercise).Error; err != nil {
		t.Fatalf("failed to seed exercise: %v", err)
	}
	if err := Instance.Model(&models.Exercise{}).Where("id = ?", exercise.ID).
		Updates(map[string]interface{}{"enabled": enabled, "is_on": isOn}).Error; err != nil {
		t.Fatalf("failed to set exercise flags: %v", err)
	}
}

func TestGetValidExercisesForUserIDsBetweenDates(t *testing.T) {
	newTestDB(t)

	userA := makeTestUser(t, "a@example.com", nil)
	userB := makeTestUser(t, "b@example.com", nil)

	rangeStart := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	rangeEnd := time.Date(2024, 1, 7, 0, 0, 0, 0, time.UTC)
	inRange := time.Date(2024, 1, 3, 0, 0, 0, 0, time.UTC)

	// User A, enabled day in range: 2 valid + 1 off + 1 disabled → 2 valid.
	dayA := seedDay(t, userA.ID, inRange, true)
	seedExercise(t, dayA, true, true)
	seedExercise(t, dayA, true, true)
	seedExercise(t, dayA, true, false) // off
	seedExercise(t, dayA, false, true) // disabled

	// User A, disabled day in range → excluded.
	dayADisabled := seedDay(t, userA.ID, inRange, false)
	seedExercise(t, dayADisabled, true, true)

	// User A, enabled day out of range → excluded.
	dayAOut := seedDay(t, userA.ID, time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC), true)
	seedExercise(t, dayAOut, true, true)

	// User B, enabled day in range: 1 valid.
	dayB := seedDay(t, userB.ID, inRange, true)
	seedExercise(t, dayB, true, true)

	onlyA, err := GetValidExercisesForUserIDsBetweenDates([]uuid.UUID{userA.ID}, rangeStart, rangeEnd)
	if err != nil {
		t.Fatalf("GetValidExercisesForUserIDsBetweenDates returned error: %v", err)
	}
	if len(onlyA) != 2 {
		t.Errorf("user A in range: got %d valid exercises, want 2", len(onlyA))
	}

	both, err := GetValidExercisesForUserIDsBetweenDates([]uuid.UUID{userA.ID, userB.ID}, rangeStart, rangeEnd)
	if err != nil {
		t.Fatalf("GetValidExercisesForUserIDsBetweenDates returned error: %v", err)
	}
	if len(both) != 3 {
		t.Errorf("users A+B in range: got %d valid exercises, want 3", len(both))
	}

	// Each returned exercise should carry its preloaded day (used for bucketing by week).
	for _, exercise := range both {
		if exercise.ExerciseDay.UserID == nil {
			t.Errorf("exercise %s: ExerciseDay not preloaded", exercise.ID)
		}
	}

	none, err := GetValidExercisesForUserIDsBetweenDates(nil, rangeStart, rangeEnd)
	if err != nil {
		t.Fatalf("GetValidExercisesForUserIDsBetweenDates(nil) returned error: %v", err)
	}
	if len(none) != 0 {
		t.Errorf("no user IDs: got %d, want 0", len(none))
	}
}

func TestGetDebtsForUserIDsBetweenDates(t *testing.T) {
	newTestDB(t)

	loserA := uuid.New()
	loserB := uuid.New()

	rangeStart := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	rangeEnd := time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC)

	seedDebt := func(loserID uuid.UUID, date time.Time, enabled bool) {
		debt := models.Debt{Date: date, LoserID: loserID, Enabled: enabled}
		debt.ID = uuid.New()
		if err := Instance.Omit("Season", "Loser", "Winner").Create(&debt).Error; err != nil {
			t.Fatalf("failed to seed debt: %v", err)
		}
		// Enabled carries a `default: true` tag, so force the column explicitly.
		if err := Instance.Model(&models.Debt{}).Where("id = ?", debt.ID).Update("enabled", enabled).Error; err != nil {
			t.Fatalf("failed to set debt enabled: %v", err)
		}
	}

	seedDebt(loserA, time.Date(2024, 1, 3, 0, 0, 0, 0, time.UTC), true)  // in range
	seedDebt(loserA, time.Date(2024, 2, 10, 0, 0, 0, 0, time.UTC), true) // out of range
	seedDebt(loserA, time.Date(2024, 1, 5, 0, 0, 0, 0, time.UTC), false) // disabled
	seedDebt(loserB, time.Date(2024, 1, 4, 0, 0, 0, 0, time.UTC), true)  // other user

	onlyA, err := GetDebtsForUserIDsBetweenDates([]uuid.UUID{loserA}, rangeStart, rangeEnd)
	if err != nil {
		t.Fatalf("GetDebtsForUserIDsBetweenDates returned error: %v", err)
	}
	if len(onlyA) != 1 {
		t.Errorf("loser A in range: got %d debts, want 1", len(onlyA))
	}

	both, err := GetDebtsForUserIDsBetweenDates([]uuid.UUID{loserA, loserB}, rangeStart, rangeEnd)
	if err != nil {
		t.Fatalf("GetDebtsForUserIDsBetweenDates returned error: %v", err)
	}
	if len(both) != 2 {
		t.Errorf("losers A+B in range: got %d debts, want 2", len(both))
	}
}

func TestGetSickleavesForGoalIDsBetweenDates(t *testing.T) {
	newTestDB(t)

	goal1 := uuid.New()
	goal2 := uuid.New()

	rangeStart := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	rangeEnd := time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC)

	seedSL := func(goalID uuid.UUID, date time.Time, enabled, used bool) {
		sickleave := models.Sickleave{GoalID: goalID, Date: date, Enabled: enabled, Used: used}
		sickleave.ID = uuid.New()
		if err := Instance.Omit("Goal").Create(&sickleave).Error; err != nil {
			t.Fatalf("failed to seed sickleave: %v", err)
		}
		// Enabled (default true) and Used (default false) carry default tags; force both.
		if err := Instance.Model(&models.Sickleave{}).Where("id = ?", sickleave.ID).
			Updates(map[string]interface{}{"enabled": enabled, "used": used}).Error; err != nil {
			t.Fatalf("failed to set sickleave flags: %v", err)
		}
	}

	seedSL(goal1, time.Date(2024, 1, 3, 0, 0, 0, 0, time.UTC), true, true)  // in range, used
	seedSL(goal1, time.Date(2024, 2, 10, 0, 0, 0, 0, time.UTC), true, true) // out of range
	seedSL(goal1, time.Date(2024, 1, 5, 0, 0, 0, 0, time.UTC), false, true) // disabled
	seedSL(goal1, time.Time{}, true, false)                                 // unused pool row (zero date)
	seedSL(goal2, time.Date(2024, 1, 4, 0, 0, 0, 0, time.UTC), true, true)  // other goal

	only1, err := GetSickleavesForGoalIDsBetweenDates([]uuid.UUID{goal1}, rangeStart, rangeEnd)
	if err != nil {
		t.Fatalf("GetSickleavesForGoalIDsBetweenDates returned error: %v", err)
	}
	if len(only1) != 1 {
		t.Errorf("goal 1 in range: got %d sick-leave rows, want 1", len(only1))
	}

	both, err := GetSickleavesForGoalIDsBetweenDates([]uuid.UUID{goal1, goal2}, rangeStart, rangeEnd)
	if err != nil {
		t.Fatalf("GetSickleavesForGoalIDsBetweenDates returned error: %v", err)
	}
	if len(both) != 2 {
		t.Errorf("goals 1+2 in range: got %d sick-leave rows, want 2", len(both))
	}
}
