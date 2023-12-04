package database

import (
	"aunefyren/treningheten/models"
	"errors"

	"github.com/google/uuid"
)

// Create new goal within a season
func CreateGoalInDB(goal models.Goal) (uuid.UUID, error) {
	record := Instance.Create(&goal)
	if record.Error != nil {
		return uuid.UUID{}, record.Error
	}
	return goal.ID, nil
}

// Verify if a user has a goal within a season
func VerifyUserGoalInSeason(userID uuid.UUID, seasonID uuid.UUID) (bool, uuid.UUID, error) {
	var goal models.Goal
	goalrecord := Instance.Where("`goals`.enabled = ?", 1).Where("`goals`.user_id = ?", userID).Where("`goals`.season_id = ?", seasonID).Find(&goal)
	if goalrecord.Error != nil {
		return false, uuid.UUID{}, goalrecord.Error
	} else if goalrecord.RowsAffected == 1 {
		return true, goal.ID, nil
	}
	return false, uuid.UUID{}, nil
}

// Get goals from within season
func GetGoalsFromWithinSeason(seasonID uuid.UUID) ([]models.Goal, error) {

	var goal []models.Goal

	goalrecord := Instance.Where("`goals`.enabled = ?", 1).Where("`goals`.season_id = ?", seasonID).Find(&goal)

	if goalrecord.Error != nil {
		return []models.Goal{}, goalrecord.Error
	} else if goalrecord.RowsAffected == 0 {
		return []models.Goal{}, nil
	}

	return goal, nil
}

// Get goal from user within season
func GetGoalFromUserWithinSeason(seasonID uuid.UUID, userID uuid.UUID) (models.Goal, error) {
	var goal models.Goal
	goalrecord := Instance.Where("`goals`.enabled = ?", 1).Where("`goals`.season_id = ?", seasonID).Where("`goals`.user_id = ?", userID).Find(&goal)
	if goalrecord.Error != nil {
		return models.Goal{}, goalrecord.Error
	} else if goalrecord.RowsAffected == 0 {
		return models.Goal{}, errors.New("User does not have a goal for the season.")
	}
	return goal, nil
}

// Set goal to disabled in DB using goal ID
func DisableGoalInDBUsingGoalID(goalID uuid.UUID) error {

	var goal models.Goal
	goalRecord := Instance.Model(goal).Where("`goals`.ID = ?", goalID).Update("enabled", 0)
	if goalRecord.Error != nil {
		return goalRecord.Error
	}
	if goalRecord.RowsAffected != 1 {
		return errors.New("Goal not changed in database.")
	}

	return nil

}

func GetGoalsForUserUsingUserID(userID uuid.UUID) ([]models.Goal, error) {

	var goals []models.Goal

	goalRecord := Instance.Order("created_at desc").Where("`goals`.enabled = ?", 1).Where("`goals`.user_id = ?", userID).Find(&goals)
	if goalRecord.Error != nil {
		return []models.Goal{}, goalRecord.Error
	} else if goalRecord.RowsAffected == 0 {
		return []models.Goal{}, nil
	}

	return goals, nil

}

// Get goal with goal ID
func GetGoalUsingGoalID(goalID uuid.UUID) (models.Goal, error) {
	var goal models.Goal

	goalrecord := Instance.Where("`goals`.enabled = ?", 1).Where("`goals`.id = ?", goalID).Find(&goal)

	if goalrecord.Error != nil {
		return models.Goal{}, goalrecord.Error
	} else if goalrecord.RowsAffected != 1 {
		return models.Goal{}, errors.New("Failed to find goal.")
	}

	return goal, nil
}
