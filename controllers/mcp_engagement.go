package controllers

import (
	"errors"
	"sort"
	"time"

	"github.com/aunefyren/treningheten/database"
	"github.com/aunefyren/treningheten/models"

	"github.com/google/uuid"
)

// --- Seasons -----------------------------------------------------------------

// assembleUserSeasons returns the seasons the authenticated user has joined. When
// activeOnly is set, only seasons currently within their start-end window are kept.
func assembleUserSeasons(userID uuid.UUID, activeOnly bool) ([]models.MCPSeason, error) {
	goals, err := database.GetGoalsForUserUsingUserID(userID)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	seasons := []models.MCPSeason{}
	for _, goal := range goals {
		season, err := database.GetSeasonByID(goal.SeasonID)
		if err != nil || season == nil {
			continue
		}
		if activeOnly && seasonStatus(*season, now) != "ongoing" {
			continue
		}

		var goalPtr *models.GoalObject
		if goalObject, err := ConvertGoalToGoalObject(goal); err == nil {
			goalPtr = &goalObject
		}
		seasons = append(seasons, seasonToMCP(*season, goalPtr))
	}

	sort.Slice(seasons, func(i, j int) bool { return seasons[i].Start.After(seasons[j].Start) })
	return seasons, nil
}

// assembleSeason returns one season with the user's personal goal data when joined.
func assembleSeason(userID uuid.UUID, seasonID uuid.UUID) (models.MCPSeason, error) {
	season, err := database.GetSeasonByID(seasonID)
	if err != nil {
		return models.MCPSeason{}, err
	}
	if season == nil {
		return models.MCPSeason{}, errors.New("season not found")
	}

	var goalPtr *models.GoalObject
	if goal, err := database.GetGoalFromUserWithinSeason(seasonID, userID); err == nil && goal != nil {
		if goalObject, err := ConvertGoalToGoalObject(*goal); err == nil {
			goalPtr = &goalObject
		}
	}

	return seasonToMCP(*season, goalPtr), nil
}

func seasonToMCP(season models.Season, goal *models.GoalObject) models.MCPSeason {
	s := models.MCPSeason{
		ID:             season.ID.String(),
		Name:           season.Name,
		Description:    season.Description,
		Start:          season.Start,
		End:            season.End,
		Status:         seasonStatus(season, time.Now()),
		SickleaveTotal: season.Sickleave,
	}
	if goal != nil {
		s.Joined = true
		weekly := goal.ExerciseInterval
		competing := goal.Competing
		sickleaveLeft := goal.SickleaveLeft
		s.WeeklyGoal = &weekly
		s.Competing = &competing
		s.SickleaveLeft = &sickleaveLeft
	}
	return s
}

func seasonStatus(season models.Season, now time.Time) string {
	if now.Before(season.Start) {
		return "upcoming"
	}
	if now.After(season.End) {
		return "ended"
	}
	return "ongoing"
}

// --- Achievements ------------------------------------------------------------

// assembleUserAchievements returns the achievement catalog with the authenticated
// user's earned state on each.
func assembleUserAchievements(userID uuid.UUID) ([]models.MCPAchievement, error) {
	achievements, err := database.GetAllEnabledAchievements()
	if err != nil {
		return nil, err
	}
	delegations, _, err := database.GetDelegatedAchievementsByUserID(userID)
	if err != nil {
		return nil, err
	}

	byAchievement := map[uuid.UUID][]models.AchievementDelegation{}
	for _, d := range delegations {
		byAchievement[d.AchievementID] = append(byAchievement[d.AchievementID], d)
	}

	result := make([]models.MCPAchievement, 0, len(achievements))
	for _, a := range achievements {
		result = append(result, achievementToMCP(a, byAchievement[a.ID]))
	}

	sort.Slice(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	return result, nil
}

// assembleAchievement returns a single achievement with the user's earned state.
func assembleAchievement(userID uuid.UUID, achievementID uuid.UUID) (models.MCPAchievement, error) {
	achievement, err := database.GetAchievementByID(achievementID)
	if err != nil {
		return models.MCPAchievement{}, err
	}
	if achievement.ID == uuid.Nil {
		return models.MCPAchievement{}, errors.New("achievement not found")
	}

	delegations, err := database.GetAchievementDelegationByAchievementIDAndUserID(userID, achievementID)
	if err != nil {
		return models.MCPAchievement{}, err
	}

	return achievementToMCP(achievement, delegations), nil
}

func achievementToMCP(a models.Achievement, delegations []models.AchievementDelegation) models.MCPAchievement {
	earned := len(delegations) > 0

	m := models.MCPAchievement{
		ID:          a.ID.String(),
		Name:        a.Name,
		Description: a.Description,
		Category:    a.Category,
		Earned:      earned,
		TimesEarned: len(delegations),
	}

	for i := range delegations {
		if m.LastEarnedAt == nil || delegations[i].GivenAt.After(*m.LastEarnedAt) {
			t := delegations[i].GivenAt
			m.LastEarnedAt = &t
		}
	}

	// Hidden achievements keep their description secret until the user earns them.
	if a.HiddenDescription != nil && *a.HiddenDescription && !earned {
		m.Description = "Hidden"
	}

	return m
}

// --- Achievement delegations -------------------------------------------------

// assembleUserDelegations returns the user's achievement awards, newest first,
// optionally filtered to one achievement, with offset/limit pagination. It also
// returns the total number of matches before pagination.
func assembleUserDelegations(userID uuid.UUID, achievementFilter *uuid.UUID, limit int, offset int) (int, []models.MCPAchievementDelegation, error) {
	delegations, _, err := database.GetDelegatedAchievementsByUserID(userID)
	if err != nil {
		return 0, nil, err
	}

	if achievementFilter != nil {
		filtered := []models.AchievementDelegation{}
		for _, d := range delegations {
			if d.AchievementID == *achievementFilter {
				filtered = append(filtered, d)
			}
		}
		delegations = filtered
	}

	total := len(delegations)

	if offset < 0 {
		offset = 0
	}
	if offset > total {
		offset = total
	}
	end := total
	if limit > 0 && offset+limit < end {
		end = offset + limit
	}
	page := delegations[offset:end]

	cache := map[uuid.UUID]models.Achievement{}
	result := make([]models.MCPAchievementDelegation, 0, len(page))
	for _, d := range page {
		result = append(result, delegationToMCP(d, cache))
	}

	return total, result, nil
}

// assembleDelegation returns one delegation owned by the user. found is false when no
// such delegation belongs to them.
func assembleDelegation(userID uuid.UUID, delegationID uuid.UUID) (models.MCPAchievementDelegation, bool, error) {
	delegation, found, err := database.GetAchievementDelegationByIDAndUserID(delegationID, userID)
	if err != nil || !found {
		return models.MCPAchievementDelegation{}, found, err
	}
	return delegationToMCP(delegation, map[uuid.UUID]models.Achievement{}), true, nil
}

func delegationToMCP(d models.AchievementDelegation, cache map[uuid.UUID]models.Achievement) models.MCPAchievementDelegation {
	achievement, ok := cache[d.AchievementID]
	if !ok {
		achievement, _ = database.GetAchievementByID(d.AchievementID)
		cache[d.AchievementID] = achievement
	}

	return models.MCPAchievementDelegation{
		ID:              d.ID.String(),
		AchievementID:   d.AchievementID.String(),
		AchievementName: achievement.Name,
		Category:        achievement.Category,
		GivenAt:         d.GivenAt,
		Seen:            d.Seen,
	}
}
