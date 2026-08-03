package controllers

import (
	"testing"
)

// rampTimes returns 0,1,2,…,n-1 — a one-sample-per-second stream, the shape Strava returns
// for a recorded activity.
func rampTimes(n int) []int {
	times := make([]int, n)
	for i := range times {
		times[i] = i
	}
	return times
}

// TestSelectStreamIndicesWindow covers the windowing half: only samples inside the inclusive
// [from, to] range are selected, and the returned total is the pre-downsampling count (what
// the caller reports as the raw sample count).
func TestSelectStreamIndicesWindow(t *testing.T) {
	times := rampTimes(100)

	tests := []struct {
		name      string
		from, to  int
		wantIdx   []int
		wantTotal int
	}{
		{"whole stream", 0, 99, nil, 100},
		{"inclusive bounds", 10, 12, []int{10, 11, 12}, 3},
		{"single sample", 42, 42, []int{42}, 1},
		{"window beyond the stream", 200, 300, []int{}, 0},
		{"window before the stream", -50, -1, []int{}, 0},
		{"inverted window selects nothing", 50, 10, []int{}, 0},
		{"window clipped by the stream end", 97, 500, []int{97, 98, 99}, 3},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			idx, total := selectStreamIndices(times, test.from, test.to, 1, 0)
			if total != test.wantTotal {
				t.Errorf("total = %d, want %d", total, test.wantTotal)
			}
			if test.wantIdx != nil {
				if len(idx) != len(test.wantIdx) {
					t.Fatalf("got %d indices %v, want %v", len(idx), idx, test.wantIdx)
				}
				for i := range idx {
					if idx[i] != test.wantIdx[i] {
						t.Fatalf("indices = %v, want %v", idx, test.wantIdx)
					}
				}
			}
		})
	}
}

// TestSelectStreamIndicesResolution covers the thinning pass: samples are kept no closer
// together than `resolution` seconds, always starting from the first in-window sample.
func TestSelectStreamIndicesResolution(t *testing.T) {
	times := rampTimes(100)

	idx, total := selectStreamIndices(times, 0, 99, 10, 0)
	if total != 100 {
		t.Errorf("total = %d, want the pre-thinning count 100", total)
	}
	// 0,10,20,…,90.
	if len(idx) != 10 {
		t.Fatalf("got %d samples at 10s resolution, want 10: %v", len(idx), idx)
	}
	for i, want := range []int{0, 10, 20, 30, 40, 50, 60, 70, 80, 90} {
		if idx[i] != want {
			t.Fatalf("indices = %v, want 10s spacing", idx)
		}
	}

	// A resolution of 1 (or 0) must not thin anything.
	for _, resolution := range []int{0, 1} {
		if idx, _ := selectStreamIndices(times, 0, 99, resolution, 0); len(idx) != 100 {
			t.Errorf("resolution %d thinned the stream to %d samples", resolution, len(idx))
		}
	}

	// Thinning is relative to the last kept sample's time, not the index, so an irregular
	// stream (a paused activity) still comes out ~resolution apart.
	irregular := []int{0, 1, 2, 3, 60, 61, 62, 120}
	idx, _ = selectStreamIndices(irregular, 0, 120, 30, 0)
	if len(idx) != 3 {
		t.Fatalf("got %v, want the three samples 30s+ apart", idx)
	}
	for i, want := range []int{0, 60, 120} {
		if irregular[idx[i]] != want {
			t.Fatalf("kept times %v, want [0 60 120]", idx)
		}
	}
}

// TestSelectStreamIndicesMaxPoints covers the final cap: the series handed to an LLM is
// strided down to at most maxPoints, and the cap is never exceeded.
func TestSelectStreamIndicesMaxPoints(t *testing.T) {
	times := rampTimes(1000)

	for _, maxPoints := range []int{1, 2, 7, 100, 999, 1000} {
		idx, total := selectStreamIndices(times, 0, 999, 1, maxPoints)
		if total != 1000 {
			t.Errorf("total = %d, want the pre-downsampling count 1000", total)
		}
		if len(idx) > maxPoints {
			t.Errorf("got %d points for maxPoints %d", len(idx), maxPoints)
		}
		if len(idx) == 0 {
			t.Errorf("maxPoints %d downsampled the stream away entirely", maxPoints)
		}
		if idx[0] != 0 {
			t.Errorf("downsampling dropped the first sample (idx[0] = %d)", idx[0])
		}
		// Indices must stay ascending and in range.
		for i := 1; i < len(idx); i++ {
			if idx[i] <= idx[i-1] {
				t.Fatalf("indices are not ascending: %v", idx[:i+1])
			}
			if idx[i] >= len(times) {
				t.Fatalf("index %d is out of range", idx[i])
			}
		}
	}

	// maxPoints 0 means "no cap".
	if idx, _ := selectStreamIndices(times, 0, 999, 1, 0); len(idx) != 1000 {
		t.Errorf("maxPoints 0 capped the stream to %d samples", len(idx))
	}
	// A cap larger than the stream leaves it untouched.
	if idx, _ := selectStreamIndices(times, 0, 999, 1, 5000); len(idx) != 1000 {
		t.Errorf("an oversized cap changed the stream to %d samples", len(idx))
	}
}

// TestSelectStreamIndicesEmpty covers the degenerate inputs: no stream at all, and a stream
// whose every sample is filtered out.
func TestSelectStreamIndicesEmpty(t *testing.T) {
	idx, total := selectStreamIndices(nil, 0, 100, 5, 10)
	if len(idx) != 0 || total != 0 {
		t.Errorf("selectStreamIndices(nil) = %v, %d, want empty", idx, total)
	}

	idx, total = selectStreamIndices([]int{}, 0, 100, 5, 10)
	if len(idx) != 0 || total != 0 {
		t.Errorf("selectStreamIndices([]) = %v, %d, want empty", idx, total)
	}

	idx, total = selectStreamIndices(rampTimes(10), 500, 600, 5, 10)
	if len(idx) != 0 || total != 0 {
		t.Errorf("an out-of-range window returned %v, %d, want empty", idx, total)
	}
}

// TestAverageSpacing covers the reported sample spacing, including the guards that keep it
// from reporting a nonsensical 0 or dividing by zero.
func TestAverageSpacing(t *testing.T) {
	times := rampTimes(100)

	if got := averageSpacing(times, []int{0, 10, 20, 30}); got != 10 {
		t.Errorf("averageSpacing of evenly spaced samples = %d, want 10", got)
	}
	// Uneven spacing is averaged over the span: (0→30) / 2 gaps.
	if got := averageSpacing(times, []int{0, 5, 30}); got != 15 {
		t.Errorf("averageSpacing = %d, want 15", got)
	}
	// Fewer than two samples has no spacing to speak of; the floor is 1.
	if got := averageSpacing(times, []int{5}); got != 1 {
		t.Errorf("averageSpacing of one sample = %d, want 1", got)
	}
	if got := averageSpacing(times, nil); got != 1 {
		t.Errorf("averageSpacing of no samples = %d, want 1", got)
	}
	// Sub-second spacing rounds up to the 1s floor rather than to 0.
	if got := averageSpacing([]int{0, 0, 0, 0}, []int{0, 1, 2, 3}); got != 1 {
		t.Errorf("averageSpacing of identical timestamps = %d, want the floor 1", got)
	}
}

// TestIntAtFloatAt covers the bounds-guarded stream accessors: an out-of-range index yields
// zero instead of panicking, which is what lets a stream with a missing channel be read
// alongside a complete one.
func TestIntAtFloatAt(t *testing.T) {
	ints := []int{10, 20, 30}
	floats := []float64{1.5, 2.5}

	if got := intAt(ints, 1); got != 20 {
		t.Errorf("intAt(1) = %d, want 20", got)
	}
	for _, i := range []int{-1, 3, 100} {
		if got := intAt(ints, i); got != 0 {
			t.Errorf("intAt(%d) = %d, want 0", i, got)
		}
	}
	if got := intAt(nil, 0); got != 0 {
		t.Errorf("intAt(nil, 0) = %d, want 0", got)
	}

	if got := floatAt(floats, 0); got != 1.5 {
		t.Errorf("floatAt(0) = %v, want 1.5", got)
	}
	for _, i := range []int{-1, 2, 100} {
		if got := floatAt(floats, i); got != 0 {
			t.Errorf("floatAt(%d) = %v, want 0", i, got)
		}
	}
	if got := floatAt(nil, 0); got != 0 {
		t.Errorf("floatAt(nil, 0) = %v, want 0", got)
	}
}
