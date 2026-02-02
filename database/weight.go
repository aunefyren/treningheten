package database

import (
	"errors"

	"github.com/aunefyren/treningheten/models"

	"github.com/google/uuid"
)

func GetEnabledWeightsForUser(userID uuid.UUID) (weights []models.WeightValue, err error) {
	weights = []models.WeightValue{}
	err = nil

	record := Instance.Where("`weight_values`.enabled = ?", 1).
		Where("`weight_values`.user_id = ?", userID).
		Find(&weights)

	if record.Error != nil {
		return weights, record.Error
	}

	return
}

func GetEnabledWeightsByWeightIDAndUserID(userID uuid.UUID, weightID uuid.UUID) (weight models.WeightValue, err error) {
	weight = models.WeightValue{}
	err = nil

	record := Instance.Where("`weight_values`.enabled = ?", 1).
		Where("`weight_values`.user_id = ?", userID).
		Where("`weight_values`.id = ?", weightID).
		Find(&weight)

	if record.Error != nil {
		return weight, record.Error
	} else if record.RowsAffected != 1 {
		return weight, errors.New("Record not found.")
	}

	return
}

func CreateWeightInDB(weight models.WeightValue) (models.WeightValue, error) {
	record := Instance.Create(&weight)
	if record.Error != nil {
		return weight, record.Error
	}
	return weight, nil
}

func UpdateWeightInDB(weight models.WeightValue) (models.WeightValue, error) {
	record := Instance.Save(&weight)
	if record.Error != nil {
		return weight, record.Error
	}
	return weight, nil
}
