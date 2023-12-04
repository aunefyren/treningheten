package database

import (
	"aunefyren/treningheten/models"
	"errors"

	"github.com/google/uuid"
)

// Register wheelview in database
func CreateWheelview(wheelview models.Wheelview) error {
	record := Instance.Create(&wheelview)
	if record.Error != nil {
		return record.Error
	}
	return nil
}

// Get wheel view for debt and user
func GetUnviewedWheelviewByDebtIDAndUserID(userID uuid.UUID, debtID uuid.UUID) (models.Wheelview, bool, error) {

	var wheelStruct models.Wheelview

	wheelviewRecord := Instance.Where("`wheelviews`.enabled = ?", 1).Where("`wheelviews`.debt_id = ?", debtID).Where("`wheelviews`.user_id = ?", userID).Where("`wheelviews`.viewed = ?", 0).Find(&wheelStruct)
	if wheelviewRecord.Error != nil {
		return models.Wheelview{}, false, wheelviewRecord.Error
	} else if wheelviewRecord.RowsAffected != 1 {
		return models.Wheelview{}, false, nil
	}

	return wheelStruct, true, nil

}

// Get wheel view for debt and user
func GetUnviewedWheelviewByUserID(userID uuid.UUID) ([]models.Wheelview, bool, error) {

	var wheelStruct []models.Wheelview

	wheelviewRecord := Instance.Where("`wheelviews`.enabled = ?", 1).Where("`wheelviews`.user_id = ?", userID).Where("`wheelviews`.viewed = ?", 0).Find(&wheelStruct)
	if wheelviewRecord.Error != nil {
		return []models.Wheelview{}, false, wheelviewRecord.Error
	} else if wheelviewRecord.RowsAffected == 0 {
		return []models.Wheelview{}, false, nil
	}

	return wheelStruct, true, nil

}

// Set wheelview to viewed by ID
func SetWheelviewToViewedByID(wheelviewID uuid.UUID) error {

	var wheelview models.Wheelview

	wheelviewRecords := Instance.Model(wheelview).Where("`wheelviews`.enabled = ?", 1).Where("`wheelviews`.ID = ?", wheelviewID).Update("viewed", 1)
	if wheelviewRecords.Error != nil {
		return wheelviewRecords.Error
	}
	if wheelviewRecords.RowsAffected != 1 {
		return errors.New("View not changed in database.")
	}

	return nil
}

// Get wheel view for debt and user
func GetWheelviewByDebtIDAndUserID(userID uuid.UUID, debtID uuid.UUID) (models.Wheelview, bool, error) {

	var wheelStruct models.Wheelview

	wheelviewRecord := Instance.Where("`wheelviews`.enabled = ?", 1).Where("`wheelviews`.debt_id = ?", debtID).Where("`wheelviews`.user_id = ?", userID).Find(&wheelStruct)
	if wheelviewRecord.Error != nil {
		return models.Wheelview{}, false, wheelviewRecord.Error
	} else if wheelviewRecord.RowsAffected != 1 {
		return models.Wheelview{}, false, nil
	}

	return wheelStruct, true, nil

}
