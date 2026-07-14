package database

import (
	"testing"
	"time"

	"github.com/aunefyren/treningheten/models"

	"github.com/google/uuid"
	"gorm.io/gorm/clause"
)

func f64Ptr(v float64) *float64        { return &v }
func durPtr(v time.Duration) *time.Duration { return &v }

// insertRow writes a single row without touching its associations, so seeding the tree by
// scalar foreign keys never upserts a blank parent.
func insertRow(t *testing.T, row interface{}) {
	t.Helper()
	if err := Instance.Omit(clause.Associations).Create(row).Error; err != nil {
		t.Fatalf("failed to insert %T: %v", row, err)
	}
}

// disableRow flips a soft-delete `Enabled` flag to false after insert. It is needed because
// GORM omits a zero-valued bool that carries a `default: true` tag on Create, so a row seeded
// with Enabled:false is actually persisted as enabled. model is a zero-value pointer to the
// GORM struct (e.g. &models.Operation{}) used only to resolve the table.
func disableRow(t *testing.T, model interface{}, id uuid.UUID) {
	t.Helper()
	if err := Instance.Model(model).Where("id = ?", id).Update("enabled", false).Error; err != nil {
		t.Fatalf("failed to disable %T %v: %v", model, id, err)
	}
}

func makeAction(t *testing.T, name, actionType string) models.Action {
	t.Helper()
	action := models.Action{Name: name, Type: actionType, Enabled: true}
	action.ID = uuid.New()
	insertRow(t, &action)
	return action
}

func makeDay(t *testing.T, userID uuid.UUID, date time.Time) models.ExerciseDay {
	t.Helper()
	day := models.ExerciseDay{Date: date, Enabled: true, UserID: &userID}
	day.ID = uuid.New()
	insertRow(t, &day)
	return day
}

func makeSession(t *testing.T, dayID uuid.UUID, at time.Time) models.Exercise {
	t.Helper()
	session := models.Exercise{ExerciseDayID: dayID, Enabled: true, IsOn: true, Time: &at}
	session.ID = uuid.New()
	insertRow(t, &session)
	return session
}

func makeOperation(t *testing.T, exerciseID uuid.UUID, actionID *uuid.UUID) models.Operation {
	t.Helper()
	op := models.Operation{ExerciseID: exerciseID, ActionID: actionID, Enabled: true, WeightUnit: "kg", DistanceUnit: "km"}
	op.ID = uuid.New()
	insertRow(t, &op)
	return op
}

func makeSet(t *testing.T, operationID uuid.UUID, distance, weight, reps *float64, dur *time.Duration) {
	t.Helper()
	set := models.OperationSet{OperationID: operationID, Enabled: true, Distance: distance, Weight: weight, Repetitions: reps, Time: dur}
	set.ID = uuid.New()
	insertRow(t, &set)
}

// seedActivityFeed builds a small but representative tree: two runs on different days, a
// padel activity sharing a session with the second run, and a 3-set lift with no distance.
func seedActivityFeed(t *testing.T) (userID uuid.UUID, runID uuid.UUID, s2ID uuid.UUID) {
	t.Helper()
	user := makeTestUser(t, "feed@test.dev", nil)

	run := makeAction(t, "Run", "cardio")
	padel := makeAction(t, "Padel", "sport")
	lift := makeAction(t, "Lifting", "strength")

	// Day 1 (newer) — a single long run.
	day1 := makeDay(t, user.ID, time.Date(2025, 5, 4, 0, 0, 0, 0, time.UTC))
	s1 := makeSession(t, day1.ID, time.Date(2025, 5, 4, 9, 0, 0, 0, time.UTC))
	run1 := makeOperation(t, s1.ID, &run.ID)
	makeSet(t, run1.ID, f64Ptr(21.1), nil, nil, durPtr(time.Duration(7110)))

	// Day 2 (older) — a session with a run AND a padel match.
	day2 := makeDay(t, user.ID, time.Date(2025, 4, 12, 0, 0, 0, 0, time.UTC))
	s2 := makeSession(t, day2.ID, time.Date(2025, 4, 12, 18, 0, 0, 0, time.UTC))
	run2 := makeOperation(t, s2.ID, &run.ID)
	makeSet(t, run2.ID, f64Ptr(18.0), nil, nil, durPtr(time.Duration(6000)))
	padel1 := makeOperation(t, s2.ID, &padel.ID)
	makeSet(t, padel1.ID, nil, nil, nil, durPtr(time.Duration(5400)))

	// Day 3 — a lift with three sets, no distance.
	day3 := makeDay(t, user.ID, time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC))
	s3 := makeSession(t, day3.ID, time.Date(2025, 6, 1, 7, 0, 0, 0, time.UTC))
	lift1 := makeOperation(t, s3.ID, &lift.ID)
	makeSet(t, lift1.ID, nil, f64Ptr(60), f64Ptr(10), nil)
	makeSet(t, lift1.ID, nil, f64Ptr(80), f64Ptr(8), nil)
	makeSet(t, lift1.ID, nil, f64Ptr(70), f64Ptr(10), nil)

	return user.ID, run.ID, s2.ID
}

func TestActivityFeedSortAndFilter(t *testing.T) {
	newTestDB(t)
	userID, runActionID, s2ID := seedActivityFeed(t)

	// "My longest run": filter to Run, sort by distance desc.
	items, total, err := GetActivityFeedForUser(userID, models.ActivityFeedFilter{
		ActionID: &runActionID, Sort: "distance", Order: "desc", Limit: 30,
	})
	if err != nil {
		t.Fatalf("feed error: %v", err)
	}
	if total != 2 {
		t.Fatalf("expected 2 runs, got total %d", total)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}
	if items[0].Distance != 21.1 || items[1].Distance != 18.0 {
		t.Errorf("distance order wrong: %v then %v", items[0].Distance, items[1].Distance)
	}
	if items[0].DurationSeconds != 7110 {
		t.Errorf("duration aggregate wrong: %d", items[0].DurationSeconds)
	}
	if items[0].ActionName != "Run" {
		t.Errorf("action name: %q", items[0].ActionName)
	}

	// The 18 km run shares a session with a padel match → session activity count 2.
	if items[1].ExerciseID != s2ID {
		t.Errorf("second run should be in session s2")
	}
	if items[1].SessionActivityCount != 2 {
		t.Errorf("session activity count should be 2, got %d", items[1].SessionActivityCount)
	}
}

func TestActivityFeedDefaultDateOrderAndAggregates(t *testing.T) {
	newTestDB(t)
	userID, _, _ := seedActivityFeed(t)

	// Full feed, default date-desc: 4 activities (2 runs, 1 padel, 1 lift), newest first.
	items, total, err := GetActivityFeedForUser(userID, models.ActivityFeedFilter{
		Sort: "date", Order: "desc", Limit: 30,
	})
	if err != nil {
		t.Fatalf("feed error: %v", err)
	}
	if total != 4 || len(items) != 4 {
		t.Fatalf("expected 4 activities, got total %d / len %d", total, len(items))
	}
	// Newest day is 2025-06-01 (the lift).
	if items[0].ActionName != "Lifting" {
		t.Errorf("newest should be the lift, got %q", items[0].ActionName)
	}
	// Lift aggregates: top weight 80, reps 28, 3 sets, no distance.
	if items[0].TopWeight != 80 {
		t.Errorf("top weight: %v", items[0].TopWeight)
	}
	if items[0].Repetitions != 28 {
		t.Errorf("reps sum: %v", items[0].Repetitions)
	}
	if items[0].SetCount != 3 {
		t.Errorf("set count: %d", items[0].SetCount)
	}
	if items[0].Distance != 0 {
		t.Errorf("lift should have no distance, got %v", items[0].Distance)
	}
}

// makeSessionOff inserts a toggled-off (is_on = false) session — the app treats these as
// deleted/not-counted, so the feed must exclude them. Toggling is done with an explicit
// column update (mirroring the app's SetExerciseOff) because GORM omits the zero-value
// false on Create and lets the `default:true` win.
func makeSessionOff(t *testing.T, dayID uuid.UUID, at time.Time) models.Exercise {
	t.Helper()
	session := makeSession(t, dayID, at)
	if err := Instance.Model(&session).Update("is_on", 0).Error; err != nil {
		t.Fatalf("failed to toggle session off: %v", err)
	}
	session.IsOn = false
	return session
}

func TestActivityFeedExcludesOffSessions(t *testing.T) {
	newTestDB(t)
	user := makeTestUser(t, "off@test.dev", nil)
	run := makeAction(t, "Run", "cardio")
	day := makeDay(t, user.ID, time.Date(2025, 5, 4, 0, 0, 0, 0, time.UTC))

	onSession := makeSession(t, day.ID, time.Date(2025, 5, 4, 9, 0, 0, 0, time.UTC))
	onOp := makeOperation(t, onSession.ID, &run.ID)
	makeSet(t, onOp.ID, f64Ptr(10), nil, nil, durPtr(time.Duration(3000)))

	offSession := makeSessionOff(t, day.ID, time.Date(2025, 5, 4, 18, 0, 0, 0, time.UTC))
	offOp := makeOperation(t, offSession.ID, &run.ID)
	makeSet(t, offOp.ID, f64Ptr(99), nil, nil, durPtr(time.Duration(9000)))

	items, total, err := GetActivityFeedForUser(user.ID, models.ActivityFeedFilter{Sort: "date", Order: "desc", Limit: 30})
	if err != nil {
		t.Fatalf("feed error: %v", err)
	}
	if total != 1 || len(items) != 1 {
		t.Fatalf("off session must be excluded: got total %d / len %d", total, len(items))
	}
	if items[0].Distance != 10 {
		t.Errorf("returned the wrong (off) activity: distance %v", items[0].Distance)
	}
}

func TestActivityFeedSearchIsCaseInsensitiveAcrossNotes(t *testing.T) {
	newTestDB(t)
	user := makeTestUser(t, "search@test.dev", nil)
	run := makeAction(t, "Run", "cardio")

	// Note lives on the OPERATION, capitalised.
	day1 := makeDay(t, user.ID, time.Date(2025, 5, 4, 0, 0, 0, 0, time.UTC))
	s1 := makeSession(t, day1.ID, time.Date(2025, 5, 4, 9, 0, 0, 0, time.UTC))
	op1 := models.Operation{ExerciseID: s1.ID, ActionID: &run.ID, Enabled: true, WeightUnit: "kg", DistanceUnit: "km", Note: strPtr("Ran with Cosmo")}
	op1.ID = uuid.New()
	insertRow(t, &op1)
	makeSet(t, op1.ID, f64Ptr(8), nil, nil, nil)

	// Note lives on the DAY, different activity with no operation note.
	day2 := makeDay(t, user.ID, time.Date(2025, 4, 1, 0, 0, 0, 0, time.UTC))
	day2.Note = "cosmo park loop"
	if err := Instance.Save(&day2).Error; err != nil {
		t.Fatalf("failed to set day note: %v", err)
	}
	s2 := makeSession(t, day2.ID, time.Date(2025, 4, 1, 9, 0, 0, 0, time.UTC))
	op2 := makeOperation(t, s2.ID, &run.ID)
	makeSet(t, op2.ID, f64Ptr(5), nil, nil, nil)

	// Lowercase query must match both the capitalised operation note and the day note.
	items, total, err := GetActivityFeedForUser(user.ID, models.ActivityFeedFilter{Query: "cosmo", Sort: "date", Order: "desc", Limit: 30})
	if err != nil {
		t.Fatalf("feed error: %v", err)
	}
	if total != 2 || len(items) != 2 {
		t.Fatalf("case-insensitive search across notes should find 2, got total %d / len %d", total, len(items))
	}
}

func TestActivityFeedHasDistanceAndPagination(t *testing.T) {
	newTestDB(t)
	userID, _, _ := seedActivityFeed(t)

	// has_distance excludes the padel (no distance) and the lift → only the 2 runs.
	items, total, err := GetActivityFeedForUser(userID, models.ActivityFeedFilter{
		HasDistance: true, Sort: "date", Order: "desc", Limit: 30,
	})
	if err != nil {
		t.Fatalf("feed error: %v", err)
	}
	if total != 2 || len(items) != 2 {
		t.Fatalf("has_distance should leave 2 runs, got total %d / len %d", total, len(items))
	}

	// Pagination: limit 1 over the full feed reports has-more via total.
	page, total, err := GetActivityFeedForUser(userID, models.ActivityFeedFilter{
		Sort: "date", Order: "desc", Limit: 1, Offset: 0,
	})
	if err != nil {
		t.Fatalf("feed error: %v", err)
	}
	if len(page) != 1 {
		t.Fatalf("expected 1 item page, got %d", len(page))
	}
	if total != 4 {
		t.Errorf("total should still be 4 regardless of limit, got %d", total)
	}
}
