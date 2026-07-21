package controllers

import (
	"testing"

	"github.com/aunefyren/treningheten/models"
)

// seq returns 0..n-1 as a 1 Hz time axis.
func seq(n int) []int {
	t := make([]int, n)
	for i := range t {
		t[i] = i
	}
	return t
}

func fillF(n int, v float64) []float64 {
	d := make([]float64, n)
	for i := range d {
		d[i] = v
	}
	return d
}

func fillI(n int, v int) []int {
	d := make([]int, n)
	for i := range d {
		d[i] = v
	}
	return d
}

// analysisStreams assembles a stream from the given channels (nil channels are omitted).
func analysisStreams(times []int, vel []float64, hr []int, alt []float64) *models.StravaActivityStreams {
	s := &models.StravaActivityStreams{Time: intStream(times)}
	if vel != nil {
		s.VelocitySmooth = f64Stream(vel)
	}
	if hr != nil {
		s.Heartrate = intStream(hr)
	}
	if alt != nil {
		s.Altitude = f64Stream(alt)
	}
	return s
}

func cumFor(s *models.StravaActivityStreams) ([]int, []float64) {
	n := streamLength(s)
	times := streamTimes(s, n)
	return times, cumulativeDistanceMeters(s, times, n)
}

func TestMidpointIndex(t *testing.T) {
	if got := midpointIndex([]int{}); got != 0 {
		t.Fatalf("empty: got %d", got)
	}
	if got := midpointIndex([]int{5}); got != 0 {
		t.Fatalf("single: got %d", got)
	}
	if got := midpointIndex(seq(101)); got != 50 {
		t.Fatalf("uniform 0..100: want 50, got %d", got)
	}
	// mid = (0+100)/2 = 50; first sample at/after 50 is index 2 (t=90).
	if got := midpointIndex([]int{0, 10, 90, 100}); got != 2 {
		t.Fatalf("non-uniform: want 2, got %d", got)
	}
}

func TestGradientBandIndex(t *testing.T) {
	cases := []struct {
		grad float64
		want int
	}{
		{-10, 0}, {-5, 1}, {-3, 1}, {-2, 2}, {0, 2}, {1.9, 2}, {2, 3}, {4.9, 3}, {5, 4}, {12, 4},
	}
	for _, c := range cases {
		if got := gradientBandIndex(c.grad); got != c.want {
			t.Errorf("gradientBandIndex(%v) = %d, want %d", c.grad, got, c.want)
		}
	}
}

func TestComputeDecoupling(t *testing.T) {
	// Steady speed and heart rate: efficiency holds, so decoupling is ~0.
	steady := steadyRun(100, 3.0, 150)
	times, cum := cumFor(steady)
	if d := computeDecoupling(steady, times, cum); d == nil || *d < -0.5 || *d > 0.5 {
		t.Fatalf("steady run decoupling: want ~0, got %v", d)
	}

	// Same pace, but heart rate drifts up in the second half: positive decoupling.
	n := 101
	hr := fillI(n, 150)
	for i := 50; i < n; i++ {
		hr[i] = 165
	}
	drift := analysisStreams(seq(n), fillF(n, 3.0), hr, nil)
	times, cum = cumFor(drift)
	d := computeDecoupling(drift, times, cum)
	if d == nil {
		t.Fatal("drift run: expected a decoupling value")
	}
	if *d < 8 || *d > 10 {
		t.Fatalf("drift run decoupling: want ~9.1, got %v", *d)
	}

	// No distance signal → nil.
	hrOnly := &models.StravaActivityStreams{Time: intStream(seq(n)), Heartrate: intStream(fillI(n, 150))}
	if d := computeDecoupling(hrOnly, seq(n), nil); d != nil {
		t.Fatalf("no distance: want nil, got %v", *d)
	}
}

func TestComputeSplitHalves(t *testing.T) {
	steady := steadyRun(100, 3.0, 150)
	times, cum := cumFor(steady)
	sh := computeSplitHalves(steady, times, cum)
	if sh == nil {
		t.Fatal("expected split halves")
	}
	if sh.First.AvgHeartrateBpm == nil || *sh.First.AvgHeartrateBpm != 150 {
		t.Fatalf("first half HR: want 150, got %v", sh.First.AvgHeartrateBpm)
	}
	if sh.First.DistanceKm <= 0 || sh.Second.DistanceKm <= 0 {
		t.Fatalf("halves should carry distance: %+v", sh)
	}
	// Too few samples → nil.
	tiny := analysisStreams([]int{0, 1}, fillF(2, 3), fillI(2, 150), nil)
	if computeSplitHalves(tiny, []int{0, 1}, []float64{0, 3}) != nil {
		t.Fatal("2 samples: want nil")
	}
}

func TestComputePaceStdDev(t *testing.T) {
	if computePaceStdDev(nil) != nil {
		t.Fatal("no segments: want nil")
	}
	// One full split → not enough to vary.
	if computePaceStdDev([]models.StreamSegment{{Distance: 1, AvgPaceMinKm: 5}}) != nil {
		t.Fatal("single split: want nil")
	}
	// Two identical full splits → 0; the partial trailing split is ignored.
	segs := []models.StreamSegment{
		{Distance: 1, AvgPaceMinKm: 5},
		{Distance: 1, AvgPaceMinKm: 5},
		{Distance: 0.4, AvgPaceMinKm: 12},
	}
	if p := computePaceStdDev(segs); p == nil || *p != 0 {
		t.Fatalf("identical splits: want 0, got %v", p)
	}
	// 5.0 and 6.0 min/km → 300s and 360s → std dev 30s.
	varied := []models.StreamSegment{
		{Distance: 1, AvgPaceMinKm: 5},
		{Distance: 1, AvgPaceMinKm: 6},
	}
	if p := computePaceStdDev(varied); p == nil || *p != 30 {
		t.Fatalf("varied splits: want 30, got %v", p)
	}
}

func TestComputeBreaks(t *testing.T) {
	// Steady run, no stops → nil.
	steady := steadyRun(100, 3.0, 150)
	times, cum := cumFor(steady)
	if computeBreaks(steady, times, cum) != nil {
		t.Fatal("steady run: want no breaks")
	}

	// A 16-second stop in the middle.
	n := 101
	vel := fillF(n, 3.0)
	for i := 40; i <= 55; i++ {
		vel[i] = 0
	}
	stopped := analysisStreams(seq(n), vel, fillI(n, 150), nil)
	times, cum = cumFor(stopped)
	b := computeBreaks(stopped, times, cum)
	if b == nil || b.Count != 1 {
		t.Fatalf("want 1 break, got %+v", b)
	}
	br := b.Breaks[0]
	if br.FromSeconds != 40 || br.ToSeconds != 56 || br.DurationSeconds != 16 || b.TotalDurationSeconds != 16 {
		t.Fatalf("break bounds wrong: %+v", br)
	}

	// A brief 3-second dip is below the minimum and ignored.
	vel2 := fillF(n, 3.0)
	for i := 40; i <= 42; i++ {
		vel2[i] = 0
	}
	brief := analysisStreams(seq(n), vel2, fillI(n, 150), nil)
	times, cum = cumFor(brief)
	if computeBreaks(brief, times, cum) != nil {
		t.Fatal("3s dip: want no break")
	}
}

func TestComputeHRByGradient(t *testing.T) {
	// No altitude → nil.
	steady := steadyRun(100, 3.0, 150)
	times, cum := cumFor(steady)
	if computeHRByGradient(steady, times, cum) != nil {
		t.Fatal("no altitude: want nil")
	}

	// Climb (HR high) → flat → descent (HR low), at 3 m/s.
	n := 151
	alt := make([]float64, n)
	hr := make([]int, n)
	for i := 0; i < n; i++ {
		switch {
		case i < 50: // climbing
			alt[i] = 0.3 * float64(i)
			hr[i] = 175
		case i < 100: // flat
			alt[i] = 0.3 * 50
			hr[i] = 140
		default: // descending
			alt[i] = 0.3*50 - 0.3*float64(i-100)
			hr[i] = 130
		}
	}
	s := analysisStreams(seq(n), fillF(n, 3.0), hr, alt)
	times, cum = cumFor(s)
	buckets := computeHRByGradient(s, times, cum)
	if len(buckets) == 0 {
		t.Fatal("expected gradient buckets")
	}
	byLabel := map[string]models.StreamHRGradientBucket{}
	for _, b := range buckets {
		byLabel[b.Label] = b
	}
	climb, okC := byLabel["steep climb"]
	descent, okD := byLabel["steep descent"]
	if !okC || !okD || climb.AvgHeartrateBpm == nil || descent.AvgHeartrateBpm == nil {
		t.Fatalf("missing climb/descent buckets: %+v", byLabel)
	}
	if *climb.AvgHeartrateBpm <= *descent.AvgHeartrateBpm {
		t.Fatalf("climb HR (%d) should exceed descent HR (%d)", *climb.AvgHeartrateBpm, *descent.AvgHeartrateBpm)
	}
	// Open-ended bands carry a null bound.
	if climb.MaxGradePct != nil {
		t.Fatalf("steep climb should have open upper bound, got %v", *climb.MaxGradePct)
	}
	if descent.MinGradePct != nil {
		t.Fatalf("steep descent should have open lower bound, got %v", *descent.MinGradePct)
	}
}

// TestComputeAnalysis_EndToEnd runs the whole analysis through SummarizeStreams so it exercises
// the real segment/cum inputs, and confirms an all-strength (channel-less) stream yields none.
func TestComputeAnalysis_EndToEnd(t *testing.T) {
	// ~2.4 km at 3 m/s so there are at least two full-km splits for the pace std dev.
	n := 801
	alt := make([]float64, n)
	for i := 0; i < n; i++ {
		alt[i] = 0.2 * float64(i) // gentle steady climb
	}
	run := analysisStreams(seq(n), fillF(n, 3.0), fillI(n, 150), alt)
	summary := SummarizeStreams(run, "km", 190, 0, "max")
	if summary == nil || summary.Analysis == nil {
		t.Fatal("expected an analysis block")
	}
	a := summary.Analysis
	if a.DecouplingPct == nil || a.SplitHalves == nil || a.PaceStdDevSeconds == nil || len(a.HRByGradient) == 0 {
		t.Fatalf("analysis missing blocks: %+v", a)
	}

	// A stream with no distance/HR/altitude (a stand-in for strength) produces no analysis.
	bare := &models.StravaActivityStreams{Time: intStream(seq(5))}
	if got := computeAnalysis(bare, seq(5), nil, nil); got != nil {
		t.Fatalf("bare stream: want nil analysis, got %+v", got)
	}
}
