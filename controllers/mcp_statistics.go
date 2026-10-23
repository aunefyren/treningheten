package controllers

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/aunefyren/treningheten/database"
	"github.com/aunefyren/treningheten/models"

	"github.com/google/uuid"
)

// MCP views of /statistics: the windowed summary (get_statistics, optionally with the
// overview), personal records, one activity type's statistics and an activity's best efforts.
// All of them reuse the page's own computations (statistics.go, records.go), so the MCP
// answers and the app can't disagree. See docs/mcp.md.

// mcpTrendLimit caps the trend points of get_activity_type_statistics; the newest are kept.
var mcpTrendLimit = 400

// mcpDefaultStatisticsDays is the default range of get_activity_type_statistics.
const mcpDefaultStatisticsDays = 365

// resolveMCPActivityType finds the activity type an MCP caller means among the types the user
// has logged: an id, an exact name (any case), or a name fragment that matches exactly one.
// Errors name the candidates so the model can retry with a valid one.
func resolveMCPActivityType(userID uuid.UUID, text string) (models.Action, error) {
	text = strings.TrimSpace(text)
	actions, err := database.GetActionsDoneUsingUserID(userID)
	if err != nil {
		return models.Action{}, err
	}

	if id, err := uuid.Parse(text); err == nil {
		for _, action := range actions {
			if action.ID == id {
				return action, nil
			}
		}
		return models.Action{}, fmt.Errorf("no activity type with id %s among the ones you have logged", text)
	}

	lower := strings.ToLower(text)
	matches := []models.Action{}
	for _, action := range actions {
		name := strings.ToLower(action.Name)
		if name == lower {
			return action, nil
		}
		if strings.Contains(name, lower) {
			matches = append(matches, action)
		}
	}
	if len(matches) == 1 {
		return matches[0], nil
	}

	names := func(list []models.Action) string {
		out := []string{}
		for _, action := range list {
			out = append(out, action.Name)
		}
		sort.Strings(out)
		return strings.Join(out, ", ")
	}
	if len(matches) > 1 {
		return models.Action{}, fmt.Errorf("%q matches several activity types: %s; pass one of these names or an action_id", text, names(matches))
	}
	return models.Action{}, fmt.Errorf("no activity type you have logged matches %q; you have logged: %s", text, names(actions))
}

// ── get_statistics ───────────────────────────────────────────────────────────

// assembleUserStatistics reports the three rolling windows and the personal streaks from the
// statistics rows /statistics uses, plus that page's overview when a period is given.
func assembleUserStatistics(userID uuid.UUID, period string, now time.Time) (models.MCPStatistics, error) {
	period = strings.TrimSpace(period)
	if period != "" {
		if _, _, _, _, _, ok := overviewPeriod(period, now, nil); !ok {
			return models.MCPStatistics{}, fmt.Errorf("unknown period %q; use 4w, 12w, year or all", period)
		}
	}

	sessions, err := database.GetStatisticsSessionRows(userID, database.StatisticsFilter{})
	if err != nil {
		return models.MCPStatistics{}, err
	}
	activities, err := database.GetStatisticsActivityRows(userID, database.StatisticsFilter{})
	if err != nil {
		return models.MCPStatistics{}, err
	}

	monthAgo := now.AddDate(0, -1, 0)
	yearAgo := now.AddDate(-1, 0, 0)
	stats := models.MCPStatistics{}
	windows := func(date time.Time, apply func(*models.MCPStatWindow)) {
		apply(&stats.AllTime)
		if date.After(yearAgo) {
			apply(&stats.PastYear)
		}
		if date.After(monthAgo) {
			apply(&stats.PastMonth)
		}
	}

	activeDates := []time.Time{}
	for _, session := range sessions {
		windows(session.Date, func(w *models.MCPStatWindow) { w.Workouts++ })
		if session.CountsTowardGoal {
			activeDates = append(activeDates, session.Date)
		}
	}
	for _, row := range activities {
		distanceKm := distanceToKm(row.Distance, row.DistanceUnit)
		seconds := activityDurationSeconds(row)
		windows(row.Date, func(w *models.MCPStatWindow) {
			w.Activities++
			w.DistanceKm += distanceKm
			w.TimeSeconds += seconds
		})
	}
	// Tidy float accumulation noise from the km conversions.
	stats.AllTime.DistanceKm = round2(stats.AllTime.DistanceKm)
	stats.PastYear.DistanceKm = round2(stats.PastYear.DistanceKm)
	stats.PastMonth.DistanceKm = round2(stats.PastMonth.DistanceKm)

	streaks := computePersonalStreaksFromDates(activeDates)
	stats.Streaks.Personal = models.MCPPersonalStreaks{
		WeekCurrent: streaks.WeekCurrent,
		WeekBest:    streaks.WeekBest,
		DayCurrent:  streaks.DayCurrent,
		DayBest:     streaks.DayBest,
	}

	if period != "" {
		groups, err := computePersonalRecords(userID, nil, now)
		if err != nil {
			return models.MCPStatistics{}, err
		}
		overview, _ := buildStatisticsOverview(period, now, sessions, activities, groups)
		stats.Overview = mcpOverview(overview)
	}

	return stats, nil
}

func mcpOverview(overview models.StatisticsOverview) *models.MCPOverview {
	out := &models.MCPOverview{
		Period:        overview.Period,
		Start:         overview.Start,
		End:           overview.End,
		Bucket:        overview.Bucket,
		Totals:        overview.Totals,
		Previous:      overview.Previous,
		Buckets:       overview.Buckets,
		Streaks:       overview.Streaks,
		Mix:           []models.MCPMixItem{},
		RecentRecords: []models.MCPRecentRecord{},
	}
	for _, item := range overview.Mix {
		out.Mix = append(out.Mix, models.MCPMixItem{ActionID: item.ActionID.String(), Action: item.Name, Activities: item.Activities, DurationSeconds: item.DurationSeconds})
	}
	for _, recent := range overview.RecentRecords {
		out.RecentRecords = append(out.RecentRecords, models.MCPRecentRecord{ActionID: recent.ActionID.String(), Action: recent.ActionName, Record: mcpPersonalRecord(recent.Record)})
	}
	return out
}

// ── get_personal_records ─────────────────────────────────────────────────────

// mcpPersonalRecord converts a record, adding pace or speed for a best effort.
func mcpPersonalRecord(record models.PersonalRecord) models.MCPPersonalRecord {
	current := mcpRecordEntry(record.Current)
	item := models.MCPPersonalRecord{
		Key:          record.Key,
		Label:        record.Label,
		Kind:         record.Kind,
		Value:        current.Value,
		Unit:         current.Unit,
		Reps:         current.Reps,
		SetAt:        current.SetAt,
		ActivityID:   current.ActivityID,
		Fresh:        record.Fresh,
		Improvements: record.Improvements,
	}
	item.PaceMinKm, item.SpeedKmh = effortPaceOrSpeed(record.Sport, effortKeyDistance(record.Key), record.Current.Value)
	if record.Previous != nil {
		previous := mcpRecordEntry(*record.Previous)
		item.Previous = &previous
	}
	return item
}

func mcpRecordEntry(entry models.PersonalRecordEntry) models.MCPRecordEntry {
	return models.MCPRecordEntry{Value: entry.Value, Unit: entry.Unit, Reps: entry.Reps, SetAt: entry.SetAt, ActivityID: entry.OperationID.String()}
}

// effortPaceOrSpeed expresses a best effort as pace (foot sports) or speed (everything else).
func effortPaceOrSpeed(stravaName string, distanceM int, seconds float64) (pace float64, speed float64) {
	if distanceM <= 0 || seconds <= 0 {
		return 0, 0
	}
	if stepCadenceSports[stravaName] {
		return round2(seconds / 60 / (float64(distanceM) / 1000)), 0
	}
	return 0, round1(float64(distanceM) / seconds * 3.6)
}

func effortKeyDistance(key string) int {
	if !strings.HasPrefix(key, "effort_") {
		return 0
	}
	meters, _ := strconv.Atoi(strings.TrimPrefix(key, "effort_"))
	return meters
}

// assemblePersonalRecords returns the user's records, optionally for one activity type; with a
// key (which needs a type) that record also carries its full history.
func assemblePersonalRecords(userID uuid.UUID, actionText string, key string, now time.Time) (models.MCPPersonalRecords, error) {
	key = strings.TrimSpace(key)
	var actionID *uuid.UUID
	if strings.TrimSpace(actionText) != "" {
		action, err := resolveMCPActivityType(userID, actionText)
		if err != nil {
			return models.MCPPersonalRecords{}, err
		}
		actionID = &action.ID
	} else if key != "" {
		return models.MCPPersonalRecords{}, fmt.Errorf("key needs an action: records are per activity type")
	}

	groups, err := computePersonalRecords(userID, actionID, now)
	if err != nil {
		return models.MCPPersonalRecords{}, err
	}

	var series map[uuid.UUID]map[string]*recordSeries
	if key != "" {
		if series, _, err = loadRecordSeries(userID, actionID); err != nil {
			return models.MCPPersonalRecords{}, err
		}
		if series[*actionID][key] == nil {
			return models.MCPPersonalRecords{}, fmt.Errorf("no %q record for this activity type; call without key to see which records exist", key)
		}
	}

	out := models.MCPPersonalRecords{ActivityTypes: []models.MCPRecordGroup{}}
	for _, group := range groups {
		mcpGroup := models.MCPRecordGroup{ActionID: group.Action.ID.String(), Action: group.Action.Name, Activities: group.Activities, Records: []models.MCPPersonalRecord{}}
		for _, record := range group.Records {
			item := mcpPersonalRecord(record)
			if key != "" && record.Key == key {
				item.History = []models.MCPRecordEntry{}
				for _, candidate := range recordProgression(series[group.Action.ID][key].candidates) {
					item.History = append(item.History, mcpRecordEntry(candidate.entry))
				}
			}
			mcpGroup.Records = append(mcpGroup.Records, item)
		}
		out.ActivityTypes = append(out.ActivityTypes, mcpGroup)
	}
	return out, nil
}

// ── get_activity_type_statistics ─────────────────────────────────────────────

func mcpBest(best *models.StatisticsBest) *models.MCPRecordEntry {
	if best == nil {
		return nil
	}
	return &models.MCPRecordEntry{Value: best.Value, Unit: best.Unit, Reps: best.Reps, SetAt: best.Date, ActivityID: best.OperationID.String()}
}

// assembleActivityTypeStatistics is the Activity types tab of /statistics for one type and
// range, with pace and speed worked out per trend point (models are poor at that arithmetic).
func assembleActivityTypeStatistics(userID uuid.UUID, actionText string, from string, to string, now time.Time) (models.MCPActivityTypeStatistics, error) {
	if strings.TrimSpace(actionText) == "" {
		return models.MCPActivityTypeStatistics{}, fmt.Errorf("action is required: an activity type name or id")
	}
	action, err := resolveMCPActivityType(userID, actionText)
	if err != nil {
		return models.MCPActivityTypeStatistics{}, err
	}

	end := localDay(now)
	start := end.AddDate(0, 0, -mcpDefaultStatisticsDays)
	if strings.TrimSpace(to) != "" {
		if end, err = parseStatisticsDate(to); err != nil {
			return models.MCPActivityTypeStatistics{}, fmt.Errorf("invalid to %q: use YYYY-MM-DD", to)
		}
	}
	if strings.TrimSpace(from) != "" {
		if start, err = parseStatisticsDate(from); err != nil {
			return models.MCPActivityTypeStatistics{}, fmt.Errorf("invalid from %q: use YYYY-MM-DD", from)
		}
	}
	if start.After(end) {
		return models.MCPActivityTypeStatistics{}, fmt.Errorf("from is after to")
	}

	filter := database.StatisticsFilter{Start: &start, End: &end, ActionID: &action.ID}
	activities, err := database.GetStatisticsActivityRows(userID, filter)
	if err != nil {
		return models.MCPActivityTypeStatistics{}, err
	}
	sets, err := database.GetStatisticsSetRows(userID, filter)
	if err != nil {
		return models.MCPActivityTypeStatistics{}, err
	}
	stats := buildActionStatistics(action, activities, sets)

	out := models.MCPActivityTypeStatistics{
		ActionID:   action.ID.String(),
		Action:     action.Name,
		From:       start.Format("2006-01-02"),
		To:         end.Format("2006-01-02"),
		Activities: stats.Activities,
		Workouts:   stats.Workouts,
		Totals:     stats.Totals,
		Averages:   stats.Averages,
		Sensors:    stats.Sensors,
		Bests: models.MCPActivityTypeBests{
			LongestDistance: mcpBest(stats.Bests.LongestDistance),
			LongestDuration: mcpBest(stats.Bests.LongestDuration),
			MostReps:        mcpBest(stats.Bests.MostReps),
			HeaviestSet:     mcpBest(stats.Bests.HeaviestSet),
		},
		Trend: []models.MCPTrendPoint{},
	}

	trend := stats.Trend
	if len(trend) > mcpTrendLimit {
		trend = trend[len(trend)-mcpTrendLimit:]
		out.TrendTruncated = true
	}
	for _, point := range trend {
		item := models.MCPTrendPoint{
			ActivityID:         point.OperationID.String(),
			Date:               point.Date,
			Distance:           point.Distance,
			DurationSeconds:    point.DurationSeconds,
			MovingSeconds:      point.MovingSeconds,
			AvgHeartrateBpm:    point.AvgHeartrate,
			Repetitions:        point.Repetitions,
			TopWeight:          point.TopWeight,
			EstimatedOneRepMax: point.EstimatedOneRepMax,
		}
		if point.Distance > 0 {
			item.DistanceUnit = point.DistanceUnit
			seconds := point.MovingSeconds
			if seconds <= 0 {
				seconds = point.DurationSeconds
			}
			if km := distanceToKm(point.Distance, point.DistanceUnit); km > 0 && seconds > 0 {
				item.PaceMinKm = round2(float64(seconds) / 60 / km)
				item.SpeedKmh = round1(km / float64(seconds) * 3600)
			}
		}
		if point.TopWeight > 0 {
			item.WeightUnit = point.WeightUnit
		}
		out.Trend = append(out.Trend, item)
	}
	return out, nil
}

// ── get_activity include "efforts" ───────────────────────────────────────────

// assembleActivityEfforts lists an activity's best efforts and flags each that is the user's
// current record for its distance.
func assembleActivityEfforts(userID uuid.UUID, operationID uuid.UUID, action *models.Action, now time.Time) ([]models.MCPBestEffort, error) {
	if action == nil {
		return []models.MCPBestEffort{}, nil
	}
	efforts, err := database.GetBestEffortsForOperation(operationID, userID)
	if err != nil {
		return nil, err
	}

	recordHolders := map[string]uuid.UUID{}
	if len(efforts) > 0 {
		groups, err := computePersonalRecords(userID, &action.ID, now)
		if err != nil {
			return nil, err
		}
		for _, group := range groups {
			for _, record := range group.Records {
				recordHolders[record.Key] = record.Current.OperationID
			}
		}
	}

	out := []models.MCPBestEffort{}
	for _, effort := range efforts {
		label := models.BestEffortLabel(action.StravaName, effort.DistanceM)
		if label == "" {
			continue
		}
		item := models.MCPBestEffort{
			Label:            label,
			DistanceM:        effort.DistanceM,
			ElapsedSeconds:   effort.ElapsedSeconds,
			IsPersonalRecord: recordHolders["effort_"+strconv.Itoa(effort.DistanceM)] == operationID,
		}
		item.PaceMinKm, item.SpeedKmh = effortPaceOrSpeed(action.StravaName, effort.DistanceM, float64(effort.ElapsedSeconds))
		out = append(out, item)
	}
	return out, nil
}
