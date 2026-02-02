package database

import (
	"github.com/aunefyren/treningheten/models"

	"github.com/google/uuid"
)

// Retrieve prize by ID
func GetPrizeByID(prizeID uuid.UUID) (models.Prize, bool, error) {

	var prizeStruct models.Prize

	prizeRecord := Instance.Where("`prizes`.enabled = ?", 1).Where("`prizes`.ID = ?", prizeID).Find(&prizeStruct)
	if prizeRecord.Error != nil {
		return models.Prize{}, false, prizeRecord.Error
	} else if prizeRecord.RowsAffected == 0 {
		return models.Prize{}, false, nil
	}

	return prizeStruct, true, nil

}

// Retrieve all enabled prizes
func GetPrizes() ([]models.Prize, bool, error) {

	var prizeStruct []models.Prize

	prizeRecord := Instance.Where("`prizes`.enabled = ?", 1).Find(&prizeStruct)
	if prizeRecord.Error != nil {
		return []models.Prize{}, false, prizeRecord.Error
	} else if prizeRecord.RowsAffected == 0 {
		return []models.Prize{}, false, nil
	}

	return prizeStruct, true, nil

}

// Retrieve prize by name and quantity
func GetPrizeByNameAndQuantity(prizeName string, prizeQuantity int) (models.Prize, bool, error) {

	var prizeStruct models.Prize

	prizeRecord := Instance.Where("`prizes`.enabled = ?", 1).Where("`prizes`.name = ?", prizeName).Where("`prizes`.quantity = ?", prizeQuantity).Find(&prizeStruct)
	if prizeRecord.Error != nil {
		return models.Prize{}, false, prizeRecord.Error
	} else if prizeRecord.RowsAffected == 0 {
		return models.Prize{}, false, nil
	}

	return prizeStruct, true, nil

}

// Create new prize
func CreatePrizeInDB(prize models.Prize) error {
	record := Instance.Create(&prize)
	if record.Error != nil {
		return record.Error
	}
	return nil
}
