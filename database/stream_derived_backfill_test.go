package database

import (
	"testing"
	"time"

	"github.com/aunefyren/treningheten/models"

	"github.com/google/uuid"
)

// seedStreamOperation builds a day → exercise → operation → set-with-streams chain and returns
// the operation id. The operation starts at stream-derived version 0, so the backfill has work.
// actionID may be nil for an activity without a type.
func seedStreamOperation(t *testing.T, userID uuid.UUID, actionID *uuid.UUID, streams models.StravaActivityStreams) uuid.UUID {
	t.Helper()

	day := models.ExerciseDay{UserID: &userID, Enabled: true, Date: time.Now()}
	day.ID = uuid.New()
	insertRow(t, &day)

	exercise := models.Exercise{ExerciseDayID: day.ID, Enabled: true, IsOn: true}
	exercise.ID = uuid.New()
	insertRow(t, &exercise)

	op := models.Operation{ExerciseID: exercise.ID, ActionID: actionID, Enabled: true, Type: "moving", DistanceUnit: "km", WeightUnit: "kg"}
	op.ID = uuid.New()
	insertRow(t, &op)

	set := models.OperationSet{
		OperationID:   op.ID,
		Enabled:       true,
		StravaStreams: &models.StravaStreamsJSON{StravaActivityStreams: streams},
	}
	set.ID = uuid.New()
	insertRow(t, &set)

	return op.ID
}

func loadOperation(t *testing.T, id uuid.UUID) models.Operation {
	t.Helper()
	var op models.Operation
	if err := Instance.Where("id = ?", id).First(&op).Error; err != nil {
		t.Fatalf("load operation: %v", err)
	}
	return op
}

func seedRunAction(t *testing.T) uuid.UUID {
	t.Helper()
	action := models.Action{Name: "Running", StravaName: "Run", Type: "moving", Enabled: true}
	action.ID = uuid.New()
	insertRow(t, &action)
	return action.ID
}

// straightRun is a GPS track heading north at a steady 4 m/s, one sample per second.
func straightRun(seconds int) models.StravaActivityStreams {
	times := make([]int, seconds+1)
	points := make([][]float64, seconds+1)
	// 4 m north ≈ 4 / 111195 degrees of latitude.
	for i := range times {
		times[i] = i
		points[i] = []float64{59.9 + float64(i)*4/111195.0, 10.7}
	}
	return models.StravaActivityStreams{
		Time:   &models.StravaStream[int]{Data: times},
		LatLng: &models.StravaStream[[]float64]{Data: points},
	}
}

func TestBackfillStreamDerivedData(t *testing.T) {
	newTestDB(t)
	user := makeTestUser(t, "backfill@test.dev", nil)
	runID := seedRunAction(t)

	// A rich stream: HR, cadence, temperature, power and a climbing altitude track.
	rich := seedStreamOperation(t, user.ID, nil, models.StravaActivityStreams{
		Heartrate: &models.StravaStream[int]{Data: []int{150, 160, 170}},
		Cadence:   &models.StravaStream[int]{Data: []int{80, 90, 100}},
		Temp:      &models.StravaStream[int]{Data: []int{10, 12, 14}},
		Watts:     &models.StravaStream[int]{Data: []int{200, 0, 250}},
		Altitude:  &models.StravaStream[float64]{Data: []float64{100, 110, 108}}, // +10, -2 → 10 gain
	})

	// A channel-less stream (just GPS, no activity type): nothing to roll up, but the version is
	// still stamped so it can't keep re-triggering the scan.
	bare := seedStreamOperation(t, user.ID, nil, models.StravaActivityStreams{
		LatLng: &models.StravaStream[[]float64]{Data: [][]float64{{59.9, 10.7}, {59.9, 10.71}}},
	})

	// A 6 km run at 4 m/s: best efforts for 1 km, 1 mile and 5 km.
	run := seedStreamOperation(t, user.ID, &runID, straightRun(1500))

	backfillStreamDerivedData()

	richOp := loadOperation(t, rich)
	tests := []struct {
		name string
		got  any
		want any
	}{
		{"avg HR", derefInt(richOp.AvgHeartrate), 160},
		{"max HR", derefInt(richOp.MaxHeartrate), 170},
		{"cadence", derefInt(richOp.AvgCadence), 90},
		{"temperature", derefInt(richOp.TempC), 12},
		{"avg power, zeros included", derefInt(richOp.AvgWatts), 150},
		{"elevation", derefFloat(richOp.ElevationGainM), 10.0},
		{"rich version", richOp.StreamDerivedVersion, models.StreamDerivedVersion},
		{"bare version", loadOperation(t, bare).StreamDerivedVersion, models.StreamDerivedVersion},
		{"bare has no HR", loadOperation(t, bare).AvgHeartrate == nil, true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if test.got != test.want {
				t.Errorf("got %v, want %v", test.got, test.want)
			}
		})
	}

	efforts, err := GetBestEffortRowsForUser(user.ID)
	if err != nil {
		t.Fatalf("GetBestEffortRowsForUser: %v", err)
	}
	byDistance := map[int]int{}
	for _, effort := range efforts {
		if effort.OperationID != run {
			t.Errorf("effort on operation %s, want only the run", effort.OperationID)
		}
		byDistance[effort.DistanceM] = effort.ElapsedSeconds
	}
	wantEfforts := map[int]int{1000: 250, 1609: 402, 5000: 1250}
	for distance, want := range wantEfforts {
		if got := byDistance[distance]; got < want-2 || got > want+2 {
			t.Errorf("effort %d m = %d s, want about %d", distance, got, want)
		}
	}
	if _, found := byDistance[10000]; found {
		t.Errorf("a 6 km run must not have a 10 km effort")
	}

	// Self-limiting: a second run finds nothing to do and leaves the efforts alone.
	backfillStreamDerivedData()
	again, _ := GetBestEffortRowsForUser(user.ID)
	if len(again) != len(efforts) {
		t.Errorf("second run changed the efforts: %d, want %d", len(again), len(efforts))
	}
}

func TestBestEffortRowsSkipHiddenWorkouts(t *testing.T) {
	newTestDB(t)
	user := makeTestUser(t, "hidden@test.dev", nil)
	runID := seedRunAction(t)
	run := seedStreamOperation(t, user.ID, &runID, straightRun(400))
	backfillStreamDerivedData()

	if rows, _ := GetBestEffortRowsForUser(user.ID); len(rows) == 0 {
		t.Fatalf("want efforts before hiding the workout")
	}

	op := loadOperation(t, run)
	if err := Instance.Model(&models.Exercise{}).Where("id = ?", op.ExerciseID).Update("enabled", false).Error; err != nil {
		t.Fatalf("hide workout: %v", err)
	}
	if rows, _ := GetBestEffortRowsForUser(user.ID); len(rows) != 0 {
		t.Errorf("efforts on a hidden workout = %d, want 0", len(rows))
	}
}

func TestReplaceBestEffortsForOperation(t *testing.T) {
	newTestDB(t)
	user := makeTestUser(t, "replace@test.dev", nil)
	runID := seedRunAction(t)
	run := seedStreamOperation(t, user.ID, &runID, straightRun(400))

	first := []models.BestEffortResult{{DistanceM: 1000, ElapsedSeconds: 300}, {DistanceM: 1609, ElapsedSeconds: 500}}
	if err := ReplaceBestEffortsForOperation(run, user.ID, first); err != nil {
		t.Fatalf("first replace: %v", err)
	}
	if err := ReplaceBestEffortsForOperation(run, user.ID, []models.BestEffortResult{{DistanceM: 1000, ElapsedSeconds: 280}}); err != nil {
		t.Fatalf("second replace: %v", err)
	}
	rows, err := GetBestEffortRowsForUser(user.ID)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if len(rows) != 1 || rows[0].ElapsedSeconds != 280 {
		t.Errorf("rows = %+v, want only the replacement", rows)
	}

	streams, sport, err := GetOperationStreamsForUser(run, user.ID)
	if err != nil || streams == nil || sport != "Run" {
		t.Errorf("GetOperationStreamsForUser = %v, %q, %v; want the run's stream", streams != nil, sport, err)
	}
	if streams, _, _ := GetOperationStreamsForUser(run, uuid.New()); streams != nil {
		t.Errorf("another user read the stream")
	}
}

func derefInt(value *int) any {
	if value == nil {
		return nil
	}
	return *value
}

func derefFloat(value *float64) any {
	if value == nil {
		return nil
	}
	return *value
}
