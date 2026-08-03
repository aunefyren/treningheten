package controllers

import (
	"testing"
	"time"

	"github.com/aunefyren/treningheten/database"
	"github.com/aunefyren/treningheten/models"

	"github.com/google/uuid"
)

// seedSessionWithOperations inserts a session under a new day for the user, plus one
// operation per entry in operationDurations (nil = no duration on that operation) and the
// given set times under the first operation. Durations are a raw seconds count throughout,
// per the repo convention.
func seedSessionWithOperations(t *testing.T, userID uuid.UUID, sessionDuration *int64, operationDurations []*int64, setTimes []int64) models.Exercise {
	t.Helper()

	now := time.Now()
	day := models.ExerciseDay{Date: now, Enabled: true, UserID: &userID}
	day.ID = uuid.New()
	if err := database.Instance.Omit("User", "Goal").Create(&day).Error; err != nil {
		t.Fatalf("failed to seed exercise day: %v", err)
	}

	start := now
	exercise := models.Exercise{ExerciseDayID: day.ID, Enabled: true, IsOn: true, Time: &start, Duration: sessionDuration}
	exercise.ID = uuid.New()
	if err := database.Instance.Omit("ExerciseDay").Create(&exercise).Error; err != nil {
		t.Fatalf("failed to seed exercise: %v", err)
	}

	for i, duration := range operationDurations {
		operation := models.Operation{ExerciseID: exercise.ID, Enabled: true, Duration: duration}
		operation.ID = uuid.New()
		if err := database.Instance.Omit("Exercise", "Action", "Gear").Create(&operation).Error; err != nil {
			t.Fatalf("failed to seed operation: %v", err)
		}

		if i > 0 {
			continue
		}
		for _, setTime := range setTimes {
			seconds := setTime
			set := models.OperationSet{OperationID: operation.ID, Enabled: true, Time: &seconds}
			set.ID = uuid.New()
			if err := database.Instance.Omit("Operation").Create(&set).Error; err != nil {
				t.Fatalf("failed to seed operation set: %v", err)
			}
		}
	}

	return exercise
}

func secondsPtr(seconds int64) *int64 { return &seconds }

// TestSessionFallbackSeconds covers the duration cascade used to size a soundtrack match
// window when the session itself carries no Duration: operation durations first (Strava
// carries these), else the sum of the logged set times (manual/Hevy), else zero so the
// caller falls back to its default window.
func TestSessionFallbackSeconds(t *testing.T) {
	newControllerTestDB(t)
	user := createTestUser(t, "fallback@example.com", "Fallback")

	t.Run("operation durations are summed", func(t *testing.T) {
		exercise := seedSessionWithOperations(t, user.ID, nil,
			[]*int64{secondsPtr(1800), secondsPtr(900)}, nil)
		if got := sessionFallbackSeconds(exercise); got != 2700 {
			t.Errorf("got %d seconds, want 2700", got)
		}
	})

	t.Run("operation durations win over set times", func(t *testing.T) {
		exercise := seedSessionWithOperations(t, user.ID, nil,
			[]*int64{secondsPtr(1800)}, []int64{60, 60})
		if got := sessionFallbackSeconds(exercise); got != 1800 {
			t.Errorf("got %d seconds, want the operation duration 1800", got)
		}
	})

	t.Run("set times are the second fallback", func(t *testing.T) {
		exercise := seedSessionWithOperations(t, user.ID, nil,
			[]*int64{nil}, []int64{90, 120, 30})
		if got := sessionFallbackSeconds(exercise); got != 240 {
			t.Errorf("got %d seconds, want the summed set times 240", got)
		}
	})

	t.Run("nothing logged returns zero", func(t *testing.T) {
		exercise := seedSessionWithOperations(t, user.ID, nil, []*int64{nil}, nil)
		if got := sessionFallbackSeconds(exercise); got != 0 {
			t.Errorf("got %d seconds, want 0", got)
		}
	})

	t.Run("a session with no operations returns zero", func(t *testing.T) {
		exercise := seedSessionWithOperations(t, user.ID, nil, nil, nil)
		if got := sessionFallbackSeconds(exercise); got != 0 {
			t.Errorf("got %d seconds, want 0", got)
		}
	})
}

// TestWindowSettledBy covers the settle gate the media reconcile cron uses to decide a
// session's listening window is final: true only once the window closed at least
// mediaSettleWindow ago, and immediately true for a session with no trustworthy window
// (nothing to wait for, so it retires from the hourly scan instead of being rescanned
// forever).
func TestWindowSettledBy(t *testing.T) {
	newControllerTestDB(t)
	user := createTestUser(t, "settle@example.com", "Settle")

	start := time.Date(2026, 8, 3, 10, 0, 0, 0, time.UTC)
	oneHour := int64(3600)

	day := models.ExerciseDay{Date: start, Enabled: true, UserID: &user.ID}
	day.ID = uuid.New()
	if err := database.Instance.Omit("User", "Goal").Create(&day).Error; err != nil {
		t.Fatalf("failed to seed exercise day: %v", err)
	}

	sessionStart := start
	exercise := models.Exercise{ExerciseDayID: day.ID, Enabled: true, IsOn: true, Time: &sessionStart, Duration: &oneHour}
	exercise.ID = uuid.New()
	if err := database.Instance.Omit("ExerciseDay").Create(&exercise).Error; err != nil {
		t.Fatalf("failed to seed exercise: %v", err)
	}

	// The window closes at 11:00, so it settles at 12:00.
	windowEnd := start.Add(time.Hour)
	tests := []struct {
		name string
		now  time.Time
		want bool
	}{
		{"mid-session", start.Add(30 * time.Minute), false},
		{"just after the window closes", windowEnd.Add(time.Minute), false},
		{"one minute short of settling", windowEnd.Add(mediaSettleWindow - time.Minute), false},
		{"exactly at the settle point", windowEnd.Add(mediaSettleWindow), true},
		{"long after", windowEnd.Add(24 * time.Hour), true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := windowSettledBy(exercise, test.now); got != test.want {
				t.Errorf("windowSettledBy at %v = %v, want %v", test.now, got, test.want)
			}
		})
	}

	// A session with no clock time has no window to wait on, so it is settled at once.
	noTime := models.Exercise{ExerciseDayID: day.ID, Enabled: true, IsOn: true}
	noTime.ID = uuid.New()
	if err := database.Instance.Omit("ExerciseDay").Create(&noTime).Error; err != nil {
		t.Fatalf("failed to seed exercise: %v", err)
	}
	if !windowSettledBy(noTime, start) {
		t.Errorf("a session with no time was not treated as settled")
	}

	// Same for a date-only midnight stamp (a manually logged past day).
	midnight := time.Date(2026, 8, 3, 0, 0, 0, 0, time.UTC)
	dateOnly := models.Exercise{ExerciseDayID: day.ID, Enabled: true, IsOn: true, Time: &midnight}
	dateOnly.ID = uuid.New()
	if err := database.Instance.Omit("ExerciseDay").Create(&dateOnly).Error; err != nil {
		t.Fatalf("failed to seed exercise: %v", err)
	}
	if !windowSettledBy(dateOnly, start) {
		t.Errorf("a date-only session was not treated as settled")
	}
}
