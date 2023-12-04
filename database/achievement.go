package database

import (
	"aunefyren/treningheten/models"
	"errors"

	"github.com/google/uuid"
)

// Get all the achivements
func GetAllEnabledAchievements() ([]models.Achievement, error) {

	var achivementStruct []models.Achievement

	achivementRecord := Instance.Order("category DESC, name ASC").Where("`achievements`.enabled = ?", 1).Find(&achivementStruct)
	if achivementRecord.Error != nil {
		return []models.Achievement{}, achivementRecord.Error
	} else if achivementRecord.RowsAffected == 0 {
		return []models.Achievement{}, nil
	}

	return achivementStruct, nil

}

// Get all achivement delegations for userID
func GetDelegatedAchievementsByUserID(userID uuid.UUID) ([]models.AchievementDelegation, bool, error) {

	var achivementStruct = []models.AchievementDelegation{}

	achivementRecord := Instance.Order("created_at desc").Where("`achievement_delegations`.enabled = ?", 1).Where("`achievement_delegations`.user_id = ?", userID).Joins("JOIN users on `achievement_delegations`.user_id = `users`.ID").Where("`users`.enabled = ?", 1).Joins("JOIN achievements on `achievement_delegations`.achievement_id = `achievements`.ID").Where("`achievements`.enabled = ?", 1).Find(&achivementStruct)
	if achivementRecord.Error != nil {
		return []models.AchievementDelegation{}, false, achivementRecord.Error
	} else if achivementRecord.RowsAffected == 0 {
		return []models.AchievementDelegation{}, false, nil
	}

	return achivementStruct, true, nil

}

func CheckIfAchivementsExistinDB() (bool, error) {

	var achivementStruct []models.Achievement

	achivementRecord := Instance.Find(&achivementStruct)
	if achivementRecord.Error != nil {
		return false, achivementRecord.Error
	} else if achivementRecord.RowsAffected == 0 {
		return false, nil
	}

	return true, nil

}

func RegisterAchievementInDB(achievement models.Achievement) (models.Achievement, error) {

	dbRecord := Instance.Create(&achievement)

	if dbRecord.Error != nil {
		return models.Achievement{}, dbRecord.Error
	} else if dbRecord.RowsAffected != 1 {
		return models.Achievement{}, errors.New("Failed to update DB.")
	}

	return achievement, nil
}

// Get achivement by ID
func GetAchievementByID(achievementID uuid.UUID) (models.Achievement, error) {

	var achivementStruct models.Achievement

	achivementRecord := Instance.Where("`achievements`.enabled = ?", 1).Where("`achievements`.ID = ?", achievementID).Find(&achivementStruct)
	if achivementRecord.Error != nil {
		return models.Achievement{}, achivementRecord.Error
	} else if achivementRecord.RowsAffected == 0 {
		return models.Achievement{}, nil
	}

	return achivementStruct, nil

}

func RegisterAchievementDelegationInDB(achievementDelegation models.AchievementDelegation) (models.AchievementDelegation, error) {

	dbRecord := Instance.Create(&achievementDelegation)

	if dbRecord.Error != nil {
		return models.AchievementDelegation{}, dbRecord.Error
	} else if dbRecord.RowsAffected != 1 {
		return models.AchievementDelegation{}, errors.New("Failed to update DB.")
	}

	return achievementDelegation, nil
}

func GetAchievementDelegationByAchivementIDAndUserID(userID uuid.UUID, achievementID uuid.UUID) (models.AchievementDelegation, bool, error) {

	var achivementStruct models.AchievementDelegation

	achivementRecord := Instance.Where("`achievement_delegations`.enabled = ?", 1).Where("`achievement_delegations`.user_id = ?", userID).Joins("JOIN users on `achievement_delegations`.user_id = `users`.ID").Where("`users`.enabled = ?", 1).Joins("JOIN achievements on `achievement_delegations`.achievement_id = `achievements`.ID").Where("`achievements`.enabled = ?", 1).Where("`achievements`.ID = ?", achievementID).Find(&achivementStruct)
	if achivementRecord.Error != nil {
		return models.AchievementDelegation{}, false, achivementRecord.Error
	} else if achivementRecord.RowsAffected == 0 {
		return models.AchievementDelegation{}, false, nil
	}

	return achivementStruct, true, nil

}

func SetAchievementsToSeenForUser(userID uuid.UUID) (updates int64, err error) {
	var achivementStruct models.AchievementDelegation
	err = nil
	updates = 0

	achivementRecord := Instance.Model(achivementStruct).Where("`achievement_delegations`.enabled = ?", 1).Where("`achievement_delegations`.user_id = ?", userID).Where("`achievement_delegations`.seen = ?", false).Update("seen", true)
	if achivementRecord.Error != nil {
		return updates, achivementRecord.Error
	}

	return achivementRecord.RowsAffected, err
}
