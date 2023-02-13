package database

import "aunefyren/treningheten/models"

// Create new goal within a season
func CreateGoalInDB(goal models.Goal) error {
	record := Instance.Create(&goal)
	if record.Error != nil {
		return record.Error
	}
	return nil
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
