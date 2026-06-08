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
			for _, op := range exercise.Operations {
				activity := operationObjectToActivity(op, exercise.Time)

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
// LLM knows it can call get_workout_streams for the time-series detail.
func operationObjectToActivity(op models.OperationObject, date time.Time) models.MCPActivity {
	actionName := "Unknown"
	if op.Action != nil {
		actionName = op.Action.Name
	}

	hasStreams := false
	for _, s := range op.OperationSets {
		if s.StravaStreams != nil {
			hasStreams = true
			break
		}
	}

	return models.MCPActivity{
		ID:              op.ID.String(),
		Date:            date,
		Action:          actionName,
		Type:            op.Type,
		Note:            derefString(op.Note),
		Equipment:       derefString(op.Equipment),
		DurationSeconds: durationToSeconds(op.Duration),
		HasStreams:      hasStreams,
		Sets:            mapSets(op.OperationSets, op.WeightUnit, op.DistanceUnit),
	}
}

// assembleSingleActivity returns the flat view of one activity (operation) owned by
// the user, resolving its exercise date with the same fallback as the list path.
func assembleSingleActivity(userID uuid.UUID, activityID uuid.UUID) (models.MCPActivity, error) {
	operation, err := database.GetOperationByIDAndUserID(activityID, userID)
	if err != nil {
		return models.MCPActivity{}, err
	}

	opObject, err := ConvertOperationToOperationObject(operation)
	if err != nil {
		return models.MCPActivity{}, err
	}

	date := resolveExerciseDate(userID, operation.ExerciseID)
	return operationObjectToActivity(opObject, date), nil
}

// resolveExerciseDate mirrors ConvertExerciseToExerciseObject's time fallback:
// the exercise's own Time, else its day's date, else now.
func resolveExerciseDate(userID uuid.UUID, exerciseID uuid.UUID) time.Time {
	exercise, err := database.GetExerciseByIDAndUserID(exerciseID, userID)
	if err != nil || exercise == nil {
		return time.Now()
	}
	if exercise.Time != nil {
		return *exercise.Time
	}
	if day, err := database.GetExerciseDayByID(exercise.ExerciseDayID); err == nil && day != nil {
		return day.Date
	}
	return time.Now()
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
			TimeSeconds:       durationToSeconds(s.Time),
			MovingTimeSeconds: durationToSeconds(s.MovingTime),
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

// durationToSeconds converts a stored duration to seconds. NOTE: these fields hold
// a raw seconds count, not real nanosecond durations, so we cast the value directly
// rather than calling .Seconds() (which would divide by 1e9 and yield 0).
func durationToSeconds(d *time.Duration) *int64 {
	if d == nil {
		return nil
	}
	seconds := int64(*d)
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
