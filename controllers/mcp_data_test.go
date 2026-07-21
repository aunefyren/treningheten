package controllers

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/aunefyren/treningheten/models"

	"github.com/google/uuid"
)

// TestCopySecondsPtr guards the duration-as-seconds contract on the MCP mapping: the
// *int64 seconds fields (Operation.Duration, OperationSet.Time/MovingTime) are copied
// straight through — not reinterpreted as nanosecond durations — and nil stays nil.
func TestCopySecondsPtr(t *testing.T) {
	if got := copySecondsPtr(nil); got != nil {
		t.Fatalf("nil input should return nil, got %v", *got)
	}

	in := int64(3600)
	got := copySecondsPtr(&in)
	if got == nil || *got != 3600 {
		t.Fatalf("expected 3600 seconds, got %v", got)
	}
	if got == &in {
		t.Fatalf("expected a copy, got the same pointer (aliasing)")
	}
}

// TestDurationFieldsSerializeAsSeconds locks the JSON wire format after the retype from
// *time.Duration to *int64: the fields marshal as a plain seconds integer (e.g. 3600),
// so the frontend's secondsToDurationString keeps working and no consumer sees the
// nanosecond value a real time.Duration would have emitted.
func TestDurationFieldsSerializeAsSeconds(t *testing.T) {
	secs := int64(3600)

	set := models.OperationSet{Time: &secs, MovingTime: &secs}
	blob, err := json.Marshal(set)
	if err != nil {
		t.Fatalf("marshal OperationSet: %v", err)
	}
	if out := string(blob); !strings.Contains(out, `"time":3600`) || !strings.Contains(out, `"moving_time":3600`) {
		t.Fatalf("expected plain seconds in OperationSet JSON, got %s", out)
	}

	exercise := models.Exercise{Duration: &secs}
	blob, err = json.Marshal(exercise)
	if err != nil {
		t.Fatalf("marshal Exercise: %v", err)
	}
	if out := string(blob); !strings.Contains(out, `"duration":3600`) {
		t.Fatalf("expected plain seconds in Exercise JSON, got %s", out)
	}
}

// TestFeedItemToSummary covers the /exercises feed → MCP search mapping: source precedence
// (strava beats hevy beats manual), that units are only carried when the metric is present,
// and that the ids and session grouping are threaded through.
func TestFeedItemToSummary(t *testing.T) {
	opID := uuid.New()
	sessionID := uuid.New()
	when := time.Date(2025, 5, 4, 9, 0, 0, 0, time.UTC)
	hevyID := "hevy-123"

	base := models.ActivityFeedItem{
		OperationID:          opID,
		ExerciseID:           sessionID,
		Date:                 when,
		Time:                 &when,
		ActionName:           "Run",
		ActionType:           "cardio",
		Distance:             21.1,
		DistanceUnit:         "km",
		DurationSeconds:      7110,
		CountsTowardGoal:     true,
		SessionActivityCount: 2,
	}

	t.Run("strava wins even with a hevy id present", func(t *testing.T) {
		item := base
		item.HasStrava = true
		item.HevyWorkoutID = &hevyID
		got := feedItemToSummary(item)
		if got.Source != "strava" {
			t.Errorf("source = %q, want strava", got.Source)
		}
		if !got.HasStreams {
			t.Errorf("has_streams should mirror has_strava")
		}
		if got.ID != opID.String() || got.SessionID != sessionID.String() {
			t.Errorf("ids not threaded: id=%q session=%q", got.ID, got.SessionID)
		}
		if got.DistanceUnit != "km" {
			t.Errorf("distance unit should be carried when distance > 0, got %q", got.DistanceUnit)
		}
	})

	t.Run("hevy when no strava", func(t *testing.T) {
		item := base
		item.HevyWorkoutID = &hevyID
		if got := feedItemToSummary(item); got.Source != "hevy" {
			t.Errorf("source = %q, want hevy", got.Source)
		}
	})

	t.Run("manual, units dropped for zero metrics", func(t *testing.T) {
		item := base
		item.Distance = 0
		item.TopWeight = 0
		item.WeightUnit = "kg"
		got := feedItemToSummary(item)
		if got.Source != "manual" {
			t.Errorf("source = %q, want manual", got.Source)
		}
		if got.DistanceUnit != "" || got.WeightUnit != "" {
			t.Errorf("units should be dropped for zero metrics: dist=%q weight=%q", got.DistanceUnit, got.WeightUnit)
		}
	})
}

// TestOperationObjectToActivityCarriesCountsTowardGoal covers that the rich get_activity path
// surfaces the session's goal-counting flag on the activity (the parity ask alongside search).
func TestOperationObjectToActivityCarriesCountsTowardGoal(t *testing.T) {
	op := models.OperationObject{Note: strPtrLocal("Bench Press")}
	op.ID = uuid.New()

	if got := operationObjectToActivity(op, time.Now(), nil, false, true); !got.CountsTowardGoal {
		t.Errorf("counts_toward_goal = false, want true")
	}
	if got := operationObjectToActivity(op, time.Now(), nil, false, false); got.CountsTowardGoal {
		t.Errorf("counts_toward_goal = true, want false")
	}
}

// TestActivityFeedFilterFromSearchArgs covers the list_activities argument validation: defaults,
// the sort/order whitelists, date parsing (both grains), the limit cap and offset floor.
func TestActivityFeedFilterFromSearchArgs(t *testing.T) {
	t.Run("defaults", func(t *testing.T) {
		filter, err := activityFeedFilterFromSearchArgs(mcpListExercisesArgs{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if filter.Sort != "date" || filter.Order != "desc" {
			t.Errorf("defaults wrong: sort=%q order=%q", filter.Sort, filter.Order)
		}
		if filter.Limit != mcpDefaultLimit {
			t.Errorf("limit default = %d, want %d", filter.Limit, mcpDefaultLimit)
		}
	})

	t.Run("action and query map through, limit capped, offset floored", func(t *testing.T) {
		filter, err := activityFeedFilterFromSearchArgs(mcpListExercisesArgs{
			Action: " Run ", Query: " loop ", Limit: 500, Offset: -5, HasDistance: true,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if filter.ActionName != "Run" || filter.Query != "loop" {
			t.Errorf("trimmed fields wrong: action=%q query=%q", filter.ActionName, filter.Query)
		}
		if filter.Limit != 100 {
			t.Errorf("limit should cap at 100, got %d", filter.Limit)
		}
		if filter.Offset != 0 {
			t.Errorf("negative offset should floor to 0, got %d", filter.Offset)
		}
		if !filter.HasDistance {
			t.Errorf("has_distance not carried")
		}
	})

	t.Run("dates accept both grains", func(t *testing.T) {
		filter, err := activityFeedFilterFromSearchArgs(mcpListExercisesArgs{
			From: "2025-01-02", To: "2025-02-03T10:00:00Z",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if filter.Start == nil || filter.Start.Year() != 2025 || filter.Start.Month() != 1 {
			t.Errorf("from date parse wrong: %v", filter.Start)
		}
		if filter.End == nil || filter.End.Hour() != 10 {
			t.Errorf("to date parse wrong: %v", filter.End)
		}
	})

	t.Run("bad inputs are rejected", func(t *testing.T) {
		for _, args := range []mcpListExercisesArgs{
			{Sort: "bogus"},
			{Order: "sideways"},
			{From: "not-a-date"},
			{To: "13/13/2025"},
		} {
			if _, err := activityFeedFilterFromSearchArgs(args); err == nil {
				t.Errorf("expected error for %+v", args)
			}
		}
	})
}
