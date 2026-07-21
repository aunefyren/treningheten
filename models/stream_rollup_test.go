package models

import "testing"

func intCh(d []int) *StravaStream[int]           { return &StravaStream[int]{Data: d} }
func floatCh(d []float64) *StravaStream[float64] { return &StravaStream[float64]{Data: d} }

func TestComputeStreamRollup_Nil(t *testing.T) {
	got := ComputeStreamRollup(nil)
	if got.AvgHeartrate != nil || got.MaxHeartrate != nil || got.AvgCadence != nil ||
		got.TempC != nil || got.ElevationGainM != nil {
		t.Fatalf("nil stream should yield an empty rollup, got %+v", got)
	}
}

func TestComputeStreamRollup_Full(t *testing.T) {
	s := &StravaActivityStreams{
		// avg = (150+160+0+170)/3 = 160 (the 0 is a paused sample, skipped); max = 170.
		Heartrate: intCh([]int{150, 160, 0, 170}),
		// avg = (80+0+90)/2 = 85.
		Cadence: intCh([]int{80, 0, 90}),
		// avg = (10+20+15)/3 = 15.
		Temp: intCh([]int{10, 20, 15}),
		// gains: +5, +5, -3 → 10 m gained.
		Altitude: floatCh([]float64{100, 105, 110, 107}),
	}
	got := ComputeStreamRollup(s)

	if got.AvgHeartrate == nil || *got.AvgHeartrate != 160 {
		t.Errorf("avg HR: want 160, got %v", got.AvgHeartrate)
	}
	if got.MaxHeartrate == nil || *got.MaxHeartrate != 170 {
		t.Errorf("max HR: want 170, got %v", got.MaxHeartrate)
	}
	if got.AvgCadence == nil || *got.AvgCadence != 85 {
		t.Errorf("avg cadence: want 85, got %v", got.AvgCadence)
	}
	if got.TempC == nil || *got.TempC != 15 {
		t.Errorf("temp: want 15, got %v", got.TempC)
	}
	if got.ElevationGainM == nil || *got.ElevationGainM != 10 {
		t.Errorf("elevation gain: want 10, got %v", got.ElevationGainM)
	}
}

func TestComputeStreamRollup_PartialChannels(t *testing.T) {
	// A dropout-only HR channel (all non-positive) yields no average, and the plausible cap
	// keeps a glitch out of the max.
	s := &StravaActivityStreams{
		Heartrate: intCh([]int{0, 0, 900}), // 900 is above the plausible cap
		Altitude:  floatCh([]float64{50}),  // a single sample can't establish gain
		Temp:      intCh([]int{-4, -6}),    // negative temps average fine
	}
	got := ComputeStreamRollup(s)
	if got.AvgHeartrate != nil {
		t.Errorf("all-paused HR should have no average, got %v", *got.AvgHeartrate)
	}
	if got.MaxHeartrate != nil {
		t.Errorf("implausible-only HR should have no max, got %v", *got.MaxHeartrate)
	}
	if got.ElevationGainM != nil {
		t.Errorf("single altitude sample should have no gain, got %v", *got.ElevationGainM)
	}
	if got.TempC == nil || *got.TempC != -5 {
		t.Errorf("temp: want -5, got %v", got.TempC)
	}
	if got.AvgCadence != nil {
		t.Errorf("absent cadence channel should be nil, got %v", *got.AvgCadence)
	}
}
