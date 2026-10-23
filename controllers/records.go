package controllers

import (
	"net/http"
	"sort"
	"strconv"
	"time"

	"github.com/aunefyren/treningheten/database"
	"github.com/aunefyren/treningheten/logger"
	"github.com/aunefyren/treningheten/middlewares"
	"github.com/aunefyren/treningheten/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Personal records: the best a user has ever done per activity type, with the history of
// how each one improved. Everything is computed on read from the statistics rows and the
// stored best efforts; see docs/statistics.md.

const (
	poundsToKg = 0.45359237
	// recordFreshDays is how long a newly set record is flagged as new.
	recordFreshDays = 30
	// oneRepMaxRepCap is the most reps a set may have to feed the estimated one-rep max;
	// Epley drifts badly beyond it.
	oneRepMaxRepCap = 10
)

// Record keys, in the order they are listed within an activity type. Best efforts come first
// as "effort_<metres>", ordered by distance.
const (
	recordLongestDistance = "longest_distance"
	recordLongestDuration = "longest_duration"
	recordMostElevation   = "most_elevation"
	recordHeaviestSet     = "heaviest_set"
	recordOneRepMax       = "estimated_one_rep_max"
	recordMostReps        = "most_reps"
	recordMostVolume      = "most_volume"
)

var recordOrder = []string{recordLongestDistance, recordLongestDuration, recordMostElevation, recordHeaviestSet, recordOneRepMax, recordMostReps, recordMostVolume}

var recordLabels = map[string]string{
	recordLongestDistance: "Longest distance",
	recordLongestDuration: "Longest duration",
	recordMostElevation:   "Most climbing",
	recordHeaviestSet:     "Heaviest set",
	recordOneRepMax:       "Estimated one-rep max",
	recordMostReps:        "Most reps in a set",
	recordMostVolume:      "Most volume in a workout",
}

var recordKinds = map[string]string{
	recordLongestDistance: "distance",
	recordLongestDuration: "duration",
	recordMostElevation:   "elevation",
	recordHeaviestSet:     "weight",
	recordOneRepMax:       "weight",
	recordMostReps:        "reps",
	recordMostVolume:      "weight",
}

// recordCandidate is one activity's (or set's) value for a record key. score is the
// comparable form — metres, kilograms, seconds negated for times — so units never mix.
type recordCandidate struct {
	entry       models.PersonalRecordEntry
	score       float64
	operationID uuid.UUID
}

// recordSeries is every candidate for one (activity type, key).
type recordSeries struct {
	actionID   uuid.UUID
	key        string
	label      string
	kind       string
	candidates []recordCandidate
}

// distanceMeters converts an activity distance to metres by its own unit.
func distanceMeters(distance float64, unit string) float64 {
	if isMileUnit(unit) {
		return distance * models.MetersPerMile
	}
	return distance * models.MetersPerKm
}

// weightKg converts a weight to kilograms by its own unit.
func weightKg(weight float64, unit string) float64 {
	if unit == "lb" {
		return weight * poundsToKg
	}
	return weight
}

// estimatedOneRepMax is the Epley estimate (weight × (1 + reps/30)), with a single rep being
// the weight itself. Sets above oneRepMaxRepCap reps, or without a weight, give 0.
func estimatedOneRepMax(weight float64, reps float64) float64 {
	if weight <= 0 || reps < 1 || reps > oneRepMaxRepCap {
		return 0
	}
	if reps == 1 {
		return weight
	}
	return weight * (1 + reps/30)
}

// activityDurationSeconds is an activity's length: its summed set times, else its own stored
// duration.
func activityDurationSeconds(row models.StatisticsActivityRow) int64 {
	if row.SetSeconds > 0 {
		return row.SetSeconds
	}
	if row.OperationDuration != nil {
		return *row.OperationDuration
	}
	return 0
}

// collectRecordSeries turns the user's rows into candidates for every record key.
func collectRecordSeries(activities []models.StatisticsActivityRow, sets []models.StatisticsSetRow, efforts []models.BestEffortRow) map[uuid.UUID]map[string]*recordSeries {
	series := map[uuid.UUID]map[string]*recordSeries{}
	add := func(actionID uuid.UUID, key string, label string, kind string, candidate recordCandidate) {
		if series[actionID] == nil {
			series[actionID] = map[string]*recordSeries{}
		}
		if series[actionID][key] == nil {
			series[actionID][key] = &recordSeries{actionID: actionID, key: key, label: label, kind: kind}
		}
		candidate.entry.OperationID = candidate.operationID
		series[actionID][key].candidates = append(series[actionID][key].candidates, candidate)
	}
	addKey := func(actionID uuid.UUID, key string, candidate recordCandidate) {
		add(actionID, key, recordLabels[key], recordKinds[key], candidate)
	}

	for _, effort := range efforts {
		label := models.BestEffortLabel(effort.StravaName, effort.DistanceM)
		if label == "" || effort.ElapsedSeconds <= 0 {
			continue
		}
		add(effort.ActionID, "effort_"+strconv.Itoa(effort.DistanceM), label, "time", recordCandidate{
			entry: models.PersonalRecordEntry{Value: float64(effort.ElapsedSeconds), Unit: "s",
				SetAt: models.RecordedAt(effort.Time, effort.Date), ExerciseDayID: effort.ExerciseDayID},
			score:       -float64(effort.ElapsedSeconds),
			operationID: effort.OperationID,
		})
	}

	for _, row := range activities {
		if row.ActionID == nil {
			continue
		}
		setAt := models.RecordedAt(row.Time, row.Date)
		entry := func(value float64, unit string) models.PersonalRecordEntry {
			return models.PersonalRecordEntry{Value: value, Unit: unit, SetAt: setAt, ExerciseDayID: row.ExerciseDayID}
		}
		if row.Type == "moving" && row.Distance > 0 {
			addKey(*row.ActionID, recordLongestDistance, recordCandidate{entry: entry(row.Distance, row.DistanceUnit), score: distanceMeters(row.Distance, row.DistanceUnit), operationID: row.OperationID})
		}
		if row.Type == "moving" || row.Type == "timing" {
			if seconds := activityDurationSeconds(row); seconds > 0 {
				addKey(*row.ActionID, recordLongestDuration, recordCandidate{entry: entry(float64(seconds), "s"), score: float64(seconds), operationID: row.OperationID})
			}
		}
		if row.ElevationGainM != nil && *row.ElevationGainM > 0 {
			addKey(*row.ActionID, recordMostElevation, recordCandidate{entry: entry(round1(*row.ElevationGainM), "m"), score: *row.ElevationGainM, operationID: row.OperationID})
		}
	}

	// Strength: per set for the heaviest set, the one-rep max and the reps; per activity for
	// the volume.
	type volume struct {
		kg, native float64
		unit       string
		set        models.StatisticsSetRow
	}
	volumes := map[uuid.UUID]*volume{}
	volumeOrder := []uuid.UUID{}
	for _, set := range sets {
		if set.Type != "lifting" {
			continue
		}
		setAt := models.RecordedAt(set.Time, set.Date)
		reps := set.Repetitions
		if set.Weight > 0 {
			repsCopy := reps
			addKey(set.ActionID, recordHeaviestSet, recordCandidate{
				entry:       models.PersonalRecordEntry{Value: set.Weight, Unit: set.WeightUnit, Reps: &repsCopy, SetAt: setAt, ExerciseDayID: set.ExerciseDayID},
				score:       weightKg(set.Weight, set.WeightUnit) + reps/1000, // more reps break a tie
				operationID: set.OperationID,
			})
			if estimate := estimatedOneRepMax(set.Weight, reps); estimate > 0 {
				addKey(set.ActionID, recordOneRepMax, recordCandidate{
					entry: models.PersonalRecordEntry{Value: round1(estimate), Unit: set.WeightUnit, Reps: &repsCopy, SetAt: setAt, ExerciseDayID: set.ExerciseDayID},
					score: weightKg(estimate, set.WeightUnit), operationID: set.OperationID,
				})
			}
			if reps > 0 {
				v := volumes[set.OperationID]
				if v == nil {
					v = &volume{unit: set.WeightUnit, set: set}
					volumes[set.OperationID] = v
					volumeOrder = append(volumeOrder, set.OperationID)
				}
				v.kg += weightKg(set.Weight*reps, set.WeightUnit)
				v.native += set.Weight * reps
			}
		}
		if reps > 0 {
			addKey(set.ActionID, recordMostReps, recordCandidate{
				entry: models.PersonalRecordEntry{Value: reps, Unit: "reps", SetAt: setAt, ExerciseDayID: set.ExerciseDayID},
				score: reps, operationID: set.OperationID,
			})
		}
	}
	for _, operationID := range volumeOrder {
		v := volumes[operationID]
		addKey(v.set.ActionID, recordMostVolume, recordCandidate{
			entry: models.PersonalRecordEntry{Value: round1(v.native), Unit: v.unit, SetAt: models.RecordedAt(v.set.Time, v.set.Date), ExerciseDayID: v.set.ExerciseDayID},
			score: v.kg, operationID: operationID,
		})
	}

	return series
}

// recordProgression is the history of a record: the candidates in date order, keeping each
// one that beat everything before it. The last entry is the record itself.
func recordProgression(candidates []recordCandidate) []recordCandidate {
	sorted := append([]recordCandidate(nil), candidates...)
	sort.SliceStable(sorted, func(i, j int) bool {
		return sorted[i].entry.SetAt.Before(sorted[j].entry.SetAt)
	})
	progression := []recordCandidate{}
	for _, candidate := range sorted {
		if len(progression) == 0 || candidate.score > progression[len(progression)-1].score {
			progression = append(progression, candidate)
		}
	}
	return progression
}

// buildRecord turns a series into its current record, the one it beat and how often it improved.
func buildRecord(s *recordSeries, now time.Time) (models.PersonalRecord, bool) {
	progression := recordProgression(s.candidates)
	if len(progression) == 0 {
		return models.PersonalRecord{}, false
	}
	current := progression[len(progression)-1]
	record := models.PersonalRecord{
		Key:          s.key,
		Label:        s.label,
		Kind:         s.kind,
		Current:      current.entry,
		Fresh:        now.Sub(current.entry.SetAt) <= recordFreshDays*24*time.Hour,
		Improvements: len(progression) - 1,
	}
	if len(progression) > 1 {
		previous := progression[len(progression)-2].entry
		record.Previous = &previous
	}
	return record, true
}

// recordKeyRank orders keys within an activity type: best efforts by distance, then the rest.
func recordKeyRank(key string) int {
	if len(key) > 7 && key[:7] == "effort_" {
		meters, _ := strconv.Atoi(key[7:])
		return meters
	}
	for i, k := range recordOrder {
		if k == key {
			return 1_000_000 + i
		}
	}
	return 2_000_000
}

// loadRecordSeries reads everything the records need for one user, optionally one type.
func loadRecordSeries(userID uuid.UUID, actionID *uuid.UUID) (map[uuid.UUID]map[string]*recordSeries, []models.StatisticsActivityRow, error) {
	filter := database.StatisticsFilter{ActionID: actionID}
	activities, err := database.GetStatisticsActivityRows(userID, filter)
	if err != nil {
		return nil, nil, err
	}
	sets, err := database.GetStatisticsSetRows(userID, filter)
	if err != nil {
		return nil, nil, err
	}
	efforts, err := database.GetBestEffortRowsForUser(userID)
	if err != nil {
		return nil, nil, err
	}
	if actionID != nil {
		kept := []models.BestEffortRow{}
		for _, effort := range efforts {
			if effort.ActionID == *actionID {
				kept = append(kept, effort)
			}
		}
		efforts = kept
	}
	return collectRecordSeries(activities, sets, efforts), activities, nil
}

// recordRange is an optional inclusive range of calendar days; a nil side is open.
type recordRange struct {
	start *time.Time
	end   *time.Time
}

func (r recordRange) set() bool {
	return r.start != nil || r.end != nil
}

func (r recordRange) contains(date time.Time) bool {
	day := localDay(date)
	return (r.start == nil || !day.Before(localDay(*r.start))) && (r.end == nil || !day.After(localDay(*r.end)))
}

// computePersonalRecords returns the user's records grouped by activity type, the most used
// types first. With a range, each record is the best inside it (its history and previous best
// too), flagged when it is also the all-time record and otherwise carrying that record in
// AllTimeBest; types with nothing in the range are left out.
func computePersonalRecords(userID uuid.UUID, actionID *uuid.UUID, period recordRange, now time.Time) ([]models.PersonalRecordGroup, error) {
	series, activities, err := loadRecordSeries(userID, actionID)
	if err != nil {
		return nil, err
	}

	activityCounts := map[uuid.UUID]int{}
	for _, row := range activities {
		if row.ActionID != nil && period.contains(models.RecordedAt(row.Time, row.Date)) {
			activityCounts[*row.ActionID]++
		}
	}

	groups := []models.PersonalRecordGroup{}
	for id, byKey := range series {
		action, err := database.GetActionByID(id)
		if err != nil {
			logger.Log.Warn("Failed to get action for records. Skipping. Error: " + err.Error())
			continue
		}
		group := models.PersonalRecordGroup{Action: action, Activities: activityCounts[id], Records: []models.PersonalRecord{}}
		for _, s := range byKey {
			allTime, ok := buildRecord(s, now)
			if !ok {
				continue
			}
			record := allTime
			record.AllTimeRecord = true
			if period.set() {
				inRange := &recordSeries{actionID: s.actionID, key: s.key, label: s.label, kind: s.kind}
				for _, candidate := range s.candidates {
					if period.contains(candidate.entry.SetAt) {
						inRange.candidates = append(inRange.candidates, candidate)
					}
				}
				if record, ok = buildRecord(inRange, now); !ok {
					continue
				}
				record.AllTimeRecord = record.Current.OperationID == allTime.Current.OperationID && record.Current.Value == allTime.Current.Value
				if !record.AllTimeRecord {
					best := allTime.Current
					record.AllTimeBest = &best
				}
			}
			record.Sport = action.StravaName
			group.Records = append(group.Records, record)
		}
		if len(group.Records) == 0 {
			continue
		}
		sort.Slice(group.Records, func(i, j int) bool {
			return recordKeyRank(group.Records[i].Key) < recordKeyRank(group.Records[j].Key)
		})
		groups = append(groups, group)
	}

	sort.Slice(groups, func(i, j int) bool {
		if groups[i].Activities != groups[j].Activities {
			return groups[i].Activities > groups[j].Activities
		}
		return groups[i].Action.Name < groups[j].Action.Name
	})
	return groups, nil
}

// APIGetPersonalRecords returns the caller's personal records; ?action_id= narrows them to
// one activity type, and ?start= / ?end= (days, inclusive) to the bests inside a period.
func APIGetPersonalRecords(context *gin.Context) {
	userID, err := middlewares.GetAuthUsername(context.GetHeader("Authorization"))
	if err != nil {
		logger.Log.Info("Failed to verify user ID. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to verify user ID."})
		context.Abort()
		return
	}

	var actionID *uuid.UUID
	if raw, ok := context.GetQuery("action_id"); ok && raw != "" {
		parsed, err := uuid.Parse(raw)
		if err != nil {
			context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse action ID."})
			context.Abort()
			return
		}
		actionID = &parsed
	}

	period := recordRange{}
	for name, target := range map[string]**time.Time{"start": &period.start, "end": &period.end} {
		if raw, ok := context.GetQuery(name); ok && raw != "" {
			parsed, err := parseStatisticsDate(raw)
			if err != nil {
				context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse " + name + " date."})
				context.Abort()
				return
			}
			*target = &parsed
		}
	}

	groups, err := computePersonalRecords(userID, actionID, period, time.Now())
	if err != nil {
		logger.Log.Info("Failed to compute personal records. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get personal records."})
		context.Abort()
		return
	}

	context.JSON(http.StatusOK, gin.H{"records": groups, "message": "Personal records retrieved."})
}

// APIGetPersonalRecordHistory returns how one record improved over time: every value that
// beat the ones before it, oldest first.
func APIGetPersonalRecordHistory(context *gin.Context) {
	userID, err := middlewares.GetAuthUsername(context.GetHeader("Authorization"))
	if err != nil {
		logger.Log.Info("Failed to verify user ID. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to verify user ID."})
		context.Abort()
		return
	}

	actionID, err := uuid.Parse(context.Query("action_id"))
	if err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse action ID."})
		context.Abort()
		return
	}
	key := context.Query("key")
	if key == "" {
		context.JSON(http.StatusBadRequest, gin.H{"error": "Missing record key."})
		context.Abort()
		return
	}

	series, _, err := loadRecordSeries(userID, &actionID)
	if err != nil {
		logger.Log.Info("Failed to load record history. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get record history."})
		context.Abort()
		return
	}

	s := series[actionID][key]
	if s == nil {
		context.JSON(http.StatusNotFound, gin.H{"error": "No such record."})
		context.Abort()
		return
	}

	history := []models.PersonalRecordEntry{}
	for _, candidate := range recordProgression(s.candidates) {
		history = append(history, candidate.entry)
	}
	context.JSON(http.StatusOK, gin.H{"history": history, "label": s.label, "kind": s.kind, "message": "Record history retrieved."})
}
