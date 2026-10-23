package database

import (
	"testing"
	"time"

	"github.com/aunefyren/treningheten/models"

	"github.com/google/uuid"
)

// statsWorkout seeds a workout on a day at local midnight, with one activity of the given type
// and sets, and returns the workout and operation ids.
func statsWorkout(t *testing.T, userID uuid.UUID, date time.Time, actionID *uuid.UUID, opType string, sets ...models.OperationSet) (uuid.UUID, uuid.UUID) {
	t.Helper()
	day := makeDay(t, userID, time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.Local))
	exercise := models.Exercise{ExerciseDayID: day.ID, Enabled: true, IsOn: true}
	exercise.ID = uuid.New()
	insertRow(t, &exercise)
	if opType == "" {
		return exercise.ID, uuid.Nil
	}
	op := models.Operation{ExerciseID: exercise.ID, ActionID: actionID, Enabled: true, Type: opType, DistanceUnit: "km", WeightUnit: "kg"}
	op.ID = uuid.New()
	insertRow(t, &op)
	for _, set := range sets {
		set.ID = uuid.New()
		set.OperationID = op.ID
		set.Enabled = true
		insertRow(t, &set)
	}
	return exercise.ID, op.ID
}

func f64(v float64) *float64 { return &v }
func i64(v int64) *int64     { return &v }

func TestStatisticsQueries(t *testing.T) {
	newTestDB(t)
	user := makeTestUser(t, "statsq@test.dev", nil)
	other := makeTestUser(t, "statsq-other@test.dev", nil)
	run := makeAction(t, "Run (stats)", "moving")
	press := makeAction(t, "Press (stats)", "lifting")

	monday := time.Date(2026, 10, 19, 0, 0, 0, 0, time.Local)
	statsWorkout(t, user.ID, monday, &run.ID, "moving", models.OperationSet{Distance: f64(5), Time: i64(1500)}, models.OperationSet{Distance: f64(1), Time: i64(300)})
	statsWorkout(t, user.ID, monday.AddDate(0, 0, 2), &press.ID, "lifting", models.OperationSet{Repetitions: f64(5), Weight: f64(60)}, models.OperationSet{Repetitions: f64(3), Weight: f64(70)}, models.OperationSet{})
	statsWorkout(t, user.ID, monday.AddDate(0, 0, 3), nil, "")                                                    // a "+" workout without activities
	statsWorkout(t, user.ID, monday.AddDate(0, 0, -10), &run.ID, "moving", models.OperationSet{Distance: f64(8)}) // outside the range below
	hiddenWorkout, _ := statsWorkout(t, user.ID, monday.AddDate(0, 0, 1), &run.ID, "moving", models.OperationSet{Distance: f64(30)})
	disableRow(t, &models.Exercise{}, hiddenWorkout)
	statsWorkout(t, other.ID, monday, &run.ID, "moving", models.OperationSet{Distance: f64(42)})

	start, end := monday, monday.AddDate(0, 0, 6)
	week := StatisticsFilter{Start: &start, End: &end}

	t.Run("sessions", func(t *testing.T) {
		rows, err := GetStatisticsSessionRows(user.ID, week)
		if err != nil {
			t.Fatal(err)
		}
		if len(rows) != 3 {
			t.Errorf("sessions = %d, want 3 (Monday's midnight day included, hidden and old ones out)", len(rows))
		}
		if rows[0].DurationSeconds != 1800 {
			t.Errorf("first session duration = %d, want the summed set times 1800", rows[0].DurationSeconds)
		}
		all, _ := GetStatisticsSessionRows(user.ID, StatisticsFilter{})
		if len(all) != 4 {
			t.Errorf("all-time sessions = %d, want 4", len(all))
		}
		runs, _ := GetStatisticsSessionRows(user.ID, StatisticsFilter{ActionID: &run.ID})
		if len(runs) != 2 {
			t.Errorf("sessions with a run = %d, want 2", len(runs))
		}
	})

	t.Run("activities", func(t *testing.T) {
		rows, err := GetStatisticsActivityRows(user.ID, week)
		if err != nil {
			t.Fatal(err)
		}
		if len(rows) != 2 {
			t.Fatalf("activities = %d, want 2", len(rows))
		}
		if rows[0].Distance != 6 || rows[0].SetSeconds != 1800 || rows[0].ActionName != "Run (stats)" {
			t.Errorf("run row = %+v, want 6 km over 1800 s", rows[0])
		}
		if rows[1].TopWeight != 70 || rows[1].Repetitions != 8 || rows[1].SetCount != 3 {
			t.Errorf("press row = %+v, want top 70, 8 reps, 3 sets", rows[1])
		}
		presses, _ := GetStatisticsActivityRows(user.ID, StatisticsFilter{ActionID: &press.ID})
		if len(presses) != 1 {
			t.Errorf("press activities = %d, want 1", len(presses))
		}
	})

	t.Run("sets", func(t *testing.T) {
		rows, err := GetStatisticsSetRows(user.ID, StatisticsFilter{ActionID: &press.ID})
		if err != nil {
			t.Fatal(err)
		}
		if len(rows) != 2 {
			t.Errorf("sets = %d, want the two with weight or reps", len(rows))
		}
		for _, row := range rows {
			if row.Type != "lifting" || row.WeightUnit != "kg" {
				t.Errorf("set row = %+v", row)
			}
		}
	})
}

func TestGetActionTracksForUser(t *testing.T) {
	newTestDB(t)
	user := makeTestUser(t, "tracks@test.dev", nil)
	run := makeAction(t, "Run (tracks)", "moving")
	day := time.Date(2026, 10, 20, 0, 0, 0, 0, time.Local)

	gps := &models.StravaStreamsJSON{StravaActivityStreams: models.StravaActivityStreams{LatLng: &models.StravaStream[[]float64]{Data: [][]float64{{59.9, 10.7}, {59.91}, {59.92, 10.72}}}}}
	statsWorkout(t, user.ID, day, &run.ID, "moving", models.OperationSet{StravaStreams: gps})
	// A treadmill run: a stream, but no GPS.
	speed := &models.StravaStreamsJSON{StravaActivityStreams: models.StravaActivityStreams{VelocitySmooth: &models.StravaStream[float64]{Data: []float64{3, 3}}}}
	statsWorkout(t, user.ID, day, &run.ID, "moving", models.OperationSet{StravaStreams: speed})

	tracks, err := GetActionTracksForUser(user.ID, StatisticsFilter{Start: &day, End: &day, ActionID: &run.ID})
	if err != nil {
		t.Fatal(err)
	}
	if len(tracks) != 1 || len(tracks[0]) != 2 {
		t.Errorf("tracks = %v, want one track with the two complete points", tracks)
	}
	if others, _ := GetActionTracksForUser(uuid.New(), StatisticsFilter{ActionID: &run.ID}); len(others) != 0 {
		t.Errorf("another user got %d tracks", len(others))
	}
}
