package database

import (
	"testing"

	"github.com/aunefyren/treningheten/models"

	"github.com/google/uuid"
)

// seedStreamOperation builds a day → exercise → operation → set-with-streams chain and returns
// the operation id, so a test can seed a stream and then inspect the rollups written to it. The
// operation's rollup columns are left NULL (nothing sets them here) so the backfill has work.
func seedStreamOperation(t *testing.T, userID uuid.UUID, streams models.StravaActivityStreams) uuid.UUID {
	t.Helper()

	day := models.ExerciseDay{UserID: &userID, Enabled: true}
	day.ID = uuid.New()
	insertRow(t, &day)

	exercise := models.Exercise{ExerciseDayID: day.ID, Enabled: true, IsOn: true}
	exercise.ID = uuid.New()
	insertRow(t, &exercise)

	op := models.Operation{ExerciseID: exercise.ID, Enabled: true, Type: "moving", DistanceUnit: "km", WeightUnit: "kg"}
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

func TestBackfillOperationStreamRollups(t *testing.T) {
	newTestDB(t)
	user := makeTestUser(t, "backfill@test.dev", nil)

	// A rich stream: HR, cadence, temperature and a climbing altitude track.
	rich := seedStreamOperation(t, user.ID, models.StravaActivityStreams{
		Heartrate: &models.StravaStream[int]{Data: []int{150, 160, 170}},
		Cadence:   &models.StravaStream[int]{Data: []int{80, 90, 100}},
		Temp:      &models.StravaStream[int]{Data: []int{10, 12, 14}},
		Altitude:  &models.StravaStream[float64]{Data: []float64{100, 110, 108}}, // +10, -2 → 10 gain
	})

	// A channel-less stream (just GPS): the rollup is empty, so the backfill must still mark it
	// with elevation_gain_m = 0 so it can't keep re-triggering the scan.
	bare := seedStreamOperation(t, user.ID, models.StravaActivityStreams{
		LatLng: &models.StravaStream[[]float64]{Data: [][]float64{{59.9, 10.7}, {59.9, 10.71}}},
	})

	backfillOperationStreamRollups()

	richOp := loadOperation(t, rich)
	if richOp.AvgHeartrate == nil || *richOp.AvgHeartrate != 160 {
		t.Errorf("rich avg HR: want 160, got %v", richOp.AvgHeartrate)
	}
	if richOp.MaxHeartrate == nil || *richOp.MaxHeartrate != 170 {
		t.Errorf("rich max HR: want 170, got %v", richOp.MaxHeartrate)
	}
	if richOp.AvgCadence == nil || *richOp.AvgCadence != 90 {
		t.Errorf("rich cadence: want 90, got %v", richOp.AvgCadence)
	}
	if richOp.TempC == nil || *richOp.TempC != 12 {
		t.Errorf("rich temp: want 12, got %v", richOp.TempC)
	}
	if richOp.ElevationGainM == nil || *richOp.ElevationGainM != 10 {
		t.Errorf("rich elevation: want 10, got %v", richOp.ElevationGainM)
	}

	bareOp := loadOperation(t, bare)
	if bareOp.AvgHeartrate != nil || bareOp.AvgCadence != nil {
		t.Errorf("bare stream should have no HR/cadence, got %+v", bareOp)
	}
	if bareOp.ElevationGainM == nil || *bareOp.ElevationGainM != 0 {
		t.Errorf("bare stream should be marked with elevation 0, got %v", bareOp.ElevationGainM)
	}

	// Idempotent: with every stream-backed operation now carrying a non-NULL rollup, a second
	// run finds nothing to do and leaves the values unchanged.
	backfillOperationStreamRollups()
	if again := loadOperation(t, rich); again.AvgHeartrate == nil || *again.AvgHeartrate != 160 {
		t.Errorf("second run changed rich avg HR: %v", again.AvgHeartrate)
	}
}
