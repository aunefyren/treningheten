package controllers

import (
	"testing"
	"time"

	"github.com/aunefyren/treningheten/models"
)

func TestCadenceUnitForSport(t *testing.T) {
	tests := []struct {
		sport string
		want  string
	}{
		{"Run", cadenceUnitSteps},
		{"TrailRun", cadenceUnitSteps},
		{"VirtualRun", cadenceUnitSteps},
		{"Walk", cadenceUnitSteps},
		{"Hike", cadenceUnitSteps},
		{"Ride", cadenceUnitRevolutions},
		{"Rowing", cadenceUnitRevolutions},
		{"", cadenceUnitRevolutions},
	}
	for _, tc := range tests {
		t.Run(tc.sport, func(t *testing.T) {
			if got := cadenceUnitForSport(tc.sport); got != tc.want {
				t.Fatalf("cadenceUnitForSport(%q) = %q, want %q", tc.sport, got, tc.want)
			}
		})
	}
}

func TestCadenceUnitForAction(t *testing.T) {
	if got := cadenceUnitForAction(nil); got != cadenceUnitRevolutions {
		t.Fatalf("nil action = %q, want rpm", got)
	}
	if got := cadenceUnitForAction(&models.Action{StravaName: "Run"}); got != cadenceUnitSteps {
		t.Fatalf("run action = %q, want spm", got)
	}
}

func TestScaleCadence(t *testing.T) {
	tests := []struct {
		name  string
		value *int
		unit  string
		want  *int
	}{
		{"nil stays nil", nil, cadenceUnitSteps, nil},
		{"steps double", intPtr(85), cadenceUnitSteps, intPtr(170)},
		{"revolutions unchanged", intPtr(90), cadenceUnitRevolutions, intPtr(90)},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := scaleCadence(tc.value, tc.unit)
			if (got == nil) != (tc.want == nil) || (got != nil && *got != *tc.want) {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
		})
	}
	// The input is never mutated.
	in := intPtr(80)
	scaleCadence(in, cadenceUnitSteps)
	if *in != 80 {
		t.Fatalf("input mutated to %d", *in)
	}
}

// cadenceRun is a steady 1 km+ run with a constant one-leg cadence, so both the header stat
// and the per-split averages carry cadence.
func cadenceRun(cadence int) *models.StravaActivityStreams {
	streams := steadyRun(400, 3.0, 150)
	data := make([]int, len(streams.Time.Data))
	for i := range data {
		data[i] = cadence
	}
	streams.Cadence = intStream(data)
	return streams
}

func TestApplyCadenceUnit(t *testing.T) {
	t.Run("nil summary and no cadence are untouched", func(t *testing.T) {
		applyCadenceUnit(nil, cadenceUnitSteps)
		s := SummarizeStreams(steadyRun(100, 3.0, 150), "km", hrAnchor{})
		applyCadenceUnit(s, cadenceUnitSteps)
		if s.Cadence != nil || s.CadenceUnit != "" {
			t.Fatalf("no cadence channel: got %+v / %q", s.Cadence, s.CadenceUnit)
		}
	})

	tests := []struct {
		name     string
		unit     string
		wantStat float64
	}{
		{"steps double header and splits", cadenceUnitSteps, 170},
		{"revolutions keep the recorded value", cadenceUnitRevolutions, 85},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s := SummarizeStreams(cadenceRun(85), "km", hrAnchor{})
			applyCadenceUnit(s, tc.unit)
			if s.CadenceUnit != tc.unit {
				t.Fatalf("unit = %q, want %q", s.CadenceUnit, tc.unit)
			}
			if s.Cadence.Avg != tc.wantStat || s.Cadence.Min != tc.wantStat || s.Cadence.Max != tc.wantStat {
				t.Fatalf("cadence = %+v, want all %v", s.Cadence, tc.wantStat)
			}
			if len(s.Segments) < 2 {
				t.Fatalf("expected at least two splits, got %d", len(s.Segments))
			}
			for _, seg := range s.Segments {
				if seg.AvgCadenceRpm == nil || float64(*seg.AvgCadenceRpm) != tc.wantStat {
					t.Fatalf("split %d cadence = %v, want %v", seg.Index, seg.AvgCadenceRpm, tc.wantStat)
				}
			}
		})
	}
}

// TestAttachStreamSummaries_CadenceUnit covers the /exercises read path: a run's cadence is
// presented as steps per minute, a ride's stays rpm.
func TestAttachStreamSummaries_CadenceUnit(t *testing.T) {
	withCadence := func(action *models.Action) models.OperationObject {
		return models.OperationObject{
			Action:       action,
			DistanceUnit: "km",
			OperationSets: []models.OperationSetObject{
				{StravaStreams: &models.StravaStreamsJSON{StravaActivityStreams: *cadenceRun(85)}},
			},
		}
	}
	day := &models.ExerciseDayObject{
		Date: time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC),
		Exercises: []models.ExerciseObject{{Operations: []models.OperationObject{
			withCadence(&models.Action{StravaName: "Run"}),
			withCadence(&models.Action{StravaName: "Ride"}),
		}}},
	}
	attachStreamSummaries(day, models.User{}, nil)

	run := day.Exercises[0].Operations[0].StreamSummary
	ride := day.Exercises[0].Operations[1].StreamSummary
	if run.CadenceUnit != cadenceUnitSteps || run.Cadence.Avg != 170 {
		t.Fatalf("run cadence = %v %q, want 170 spm", run.Cadence.Avg, run.CadenceUnit)
	}
	if ride.CadenceUnit != cadenceUnitRevolutions || ride.Cadence.Avg != 85 {
		t.Fatalf("ride cadence = %v %q, want 85 rpm", ride.Cadence.Avg, ride.CadenceUnit)
	}
}

func TestFeedItemToSummary_CadenceUnit(t *testing.T) {
	tests := []struct {
		name     string
		sport    string
		cadence  *int
		wantVal  *int
		wantUnit string
	}{
		{"run doubles to steps", "Run", intPtr(86), intPtr(172), cadenceUnitSteps},
		{"ride keeps rpm", "Ride", intPtr(88), intPtr(88), cadenceUnitRevolutions},
		{"no cadence, no unit", "Run", nil, nil, ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := feedItemToSummary(models.ActivityFeedItem{ActionStravaName: tc.sport, AvgCadence: tc.cadence})
			if got.CadenceUnit != tc.wantUnit {
				t.Fatalf("unit = %q, want %q", got.CadenceUnit, tc.wantUnit)
			}
			if (got.AvgCadenceRpm == nil) != (tc.wantVal == nil) || (got.AvgCadenceRpm != nil && *got.AvgCadenceRpm != *tc.wantVal) {
				t.Fatalf("cadence = %v, want %v", got.AvgCadenceRpm, tc.wantVal)
			}
		})
	}
}
