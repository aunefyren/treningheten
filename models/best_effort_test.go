package models

import (
	"testing"
	"time"
)

// metresToLat converts metres north into degrees of latitude (haversine's earth radius).
const metresPerDegreeLat = 6371000.0 * 3.141592653589793 / 180

// track builds a north-heading GPS stream from per-second speeds (m/s); a negative speed
// marks a paused second: the clock jumps 60 s and the position stays put.
func track(speeds []float64) *StravaActivityStreams {
	times := []int{0}
	points := [][]float64{{59.9, 10.7}}
	lat, clock := 59.9, 0
	for _, speed := range speeds {
		if speed < 0 {
			clock += 60
		} else {
			clock++
			lat += speed / metresPerDegreeLat
		}
		times = append(times, clock)
		points = append(points, []float64{lat, 10.7})
	}
	return &StravaActivityStreams{
		Time:   &StravaStream[int]{Data: times},
		LatLng: &StravaStream[[]float64]{Data: points},
	}
}

func steady(speed float64, seconds int) []float64 {
	speeds := make([]float64, seconds)
	for i := range speeds {
		speeds[i] = speed
	}
	return speeds
}

func effortsByDistance(results []BestEffortResult) map[int]int {
	byDistance := map[int]int{}
	for _, result := range results {
		byDistance[result.DistanceM] = result.ElapsedSeconds
	}
	return byDistance
}

func TestComputeBestEfforts(t *testing.T) {
	// 1000 s at 3 m/s, then 400 s at 5 m/s: the fastest 1 km is all in the fast part.
	negativeSplit := append(steady(3, 1000), steady(5, 400)...)

	// A 30 s pause in the middle of a 1 km stretch at 4 m/s.
	paused := append(append(steady(4, 125), -1), steady(4, 400)...)

	// A GPS jump: one sample 900 m ahead (900 m/s), inside an otherwise 3 m/s run.
	glitch := append(append(steady(3, 300), 900), steady(3, 1000)...)

	tests := []struct {
		name   string
		sport  string
		stream *StravaActivityStreams
		want   map[int]int // distance → seconds; 0 means "must be absent"
	}{
		{"nil stream", "Run", nil, map[int]int{}},
		{"sport without efforts", "Yoga", track(steady(4, 2000)), map[int]int{}},
		{"too short for any target", "Run", track(steady(4, 200)), map[int]int{1000: 0}},
		{"steady run", "Run", track(steady(4, 1300)), map[int]int{1000: 250, 1609: 402, 5000: 1250, 10000: 0}},
		{"fastest window wins", "Run", track(negativeSplit), map[int]int{1000: 200}},
		{"a pause counts", "Run", track(paused), map[int]int{1000: 250, 1609: 462}},
		{"GPS jump skipped", "Run", track(glitch), map[int]int{1000: 333}},
		{"ride targets", "Ride", track(steady(10, 2100)), map[int]int{10000: 1000, 20000: 2000, 1000: 0}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := effortsByDistance(ComputeBestEfforts(test.stream, test.sport))
			for distance, want := range test.want {
				seconds, found := got[distance]
				if want == 0 {
					if found {
						t.Errorf("%d m: got %d s, want none", distance, seconds)
					}
					continue
				}
				if !found || seconds < want-2 || seconds > want+2 {
					t.Errorf("%d m: got %d s (found %v), want about %d", distance, seconds, found, want)
				}
			}
		})
	}
}

func TestComputeBestEffortsFromSpeedOnly(t *testing.T) {
	// A treadmill run: no GPS, only velocity_smooth at 5 m/s.
	n := 401
	times := make([]int, n)
	velocity := make([]float64, n)
	for i := range times {
		times[i] = i
		velocity[i] = 5
	}
	streams := &StravaActivityStreams{
		Time:           &StravaStream[int]{Data: times},
		VelocitySmooth: &StravaStream[float64]{Data: velocity},
	}
	got := effortsByDistance(ComputeBestEfforts(streams, "Run"))
	if got[1000] != 200 {
		t.Errorf("1 km from speed = %d s, want 200", got[1000])
	}
}

func TestBestEffortLabel(t *testing.T) {
	tests := []struct {
		sport    string
		distance int
		want     string
	}{
		{"Run", 5000, "5 km"},
		{"Walk", 21097, "Half marathon"},
		{"Ride", 40000, "40 km"},
		{"Ride", 5000, ""},
		{"Yoga", 1000, ""},
	}
	for _, test := range tests {
		if got := BestEffortLabel(test.sport, test.distance); got != test.want {
			t.Errorf("BestEffortLabel(%q, %d) = %q, want %q", test.sport, test.distance, got, test.want)
		}
	}
}

func TestRecordedAt(t *testing.T) {
	day := time.Date(2026, 10, 20, 0, 0, 0, 0, time.UTC)
	clock := time.Date(2026, 10, 20, 17, 30, 0, 0, time.UTC)
	zero := time.Time{}
	if got := RecordedAt(&clock, day); !got.Equal(clock) {
		t.Errorf("with a time: got %v", got)
	}
	if got := RecordedAt(nil, day); !got.Equal(day) {
		t.Errorf("without a time: got %v", got)
	}
	if got := RecordedAt(&zero, day); !got.Equal(day) {
		t.Errorf("with a zero time: got %v", got)
	}
}
