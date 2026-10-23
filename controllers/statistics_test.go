package controllers

import (
	"math"
	"net/http"
	"testing"
	"time"

	"github.com/aunefyren/treningheten/database"
	"github.com/aunefyren/treningheten/files"
	"github.com/aunefyren/treningheten/models"

	"github.com/google/uuid"
	"gorm.io/gorm/clause"
)

// ── Seeding ──────────────────────────────────────────────────────────────────

type seedSet struct {
	reps, weight float64
}

// seedActivity describes one activity in a seeded workout.
type seedActivity struct {
	actionID     uuid.UUID
	kind         string // operation type: moving, timing or lifting
	distance     float64
	distanceUnit string
	seconds      int64
	weightUnit   string
	sets         []seedSet
	avgHR        *int
	maxHR        *int
	cadence      *int
	elevation    *float64
	streams      *models.StravaActivityStreams
}

// seedWorkout inserts a day, a workout on it and its activities, returning the operation ids.
func seedWorkout(t *testing.T, userID uuid.UUID, date time.Time, countsTowardGoal bool, activities ...seedActivity) []uuid.UUID {
	t.Helper()

	day := models.ExerciseDay{Date: time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.Local), Enabled: true, UserID: &userID}
	day.ID = uuid.New()
	if err := database.Instance.Omit(clause.Associations).Create(&day).Error; err != nil {
		t.Fatalf("seed day: %v", err)
	}
	clock := time.Date(date.Year(), date.Month(), date.Day(), 17, 0, 0, 0, time.Local)
	exercise := models.Exercise{Enabled: true, IsOn: true, CountsTowardGoal: true, ExerciseDayID: day.ID, Time: &clock}
	exercise.ID = uuid.New()
	if err := database.Instance.Omit(clause.Associations).Create(&exercise).Error; err != nil {
		t.Fatalf("seed workout: %v", err)
	}
	if !countsTowardGoal {
		database.Instance.Model(&exercise).Update("counts_toward_goal", false)
	}

	ids := []uuid.UUID{}
	for _, activity := range activities {
		actionID := activity.actionID
		unit := activity.distanceUnit
		if unit == "" {
			unit = "km"
		}
		weightUnit := activity.weightUnit
		if weightUnit == "" {
			weightUnit = "kg"
		}
		op := models.Operation{
			Enabled: true, ExerciseID: exercise.ID, ActionID: &actionID, Type: activity.kind,
			DistanceUnit: unit, WeightUnit: weightUnit,
			AvgHeartrate: activity.avgHR, MaxHeartrate: activity.maxHR, AvgCadence: activity.cadence, ElevationGainM: activity.elevation,
		}
		op.ID = uuid.New()
		if err := database.Instance.Omit(clause.Associations).Create(&op).Error; err != nil {
			t.Fatalf("seed activity: %v", err)
		}
		ids = append(ids, op.ID)

		sets := activity.sets
		if len(sets) == 0 {
			sets = []seedSet{{}}
		}
		for i, s := range sets {
			set := models.OperationSet{Enabled: true, OperationID: op.ID}
			set.ID = uuid.New()
			if s.reps > 0 {
				reps := s.reps
				set.Repetitions = &reps
			}
			if s.weight > 0 {
				weight := s.weight
				set.Weight = &weight
			}
			if i == 0 {
				if activity.distance > 0 {
					distance := activity.distance
					set.Distance = &distance
				}
				if activity.seconds > 0 {
					seconds := activity.seconds
					set.Time = &seconds
					set.MovingTime = &seconds
				}
				if activity.streams != nil {
					set.StravaStreams = &models.StravaStreamsJSON{StravaActivityStreams: *activity.streams}
				}
			}
			if err := database.Instance.Omit(clause.Associations).Create(&set).Error; err != nil {
				t.Fatalf("seed set: %v", err)
			}
		}
	}
	return ids
}

func hideWorkoutOf(t *testing.T, operationID uuid.UUID) {
	t.Helper()
	var op models.Operation
	database.Instance.Where("id = ?", operationID).First(&op)
	if err := database.Instance.Model(&models.Exercise{}).Where("id = ?", op.ExerciseID).Update("enabled", false).Error; err != nil {
		t.Fatalf("hide workout: %v", err)
	}
}

func seedLiftingAction(t *testing.T, name string) uuid.UUID {
	t.Helper()
	action := models.Action{Name: name, Type: "lifting", Enabled: true, Category: "strength"}
	action.ID = uuid.New()
	if err := database.Instance.Create(&action).Error; err != nil {
		t.Fatalf("seed action: %v", err)
	}
	return action.ID
}

func actionIDByStravaName(t *testing.T, name string) uuid.UUID {
	t.Helper()
	action, err := database.GetActionByStravaName(name)
	if err != nil || action == nil {
		t.Fatalf("no seeded action for %s: %v", name, err)
	}
	return action.ID
}

func daysAgo(days int) time.Time { return time.Now().AddDate(0, 0, -days) }
func near(a, b float64) bool     { return math.Abs(a-b) < 0.06 }
func asMap(v any) map[string]any { m, _ := v.(map[string]any); return m }
func asList(v any) []any         { l, _ := v.([]any); return l }
func asFloat(v any) float64      { f, _ := v.(float64); return f }

// recordsByKey indexes one activity type's records from a records response.
func recordsByKey(t *testing.T, body map[string]any, actionID uuid.UUID) map[string]map[string]any {
	t.Helper()
	for _, raw := range asList(body["records"]) {
		group := asMap(raw)
		if asMap(group["action"])["id"] != actionID.String() {
			continue
		}
		byKey := map[string]map[string]any{}
		for _, record := range asList(group["records"]) {
			r := asMap(record)
			byKey[r["key"].(string)] = r
		}
		return byKey
	}
	return nil
}

// ── Pure helpers ─────────────────────────────────────────────────────────────

func TestEstimatedOneRepMax(t *testing.T) {
	tests := []struct {
		name         string
		weight, reps float64
		want         float64
	}{
		{"single rep is the weight", 100, 1, 100},
		{"five reps", 80, 5, 80 * (1 + 5.0/30)},
		{"ten reps is the cap", 60, 10, 80},
		{"eleven reps is too many", 60, 11, 0},
		{"no weight", 0, 5, 0},
		{"no reps", 50, 0, 0},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := estimatedOneRepMax(test.weight, test.reps); !near(got, test.want) {
				t.Errorf("got %v, want %v", got, test.want)
			}
		})
	}
}

func TestUnitConversions(t *testing.T) {
	if got := distanceMeters(2, "miles"); !near(got, 3218.688) {
		t.Errorf("2 miles = %v m", got)
	}
	if got := distanceMeters(2, "km"); got != 2000 {
		t.Errorf("2 km = %v m", got)
	}
	if got := weightKg(100, "lb"); !near(got, 45.359) {
		t.Errorf("100 lb = %v kg", got)
	}
	if got := weightKg(100, "kg"); got != 100 {
		t.Errorf("100 kg = %v kg", got)
	}
}

func TestBuildRecord(t *testing.T) {
	now := time.Date(2026, 10, 23, 12, 0, 0, 0, time.Local)
	candidate := func(daysBefore int, score float64) recordCandidate {
		return recordCandidate{entry: models.PersonalRecordEntry{Value: score, SetAt: now.AddDate(0, 0, -daysBefore)}, score: score}
	}

	tests := []struct {
		name         string
		candidates   []recordCandidate
		want         float64
		previous     float64 // 0 means none
		improvements int
		fresh        bool
	}{
		{"single value", []recordCandidate{candidate(5, 10)}, 10, 0, 0, true},
		{"improved twice", []recordCandidate{candidate(90, 10), candidate(60, 8), candidate(40, 12), candidate(10, 15)}, 15, 12, 2, true},
		{"old record stands", []recordCandidate{candidate(60, 20), candidate(5, 19)}, 20, 0, 0, false},
		{"a tie doesn't improve", []recordCandidate{candidate(60, 20), candidate(5, 20)}, 20, 0, 0, false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			record, ok := buildRecord(&recordSeries{key: "k", candidates: test.candidates}, now)
			if !ok {
				t.Fatal("no record")
			}
			if record.Current.Value != test.want {
				t.Errorf("current = %v, want %v", record.Current.Value, test.want)
			}
			if (record.Previous == nil) != (test.previous == 0) || (record.Previous != nil && record.Previous.Value != test.previous) {
				t.Errorf("previous = %+v, want %v", record.Previous, test.previous)
			}
			if record.Improvements != test.improvements {
				t.Errorf("improvements = %d, want %d", record.Improvements, test.improvements)
			}
			if record.Fresh != test.fresh {
				t.Errorf("fresh = %v, want %v", record.Fresh, test.fresh)
			}
		})
	}

	if _, ok := buildRecord(&recordSeries{key: "k"}, now); ok {
		t.Errorf("an empty series made a record")
	}
}

func TestRecordKeyRank(t *testing.T) {
	keys := []string{"effort_1000", "effort_5000", recordLongestDistance, recordHeaviestSet, recordMostVolume, "unknown"}
	for i := 1; i < len(keys); i++ {
		if recordKeyRank(keys[i-1]) >= recordKeyRank(keys[i]) {
			t.Errorf("%s should sort before %s", keys[i-1], keys[i])
		}
	}
}

func TestWeightedAverage(t *testing.T) {
	var w weightedAverage
	if w.result() != nil {
		t.Errorf("empty average should be nil")
	}
	w.add(intPtr(150), 600)
	w.add(intPtr(170), 1800)
	w.add(nil, 1000)
	w.add(intPtr(0), 1000)
	if got := w.result(); got == nil || *got != 165 {
		t.Errorf("weighted = %v, want 165", got)
	}

	var plain weightedAverage
	plain.add(intPtr(100), 0)
	plain.add(intPtr(120), 0)
	if got := plain.result(); got == nil || *got != 110 {
		t.Errorf("plain fallback = %v, want 110", got)
	}
}

func TestOverviewPeriod(t *testing.T) {
	now := time.Date(2026, 10, 23, 12, 0, 0, 0, time.Local) // a Friday
	first := time.Date(2025, 3, 14, 0, 0, 0, 0, time.Local)

	tests := []struct {
		period        string
		start         time.Time
		previousStart time.Time // zero means none
		bucket        string
	}{
		{"4w", time.Date(2026, 9, 28, 0, 0, 0, 0, time.Local), time.Date(2026, 8, 31, 0, 0, 0, 0, time.Local), "week"},
		{"12w", time.Date(2026, 8, 3, 0, 0, 0, 0, time.Local), time.Date(2026, 5, 11, 0, 0, 0, 0, time.Local), "week"},
		{"year", time.Date(2026, 1, 1, 0, 0, 0, 0, time.Local), time.Date(2025, 1, 1, 0, 0, 0, 0, time.Local), "week"},
		{"all", time.Date(2025, 3, 1, 0, 0, 0, 0, time.Local), time.Time{}, "month"},
	}
	for _, test := range tests {
		t.Run(test.period, func(t *testing.T) {
			start, end, previousStart, _, bucket, ok := overviewPeriod(test.period, now, &first)
			if !ok {
				t.Fatal("rejected")
			}
			if !start.Equal(test.start) {
				t.Errorf("start = %v, want %v", start, test.start)
			}
			if !end.Equal(time.Date(2026, 10, 23, 0, 0, 0, 0, time.Local)) {
				t.Errorf("end = %v", end)
			}
			if test.previousStart.IsZero() != (previousStart == nil) || (previousStart != nil && !previousStart.Equal(test.previousStart)) {
				t.Errorf("previous start = %v, want %v", previousStart, test.previousStart)
			}
			if bucket != test.bucket {
				t.Errorf("bucket = %s, want %s", bucket, test.bucket)
			}
		})
	}

	if _, _, _, _, _, ok := overviewPeriod("fortnight", now, nil); ok {
		t.Errorf("unknown period accepted")
	}
	if start, _, _, _, _, _ := overviewPeriod("all", now, nil); !start.Equal(time.Date(2026, 10, 1, 0, 0, 0, 0, time.Local)) {
		t.Errorf("all time without workouts starts %v, want this month", start)
	}
}

// ── Personal records API ─────────────────────────────────────────────────────

func TestPersonalRecordsAPI(t *testing.T) {
	h := newAPIHarness(t)
	user, token := h.user("records@stats.test", false)
	stranger, strangerToken := h.user("stranger@stats.test", false)
	run := actionIDByStravaName(t, "Run")
	bench := seedLiftingAction(t, "Bench press (test)")

	old := seedWorkout(t, user.ID, daysAgo(60), true, seedActivity{actionID: run, kind: "moving", distance: 5, seconds: 1500},
		seedActivity{actionID: bench, kind: "lifting", sets: []seedSet{{5, 80}, {5, 80}}})
	seedWorkout(t, user.ID, daysAgo(40), true, seedActivity{actionID: run, kind: "moving", distance: 8, seconds: 2400})
	recent := seedWorkout(t, user.ID, daysAgo(10), true, seedActivity{actionID: run, kind: "moving", distance: 6, seconds: 1700},
		seedActivity{actionID: bench, kind: "lifting", sets: []seedSet{{3, 85}, {12, 70}}})
	seedWorkout(t, user.ID, daysAgo(5), true, seedActivity{actionID: bench, kind: "lifting", weightUnit: "lb", sets: []seedSet{{1, 200}}})
	hidden := seedWorkout(t, user.ID, daysAgo(3), true, seedActivity{actionID: run, kind: "moving", distance: 20, seconds: 6000})
	hideWorkoutOf(t, hidden[0])
	seedWorkout(t, stranger.ID, daysAgo(2), true, seedActivity{actionID: run, kind: "moving", distance: 42, seconds: 15000})

	if err := database.ReplaceBestEffortsForOperation(old[0], user.ID, []models.BestEffortResult{{DistanceM: 1000, ElapsedSeconds: 300}}); err != nil {
		t.Fatal(err)
	}
	if err := database.ReplaceBestEffortsForOperation(recent[0], user.ID, []models.BestEffortResult{{DistanceM: 1000, ElapsedSeconds: 280}}); err != nil {
		t.Fatal(err)
	}
	if err := database.ReplaceBestEffortsForOperation(hidden[0], user.ID, []models.BestEffortResult{{DistanceM: 1000, ElapsedSeconds: 200}}); err != nil {
		t.Fatal(err)
	}

	body := h.ok("GET", "/api/auth/records", token, nil)
	runs := recordsByKey(t, body, run)
	lifts := recordsByKey(t, body, bench)
	if runs == nil || lifts == nil {
		t.Fatalf("missing groups: %v", body)
	}

	type want struct {
		group    map[string]map[string]any
		key      string
		value    float64
		unit     string
		previous float64 // 0 means none
		fresh    bool
	}
	tests := []want{
		{runs, "effort_1000", 280, "s", 300, true},
		{runs, recordLongestDistance, 8, "km", 5, false},
		{runs, recordLongestDuration, 2400, "s", 1500, false},
		{lifts, recordHeaviestSet, 200, "lb", 85, true},
		{lifts, recordOneRepMax, 93.5, "kg", 93.3, true},
		{lifts, recordMostReps, 12, "reps", 5, true},
		{lifts, recordMostVolume, 1095, "kg", 800, true},
	}
	for _, test := range tests {
		t.Run(test.key, func(t *testing.T) {
			record := test.group[test.key]
			if record == nil {
				t.Fatalf("missing record %s", test.key)
			}
			current := asMap(record["current"])
			if !near(asFloat(current["value"]), test.value) || current["unit"] != test.unit {
				t.Errorf("current = %v %v, want %v %s", current["value"], current["unit"], test.value, test.unit)
			}
			previous := asMap(record["previous"])
			if test.previous == 0 && previous != nil {
				t.Errorf("previous = %v, want none", previous)
			}
			if test.previous != 0 && (previous == nil || !near(asFloat(previous["value"]), test.previous)) {
				t.Errorf("previous = %v, want %v", previous, test.previous)
			}
			if record["fresh"] != test.fresh {
				t.Errorf("fresh = %v, want %v", record["fresh"], test.fresh)
			}
		})
	}
	if runs[recordHeaviestSet] != nil || lifts[recordLongestDistance] != nil {
		t.Errorf("records leaked across activity kinds")
	}

	// One activity type only.
	only := h.ok("GET", "/api/auth/records?action_id="+bench.String(), token, nil)
	if groups := asList(only["records"]); len(groups) != 1 {
		t.Errorf("filtered groups = %d, want 1", len(groups))
	}
	h.expect(http.StatusBadRequest, "GET", "/api/auth/records?action_id=nope", token, nil)

	// The stranger sees only their own marathon.
	theirs := h.ok("GET", "/api/auth/records", strangerToken, nil)
	if longest := recordsByKey(t, theirs, run)[recordLongestDistance]; longest == nil || asFloat(asMap(longest["current"])["value"]) != 42 {
		t.Errorf("stranger's longest = %v, want their own 42 km", longest)
	}

	// History.
	history := h.ok("GET", "/api/auth/records/history?action_id="+run.String()+"&key=effort_1000", token, nil)
	if entries := asList(history["history"]); len(entries) != 2 || asFloat(asMap(entries[1])["value"]) != 280 {
		t.Errorf("history = %v, want 300 then 280", entries)
	}
	h.expect(http.StatusNotFound, "GET", "/api/auth/records/history?action_id="+run.String()+"&key=heaviest_set", token, nil)
	h.expect(http.StatusBadRequest, "GET", "/api/auth/records/history?action_id="+run.String(), token, nil)
	h.expect(http.StatusBadRequest, "GET", "/api/auth/records/history?action_id=nope&key=effort_1000", token, nil)
}

// ── Overview API ─────────────────────────────────────────────────────────────

func TestStatisticsOverviewAPI(t *testing.T) {
	h := newAPIHarness(t)
	user, token := h.user("overview@stats.test", false)
	run := actionIDByStravaName(t, "Run")
	ride := actionIDByStravaName(t, "Ride")
	bench := seedLiftingAction(t, "Squat (test)")

	today := time.Now()
	weekStart := localWeekStart(today)
	// Inside the 4-week period: two workouts this week, one in an earlier week of the period.
	seedWorkout(t, user.ID, today, true, seedActivity{actionID: run, kind: "moving", distance: 10, seconds: 3000})
	seedWorkout(t, user.ID, weekStart, true, seedActivity{actionID: ride, kind: "moving", distance: 5, distanceUnit: "miles", seconds: 1200},
		seedActivity{actionID: bench, kind: "lifting", sets: []seedSet{{5, 100}}})
	seedWorkout(t, user.ID, weekStart.AddDate(0, 0, -14), false, seedActivity{actionID: run, kind: "moving", distance: 4, seconds: 1300})
	// The previous four weeks.
	seedWorkout(t, user.ID, weekStart.AddDate(0, 0, -35), true, seedActivity{actionID: run, kind: "moving", distance: 7, seconds: 2100})
	// A workout with no activities still counts as a workout.
	seedWorkout(t, user.ID, weekStart.AddDate(0, 0, -15), true)

	body := h.ok("GET", "/api/auth/statistics/overview", token, nil)
	overview := asMap(body["overview"])
	totals := asMap(overview["totals"])

	if overview["period"] != "4w" || overview["bucket"] != "week" {
		t.Errorf("period/bucket = %v/%v, want 4w/week", overview["period"], overview["bucket"])
	}
	if asFloat(totals["workouts"]) != 4 {
		t.Errorf("workouts = %v, want 4", totals["workouts"])
	}
	if asFloat(totals["duration_seconds"]) != 3000+1200+1300 {
		t.Errorf("duration = %v", totals["duration_seconds"])
	}
	distances := asList(totals["distances"])
	if len(distances) != 2 || asMap(distances[0])["unit"] != "km" || asFloat(asMap(distances[0])["amount"]) != 14 || asMap(distances[1])["unit"] != "miles" {
		t.Errorf("distances = %v, want 14 km then 5 miles, never mixed", distances)
	}
	if previous := asMap(overview["previous"]); previous == nil || asFloat(previous["workouts"]) != 1 {
		t.Errorf("previous = %v, want 1 workout", overview["previous"])
	}
	if buckets := asList(overview["buckets"]); len(buckets) != 4 || asFloat(asMap(buckets[3])["workouts"]) != 2 {
		t.Errorf("buckets = %v, want 4 weeks with 2 workouts in the last", buckets)
	}
	mix := asList(overview["mix"])
	if len(mix) != 3 || asMap(mix[0])["action_id"] != run.String() {
		t.Errorf("mix = %v, want running first by time", mix)
	}
	streaks := asMap(overview["streaks"])
	if asFloat(streaks["week_current"]) < 1 {
		t.Errorf("week streak = %v, want at least 1", streaks["week_current"])
	}
	if records := asList(overview["recent_records"]); len(records) == 0 {
		t.Errorf("want recent records in the period")
	}

	all := asMap(h.ok("GET", "/api/auth/statistics/overview?period=all", token, nil)["overview"])
	if all["bucket"] != "month" || all["previous"] != nil || asFloat(asMap(all["totals"])["workouts"]) != 5 {
		t.Errorf("all time = %v", all)
	}
	h.ok("GET", "/api/auth/statistics/overview?period=12w", token, nil)
	h.ok("GET", "/api/auth/statistics/overview?period=year", token, nil)
	h.expect(http.StatusBadRequest, "GET", "/api/auth/statistics/overview?period=fortnight", token, nil)
}

// ── Activity type statistics API ─────────────────────────────────────────────

func TestActionStatisticsAPI(t *testing.T) {
	h := newAPIHarness(t)
	user, token := h.user("action@stats.test", false)
	run := actionIDByStravaName(t, "Run")
	bench := seedLiftingAction(t, "Deadlift (test)")

	gps := &models.StravaActivityStreams{LatLng: &models.StravaStream[[]float64]{Data: [][]float64{{59.9, 10.7}, {59.91, 10.7}, {59.92, 10.71}}}}
	seedWorkout(t, user.ID, daysAgo(2), true, seedActivity{actionID: run, kind: "moving", distance: 10, seconds: 3000, avgHR: intPtr(150), maxHR: intPtr(181), cadence: intPtr(85), elevation: floatPtr(40), streams: gps})
	seedWorkout(t, user.ID, daysAgo(4), true, seedActivity{actionID: run, kind: "moving", distance: 5, seconds: 1000, avgHR: intPtr(170), maxHR: intPtr(175), cadence: intPtr(80), elevation: floatPtr(10)})
	seedWorkout(t, user.ID, daysAgo(6), true, seedActivity{actionID: run, kind: "moving", distance: 3, distanceUnit: "miles", seconds: 1500})
	seedWorkout(t, user.ID, daysAgo(3), true, seedActivity{actionID: bench, kind: "lifting", sets: []seedSet{{5, 140}, {3, 150}}})
	seedWorkout(t, user.ID, daysAgo(5), true, seedActivity{actionID: bench, kind: "lifting", weightUnit: "lb", sets: []seedSet{{2, 320}}})
	seedWorkout(t, user.ID, daysAgo(60), true, seedActivity{actionID: run, kind: "moving", distance: 30, seconds: 9000})

	start := daysAgo(30).Format("2006-01-02")
	end := time.Now().Format("2006-01-02")
	runStats := asMap(h.ok("GET", "/api/auth/actions/"+run.String()+"/statistics?start="+start+"&end="+end, token, nil)["statistics"])

	if asFloat(runStats["activities"]) != 3 {
		t.Errorf("activities = %v, want 3 (the old run is outside the period)", runStats["activities"])
	}
	totals := asMap(runStats["totals"])
	distances := asList(totals["distances"])
	if len(distances) != 2 || asFloat(asMap(distances[0])["amount"]) != 15 || asMap(distances[1])["unit"] != "miles" {
		t.Errorf("distances = %v, want 15 km and 3 miles apart", distances)
	}
	if avg := asMap(asMap(runStats["averages"])["distance"]); avg["unit"] != "km" || asFloat(avg["amount"]) != 7.5 {
		t.Errorf("average distance = %v, want 7.5 km over the km runs", avg)
	}
	if longest := asMap(asMap(runStats["bests"])["longest_distance"]); asFloat(longest["value"]) != 10 {
		t.Errorf("longest = %v, want 10 km (3 miles is shorter)", longest)
	}
	sensors := asMap(runStats["sensors"])
	if asFloat(sensors["avg_heartrate"]) != 155 || asFloat(sensors["max_heartrate"]) != 181 {
		t.Errorf("heart rate = %v / %v, want 155 weighted by duration and 181", sensors["avg_heartrate"], sensors["max_heartrate"])
	}
	if sensors["cadence_unit"] != "spm" || asFloat(sensors["avg_cadence"]) < 166 || asFloat(sensors["avg_cadence"]) > 170 {
		t.Errorf("cadence = %v %v, want about 168 spm", sensors["avg_cadence"], sensors["cadence_unit"])
	}
	if asFloat(sensors["elevation_gain_m"]) != 50 {
		t.Errorf("elevation = %v, want 50", sensors["elevation_gain_m"])
	}
	if trend := asList(runStats["trend"]); len(trend) != 3 {
		t.Errorf("trend points = %d, want 3", len(trend))
	}

	liftStats := asMap(h.ok("GET", "/api/auth/actions/"+bench.String()+"/statistics?start="+start+"&end="+end, token, nil)["statistics"])
	heaviest := asMap(asMap(liftStats["bests"])["heaviest_set"])
	if asFloat(heaviest["value"]) != 150 || heaviest["unit"] != "kg" {
		t.Errorf("heaviest = %v, want 150 kg (320 lb is lighter)", heaviest)
	}
	volume := asList(asMap(liftStats["totals"])["volume"])
	if len(volume) != 2 || asFloat(asMap(volume[0])["amount"]) != 1150 {
		t.Errorf("volume = %v, want 1150 kg and 640 lb apart", volume)
	}

	heatmap := h.ok("GET", "/api/auth/actions/"+run.String()+"/heatmap?start="+start+"&end="+end, token, nil)
	if tracks := asList(heatmap["tracks"]); len(tracks) != 1 || len(asList(tracks[0])) != 3 {
		t.Errorf("tracks = %v, want the one GPS run", tracks)
	}

	for _, path := range []string{
		"/api/auth/actions/" + run.String() + "/statistics?start=" + start,
		"/api/auth/actions/" + run.String() + "/heatmap?end=" + end,
		"/api/auth/actions/" + run.String() + "/heatmap?start=soon&end=" + end,
		"/api/auth/actions/nope/heatmap?start=" + start + "&end=" + end,
	} {
		h.expect(http.StatusBadRequest, "GET", path, token, nil)
	}
}

// Changing an activity's type recomputes its best efforts for the new sport's distances.
func TestUpdateOperationRefreshesBestEfforts(t *testing.T) {
	h := newAPIHarness(t)
	user, token := h.user("refresh@stats.test", false)
	run := actionIDByStravaName(t, "Run")

	// 2.5 km north at 5 m/s.
	times := make([]int, 501)
	points := make([][]float64, 501)
	for i := range times {
		times[i] = i
		points[i] = []float64{59.9 + float64(i)*5/111195.0, 10.7}
	}
	streams := &models.StravaActivityStreams{Time: &models.StravaStream[int]{Data: times}, LatLng: &models.StravaStream[[]float64]{Data: points}}
	ids := seedWorkout(t, user.ID, daysAgo(1), true, seedActivity{actionID: run, kind: "moving", distance: 2.5, seconds: 500, streams: streams})

	if err := refreshBestEfforts(ids[0], user.ID); err != nil {
		t.Fatal(err)
	}
	if efforts, _ := database.GetBestEffortRowsForUser(user.ID); len(efforts) != 2 {
		t.Fatalf("run efforts = %d, want 1 km and 1 mile", len(efforts))
	}

	h.ok("PUT", "/api/auth/operations/"+ids[0].String(), token, map[string]any{"action": "Bicycling", "type": "moving", "distance_unit": "km", "weight_unit": "kg"})
	if efforts, _ := database.GetBestEffortRowsForUser(user.ID); len(efforts) != 0 {
		t.Errorf("ride efforts = %d, want none for a 2.5 km ride", len(efforts))
	}
}

// With media on, the soundtrack is gathered once per workout; a period without playback
// leaves it null rather than an empty panel.
func TestActionStatisticsSoundtrackBlock(t *testing.T) {
	h := newAPIHarness(t)
	previous := files.ConfigFile.Media.Enabled
	files.ConfigFile.Media.Enabled = true
	restoreAfterBackgroundWork(t, func() { files.ConfigFile.Media.Enabled = previous })

	user, token := h.user("soundtrack@stats.test", false)
	run := actionIDByStravaName(t, "Run")
	seedWorkout(t, user.ID, daysAgo(1), true,
		seedActivity{actionID: run, kind: "moving", distance: 3, seconds: 900},
		seedActivity{actionID: run, kind: "moving", distance: 2, seconds: 600})

	day := time.Now().Format("2006-01-02")
	stats := asMap(h.ok("GET", "/api/auth/actions/"+run.String()+"/statistics?start="+daysAgo(7).Format("2006-01-02")+"&end="+day, token, nil)["statistics"])
	if asFloat(stats["workouts"]) != 1 || asFloat(stats["activities"]) != 2 {
		t.Errorf("workouts/activities = %v/%v, want 1/2", stats["workouts"], stats["activities"])
	}
	if stats["media"] != nil {
		t.Errorf("media = %v, want null without playback", stats["media"])
	}
}
