package models

import (
	"math"
	"time"

	"github.com/google/uuid"
)

// StreamDerivedVersion is the version of everything derived from an activity's stored stream
// (the Operation rollup columns and its BestEffort rows). Operation.StreamDerivedVersion
// records which version an operation was last processed with; the Migrate backfill
// reprocesses any stream-bearing operation below it. Bump it when a derivation changes so
// existing activities are recomputed once on the next boot.
const StreamDerivedVersion = 1

// BestEffort is an activity's fastest time over one target distance (its 5 km inside a
// 12 km run), found by a sliding window over the stream. One row per (operation, distance).
// Rows are derived data: written by the Strava sync and the stream backfill, replaced
// whenever the activity is reprocessed, and filtered at read time through the usual
// enabled/on joins, so a deleted or hidden workout drops out without touching them.
// See docs/statistics.md.
type BestEffort struct {
	GormModel
	UserID         uuid.UUID `json:"user_id" gorm:"type:varchar(100); not null; index"`
	OperationID    uuid.UUID `json:"operation_id" gorm:"type:varchar(100); not null; index"`
	DistanceM      int       `json:"distance_m" gorm:"not null"`
	ElapsedSeconds int       `json:"elapsed_seconds" gorm:"not null"`
	StartPoint     int       `json:"start_point" gorm:"not null"`
	EndPoint       int       `json:"end_point" gorm:"not null"`
}

// BestEffortTarget is one distance best efforts are tracked for.
type BestEffortTarget struct {
	DistanceM int
	Label     string
}

var footBestEffortTargets = []BestEffortTarget{
	{1000, "1 km"},
	{1609, "1 mile"},
	{5000, "5 km"},
	{10000, "10 km"},
	{21097, "Half marathon"},
	{42195, "Marathon"},
}

var rideBestEffortTargets = []BestEffortTarget{
	{10000, "10 km"},
	{20000, "20 km"},
	{40000, "40 km"},
	{100000, "100 km"},
}

// Fastest believable average speed over a whole best-effort window, in metres per second.
// A window faster than this is a GPS glitch, not an effort, and is skipped. Foot: 30 km/h,
// above the 1 km world record. Ride: 100 km/h.
const (
	footBestEffortMaxSpeed = 30 / 3.6
	rideBestEffortMaxSpeed = 100 / 3.6
)

// bestEffortSampleSlack is how far a single step between two samples may exceed the window
// cap before it is treated as a GPS jump and contributes no distance. Looser than the window
// cap because honest GPS jitters from sample to sample.
const bestEffortSampleSlack = 1.5

var footBestEffortSports = map[string]bool{"Run": true, "TrailRun": true, "VirtualRun": true, "Walk": true, "Hike": true}
var rideBestEffortSports = map[string]bool{"Ride": true, "VirtualRide": true, "EBikeRide": true, "GravelRide": true, "MountainBikeRide": true}

// BestEffortTargets returns the distances tracked for a sport (an Action.StravaName) and the
// speed cap that rejects glitches. Sports without best efforts return nil.
func BestEffortTargets(stravaName string) ([]BestEffortTarget, float64) {
	switch {
	case footBestEffortSports[stravaName]:
		return footBestEffortTargets, footBestEffortMaxSpeed
	case rideBestEffortSports[stravaName]:
		return rideBestEffortTargets, rideBestEffortMaxSpeed
	}
	return nil, 0
}

// BestEffortLabel names a tracked distance for a sport ("5 km", "Half marathon"), or "" when
// the sport doesn't track it.
func BestEffortLabel(stravaName string, distanceM int) string {
	targets, _ := BestEffortTargets(stravaName)
	for _, target := range targets {
		if target.DistanceM == distanceM {
			return target.Label
		}
	}
	return ""
}

// BestEffortResult is one computed effort, before it is stored.
type BestEffortResult struct {
	DistanceM      int
	ElapsedSeconds int
	StartPoint     int
	EndPoint       int
}

// ComputeBestEfforts finds the fastest window over each target distance the activity covers.
// Distance is the same cumulative GPS (or integrated speed) distance the splits use; time is
// the recorded clock, so a pause inside a window counts, as on a stopwatch. The window start
// is interpolated between samples so it covers exactly the target distance. Windows faster
// than the sport's cap are skipped, and a single step faster than 1.5× the cap adds no
// distance, so a GPS jump can't hide inside a longer window. Pure; shared by the sync and
// the backfill.
func ComputeBestEfforts(streams *StravaActivityStreams, stravaName string) []BestEffortResult {
	targets, maxSpeed := BestEffortTargets(stravaName)
	if streams == nil || len(targets) == 0 {
		return nil
	}

	n := StreamLength(streams)
	if n < 2 {
		return nil
	}
	times := StreamTimes(streams, n)
	cum := CumulativeDistanceMeters(streams, times, n)
	if cum == nil {
		return nil
	}
	cum = withoutGPSJumps(cum, times, maxSpeed*bestEffortSampleSlack)
	if cum[n-1] <= 0 {
		return nil
	}

	results := []BestEffortResult{}
	for _, target := range targets {
		if result, ok := fastestWindow(cum, times, float64(target.DistanceM), maxSpeed); ok {
			result.DistanceM = target.DistanceM
			results = append(results, result)
		}
	}
	return results
}

// withoutGPSJumps rebuilds a cumulative distance with every step faster than maxStepSpeed
// (or covering distance in no time) counted as zero, so jumps between two points vanish.
func withoutGPSJumps(cum []float64, times []int, maxStepSpeed float64) []float64 {
	cleaned := make([]float64, len(cum))
	for i := 1; i < len(cum); i++ {
		step := cum[i] - cum[i-1]
		dt := times[i] - times[i-1]
		if step > 0 && (dt <= 0 || step/float64(dt) > maxStepSpeed) {
			step = 0
		}
		cleaned[i] = cleaned[i-1] + step
	}
	return cleaned
}

// fastestWindow is a two-pointer sweep: for every end sample j it keeps start as the last
// sample with at least `distance` metres between it and j, interpolates the exact start time
// at cum[j]-distance, and keeps the shortest elapsed time.
func fastestWindow(cum []float64, times []int, distance float64, maxSpeed float64) (BestEffortResult, bool) {
	best := BestEffortResult{}
	bestSeconds := math.MaxFloat64
	start := 0

	for end := 1; end < len(cum); end++ {
		if cum[end] < distance {
			continue
		}
		for start+1 < end && cum[end]-cum[start+1] >= distance {
			start++
		}

		// Interpolate the moment the window covered exactly `distance` metres.
		from := cum[end] - distance
		startTime := float64(times[start])
		if span := cum[start+1] - cum[start]; span > 0 {
			fraction := (from - cum[start]) / span
			startTime += fraction * float64(times[start+1]-times[start])
		}

		elapsed := float64(times[end]) - startTime
		if elapsed <= 0 || distance/elapsed > maxSpeed {
			continue
		}
		if elapsed < bestSeconds {
			bestSeconds = elapsed
			best = BestEffortResult{StartPoint: start, EndPoint: end}
		}
	}

	if bestSeconds == math.MaxFloat64 {
		return BestEffortResult{}, false
	}
	best.ElapsedSeconds = int(math.Round(bestSeconds))
	return best, true
}

// RecordedAt is a helper for callers that need the activity's own moment: the session time
// when it has one, otherwise the day.
func RecordedAt(sessionTime *time.Time, day time.Time) time.Time {
	if sessionTime != nil && !sessionTime.IsZero() {
		return *sessionTime
	}
	return day
}
