package controllers

import (
	"sort"
	"strings"
	"time"

	"github.com/aunefyren/treningheten/database"
	"github.com/aunefyren/treningheten/models"

	"github.com/google/uuid"
)

// assembleUserProfile returns a flattened profile for the authenticated user.
func assembleUserProfile(userID uuid.UUID) (models.MCPProfile, error) {
	user, err := database.GetUserInformation(userID)
	if err != nil {
		return models.MCPProfile{}, err
	}
	return models.MCPProfile{
		ID:          user.ID.String(),
		FirstName:   user.FirstName,
		LastName:    user.LastName,
		Email:       user.Email,
		Admin:       user.Admin != nil && *user.Admin,
		MemberSince: user.CreatedAt,
	}, nil
}

// assembleUserWeights returns the user's weight history, newest first.
func assembleUserWeights(userID uuid.UUID, limit int) ([]models.MCPWeight, error) {
	weights, err := database.GetEnabledWeightsForUser(userID)
	if err != nil {
		return nil, err
	}
	sort.Slice(weights, func(i, j int) bool {
		return weights[i].Date.After(weights[j].Date)
	})
	result := make([]models.MCPWeight, 0, len(weights))
	for _, w := range weights {
		result = append(result, models.MCPWeight{Date: w.Date, Weight: w.Weight})
	}
	if limit > 0 && len(result) > limit {
		result = result[:limit]
	}
	return result, nil
}

// assembleUserActivities flattens the ExerciseDay -> Exercise -> Operation -> Set
// model into a date-sorted list of activities. It walks the enriched *Object tree
// (see ConvertExerciseDaysToExerciseDayObjects) rather than raw GORM models, so it
// inherits action resolution and the exercise-time fallback for free.
// actionFilter (case-insensitive) limits to a single exercise type when non-empty;
// limit caps the result count.
func assembleUserActivities(userID uuid.UUID, actionFilter string, limit int) ([]models.MCPActivity, error) {
	dayObjects, err := loadUserExerciseDayObjects(userID)
	if err != nil {
		return nil, err
	}
	return flattenActivities(dayObjects, actionFilter, limit), nil
}

// assembleActivitySearch runs the query-time /exercises feed (database.GetActivityFeedForUser)
// and maps each pre-aggregated row to a slim MCPActivitySummary. Unlike assembleUserActivities
// it does NOT load or convert the whole exercise-day tree — the filtering, sorting and
// pagination happen in the database, so a client can find relevant activities without pulling
// everything. It returns the summaries plus the total match count and whether more pages remain.
func assembleActivitySearch(userID uuid.UUID, filter models.ActivityFeedFilter) ([]models.MCPActivitySummary, int64, bool, error) {
	items, total, err := database.GetActivityFeedForUser(userID, filter)
	if err != nil {
		return nil, 0, false, err
	}

	summaries := make([]models.MCPActivitySummary, 0, len(items))
	for _, item := range items {
		summaries = append(summaries, feedItemToSummary(item))
	}

	hasMore := int64(filter.Offset+len(items)) < total
	return summaries, total, hasMore, nil
}

// feedItemToSummary flattens one ActivityFeedItem into the MCP search shape. Source mirrors the
// rich path's precedence (strava first, then hevy, else manual). Distance/weight units are only
// meaningful when a value is present, so they are dropped for zero metrics to keep the payload lean.
func feedItemToSummary(item models.ActivityFeedItem) models.MCPActivitySummary {
	source := "manual"
	if item.HasStrava {
		source = "strava"
	} else if item.HevyWorkoutID != nil && *item.HevyWorkoutID != "" {
		source = "hevy"
	}

	summary := models.MCPActivitySummary{
		ID:                   item.OperationID.String(),
		Date:                 item.Date,
		Time:                 item.Time,
		Action:               item.ActionName,
		Type:                 item.ActionType,
		Note:                 derefString(item.Note),
		Distance:             item.Distance,
		DurationSeconds:      item.DurationSeconds,
		MovingSeconds:        item.MovingSeconds,
		Repetitions:          item.Repetitions,
		TopWeight:            item.TopWeight,
		SetCount:             item.SetCount,
		HasStreams:           item.HasStrava,
		Source:               source,
		CountsTowardGoal:     item.CountsTowardGoal,
		SessionID:            item.ExerciseID.String(),
		SessionActivityCount: item.SessionActivityCount,
		AvgHeartrateBpm:      item.AvgHeartrate,
		MaxHeartrateBpm:      item.MaxHeartrate,
		AvgCadenceRpm:        item.AvgCadence,
		TemperatureC:         item.TempC,
		ElevationGainM:       item.ElevationGainM,
	}
	if item.Distance > 0 {
		summary.DistanceUnit = item.DistanceUnit
	}
	if item.TopWeight > 0 {
		summary.WeightUnit = item.WeightUnit
	}
	// Derive average pace from moving time when present (more accurate), else elapsed time.
	// Only meaningful for distance activities, so it stays 0 for strength/timed work.
	if distKm := distanceToKm(item.Distance, item.DistanceUnit); distKm > 0 {
		secs := item.MovingSeconds
		if secs <= 0 {
			secs = item.DurationSeconds
		}
		if secs > 0 {
			summary.AvgPaceMinKm = round2(float64(secs) / 60.0 / distKm)
		}
	}
	return summary
}

// loadUserExerciseDayObjects loads and enriches the user's exercise days. Callers
// that need both the flat activities and the day grouping (e.g. streaks) load once
// and reuse, since the conversion is the expensive part.
func loadUserExerciseDayObjects(userID uuid.UUID) ([]models.ExerciseDayObject, error) {
	days, err := database.GetExerciseDaysForUserUsingUserID(userID)
	if err != nil {
		return nil, err
	}
	return ConvertExerciseDaysToExerciseDayObjects(days)
}

// flattenActivities walks the enriched day tree into a date-sorted activity list.
// actionFilter (case-insensitive) limits to a single exercise type when non-empty;
// limit caps the result count.
func flattenActivities(dayObjects []models.ExerciseDayObject, actionFilter string, limit int) []models.MCPActivity {
	actionFilter = strings.ToLower(strings.TrimSpace(actionFilter))

	activities := []models.MCPActivity{}
	for _, day := range dayObjects {
		for _, exercise := range day.Exercises {
			// Soundtrack is a session-level fact, so it is the same for every operation
			// of the exercise. The day tree already carries it (media is filled during
			// ConvertExerciseToExerciseObject when Media.Enabled), so this is free here.
			hasSoundtrack := len(exercise.MediaPlayback) > 0
			for _, op := range exercise.Operations {
				activity := operationObjectToActivity(op, exercise.Time, exercise.HevyWorkoutID, hasSoundtrack, exercise.CountsTowardGoal)

				if actionFilter != "" && !strings.Contains(strings.ToLower(activity.Action), actionFilter) {
					continue
				}

				activities = append(activities, activity)
			}
		}
	}

	sort.Slice(activities, func(i, j int) bool {
		return activities[i].Date.After(activities[j].Date)
	})
	if limit > 0 && len(activities) > limit {
		activities = activities[:limit]
	}
	return activities
}

// operationObjectToActivity flattens one enriched OperationObject into an
// MCPActivity. HasStreams flags whether any set carries Strava sensor data, so the
// LLM knows it can call get_activity_streams for the time-series detail. hevyWorkoutID
// is the parent exercise's Hevy id (provenance), used to set Source. hasSoundtrack is
// the session-level listening-history flag; the tracks are fetched on demand via
// get_activity_soundtrack (mirrors the streams pattern).
func operationObjectToActivity(op models.OperationObject, date time.Time, hevyWorkoutID *string, hasSoundtrack bool, countsTowardGoal bool) models.MCPActivity {
	note := derefString(op.Note)
	actionName := "Unknown"
	if op.Action != nil {
		actionName = op.Action.Name
	} else if name := strings.TrimSpace(note); name != "" {
		// No global Action (e.g. a Hevy custom exercise): the exercise name is kept on
		// the operation's note, with any real note moved to the description. Promote it
		// so the activity's action reflects the exercise instead of "Unknown".
		actionName = name
		note = ""
	}

	hasStreams := false
	for _, s := range op.OperationSets {
		if s.StravaStreams != nil {
			hasStreams = true
			break
		}
	}

	source := "manual"
	if op.StravaID != nil && *op.StravaID != "" {
		source = "strava"
	} else if hevyWorkoutID != nil && *hevyWorkoutID != "" {
		source = "hevy"
	}

	return models.MCPActivity{
		ID:               op.ID.String(),
		Date:             date,
		Action:           actionName,
		Type:             op.Type,
		Source:           source,
		Note:             note,
		Description:      derefString(op.Description),
		Tags:             op.Tags,
		Equipment:        derefString(op.Equipment),
		DurationSeconds:  copySecondsPtr(op.Duration),
		HasStreams:       hasStreams,
		HasSoundtrack:    hasSoundtrack,
		CountsTowardGoal: countsTowardGoal,
		Sets:             mapSets(op.OperationSets, op.WeightUnit, op.DistanceUnit),
	}
}

// assembleSingleActivity returns the flat view of one activity (operation) owned by
// the user, resolving its exercise date with the same fallback as the list path. When
// include is non-empty and the activity carries Strava streams, the requested processed
// blocks (segments, zones, elevation, route, profile, analysis) are attached under
// StreamSummary — the summary-first path that avoids pulling the raw series via
// get_activity_streams.
func assembleSingleActivity(userID uuid.UUID, activityID uuid.UUID, include []string) (models.MCPActivity, error) {
	operation, err := database.GetOperationByIDAndUserID(activityID, userID)
	if err != nil {
		return models.MCPActivity{}, err
	}

	opObject, err := ConvertOperationToOperationObject(operation)
	if err != nil {
		return models.MCPActivity{}, err
	}

	date, hevyWorkoutID, countsTowardGoal := resolveExerciseDate(userID, operation.ExerciseID)
	// Unlike the list path, this operation-level path doesn't carry the session's
	// media, so resolve the flag directly (a no-op query when Media is disabled).
	hasSoundtrack := exerciseHasSoundtrack(operation.ExerciseID)
	activity := operationObjectToActivity(opObject, date, hevyWorkoutID, hasSoundtrack, countsTowardGoal)

	if activity.HasStreams {
		if len(include) > 0 {
			summary, err := assembleActivityStreamSummary(userID, activityID)
			if err != nil {
				return models.MCPActivity{}, err
			}
			activity.StreamSummary = filterStreamSummary(summary, include)
		} else {
			// Fetched flat on a stream-backed activity: surface the analysis layer in-band so the
			// caller discovers it here, not only from the tool description.
			activity.AnalysisHint = mcpAnalysisHint
		}
	}

	return activity, nil
}

// mcpAnalysisHint is returned on a stream-backed activity fetched without include, telling the
// caller exactly how to pull the analysis blocks without the raw series.
const mcpAnalysisHint = "This activity has Strava sensor streams. Re-call get_activity with include for the analysis layer (no raw samples pulled): " +
	`include:["segments","zones","analysis"] gives per-km/mile splits (pace, HR, cadence, elevation per split), the heart-rate zone breakdown, and derived metrics (aerobic decoupling, pace consistency, walk/stop breaks with count and timing, HR-by-gradient). ` +
	`Also available: "elevation", "route", "profile". Prefer this over get_activity_streams for splits, zones and drift — streams are only for the raw second-by-second series.`

// resolveExerciseDate mirrors ConvertExerciseToExerciseObject's time fallback:
// the exercise's own Time, else its day's date, else now. It also returns the
// exercise's Hevy workout id (provenance) so the activity can report its source,
// and the session's counts-toward-goal flag.
func resolveExerciseDate(userID uuid.UUID, exerciseID uuid.UUID) (time.Time, *string, bool) {
	exercise, err := database.GetExerciseByIDAndUserID(exerciseID, userID)
	if err != nil || exercise == nil {
		return time.Now(), nil, false
	}
	date := time.Now()
	if exercise.Time != nil {
		date = *exercise.Time
	} else if day, err := database.GetExerciseDayByID(exercise.ExerciseDayID); err == nil && day != nil {
		date = day.Date
	}
	return date, exercise.HevyWorkoutID, exercise.CountsTowardGoal
}

// assembleUserStatistics derives a focused set of counts, totals and streaks from
// the user's full activity history. It loads the enriched day tree once and uses it
// for both the windowed totals and the streak computation.
func assembleUserStatistics(userID uuid.UUID) (models.MCPStatistics, error) {
	dayObjects, err := loadUserExerciseDayObjects(userID)
	if err != nil {
		return models.MCPStatistics{}, err
	}

	activities := flattenActivities(dayObjects, "", 0)

	now := time.Now()
	monthAgo := now.AddDate(0, -1, 0)
	yearAgo := now.AddDate(-1, 0, 0)

	add := func(w *models.MCPStatWindow, distanceKm float64, timeSeconds int64) {
		w.Activities++
		w.DistanceKm += distanceKm
		w.TimeSeconds += timeSeconds
	}

	stats := models.MCPStatistics{}
	for _, a := range activities {
		distanceKm := 0.0
		for _, s := range a.Sets {
			if s.Distance != nil {
				distanceKm += distanceToKm(*s.Distance, s.DistanceUnit)
			}
		}

		// Count time once per activity: the activity duration when present, otherwise
		// the sum of set times. (Strava sets both to the same value, so adding both
		// would double-count.)
		var timeSeconds int64
		if a.DurationSeconds != nil {
			timeSeconds = *a.DurationSeconds
		} else {
			for _, s := range a.Sets {
				if s.TimeSeconds != nil {
					timeSeconds += *s.TimeSeconds
				}
			}
		}

		add(&stats.AllTime, distanceKm, timeSeconds)
		if a.Date.After(yearAgo) {
			add(&stats.PastYear, distanceKm, timeSeconds)
		}
		if a.Date.After(monthAgo) {
			add(&stats.PastMonth, distanceKm, timeSeconds)
		}
	}

	// Tidy float accumulation noise from the km conversions.
	stats.AllTime.DistanceKm = round2(stats.AllTime.DistanceKm)
	stats.PastYear.DistanceKm = round2(stats.PastYear.DistanceKm)
	stats.PastMonth.DistanceKm = round2(stats.PastMonth.DistanceKm)

	streaks := computePersonalStreaks(dayObjects)
	stats.Streaks.Personal = models.MCPPersonalStreaks{
		WeekCurrent: streaks.WeekCurrent,
		WeekBest:    streaks.WeekBest,
		DayCurrent:  streaks.DayCurrent,
		DayBest:     streaks.DayBest,
	}

	return stats, nil
}

func mapSets(sets []models.OperationSetObject, weightUnit string, distanceUnit string) []models.MCPActivitySet {
	result := make([]models.MCPActivitySet, 0, len(sets))
	for _, s := range sets {
		set := models.MCPActivitySet{
			Repetitions:       s.Repetitions,
			Weight:            s.Weight,
			Distance:          s.Distance,
			TimeSeconds:       copySecondsPtr(s.Time),
			MovingTimeSeconds: copySecondsPtr(s.MovingTime),
		}
		if s.Weight != nil {
			set.WeightUnit = weightUnit
		}
		if s.Distance != nil {
			set.DistanceUnit = distanceUnit
		}
		result = append(result, set)
	}
	return result
}

// copySecondsPtr returns a copy of a nullable seconds value. The duration-ish fields
// (Operation.Duration, OperationSet.Time/MovingTime) store a raw seconds count in an
// int64, and the MCP *Seconds fields are seconds too, so this is a straight copy.
func copySecondsPtr(v *int64) *int64 {
	if v == nil {
		return nil
	}
	seconds := *v
	return &seconds
}

// distanceToKm normalizes a distance value to kilometres based on its unit. Units
// are free-form strings (default "km"); unrecognized units are treated as km, which
// is the overwhelmingly common case (Strava import and the model default).
func distanceToKm(value float64, unit string) float64 {
	switch strings.ToLower(strings.TrimSpace(unit)) {
	case "mi", "mile", "miles":
		return value * 1.609344
	case "m", "meter", "meters", "metre", "metres":
		return value / 1000.0
	case "yd", "yard", "yards":
		return value * 0.0009144
	case "ft", "foot", "feet":
		return value * 0.0003048
	default:
		return value
	}
}

func derefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
