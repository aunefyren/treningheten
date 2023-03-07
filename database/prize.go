package database

import "aunefyren/treningheten/models"

// Retrieve prize by ID
func GetPrizeByID(prizeID int) (models.Prize, bool, error) {

	var prizeStruct models.Prize

	prizeRecord := Instance.Where("`prizes`.enabled = ?", 1).Where("`prizes`.ID = ?", prizeID).Find(&prizeStruct)
	if prizeRecord.Error != nil {
		return models.Prize{}, false, prizeRecord.Error
	} else if prizeRecord.RowsAffected == 0 {
		return models.Prize{}, false, nil
	}

	return prizeStruct, true, nil

}
