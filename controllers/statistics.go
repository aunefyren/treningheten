package controllers

import (
	"errors"
	"net/http"
	"sort"
	"time"

	"github.com/aunefyren/treningheten/database"
	"github.com/aunefyren/treningheten/files"
	"github.com/aunefyren/treningheten/logger"
	"github.com/aunefyren/treningheten/middlewares"
	"github.com/aunefyren/treningheten/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// The /statistics endpoints: the overview of a period, the statistics of one activity type
// and its GPS heatmap. Personal records live in records.go. See docs/statistics.md.

// statisticsRecentRecords caps the records listed on the overview.
const statisticsRecentRecords = 6

// localDay is midnight of t's calendar day in the server zone.
func localDay(t time.Time) time.Time {
	l := t.In(time.Local)
	return time.Date(l.Year(), l.Month(), l.Day(), 0, 0, 0, 0, time.Local)
}

// localWeekStart is midnight on the Monday of t's week, in the server zone.
func localWeekStart(t time.Time) time.Time {
	d := localDay(t)
	return d.AddDate(0, 0, -((int(d.Weekday()) + 6) % 7))
}

// localMonthStart is midnight on the first of t's month, in the server zone.
func localMonthStart(t time.Time) time.Time {
	d := localDay(t)
	return time.Date(d.Year(), d.Month(), 1, 0, 0, 0, 0, time.Local)
}

// parseStatisticsDate reads a day bound: a plain date (YYYY-MM-DD, in the server zone) or a
// full timestamp, which older clients send.
func parseStatisticsDate(value string) (time.Time, error) {
	if parsed, err := time.ParseInLocation("2006-01-02", value, time.Local); err == nil {
		return parsed, nil
	}
	if parsed, err := time.Parse(time.RFC3339, value); err == nil {
		return parsed, nil
	}
	return time.Time{}, errors.New("unparseable date")
}

// statisticsRange reads the required start and end query values.
func statisticsRange(context *gin.Context) (time.Time, time.Time, string) {
	startString, okay := context.GetQuery("start")
	if !okay {
		return time.Time{}, time.Time{}, "Start query is missing."
	}
	start, err := parseStatisticsDate(startString)
	if err != nil {
		return time.Time{}, time.Time{}, "Failed to parse start time string."
	}
	endString, okay := context.GetQuery("end")
	if !okay {
		return time.Time{}, time.Time{}, "End query is missing."
	}
	end, err := parseStatisticsDate(endString)
	if err != nil {
		return time.Time{}, time.Time{}, "Failed to parse end time string."
	}
	return start, end, ""
}

// addUnitAmount adds an amount to the entry for its unit.
func addUnitAmount(amounts []models.UnitAmount, unit string, amount float64) []models.UnitAmount {
	if amount == 0 {
		return amounts
	}
	for i := range amounts {
		if amounts[i].Unit == unit {
			amounts[i].Amount += amount
			return amounts
		}
	}
	return append(amounts, models.UnitAmount{Unit: unit, Amount: amount})
}

// roundUnitAmounts rounds each amount to two decimals and orders the biggest first, so the
// first entry is the period's main unit.
func roundUnitAmounts(amounts []models.UnitAmount) []models.UnitAmount {
	for i := range amounts {
		amounts[i].Amount = round2(amounts[i].Amount)
	}
	sort.SliceStable(amounts, func(i, j int) bool { return amounts[i].Amount > amounts[j].Amount })
	return amounts
}

// weightedAverage averages rollup values weighted by duration, falling back to a plain mean
// when no activity carries a duration.
type weightedAverage struct {
	sum, weight float64
	plainSum    float64
	count       int
}

func (w *weightedAverage) add(value *int, seconds int64) {
	if value == nil || *value <= 0 {
		return
	}
	w.sum += float64(*value) * float64(seconds)
	w.weight += float64(seconds)
	w.plainSum += float64(*value)
	w.count++
}

func (w *weightedAverage) result() *int {
	if w.count == 0 {
		return nil
	}
	var value float64
	if w.weight > 0 {
		value = w.sum / w.weight
	} else {
		value = w.plainSum / float64(w.count)
	}
	rounded := int(value + 0.5)
	return &rounded
}

// ── Activity type statistics ─────────────────────────────────────────────────

// buildActionStatistics folds one activity type's rows for a period into the statistics:
// totals, per-activity averages, bests, the sensor rollups and one trend point per activity.
func buildActionStatistics(action models.Action, activities []models.StatisticsActivityRow, sets []models.StatisticsSetRow) models.ActionStatistics {
	stats := models.ActionStatistics{
		Action: action,
		Totals: models.ActionStatisticsTotals{Distances: []models.UnitAmount{}, Volume: []models.UnitAmount{}},
		Trend:  []models.ActionTrendPoint{},
	}

	workouts := map[uuid.UUID]bool{}
	distanceCounts := map[string]int{}
	var durationSum, durationCount int64
	var repsSum float64
	var repsCount int
	var heartrate, cadence, watts weightedAverage
	var maxHeartrate *int
	var elevation float64
	hasElevation := false

	// Per-activity strength figures from the sets: volume and the best one-rep max estimate.
	type strength struct {
		volume, oneRepMax float64
	}
	byOperation := map[uuid.UUID]*strength{}
	for _, set := range sets {
		s := byOperation[set.OperationID]
		if s == nil {
			s = &strength{}
			byOperation[set.OperationID] = s
		}
		if set.Weight > 0 && set.Repetitions > 0 {
			s.volume += set.Weight * set.Repetitions
			stats.Totals.Volume = addUnitAmount(stats.Totals.Volume, set.WeightUnit, set.Weight*set.Repetitions)
		}
		if estimate := estimatedOneRepMax(set.Weight, set.Repetitions); estimate > s.oneRepMax {
			s.oneRepMax = estimate
		}

		if set.Repetitions > 0 && (stats.Bests.MostReps == nil || set.Repetitions > stats.Bests.MostReps.Value) {
			stats.Bests.MostReps = &models.StatisticsBest{Value: set.Repetitions, Unit: "reps", Date: models.RecordedAt(set.Time, set.Date), ExerciseDayID: set.ExerciseDayID}
		}
		if set.Weight > 0 {
			better := stats.Bests.HeaviestSet == nil
			if !better {
				current := weightKg(stats.Bests.HeaviestSet.Value, stats.Bests.HeaviestSet.Unit)
				candidate := weightKg(set.Weight, set.WeightUnit)
				better = candidate > current || (candidate == current && set.Repetitions > *stats.Bests.HeaviestSet.Reps)
			}
			if better {
				reps := set.Repetitions
				stats.Bests.HeaviestSet = &models.StatisticsBest{Value: set.Weight, Unit: set.WeightUnit, Reps: &reps, Date: models.RecordedAt(set.Time, set.Date), ExerciseDayID: set.ExerciseDayID}
			}
		}
	}

	for _, row := range activities {
		stats.Activities++
		workouts[row.ExerciseID] = true
		seconds := activityDurationSeconds(row)
		date := models.RecordedAt(row.Time, row.Date)

		if row.Distance > 0 {
			stats.Totals.Distances = addUnitAmount(stats.Totals.Distances, row.DistanceUnit, row.Distance)
			distanceCounts[row.DistanceUnit]++
			if stats.Bests.LongestDistance == nil || distanceMeters(row.Distance, row.DistanceUnit) > distanceMeters(stats.Bests.LongestDistance.Value, stats.Bests.LongestDistance.Unit) {
				stats.Bests.LongestDistance = &models.StatisticsBest{Value: round2(row.Distance), Unit: row.DistanceUnit, Date: date, ExerciseDayID: row.ExerciseDayID}
			}
		}
		if seconds > 0 {
			stats.Totals.DurationSeconds += seconds
			durationSum += seconds
			durationCount++
			if stats.Bests.LongestDuration == nil || float64(seconds) > stats.Bests.LongestDuration.Value {
				stats.Bests.LongestDuration = &models.StatisticsBest{Value: float64(seconds), Unit: "s", Date: date, ExerciseDayID: row.ExerciseDayID}
			}
		}
		stats.Totals.MovingSeconds += row.MovingSeconds
		if row.Repetitions > 0 {
			stats.Totals.Repetitions += row.Repetitions
			repsSum += row.Repetitions
			repsCount++
		}

		weight := seconds
		if row.MovingSeconds > 0 {
			weight = row.MovingSeconds
		}
		heartrate.add(row.AvgHeartrate, weight)
		cadence.add(row.AvgCadence, weight)
		watts.add(row.AvgWatts, weight)
		if row.MaxHeartrate != nil && (maxHeartrate == nil || *row.MaxHeartrate > *maxHeartrate) {
			peak := *row.MaxHeartrate
			maxHeartrate = &peak
		}
		if row.ElevationGainM != nil && *row.ElevationGainM > 0 {
			elevation += *row.ElevationGainM
			hasElevation = true
		}

		point := models.ActionTrendPoint{
			Date:            date,
			ExerciseDayID:   row.ExerciseDayID,
			Distance:        round2(row.Distance),
			DistanceUnit:    row.DistanceUnit,
			DurationSeconds: seconds,
			MovingSeconds:   row.MovingSeconds,
			AvgHeartrate:    row.AvgHeartrate,
			Repetitions:     row.Repetitions,
			TopWeight:       row.TopWeight,
			WeightUnit:      row.WeightUnit,
		}
		if s := byOperation[row.OperationID]; s != nil {
			point.EstimatedOneRepMax = round1(s.oneRepMax)
		}
		stats.Trend = append(stats.Trend, point)
	}

	stats.Workouts = len(workouts)
	stats.Totals.Distances = roundUnitAmounts(stats.Totals.Distances)
	stats.Totals.Volume = roundUnitAmounts(stats.Totals.Volume)

	// Averages per activity, in the period's main unit only.
	if len(stats.Totals.Distances) > 0 {
		main := stats.Totals.Distances[0]
		stats.Averages.Distance = &models.UnitAmount{Unit: main.Unit, Amount: round2(main.Amount / float64(distanceCounts[main.Unit]))}
	}
	if durationCount > 0 {
		average := durationSum / durationCount
		stats.Averages.DurationSeconds = &average
	}
	if repsCount > 0 {
		average := round1(repsSum / float64(repsCount))
		stats.Averages.Repetitions = &average
	}
	if len(stats.Totals.Volume) > 0 {
		main := stats.Totals.Volume[0]
		count := 0
		for _, s := range byOperation {
			if s.volume > 0 {
				count++
			}
		}
		if count > 0 {
			stats.Averages.Volume = &models.UnitAmount{Unit: main.Unit, Amount: round1(main.Amount / float64(count))}
		}
	}

	stats.Sensors.AvgHeartrate = heartrate.result()
	stats.Sensors.MaxHeartrate = maxHeartrate
	stats.Sensors.AvgWatts = watts.result()
	if avg := cadence.result(); avg != nil {
		unit := cadenceUnitForAction(&action)
		stats.Sensors.AvgCadence = scaleCadence(avg, unit)
		stats.Sensors.CadenceUnit = unit
	}
	if hasElevation {
		total := round1(elevation)
		stats.Sensors.ElevationGainM = &total
	}

	return stats
}

// APIGetActionStatistics returns the caller's statistics for one activity type over a period
// (?start= and ?end=, inclusive days).
func APIGetActionStatistics(context *gin.Context) {
	actionID, err := uuid.Parse(context.Param("action_id"))
	if err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse ID."})
		context.Abort()
		return
	}

	action, err := database.GetActionByID(actionID)
	if err != nil {
		logger.Log.Error("Failed to get action by ID. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to get action by ID."})
		context.Abort()
		return
	}

	userID, err := middlewares.GetAuthUsername(context.GetHeader("Authorization"))
	if err != nil {
		logger.Log.Error("Failed to verify user ID. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to verify user ID."})
		context.Abort()
		return
	}

	start, end, problem := statisticsRange(context)
	if problem != "" {
		context.JSON(http.StatusBadRequest, gin.H{"error": problem})
		context.Abort()
		return
	}

	filter := database.StatisticsFilter{Start: &start, End: &end, ActionID: &actionID}
	activities, err := database.GetStatisticsActivityRows(userID, filter)
	if err != nil {
		logger.Log.Error("Failed to get activities for statistics. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get activities."})
		context.Abort()
		return
	}
	sets, err := database.GetStatisticsSetRows(userID, filter)
	if err != nil {
		logger.Log.Error("Failed to get sets for statistics. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get sets."})
		context.Abort()
		return
	}

	stats := buildActionStatistics(action, activities, sets)

	// The soundtrack is a workout-level fact, so it aggregates over the distinct workouts the
	// matched activities belong to, each counted once.
	if files.ConfigFile.Media.Enabled {
		seen := map[uuid.UUID]bool{}
		playback := []models.MediaPlaybackObject{}
		for _, row := range activities {
			if seen[row.ExerciseID] {
				continue
			}
			seen[row.ExerciseID] = true
			rows, err := database.GetMediaPlaybackForExercise(row.ExerciseID)
			if err != nil {
				logger.Log.Warn("Failed to get media playback for session. Error: " + err.Error())
				continue
			}
			playback = append(playback, ConvertMediaPlaybackToObjects(rows)...)
		}
		stats.Media = computeActionMediaStatistics(playback)
	}

	context.JSON(http.StatusOK, gin.H{"message": "Action statistics retrieved.", "statistics": stats})
}

// APIGetActionHeatmap returns the GPS tracks of one activity type over a period, for the
// heatmap. Kept apart from the statistics so those never carry raw streams.
func APIGetActionHeatmap(context *gin.Context) {
	actionID, err := uuid.Parse(context.Param("action_id"))
	if err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse ID."})
		context.Abort()
		return
	}

	userID, err := middlewares.GetAuthUsername(context.GetHeader("Authorization"))
	if err != nil {
		logger.Log.Error("Failed to verify user ID. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to verify user ID."})
		context.Abort()
		return
	}

	start, end, problem := statisticsRange(context)
	if problem != "" {
		context.JSON(http.StatusBadRequest, gin.H{"error": problem})
		context.Abort()
		return
	}

	tracks, err := database.GetActionTracksForUser(userID, database.StatisticsFilter{Start: &start, End: &end, ActionID: &actionID})
	if err != nil {
		logger.Log.Error("Failed to get tracks for heatmap. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get tracks."})
		context.Abort()
		return
	}

	context.JSON(http.StatusOK, gin.H{"message": "Heatmap retrieved.", "tracks": tracks})
}

// ── Overview ─────────────────────────────────────────────────────────────────

// overviewPeriod resolves a period name to its range, the previous range of the same length
// (nil for all time) and the bucket size. "all" starts at the first workout.
func overviewPeriod(period string, now time.Time, firstWorkout *time.Time) (start time.Time, end time.Time, previousStart *time.Time, previousEnd *time.Time, bucket string, ok bool) {
	today := localDay(now)
	end = today
	bucket = "week"
	switch period {
	case "4w", "12w":
		weeks := 4
		if period == "12w" {
			weeks = 12
		}
		start = localWeekStart(today).AddDate(0, 0, -7*(weeks-1))
		ps := start.AddDate(0, 0, -7*weeks)
		pe := start.AddDate(0, 0, -1)
		previousStart, previousEnd = &ps, &pe
	case "year":
		start = time.Date(today.Year(), 1, 1, 0, 0, 0, 0, time.Local)
		ps := start.AddDate(-1, 0, 0)
		pe := today.AddDate(-1, 0, 0)
		previousStart, previousEnd = &ps, &pe
	case "all":
		bucket = "month"
		start = localMonthStart(today)
		if firstWorkout != nil {
			start = localMonthStart(*firstWorkout)
		}
	default:
		return time.Time{}, time.Time{}, nil, nil, "", false
	}
	return start, end, previousStart, previousEnd, bucket, true
}

func inDays(date time.Time, start time.Time, end time.Time) bool {
	day := localDay(date)
	return !day.Before(start) && !day.After(end)
}

// overviewTotals sums the sessions and activities inside a range.
func overviewTotals(sessions []models.StatisticsSessionRow, activities []models.StatisticsActivityRow, start time.Time, end time.Time) models.StatisticsTotals {
	totals := models.StatisticsTotals{Distances: []models.UnitAmount{}}
	days := map[time.Time]bool{}
	for _, session := range sessions {
		if !inDays(session.Date, start, end) {
			continue
		}
		totals.Workouts++
		totals.DurationSeconds += session.DurationSeconds
		days[localDay(session.Date)] = true
	}
	totals.ActiveDays = len(days)
	for _, row := range activities {
		if inDays(row.Date, start, end) && row.Distance > 0 {
			totals.Distances = addUnitAmount(totals.Distances, row.DistanceUnit, row.Distance)
		}
	}
	totals.Distances = roundUnitAmounts(totals.Distances)
	return totals
}

// buildStatisticsOverview assembles the overview of one period from the user's rows.
func buildStatisticsOverview(period string, now time.Time, sessions []models.StatisticsSessionRow, activities []models.StatisticsActivityRow, groups []models.PersonalRecordGroup) (models.StatisticsOverview, bool) {
	var firstWorkout *time.Time
	if len(sessions) > 0 {
		first := sessions[0].Date
		for _, session := range sessions {
			if session.Date.Before(first) {
				first = session.Date
			}
		}
		firstWorkout = &first
	}

	start, end, previousStart, previousEnd, bucket, ok := overviewPeriod(period, now, firstWorkout)
	if !ok {
		return models.StatisticsOverview{}, false
	}

	overview := models.StatisticsOverview{
		Period:        period,
		Start:         start,
		End:           end,
		Bucket:        bucket,
		Totals:        overviewTotals(sessions, activities, start, end),
		Buckets:       []models.StatisticsBucket{},
		Mix:           []models.StatisticsMixItem{},
		RecentRecords: []models.RecentPersonalRecord{},
	}
	if previousStart != nil {
		previous := overviewTotals(sessions, activities, *previousStart, *previousEnd)
		overview.Previous = &previous
	}

	// Buckets: every week (or month) of the period, empty ones included.
	bucketStart := func(t time.Time) time.Time {
		if bucket == "month" {
			return localMonthStart(t)
		}
		return localWeekStart(t)
	}
	index := map[time.Time]int{}
	for cursor := bucketStart(start); !cursor.After(end); {
		index[cursor] = len(overview.Buckets)
		overview.Buckets = append(overview.Buckets, models.StatisticsBucket{Start: cursor, Distances: []models.UnitAmount{}})
		if bucket == "month" {
			cursor = cursor.AddDate(0, 1, 0)
		} else {
			cursor = cursor.AddDate(0, 0, 7)
		}
	}
	for _, session := range sessions {
		if !inDays(session.Date, start, end) {
			continue
		}
		if i, found := index[bucketStart(session.Date)]; found {
			overview.Buckets[i].Workouts++
			overview.Buckets[i].DurationSeconds += session.DurationSeconds
		}
	}

	// Activity mix and bucket distances.
	mix := map[uuid.UUID]*models.StatisticsMixItem{}
	for _, row := range activities {
		if !inDays(row.Date, start, end) {
			continue
		}
		if i, found := index[bucketStart(row.Date)]; found && row.Distance > 0 {
			overview.Buckets[i].Distances = addUnitAmount(overview.Buckets[i].Distances, row.DistanceUnit, row.Distance)
		}
		if row.ActionID == nil {
			continue
		}
		item := mix[*row.ActionID]
		if item == nil {
			item = &models.StatisticsMixItem{ActionID: *row.ActionID, Name: row.ActionName}
			mix[*row.ActionID] = item
		}
		item.Activities++
		item.DurationSeconds += activityDurationSeconds(row)
	}
	for i := range overview.Buckets {
		overview.Buckets[i].Distances = roundUnitAmounts(overview.Buckets[i].Distances)
	}
	for _, item := range mix {
		overview.Mix = append(overview.Mix, *item)
	}
	sort.Slice(overview.Mix, func(i, j int) bool {
		if overview.Mix[i].DurationSeconds != overview.Mix[j].DurationSeconds {
			return overview.Mix[i].DurationSeconds > overview.Mix[j].DurationSeconds
		}
		if overview.Mix[i].Activities != overview.Mix[j].Activities {
			return overview.Mix[i].Activities > overview.Mix[j].Activities
		}
		return overview.Mix[i].Name < overview.Mix[j].Name
	})

	// Streaks are all-time, from the workouts that count toward the goal.
	activeDates := []time.Time{}
	for _, session := range sessions {
		if session.CountsTowardGoal {
			activeDates = append(activeDates, session.Date)
		}
	}
	streaks := computePersonalStreaksFromDates(activeDates)
	overview.Streaks = models.StatisticsStreaks{WeekCurrent: streaks.WeekCurrent, WeekBest: streaks.WeekBest, DayCurrent: streaks.DayCurrent, DayBest: streaks.DayBest}

	// Records set inside the period, newest first.
	for _, group := range groups {
		for _, record := range group.Records {
			if inDays(record.Current.SetAt, start, end) {
				overview.RecentRecords = append(overview.RecentRecords, models.RecentPersonalRecord{ActionID: group.Action.ID, ActionName: group.Action.Name, Record: record})
			}
		}
	}
	sort.SliceStable(overview.RecentRecords, func(i, j int) bool {
		return overview.RecentRecords[i].Record.Current.SetAt.After(overview.RecentRecords[j].Record.Current.SetAt)
	})
	if len(overview.RecentRecords) > statisticsRecentRecords {
		overview.RecentRecords = overview.RecentRecords[:statisticsRecentRecords]
	}

	return overview, true
}

// APIGetStatisticsOverview returns the caller's overview for ?period= (4w, 12w, year or all;
// 4w when omitted).
func APIGetStatisticsOverview(context *gin.Context) {
	userID, err := middlewares.GetAuthUsername(context.GetHeader("Authorization"))
	if err != nil {
		logger.Log.Info("Failed to verify user ID. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to verify user ID."})
		context.Abort()
		return
	}

	period := context.DefaultQuery("period", "4w")
	now := time.Now()
	if _, _, _, _, _, ok := overviewPeriod(period, now, nil); !ok {
		context.JSON(http.StatusBadRequest, gin.H{"error": "Unknown period. Use 4w, 12w, year or all."})
		context.Abort()
		return
	}

	sessions, err := database.GetStatisticsSessionRows(userID, database.StatisticsFilter{})
	if err != nil {
		logger.Log.Info("Failed to get workouts for overview. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get workouts."})
		context.Abort()
		return
	}
	activities, err := database.GetStatisticsActivityRows(userID, database.StatisticsFilter{})
	if err != nil {
		logger.Log.Info("Failed to get activities for overview. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get activities."})
		context.Abort()
		return
	}
	groups, err := computePersonalRecords(userID, nil, now)
	if err != nil {
		logger.Log.Info("Failed to get records for overview. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get personal records."})
		context.Abort()
		return
	}

	overview, _ := buildStatisticsOverview(period, now, sessions, activities, groups)
	context.JSON(http.StatusOK, gin.H{"overview": overview, "message": "Overview retrieved."})
}
