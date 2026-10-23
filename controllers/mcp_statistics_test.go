package controllers

import (
	"strings"
	"testing"
	"time"

	"github.com/aunefyren/treningheten/database"
	"github.com/aunefyren/treningheten/files"
	"github.com/aunefyren/treningheten/models"

	"github.com/google/uuid"
)

// mcpStatsWorld seeds a user with runs, rides and bench presses for the MCP statistics tests.
type mcpStatsWorld struct {
	h            *apiHarness
	user         models.User
	token        string
	run, ride    uuid.UUID
	bench        uuid.UUID
	recentRun    uuid.UUID
	oldRun       uuid.UUID
	recentRide   uuid.UUID
	recentBench  uuid.UUID
	hiddenRunDay uuid.UUID
}

func newMCPStatsWorld(t *testing.T) mcpStatsWorld {
	t.Helper()
	h := newAPIHarness(t)
	user, token := h.user("mcpstats@stats.test", false)
	w := mcpStatsWorld{h: h, user: user, token: token,
		run: actionIDByStravaName(t, "Run"), ride: actionIDByStravaName(t, "Ride"), bench: seedLiftingAction(t, "Bench Press (mcp)")}

	w.oldRun = seedWorkout(t, user.ID, daysAgo(200), true, seedActivity{actionID: w.run, kind: "moving", distance: 5, seconds: 1500})[0]
	w.recentRun = seedWorkout(t, user.ID, daysAgo(10), true, seedActivity{actionID: w.run, kind: "moving", distance: 10, seconds: 3000, avgHR: intPtr(150)})[0]
	w.recentRide = seedWorkout(t, user.ID, daysAgo(5), false, seedActivity{actionID: w.ride, kind: "moving", distance: 10, distanceUnit: "miles", seconds: 1800})[0]
	w.recentBench = seedWorkout(t, user.ID, daysAgo(3), true, seedActivity{actionID: w.bench, kind: "lifting", sets: []seedSet{{5, 80}, {3, 90}}})[0]
	seedWorkout(t, user.ID, daysAgo(2), true) // a workout without activities
	hidden := seedWorkout(t, user.ID, daysAgo(1), true, seedActivity{actionID: w.run, kind: "moving", distance: 42, seconds: 12000})
	hideWorkoutOf(t, hidden[0])

	mustEfforts := func(operationID uuid.UUID, efforts ...models.BestEffortResult) {
		if err := database.ReplaceBestEffortsForOperation(operationID, user.ID, efforts); err != nil {
			t.Fatal(err)
		}
	}
	mustEfforts(w.oldRun, models.BestEffortResult{DistanceM: 1000, ElapsedSeconds: 300}, models.BestEffortResult{DistanceM: 5000, ElapsedSeconds: 1500})
	mustEfforts(w.recentRun, models.BestEffortResult{DistanceM: 1000, ElapsedSeconds: 270}, models.BestEffortResult{DistanceM: 5000, ElapsedSeconds: 1520})
	mustEfforts(w.recentRide, models.BestEffortResult{DistanceM: 10000, ElapsedSeconds: 1200})
	return w
}

func TestResolveMCPActivityType(t *testing.T) {
	w := newMCPStatsWorld(t)
	tests := []struct {
		name    string
		text    string
		want    uuid.UUID
		wantErr string
	}{
		{"exact name, any case", "rUn", w.run, ""},
		{"by id", w.bench.String(), w.bench, ""},
		{"unique fragment", "bicyc", w.ride, ""},
		{"fragment matching several", "n", uuid.Nil, "matches several"},
		{"nothing logged by that name", "Yoga", uuid.Nil, "you have logged:"},
		{"an id never logged", uuid.NewString(), uuid.Nil, "no activity type with id"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			action, err := resolveMCPActivityType(w.user.ID, test.text)
			if test.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), test.wantErr) {
					t.Errorf("err = %v, want it to mention %q", err, test.wantErr)
				}
				return
			}
			if err != nil || action.ID != test.want {
				t.Errorf("got %v, %v; want %v", action.Name, err, test.want)
			}
		})
	}
}

func TestAssembleUserStatisticsMCP(t *testing.T) {
	w := newMCPStatsWorld(t)
	now := time.Now()

	stats, err := assembleUserStatistics(w.user.ID, "", now)
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name      string
		got, want any
	}{
		{"all-time workouts include the one without activities", stats.AllTime.Workouts, 5},
		{"all-time activities leave the hidden marathon out", stats.AllTime.Activities, 4},
		{"past month workouts", stats.PastMonth.Workouts, 4},
		{"past month km, miles converted", stats.PastMonth.DistanceKm, round2(10 + 10*1.609344)},
		{"past year time, the 200-day-old run included", stats.PastYear.TimeSeconds, int64(1500 + 3000 + 1800)},
		{"no overview without a period", stats.Overview == nil, true},
		{"a best week streak", stats.Streaks.Personal.WeekBest >= 1, true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if test.got != test.want {
				t.Errorf("got %v, want %v", test.got, test.want)
			}
		})
	}

	withOverview, err := assembleUserStatistics(w.user.ID, "12w", now)
	if err != nil || withOverview.Overview == nil || withOverview.Overview.Period != "12w" {
		t.Fatalf("overview = %+v, err = %v", withOverview.Overview, err)
	}
	if len(withOverview.Overview.RecentRecords) == 0 {
		t.Errorf("want records set in the period")
	}
	if _, err := assembleUserStatistics(w.user.ID, "fortnight", now); err == nil {
		t.Errorf("an unknown period was accepted")
	}
}

func TestAssemblePersonalRecordsMCP(t *testing.T) {
	w := newMCPStatsWorld(t)
	now := time.Now()

	all, err := assemblePersonalRecords(w.user.ID, "", "", now)
	if err != nil {
		t.Fatal(err)
	}
	if len(all.ActivityTypes) != 3 {
		t.Fatalf("activity types = %d, want 3", len(all.ActivityTypes))
	}

	runs, err := assemblePersonalRecords(w.user.ID, "run", "effort_1000", now)
	if err != nil {
		t.Fatal(err)
	}
	if len(runs.ActivityTypes) != 1 {
		t.Fatalf("groups = %d, want only runs", len(runs.ActivityTypes))
	}
	records := map[string]models.MCPPersonalRecord{}
	for _, record := range runs.ActivityTypes[0].Records {
		records[record.Key] = record
	}

	km := records["effort_1000"]
	if km.Value != 270 || km.ActivityID != w.recentRun.String() || km.Previous == nil || km.Previous.Value != 300 {
		t.Errorf("1 km = %+v, want 270 s in the recent run, beating 300", km)
	}
	if km.PaceMinKm != 4.5 || km.SpeedKmh != 0 {
		t.Errorf("1 km pace/speed = %v/%v, want 4.5 min/km and no speed", km.PaceMinKm, km.SpeedKmh)
	}
	if len(km.History) != 2 {
		t.Errorf("history = %v, want 300 then 270", km.History)
	}
	if five := records["effort_5000"]; five.Value != 1500 || five.Previous != nil || five.History != nil {
		t.Errorf("5 km = %+v, want the old 1500 s standing, no history without its key", five)
	}
	if longest := records["longest_distance"]; longest.Value != 10 {
		t.Errorf("longest = %v, want 10 (the hidden marathon is out)", longest.Value)
	}

	rides, _ := assemblePersonalRecords(w.user.ID, "bicycling", "", now)
	for _, record := range rides.ActivityTypes[0].Records {
		if record.Key == "effort_10000" && (record.SpeedKmh != 30 || record.PaceMinKm != 0) {
			t.Errorf("ride 10 km = %+v, want 30 km/h", record)
		}
	}

	for name, args := range map[string][2]string{
		"key without action": {"", "effort_1000"},
		"unknown key":        {"run", "heaviest_set"},
		"unknown type":       {"Yoga", ""},
	} {
		if _, err := assemblePersonalRecords(w.user.ID, args[0], args[1], now); err == nil {
			t.Errorf("%s: no error", name)
		}
	}
}

func TestAssembleActivityTypeStatisticsMCP(t *testing.T) {
	w := newMCPStatsWorld(t)
	now := time.Now()

	runs, err := assembleActivityTypeStatistics(w.user.ID, "Run", "", "", now)
	if err != nil {
		t.Fatal(err)
	}
	if runs.Activities != 2 || len(runs.Trend) != 2 {
		t.Errorf("activities/trend = %d/%d, want both runs in the default year", runs.Activities, len(runs.Trend))
	}
	latest := runs.Trend[len(runs.Trend)-1]
	if latest.ActivityID != w.recentRun.String() || latest.PaceMinKm != 5 || latest.SpeedKmh != 12 {
		t.Errorf("latest point = %+v, want the recent run at 5:00/km and 12 km/h", latest)
	}
	if runs.Bests.LongestDistance == nil || runs.Bests.LongestDistance.ActivityID != w.recentRun.String() {
		t.Errorf("longest = %+v", runs.Bests.LongestDistance)
	}

	narrow, err := assembleActivityTypeStatistics(w.user.ID, w.run.String(), daysAgo(30).Format("2006-01-02"), time.Now().Format("2006-01-02"), now)
	if err != nil || narrow.Activities != 1 {
		t.Errorf("last 30 days = %d activities, err %v; want 1", narrow.Activities, err)
	}

	bench, err := assembleActivityTypeStatistics(w.user.ID, "bench", "", "", now)
	if err != nil || bench.Bests.HeaviestSet == nil || bench.Bests.HeaviestSet.Value != 90 || bench.Trend[0].EstimatedOneRepMax == 0 {
		t.Errorf("bench = %+v, err %v", bench, err)
	}

	previous := mcpTrendLimit
	mcpTrendLimit = 1
	t.Cleanup(func() { mcpTrendLimit = previous })
	capped, _ := assembleActivityTypeStatistics(w.user.ID, "Run", "", "", now)
	if !capped.TrendTruncated || len(capped.Trend) != 1 || capped.Trend[0].ActivityID != w.recentRun.String() {
		t.Errorf("capped trend = %+v, want only the newest point", capped.Trend)
	}

	for name, args := range map[string][3]string{
		"no action":     {"", "", ""},
		"bad from":      {"Run", "soon", ""},
		"bad to":        {"Run", "", "later"},
		"from after to": {"Run", "2026-02-01", "2026-01-01"},
	} {
		if _, err := assembleActivityTypeStatistics(w.user.ID, args[0], args[1], args[2], now); err == nil {
			t.Errorf("%s: no error", name)
		}
	}
}

func TestAssembleActivityEffortsMCP(t *testing.T) {
	w := newMCPStatsWorld(t)
	run, _ := database.GetActionByID(w.run)

	efforts, err := assembleActivityEfforts(w.user.ID, w.recentRun, &run, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	byLabel := map[string]models.MCPBestEffort{}
	for _, effort := range efforts {
		byLabel[effort.Label] = effort
	}
	if !byLabel["1 km"].IsPersonalRecord || byLabel["5 km"].IsPersonalRecord {
		t.Errorf("efforts = %+v, want the 1 km as a record and the 5 km not", efforts)
	}
	if byLabel["1 km"].PaceMinKm != 4.5 {
		t.Errorf("1 km pace = %v, want 4.5", byLabel["1 km"].PaceMinKm)
	}
	if none, _ := assembleActivityEfforts(w.user.ID, w.recentRun, nil, time.Now()); len(none) != 0 {
		t.Errorf("no activity type should give no efforts")
	}
	if theirs, _ := assembleActivityEfforts(uuid.New(), w.recentRun, &run, time.Now()); len(theirs) != 0 {
		t.Errorf("another user read the efforts")
	}
}

// The tools end to end through the MCP transport.
func TestMCPStatisticsToolsEndToEnd(t *testing.T) {
	w := newMCPStatsWorld(t)
	previous := files.ConfigFile.MCPEnabled
	enabled := true
	files.ConfigFile.MCPEnabled = &enabled
	t.Cleanup(func() { files.ConfigFile.MCPEnabled = previous })
	database.Instance.Model(&models.Operation{}).Where("id = ?", w.recentRide).Update("avg_watts", 180)

	session := mcpSession(t, w.token)

	stats := callTool(t, session, "get_statistics", map[string]any{"period": "4w"}, false)
	if stats["overview"] == nil || asMap(stats["all_time"])["workouts"] != float64(5) {
		t.Errorf("get_statistics = %v", stats)
	}
	callTool(t, session, "get_statistics", map[string]any{"period": "fortnight"}, true)

	records := callTool(t, session, "get_personal_records", map[string]any{"action": "Run", "key": "effort_1000"}, false)
	if groups := asList(records["activity_types"]); len(groups) != 1 {
		t.Errorf("get_personal_records = %v", records)
	}
	callTool(t, session, "get_personal_records", map[string]any{"key": "effort_1000"}, true)

	typeStats := callTool(t, session, "get_activity_type_statistics", map[string]any{"action": "Run"}, false)
	if len(asList(typeStats["trend"])) != 2 {
		t.Errorf("get_activity_type_statistics = %v", typeStats)
	}
	callTool(t, session, "get_activity_type_statistics", map[string]any{"action": "Yoga"}, true)

	activity := callTool(t, session, "get_activity", map[string]any{"activity_id": w.recentRun.String(), "include": []string{"efforts"}}, false)
	if efforts := asList(asMap(activity["activity"])["best_efforts"]); len(efforts) != 2 {
		t.Errorf("best_efforts = %v, want 1 km and 5 km", efforts)
	}

	workouts := callTool(t, session, "list_workouts", map[string]any{"action": "bicycling"}, false)
	ride := asMap(asList(asMap(asList(workouts["workouts"])[0])["activities"])[0])
	if ride["avg_watts"] != float64(180) {
		t.Errorf("ride avg_watts = %v, want 180", ride["avg_watts"])
	}
}
