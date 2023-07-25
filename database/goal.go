package database

import (
	"aunefyren/treningheten/models"
	"errors"
)

// Create new goal within a season
func CreateGoalInDB(goal models.Goal) (uint, error) {
	record := Instance.Create(&goal)
	if record.Error != nil {
		return 0, record.Error
	}
	return goal.ID, nil
}

// Verify if a user has a goal within a season
func VerifyUserGoalInSeason(userID int, seasonID int) (bool, int, error) {
	var goal models.Goal
	goalrecord := Instance.Where("`goals`.enabled = ?", 1).Where("`goals`.user = ?", userID).Where("`goals`.season = ?", seasonID).Find(&goal)
	if goalrecord.Error != nil {
		return false, 0, goalrecord.Error
	} else if goalrecord.RowsAffected == 1 {
		return true, int(goal.ID), nil
	}
	return false, 0, nil
}

// Get goals from within season
func GetGoalsFromWithinSeason(seasonID int) ([]models.Goal, error) {

	var goal []models.Goal

	goalrecord := Instance.Where("`goals`.enabled = ?", 1).Where("`goals`.season = ?", seasonID).Find(&goal)

	if goalrecord.Error != nil {
		return []models.Goal{}, goalrecord.Error
	} else if goalrecord.RowsAffected == 0 {
		return []models.Goal{}, nil
	}

	return goal, nil
}

// Get goal from user within season
func GetGoalFromUserWithinSeason(seasonID int, userID int) (models.Goal, error) {
	var goal models.Goal
	goalrecord := Instance.Where("`goals`.enabled = ?", 1).Where("`goals`.season = ?", seasonID).Where("`goals`.user = ?", userID).Find(&goal)
	if goalrecord.Error != nil {
		return models.Goal{}, goalrecord.Error
	} else if goalrecord.RowsAffected == 0 {
		return models.Goal{}, errors.New("User does not have a goal for the season.")
	}
	return goal, nil
}

// Set goal to disabled in DB using goal ID
func DisableGoalInDBUsingGoalID(goalID int) error {

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
