package controllers

import (
	"math"
	"testing"
	"time"

	"github.com/aunefyren/treningheten/models"
)

func f64Stream(d []float64) *models.StravaStream[float64] {
	return &models.StravaStream[float64]{Data: d}
}
func intStream(d []int) *models.StravaStream[int] {
	return &models.StravaStream[int]{Data: d}
}
func llStream(d [][]float64) *models.StravaStream[[]float64] {
	return &models.StravaStream[[]float64]{Data: d}
}

// steadyRun builds a synthetic run: seconds seconds long, 1 Hz, moving at speedMps with a
// constant heart rate. Distance comes from velocity so tests can assert exact splits.
func steadyRun(seconds int, speedMps float64, hr int) *models.StravaActivityStreams {
	n := seconds + 1
	times := make([]int, n)
	vel := make([]float64, n)
	hrs := make([]int, n)
	for i := 0; i < n; i++ {
		times[i] = i
		vel[i] = speedMps
		hrs[i] = hr
	}
	return &models.StravaActivityStreams{
		Time:           intStream(times),
		VelocitySmooth: f64Stream(vel),
		Heartrate:      intStream(hrs),
	}
}

func TestSummarizeStreams_NilAndEmpty(t *testing.T) {
	if SummarizeStreams(nil, "km", hrAnchor{}) != nil {
		t.Fatal("nil streams should summarize to nil")
	}
	// Streams present but every channel empty: header stats nil, no segments/route/zones.
	empty := &models.StravaActivityStreams{Heartrate: intStream([]int{})}
	s := SummarizeStreams(empty, "km", hrAnchor{})
	if s == nil {
		t.Fatal("empty (non-nil) streams should still return a summary")
	}
	if s.Heartrate != nil || len(s.Segments) != 0 || s.Route != nil || len(s.HRZones) != 0 {
		t.Fatalf("empty streams produced content: %+v", s)
	}
}

func TestSummarizeStreams_Header(t *testing.T) {
	streams := steadyRun(100, 3.0, 150) // 3 m/s -> 10.8 km/h
	streams.Altitude = f64Stream(func() []float64 {
		d := make([]float64, 101)
		for i := range d {
			d[i] = float64(i) // climbs 1 m/s -> 100 m gain
		}
		return d
	}())
	streams.Watts = intStream(func() []int {
		d := make([]int, 101)
		for i := range d {
			d[i] = 200
		}
		return d
	}())

	s := SummarizeStreams(streams, "km", hrAnchor{MaxBpm: 200, Basis: "age"})
	if s.Heartrate == nil || s.Heartrate.Avg != 150 {
		t.Fatalf("avg HR = %+v, want 150", s.Heartrate)
	}
	if s.Speed == nil || math.Abs(s.Speed.AvgKmh-10.8) > 0.1 {
		t.Fatalf("avg speed = %+v, want ~10.8 km/h", s.Speed)
	}
	if s.Elevation == nil || math.Abs(s.Elevation.GainM-100) > 0.1 {
		t.Fatalf("elevation gain = %+v, want 100 m", s.Elevation)
	}
	if s.Power == nil || s.Power.AvgW != 200 {
		t.Fatalf("avg power = %+v, want 200 W", s.Power)
	}
	if s.DurationSeconds != 100 {
		t.Fatalf("duration = %d, want 100", s.DurationSeconds)
	}
}

func TestSummarizeStreams_Segments(t *testing.T) {
	// 1000 s at 3 m/s = ~3000 m -> three full km splits plus a small trailing split.
	streams := steadyRun(1000, 3.0, 150)
	s := SummarizeStreams(streams, "km", hrAnchor{})
	if len(s.Segments) < 3 {
		t.Fatalf("got %d segments, want >= 3", len(s.Segments))
	}
	// First full split is ~1 km and its indices/pace are set.
	first := s.Segments[0]
	if math.Abs(first.Distance-1.0) > 0.02 {
		t.Fatalf("first split distance = %v, want ~1.0", first.Distance)
	}
	if first.Index != 1 || first.ToPoint <= first.FromPoint {
		t.Fatalf("first split indices bad: %+v", first)
	}
	// 3 m/s -> 12000 s per... pace = 1000 m / 3 (m/s) = 333 s/km = 5.56 min/km.
	if math.Abs(first.AvgPaceMinKm-5.56) > 0.1 {
		t.Fatalf("first split pace = %v, want ~5.56 min/km", first.AvgPaceMinKm)
	}
	if first.AvgHeartrateBpm == nil || *first.AvgHeartrateBpm != 150 {
		t.Fatalf("first split HR = %v, want 150", first.AvgHeartrateBpm)
	}
	// Total split distance ~ 3.0 km.
	total := 0.0
	for _, seg := range s.Segments {
		total += seg.Distance
	}
	if math.Abs(total-3.0) > 0.05 {
		t.Fatalf("summed split distance = %v, want ~3.0", total)
	}
}

func TestSummarizeStreams_SegmentsMileUnit(t *testing.T) {
	// Same distance measured in miles yields fewer (longer) splits than in km.
	streams := steadyRun(1000, 3.0, 150)
	km := SummarizeStreams(streams, "km", hrAnchor{})
	mi := SummarizeStreams(steadyRun(1000, 3.0, 150), "mi", hrAnchor{})
	if !(len(mi.Segments) < len(km.Segments)) {
		t.Fatalf("mile splits (%d) should be fewer than km splits (%d)", len(mi.Segments), len(km.Segments))
	}
	if mi.Segments[0].DistanceUnit != "mi" {
		t.Fatalf("segment unit = %q, want mi", mi.Segments[0].DistanceUnit)
	}
}

func TestSummarizeStreams_NoDistanceNoSegments(t *testing.T) {
	// HR only (e.g. a treadmill run): no distance signal, so no segments and no route.
	streams := &models.StravaActivityStreams{
		Time:      intStream([]int{0, 1, 2, 3}),
		Heartrate: intStream([]int{120, 130, 140, 150}),
	}
	s := SummarizeStreams(streams, "km", hrAnchor{MaxBpm: 190, Basis: "age"})
	if len(s.Segments) != 0 {
		t.Fatalf("expected no segments without distance, got %d", len(s.Segments))
	}
	if s.Route != nil {
		t.Fatal("expected no route without GPS")
	}
	if len(s.HRZones) == 0 {
		t.Fatal("expected HR zones from the heart-rate channel")
	}
}

func TestSummarizeStreams_Route(t *testing.T) {
	pts := [][]float64{{59.9, 10.7}, {59.9, 10.701}, {59.901, 10.701}}
	streams := &models.StravaActivityStreams{
		Time:   intStream([]int{0, 10, 20}),
		LatLng: llStream(pts),
	}
	s := SummarizeStreams(streams, "km", hrAnchor{})
	if !s.HasGPS || s.Route == nil {
		t.Fatal("expected a route with GPS")
	}
	r := s.Route
	if r.PointCount != 3 {
		t.Fatalf("point count = %d, want 3", r.PointCount)
	}
	if len(r.Start) != 2 || r.Start[0] != 59.9 || r.Start[1] != 10.7 {
		t.Fatalf("start = %v", r.Start)
	}
	if len(r.End) != 2 || r.End[0] != 59.901 || r.End[1] != 10.701 {
		t.Fatalf("end = %v", r.End)
	}
	if r.BoundingBox == nil || r.BoundingBox.MinLat != 59.9 || r.BoundingBox.MaxLat != 59.901 {
		t.Fatalf("bbox = %+v", r.BoundingBox)
	}
	if r.DistanceKm <= 0 {
		t.Fatalf("route distance = %v, want > 0", r.DistanceKm)
	}
}

func TestSummarizeStreams_RouteDownsample(t *testing.T) {
	// A long trace is strided to the overview cap and still keeps the true final point.
	n := routeOverviewMaxPoints * 3
	pts := make([][]float64, n)
	for i := range pts {
		pts[i] = []float64{59.9 + float64(i)*1e-5, 10.7}
	}
	streams := &models.StravaActivityStreams{LatLng: llStream(pts)}
	s := SummarizeStreams(streams, "km", hrAnchor{})
	if len(s.Route.Polyline) > routeOverviewMaxPoints+1 {
		t.Fatalf("polyline has %d points, want <= %d", len(s.Route.Polyline), routeOverviewMaxPoints+1)
	}
	last := s.Route.Polyline[len(s.Route.Polyline)-1]
	if !sameLatLng(last, s.Route.End) {
		t.Fatalf("polyline end %v != route end %v", last, s.Route.End)
	}
}

func TestSummarizeStreams_Elevation(t *testing.T) {
	// A climb of 100 m over 200 s then a descent of 40 m: gain 100, loss 40, and the
	// biggest climb is that 100 m ascent. Distance from a steady 3 m/s.
	n := 301
	times := make([]int, n)
	vel := make([]float64, n)
	alt := make([]float64, n)
	for i := 0; i < n; i++ {
		times[i] = i
		vel[i] = 3.0
		if i <= 200 {
			alt[i] = float64(i) * 0.5 // 0 -> 100 m over 200 s
		} else {
			alt[i] = 100 - float64(i-200)*0.4 // 100 -> 60 m over 100 s
		}
	}
	streams := &models.StravaActivityStreams{
		Time:           intStream(times),
		VelocitySmooth: f64Stream(vel),
		Altitude:       f64Stream(alt),
	}
	s := SummarizeStreams(streams, "km", hrAnchor{})

	if s.Elevation == nil {
		t.Fatal("expected elevation stats")
	}
	if math.Abs(s.Elevation.GainM-100) > 0.5 {
		t.Fatalf("gain = %v, want ~100", s.Elevation.GainM)
	}
	if math.Abs(s.Elevation.LossM-40) > 0.5 {
		t.Fatalf("loss = %v, want ~40", s.Elevation.LossM)
	}
	if s.Elevation.BiggestClimb == nil {
		t.Fatal("expected a biggest climb")
	}
	c := s.Elevation.BiggestClimb
	if math.Abs(c.GainM-100) > 0.5 || c.FromPoint != 0 || c.ToPoint != 200 {
		t.Fatalf("biggest climb = %+v, want ~100 m over [0,200]", c)
	}
	if c.GradePct <= 0 {
		t.Fatalf("climb grade = %v, want > 0", c.GradePct)
	}
	if len(s.ElevationProfile) < 2 {
		t.Fatalf("expected an elevation profile, got %d points", len(s.ElevationProfile))
	}
	// Profile is ordered by distance and ends at the final sample's altitude.
	if s.ElevationProfile[len(s.ElevationProfile)-1].AltitudeM != 60 {
		t.Fatalf("profile should end at 60 m, got %v", s.ElevationProfile[len(s.ElevationProfile)-1])
	}
}

func TestSummarizeStreams_FlatNoClimb(t *testing.T) {
	// Flat altitude: no meaningful climb to report.
	streams := steadyRun(100, 3.0, 150)
	flat := make([]float64, 101)
	for i := range flat {
		flat[i] = 12
	}
	streams.Altitude = f64Stream(flat)
	s := SummarizeStreams(streams, "km", hrAnchor{})
	if s.Elevation != nil && s.Elevation.BiggestClimb != nil {
		t.Fatalf("flat course should have no biggest climb, got %+v", s.Elevation.BiggestClimb)
	}
}

func TestAttachStreamSummaries(t *testing.T) {
	streams := steadyRun(100, 3.0, 150)
	day := &models.ExerciseDayObject{
		Date: time.Date(2020, 6, 1, 0, 0, 0, 0, time.UTC), // age drives the "age" basis

		Exercises: []models.ExerciseObject{{
			Operations: []models.OperationObject{
				{
					DistanceUnit: "km",
					OperationSets: []models.OperationSetObject{
						{StravaStreams: &models.StravaStreamsJSON{StravaActivityStreams: *streams}},
					},
				},
				{DistanceUnit: "km"}, // no streams -> left nil
			},
		}},
	}
	// The owner is passed separately: day.User is the public view and carries no heart-rate
	// settings, so the age-based HR basis has to come from the real row.
	owner := models.User{BirthDate: timePtr(time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC))}
	attachStreamSummaries(day, owner)

	withStreams := day.Exercises[0].Operations[0]
	if withStreams.StreamSummary == nil {
		t.Fatal("operation with streams should have a summary")
	}
	if withStreams.StreamSummary.HRMaxBasis != "age" {
		t.Fatalf("HR basis = %q, want age (birth date present)", withStreams.StreamSummary.HRMaxBasis)
	}
	if withStreams.StreamSummary.HRZoneSystem != models.HRZoneSystemPercentMax {
		t.Fatalf("HR zone system = %q, want percent_max (nothing chosen, no resting HR)", withStreams.StreamSummary.HRZoneSystem)
	}
	if day.Exercises[0].Operations[1].StreamSummary != nil {
		t.Fatal("operation without streams should have no summary")
	}
}

func timePtr(t time.Time) *time.Time { return &t }
