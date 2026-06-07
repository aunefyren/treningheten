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
// model into a date-sorted list of activities. actionFilter (case-insensitive)
// limits to a single exercise type when non-empty; limit caps the result count.
func assembleUserActivities(userID uuid.UUID, actionFilter string, limit int) ([]models.MCPActivity, error) {
	days, err := database.GetAllExerciseDaysWithExerciseByUserID(userID)
	if err != nil {
		return nil, err
	}

	actionFilter = strings.ToLower(strings.TrimSpace(actionFilter))
	actionCache := map[uuid.UUID]string{}

	activities := []models.MCPActivity{}
	for _, day := range days {
		exercises, err := database.GetExerciseByExerciseDayID(day.ID)
		if err != nil {
			return nil, err
		}
		for _, exercise := range exercises {
			operations, err := database.GetOperationsByExerciseID(exercise.ID)
			if err != nil {
				return nil, err
			}
			for _, op := range operations {
				actionName := resolveActionName(op.ActionID, actionCache)

				if actionFilter != "" && !strings.Contains(strings.ToLower(actionName), actionFilter) {
					continue
				}

				sets, err := database.GetOperationSetsByOperationID(op.ID)
				if err != nil {
					return nil, err
				}

				activities = append(activities, models.MCPActivity{
					Date:            day.Date,
					Action:          actionName,
					Type:            op.Type,
					Note:            derefString(op.Note),
					DurationSeconds: durationToSeconds(op.Duration),
					Sets:            mapSets(sets, op.WeightUnit, op.DistanceUnit),
				})
			}
		}
	}

	sort.Slice(activities, func(i, j int) bool {
		return activities[i].Date.After(activities[j].Date)
	})
	if limit > 0 && len(activities) > limit {
		activities = activities[:limit]
	}
	return activities, nil
}

// assembleUserStatistics derives a focused set of counts and totals from the
// user's full activity history.
func assembleUserStatistics(userID uuid.UUID) (models.MCPStatistics, error) {
	activities, err := assembleUserActivities(userID, "", 0)
	if err != nil {
		return models.MCPStatistics{}, err
	}

	now := time.Now()
	monthAgo := now.AddDate(0, -1, 0)
	yearAgo := now.AddDate(-1, 0, 0)

	stats := models.MCPStatistics{}
	for _, a := range activities {
		stats.ActivitiesAllTime++
		if a.Date.After(yearAgo) {
			stats.ActivitiesPastYear++
		}
		if a.Date.After(monthAgo) {
			stats.ActivitiesPastMonth++
		}
		if a.DurationSeconds != nil {
			stats.TotalTimeSeconds += *a.DurationSeconds
		}
		for _, s := range a.Sets {
			if s.Distance != nil {
				stats.TotalDistance += *s.Distance
			}
			if s.TimeSeconds != nil {
				stats.TotalTimeSeconds += *s.TimeSeconds
			}
		}
	}
	return stats, nil
}

func resolveActionName(actionID *uuid.UUID, cache map[uuid.UUID]string) string {
	if actionID == nil {
		return "Unknown"
	}
	if name, ok := cache[*actionID]; ok {
		return name
	}
	action, err := database.GetActionByID(*actionID)
	name := "Unknown"
	if err == nil {
		name = action.Name
	}
	cache[*actionID] = name
	return name
}

func mapSets(sets []models.OperationSet, weightUnit string, distanceUnit string) []models.MCPActivitySet {
	result := make([]models.MCPActivitySet, 0, len(sets))
	for _, s := range sets {
		set := models.MCPActivitySet{
			Repetitions: s.Repetitions,
			Weight:      s.Weight,
			Distance:    s.Distance,
			TimeSeconds: durationToSeconds(s.Time),
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

func durationToSeconds(d *time.Duration) *int64 {
	if d == nil {
		return nil
	}
	seconds := int64(d.Seconds())
	return &seconds
}

func derefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
