package controllers

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/aunefyren/treningheten/models"
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
