package controllers

import (
	"testing"
	"time"

	"github.com/aunefyren/treningheten/database"
	"github.com/aunefyren/treningheten/models"
)

func TestIsDistanceActivityType(t *testing.T) {
	tests := []struct {
		operationType string
		want          bool
	}{
		{"moving", true},
		{"", true}, // legacy/unknown rows keep their distance views
		{"timing", false},
		{"lifting", false},
	}
	for _, tc := range tests {
		t.Run(tc.operationType, func(t *testing.T) {
			if got := isDistanceActivityType(tc.operationType); got != tc.want {
				t.Fatalf("isDistanceActivityType(%q) = %v, want %v", tc.operationType, got, tc.want)
			}
		})
	}
}

// scopableSummary returns a summary with every block scopeSummaryToActivityType touches populated;
// withGradient controls whether the terrain-based HR-by-gradient view is present.
func scopableSummary(withGradient bool) *models.StreamSummary {
	pct := 4.2
	summary := &models.StreamSummary{
		Heartrate: &models.StreamStat{Avg: 140},
		Cadence:   &models.StreamStat{Avg: 60},
		Speed:     &models.StreamSpeedStat{},
		Segments:  []models.StreamSegment{{Index: 1}},
		HRZones:   []models.StreamHRZone{{Zone: 1}},
		Route:     &models.StreamRoute{},
		Analysis: &models.StreamAnalysis{
			DecouplingPct:     &pct,
			SplitHalves:       &models.StreamSplitHalves{},
			PaceStdDevSeconds: &pct,
			PaceVariationPct:  &pct,
			Breaks:            &models.StreamBreaks{},
		},
	}
	if withGradient {
		summary.Analysis.HRByGradient = []models.StreamHRGradientBucket{{}}
	}
	return summary
}

func TestScopeSummaryToActivityType(t *testing.T) {
	t.Run("nil summary is a no-op", func(t *testing.T) {
		scopeSummaryToActivityType(nil, "timing")
	})

	t.Run("moving keeps everything", func(t *testing.T) {
		summary := scopableSummary(false)
		scopeSummaryToActivityType(summary, "moving")
		if summary.Speed == nil || summary.Segments == nil || summary.Analysis == nil || summary.Analysis.Breaks == nil {
			t.Fatalf("moving summary was scoped: %+v", summary)
		}
	})

	for _, operationType := range []string{"timing", "lifting"} {
		t.Run(operationType+" drops distance views but keeps sensors", func(t *testing.T) {
			summary := scopableSummary(false)
			scopeSummaryToActivityType(summary, operationType)
			if summary.Speed != nil || summary.Segments != nil {
				t.Errorf("speed/segments kept: %+v / %+v", summary.Speed, summary.Segments)
			}
			if summary.Analysis != nil {
				t.Errorf("analysis = %+v, want nil once only pace-based blocks remained", summary.Analysis)
			}
			if summary.Heartrate == nil || summary.Cadence == nil || summary.HRZones == nil || summary.Route == nil {
				t.Errorf("sensor header, zones or route dropped: %+v", summary)
			}
		})
	}

	t.Run("timing keeps HR by gradient", func(t *testing.T) {
		summary := scopableSummary(true)
		scopeSummaryToActivityType(summary, "timing")
		a := summary.Analysis
		if a == nil || len(a.HRByGradient) != 1 {
			t.Fatalf("analysis = %+v, want HR-by-gradient kept", a)
		}
		if a.DecouplingPct != nil || a.SplitHalves != nil || a.PaceStdDevSeconds != nil || a.PaceVariationPct != nil || a.Breaks != nil {
			t.Fatalf("pace-based analysis kept: %+v", a)
		}
	})

	t.Run("timing without analysis", func(t *testing.T) {
		summary := scopableSummary(false)
		summary.Analysis = nil
		scopeSummaryToActivityType(summary, "timing")
		if summary.Speed != nil || summary.Analysis != nil {
			t.Fatalf("summary = %+v, want speed and analysis nil", summary)
		}
	})
}

// TestAttachStreamSummaries_ActivityType covers the /exercises read path: a moving activity
// keeps its speed and splits, a timing activity (padel) carries only the sensor views, and
// the raw stream on the set is left untouched.
func TestAttachStreamSummaries_ActivityType(t *testing.T) {
	withType := func(operationType string) models.OperationObject {
		return models.OperationObject{
			Type:         operationType,
			DistanceUnit: "km",
			OperationSets: []models.OperationSetObject{
				{StravaStreams: &models.StravaStreamsJSON{StravaActivityStreams: *steadyRun(100, 3.0, 150)}},
			},
		}
	}
	day := &models.ExerciseDayObject{
		Date:      time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC),
		Exercises: []models.ExerciseObject{{Operations: []models.OperationObject{withType("moving"), withType("timing")}}},
	}
	attachStreamSummaries(day, models.User{}, nil)

	moving := day.Exercises[0].Operations[0].StreamSummary
	timingOperation := day.Exercises[0].Operations[1]
	timing := timingOperation.StreamSummary
	if moving == nil || moving.Speed == nil || len(moving.Segments) == 0 {
		t.Fatalf("moving summary = %+v, want speed and segments", moving)
	}
	if timing == nil || timing.Speed != nil || timing.Segments != nil {
		t.Fatalf("timing summary = %+v, want no speed or segments", timing)
	}
	if timing.Heartrate == nil || len(timing.HRZones) == 0 {
		t.Fatalf("timing summary = %+v, want heart rate and zones kept", timing)
	}
	if raw := timingOperation.OperationSets[0].StravaStreams.VelocitySmooth; raw == nil || len(raw.Data) != 101 {
		t.Fatal("timing raw velocity stream was altered")
	}
}

// TestAssembleActivityStreamSummary_ActivityType covers the MCP path: the operation type is
// read from the stored row, so a timing activity's summary carries no speed or splits.
func TestAssembleActivityStreamSummary_ActivityType(t *testing.T) {
	h := newAPIHarness(t)
	me, _ := h.user("padel@scope.test", false)
	_, operationID := seedHRActivity(t, me.ID, sportActionID(t, "Run"), 150, true)

	summary, err := assembleActivityStreamSummary(me.ID, operationID)
	if err != nil || summary == nil || summary.Speed == nil {
		t.Fatalf("moving summary = %+v (err %v), want speed", summary, err)
	}

	if err := database.Instance.Model(&models.Operation{}).Where("id = ?", operationID).Update("type", "timing").Error; err != nil {
		t.Fatalf("retype operation: %v", err)
	}
	summary, err = assembleActivityStreamSummary(me.ID, operationID)
	if err != nil || summary == nil {
		t.Fatalf("timing summary = %+v (err %v)", summary, err)
	}
	if summary.Speed != nil || summary.Segments != nil {
		t.Fatalf("timing summary = %+v, want no speed or segments", summary)
	}
	if summary.Heartrate == nil {
		t.Fatal("timing summary dropped heart rate")
	}
}
