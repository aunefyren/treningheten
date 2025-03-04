package database

import (
	"aunefyren/treningheten/models"
	"errors"

	"github.com/google/uuid"
)

// Get all the achievements
func GetAllEnabledAchievements() ([]models.Achievement, error) {

	var achievementStruct []models.Achievement

	achievementRecord := Instance.Order("category DESC, achievement_order ASC").Where("`achievements`.enabled = ?", 1).Find(&achievementStruct)
	if achievementRecord.Error != nil {
		return []models.Achievement{}, achievementRecord.Error
	} else if achievementRecord.RowsAffected == 0 {
		return []models.Achievement{}, nil
	}

	return achievementStruct, nil

}

// Get all achievement delegations for userID
func GetDelegatedAchievementsByUserID(userID uuid.UUID) ([]models.AchievementDelegation, bool, error) {
	var achievementStruct = []models.AchievementDelegation{}

	achievementRecord := Instance.Order("created_at desc").Where("`achievement_delegations`.enabled = ?", 1).Where("`achievement_delegations`.user_id = ?", userID).Joins("JOIN users on `achievement_delegations`.user_id = `users`.ID").Where("`users`.enabled = ?", 1).Joins("JOIN achievements on `achievement_delegations`.achievement_id = `achievements`.ID").Where("`achievements`.enabled = ?", 1).Find(&achievementStruct)
	if achievementRecord.Error != nil {
		return []models.AchievementDelegation{}, false, achievementRecord.Error
	} else if achievementRecord.RowsAffected == 0 {
		return []models.AchievementDelegation{}, false, nil
	}

	return achievementStruct, true, nil
}

// Get all unique achievement delegations for userID
func GetDistinctDelegatedAchievementsByUserID(userID uuid.UUID) ([]models.AchievementDelegation, bool, error) {
	var finalAchievementDelegations = []models.AchievementDelegation{}
	var achievementStruct = []models.AchievementDelegation{}

	achievementRecord := Instance.Order("created_at desc").
		Where("`achievement_delegations`.enabled = ?", 1).
		Where("`achievement_delegations`.user_id = ?", userID).
		Joins("JOIN users on `achievement_delegations`.user_id = `users`.ID").
		Where("`users`.enabled = ?", 1).
		Joins("JOIN achievements on `achievement_delegations`.achievement_id = `achievements`.ID").
		Where("`achievements`.enabled = ?", 1).
		Find(&achievementStruct)

	if achievementRecord.Error != nil {
		return []models.AchievementDelegation{}, false, achievementRecord.Error
	} else if achievementRecord.RowsAffected == 0 {
		return []models.AchievementDelegation{}, false, nil
	}

	// Remove duplicate achievements
	for _, achievementDelegation := range achievementStruct {
		added := false
		for _, finalAchievementDelegation := range finalAchievementDelegations {
			if achievementDelegation.AchievementID == finalAchievementDelegation.AchievementID {
				added = true
				break
			}
		}
		if !added {
			finalAchievementDelegations = append(finalAchievementDelegations, achievementDelegation)
		}
	}

	return finalAchievementDelegations, true, nil
}

func CheckIfAchievementsExistsInDB() (bool, error) {

	var achievementStruct []models.Achievement

	achievementRecord := Instance.Find(&achievementStruct)
	if achievementRecord.Error != nil {
		return false, achievementRecord.Error
	} else if achievementRecord.RowsAffected == 0 {
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

func SaveAchievementInDB(achievement models.Achievement) (models.Achievement, error) {
	dbRecord := Instance.Save(&achievement)

	if dbRecord.Error != nil {
		return models.Achievement{}, dbRecord.Error
	} else if dbRecord.RowsAffected != 1 {
		return models.Achievement{}, errors.New("Failed to update DB.")
	}

	return achievement, nil
}

// Get achievement by ID
func GetAchievementByID(achievementID uuid.UUID) (models.Achievement, error) {

	var achievementStruct models.Achievement

	achievementRecord := Instance.Where("`achievements`.enabled = ?", 1).Where("`achievements`.ID = ?", achievementID).Find(&achievementStruct)
	if achievementRecord.Error != nil {
		return models.Achievement{}, achievementRecord.Error
	} else if achievementRecord.RowsAffected == 0 {
		return models.Achievement{}, nil
	}

	return achievementStruct, nil

}

// Get all achievements, enabled or disabled
func GetAllAchievements() ([]models.Achievement, error) {
	var achievementStruct = []models.Achievement{}

	achievementRecord := Instance.Find(&achievementStruct)
	if achievementRecord.Error != nil {
		return achievementStruct, achievementRecord.Error
	}

	return achievementStruct, nil
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

func GetAchievementDelegationByAchievementIDAndUserID(userID uuid.UUID, achievementID uuid.UUID) (achievementDelegations []models.AchievementDelegation, err error) {
	achievementDelegations = []models.AchievementDelegation{}

	achievementRecord := Instance.Where("`achievement_delegations`.enabled = ?", 1).
		Where("`achievement_delegations`.user_id = ?", userID).
		Joins("JOIN users on `achievement_delegations`.user_id = `users`.ID").
		Where("`users`.enabled = ?", 1).
		Joins("JOIN achievements on `achievement_delegations`.achievement_id = `achievements`.ID").
		Where("`achievements`.enabled = ?", 1).
		Where("`achievements`.ID = ?", achievementID).
		Find(&achievementDelegations)

	if achievementRecord.Error != nil {
		return achievementDelegations, achievementRecord.Error
	}

	return
}

func SetAchievementsToSeenForUser(userID uuid.UUID) (updates int64, err error) {
	var achievementStruct models.AchievementDelegation
	err = nil
	updates = 0

	achievementRecord := Instance.Model(achievementStruct).Where("`achievement_delegations`.enabled = ?", 1).Where("`achievement_delegations`.user_id = ?", userID).Where("`achievement_delegations`.seen = ?", false).Update("seen", true)
	if achievementRecord.Error != nil {
		return updates, achievementRecord.Error
	}

	return achievementRecord.RowsAffected, err
}

func VerifyIfAchievedByUser(achievementID uuid.UUID, userID uuid.UUID) (bool, error) {
	achievementDelegations := []models.AchievementDelegation{}

	achievementRecord := Instance.
		Where("`achievement_delegations`.enabled = ?", 1).
		Where("`achievement_delegations`.user_id = ?", userID).
		Joins("JOIN users on `achievement_delegations`.user_id = `users`.ID").
		Where("`users`.enabled = ?", 1).
		Where("`users`.id = ?", userID).
		Joins("JOIN achievements on `achievement_delegations`.achievement_id = `achievements`.ID").
		Where("`achievements`.enabled = ?", 1).
		Where("`achievements`.ID = ?", achievementID).
		Find(&achievementDelegations)

	if achievementRecord.Error != nil {
		return false, achievementRecord.Error
	}

	return (len(achievementDelegations) > 0), nil
}
