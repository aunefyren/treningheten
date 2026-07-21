package controllers

import (
	"testing"
	"time"

	"github.com/aunefyren/treningheten/database"
	"github.com/aunefyren/treningheten/models"

	"github.com/google/uuid"
)

func fullSummary() *models.StreamSummary {
	return &models.StreamSummary{
		Available:        []string{"heartrate", "altitude"},
		HasGPS:           true,
		DurationSeconds:  100,
		Heartrate:        &models.StreamStat{Avg: 150},
		Segments:         []models.StreamSegment{{Index: 1}},
		HRZones:          []models.StreamHRZone{{Zone: 1}},
		HRMaxBpm:         190,
		Elevation:        &models.StreamElevationStat{GainM: 12},
		Route:            &models.StreamRoute{PointCount: 5},
		ElevationProfile: []models.StreamElevationPoint{{DistanceKm: 1}},
		Analysis:         &models.StreamAnalysis{},
	}
}

func TestFilterStreamSummary(t *testing.T) {
	if filterStreamSummary(nil, []string{"segments"}) != nil {
		t.Fatal("nil summary should stay nil")
	}
	if filterStreamSummary(fullSummary(), nil) != nil {
		t.Fatal("empty include should select nothing")
	}
	if filterStreamSummary(fullSummary(), []string{"nonsense"}) != nil {
		t.Fatal("unknown token should select nothing")
	}

	got := filterStreamSummary(fullSummary(), []string{"segments", "zones"})
	if got == nil {
		t.Fatal("expected a filtered summary")
	}
	// Header stats always ride along.
	if got.Heartrate == nil || !got.HasGPS || got.DurationSeconds != 100 {
		t.Errorf("header should be retained: %+v", got)
	}
	if len(got.Segments) == 0 || len(got.HRZones) == 0 || got.HRMaxBpm != 190 {
		t.Errorf("requested blocks missing: %+v", got)
	}
	// Unrequested heavy blocks are dropped.
	if got.Elevation != nil || got.Route != nil || got.ElevationProfile != nil || got.Analysis != nil {
		t.Errorf("unrequested blocks leaked: %+v", got)
	}

	// Case-insensitive, and every block selectable.
	all := filterStreamSummary(fullSummary(), []string{"SEGMENTS", "Zones", "elevation", "route", "profile", "analysis"})
	if all.Elevation == nil || all.Route == nil || len(all.ElevationProfile) == 0 || all.Analysis == nil {
		t.Errorf("all-blocks filter dropped something: %+v", all)
	}
}

func TestFeedItemToSummary_ScalarsAndPace(t *testing.T) {
	avgHR, maxHR, cad, temp, elev := 150, 175, 85, 20, 120.0
	item := models.ActivityFeedItem{
		OperationID:     uuid.New(),
		ExerciseID:      uuid.New(),
		Date:            time.Now(),
		ActionName:      "Run",
		ActionType:      "moving",
		Distance:        10,
		DistanceUnit:    "km",
		DurationSeconds: 3600,
		MovingSeconds:   3000,
		HasStrava:       true,
		AvgHeartrate:    &avgHR,
		MaxHeartrate:    &maxHR,
		AvgCadence:      &cad,
		TempC:           &temp,
		ElevationGainM:  &elev,
	}
	got := feedItemToSummary(item)
	if got.Source != "strava" || !got.HasStreams {
		t.Errorf("source/has_streams wrong: %+v", got)
	}
	if got.MovingSeconds != 3000 {
		t.Errorf("moving seconds: want 3000, got %d", got.MovingSeconds)
	}
	// Pace prefers moving time: 3000 s / 10 km = 5.0 min/km.
	if got.AvgPaceMinKm != 5.0 {
		t.Errorf("pace: want 5.0, got %v", got.AvgPaceMinKm)
	}
	if got.AvgHeartrateBpm == nil || *got.AvgHeartrateBpm != 150 || got.MaxHeartrateBpm == nil || *got.MaxHeartrateBpm != 175 {
		t.Errorf("HR scalars wrong: %+v", got)
	}
	if got.ElevationGainM == nil || *got.ElevationGainM != 120 {
		t.Errorf("elevation: %+v", got)
	}

	// No moving time → pace falls back to elapsed duration (3600 s / 12 km = 5.0).
	noMoving := item
	noMoving.MovingSeconds = 0
	noMoving.Distance = 12
	noMoving.DurationSeconds = 3600
	if p := feedItemToSummary(noMoving).AvgPaceMinKm; p != 5.0 {
		t.Errorf("fallback pace: want 5.0, got %v", p)
	}

	// A distance-less strength item → no pace, no stream scalars.
	lift := models.ActivityFeedItem{OperationID: uuid.New(), ExerciseID: uuid.New(), ActionName: "Lifting", ActionType: "lifting", TopWeight: 100, WeightUnit: "kg"}
	ls := feedItemToSummary(lift)
	if ls.AvgPaceMinKm != 0 || ls.AvgHeartrateBpm != nil || ls.MovingSeconds != 0 {
		t.Errorf("lift should have no pace/scalars: %+v", ls)
	}
}

// TestAssembleSingleActivityInclude drives the get_activity summary-first path end to end: an
// operation carrying a Strava stream returns the requested processed blocks only when include
// asks for them, and never the ones it didn't.
func TestAssembleSingleActivityInclude(t *testing.T) {
	newControllerTestDB(t)
	user := createTestUser(t, "include@test.dev", "Inc")

	// day → exercise → operation → set with a synthetic climbing run stream.
	day := models.ExerciseDay{Date: time.Now(), Enabled: true, UserID: &user.ID}
	day.ID = uuid.New()
	if err := database.Instance.Omit("User", "Goal").Create(&day).Error; err != nil {
		t.Fatalf("seed day: %v", err)
	}
	exercise := models.Exercise{Enabled: true, IsOn: true, ExerciseDayID: day.ID}
	exercise.ID = uuid.New()
	if err := database.Instance.Omit("ExerciseDay").Create(&exercise).Error; err != nil {
		t.Fatalf("seed exercise: %v", err)
	}
	op := models.Operation{Enabled: true, Type: "moving", DistanceUnit: "km", WeightUnit: "kg", ExerciseID: exercise.ID}
	op.ID = uuid.New()
	if err := database.Instance.Omit("Exercise", "Action", "Gear").Create(&op).Error; err != nil {
		t.Fatalf("seed operation: %v", err)
	}

	n := 801
	alt := make([]float64, n)
	for i := 0; i < n; i++ {
		alt[i] = 0.2 * float64(i)
	}
	streams := analysisStreams(seq(n), fillF(n, 3.0), fillI(n, 150), alt)
	set := models.OperationSet{
		Enabled:       true,
		OperationID:   op.ID,
		Distance:      f64p(2.4),
		Time:          i64p(800),
		StravaID:      sp("999"),
		StravaStreams: &models.StravaStreamsJSON{StravaActivityStreams: *streams},
	}
	set.ID = uuid.New()
	if err := database.Instance.Omit("Operation").Create(&set).Error; err != nil {
		t.Fatalf("seed set: %v", err)
	}

	// No include → flat activity, no stream summary.
	flat, err := assembleSingleActivity(user.ID, op.ID, nil)
	if err != nil {
		t.Fatalf("assemble (no include): %v", err)
	}
	if !flat.HasStreams {
		t.Fatal("activity should report has_streams")
	}
	if flat.StreamSummary != nil {
		t.Fatalf("no include should attach no summary, got %+v", flat.StreamSummary)
	}

	// include=[segments, analysis] → those blocks present, others absent.
	rich, err := assembleSingleActivity(user.ID, op.ID, []string{"segments", "analysis"})
	if err != nil {
		t.Fatalf("assemble (include): %v", err)
	}
	if rich.StreamSummary == nil {
		t.Fatal("include should attach a stream summary")
	}
	if len(rich.StreamSummary.Segments) == 0 {
		t.Error("segments requested but missing")
	}
	if rich.StreamSummary.Analysis == nil {
		t.Error("analysis requested but missing")
	}
	if len(rich.StreamSummary.HRZones) != 0 || rich.StreamSummary.Route != nil {
		t.Errorf("unrequested blocks leaked: %+v", rich.StreamSummary)
	}
}

func f64p(v float64) *float64 { return &v }
func i64p(v int64) *int64     { return &v }
func sp(v string) *string     { return &v }
