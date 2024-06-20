package database

import (
	"aunefyren/treningheten/models"
	"errors"
	"strings"

	"github.com/google/uuid"
)

// Get all exercise for exercise-day
func GetOperationsByExerciseID(exerciseID uuid.UUID) ([]models.Operation, error) {
	var operations []models.Operation

	exerciseRecord := Instance.Where("`operations`.enabled = ?", 1).Where("`operations`.exercise_id = ?", exerciseID).Find(&operations)
	if exerciseRecord.Error != nil {
		return []models.Operation{}, exerciseRecord.Error
	}

	return operations, nil
}

func GetOperationSetsByOperationID(operationID uuid.UUID) (operationSets []models.OperationSet, err error) {
	operationSets = []models.OperationSet{}
	err = nil

	record := Instance.Where("`operation_sets`.enabled = ?", 1).Where("`operation_sets`.operation_id = ?", operationID).Find(&operationSets)
	if record.Error != nil {
		return operationSets, record.Error
	}

	return
}

// Get all operations for user
func GetOperationsByUserID(userID uuid.UUID) ([]models.Operation, error) {
	var operations []models.Operation

	exerciseRecord := Instance.Where("`operations`.enabled = ?", 1).
		Joins("JOIN `exercises` on `operations`.exercise_id = `exercises`.id").
		Where("`exercises`.enabled = ?", 1).
		Joins("JOIN `exercise_days` on `exercises`.exercise_day_id = `exercise_days`.id").
		Where("`exercise_days`.enabled = ?", 1).
		Joins("JOIN `users` on `exercise_days`.user_id = `users`.id").
		Where("`users`.enabled = ?", 1).
		Where("`users`.id = ?", userID).
		Find(&operations)

	if exerciseRecord.Error != nil {
		return []models.Operation{}, exerciseRecord.Error
	}

	return operations, nil
}

func GetOperationByIDAndUserID(operationID uuid.UUID, userID uuid.UUID) (operation models.Operation, err error) {
	operation = models.Operation{}
	err = nil

	record := Instance.Where("`operations`.enabled = ?", 1).
		Where("`operations`.id = ?", operationID).
		Joins("JOIN `exercises` on `operations`.exercise_id = `exercises`.id").
		Where("`exercises`.enabled = ?", 1).
		Joins("JOIN `exercise_days` on `exercises`.exercise_day_id = `exercise_days`.id").
		Where("`exercise_days`.enabled = ?", 1).
		Joins("JOIN `users` on `exercise_days`.user_id = `users`.id").
		Where("`users`.enabled = ?", 1).
		Where("`users`.id = ?", userID).
		Find(&operation)

	if record.Error != nil {
		return operation, record.Error
	} else if record.RowsAffected != 1 {
		return operation, errors.New("Record not found.")
	}

	return
}

func GetOperationSetsByOperationIDAndUserID(operationID uuid.UUID, userID uuid.UUID) (operationSets []models.OperationSet, err error) {
	operationSets = []models.OperationSet{}
	err = nil

	record := Instance.Where("`operation_sets`.enabled = ?", 1).
		Where("`operation_sets`.operation_id = ?", operationID).
		Joins("JOIN `operations` on `operation_sets`.operation_id = `operations`.id").
		Where("`operations`.enabled = ?", 1).
		Joins("JOIN `exercises` on `operations`.exercise_id = `exercises`.id").
		Where("`exercises`.enabled = ?", 1).
		Joins("JOIN `exercise_days` on `exercises`.exercise_day_id = `exercise_days`.id").
		Where("`exercise_days`.enabled = ?", 1).
		Joins("JOIN `users` on `exercise_days`.user_id = `users`.id").
		Where("`users`.enabled = ?", 1).
		Where("`users`.id = ?", userID).
		Find(&operationSets)

	if record.Error != nil {
		return operationSets, record.Error
	}

	return
}

func GetOperationSetsByUserID(userID uuid.UUID) (operationSets []models.OperationSet, err error) {
	operationSets = []models.OperationSet{}
	err = nil

	record := Instance.Where("`operation_sets`.enabled = ?", 1).
		Joins("JOIN `operations` on `operation_sets`.operation_id = `operations`.id").
		Where("`operations`.enabled = ?", 1).
		Joins("JOIN `exercises` on `operations`.exercise_id = `exercises`.id").
		Where("`exercises`.enabled = ?", 1).
		Joins("JOIN `exercise_days` on `exercises`.exercise_day_id = `exercise_days`.id").
		Where("`exercise_days`.enabled = ?", 1).
		Joins("JOIN `users` on `exercise_days`.user_id = `users`.id").
		Where("`users`.enabled = ?", 1).
		Where("`users`.id = ?", userID).
		Find(&operationSets)

	if record.Error != nil {
		return operationSets, record.Error
	}

	return
}

func CreateOperationInDB(operation models.Operation) (models.Operation, error) {
	record := Instance.Create(&operation)
	if record.Error != nil {
		return operation, record.Error
	}
	return operation, nil
}

func CreateOperationSetInDB(operationSet models.OperationSet) (models.OperationSet, error) {
	record := Instance.Create(&operationSet)
	if record.Error != nil {
		return operationSet, record.Error
	}
	return operationSet, nil
}

func UpdateOperationInDB(operation models.Operation) (models.Operation, error) {
	record := Instance.Save(&operation)
	if record.Error != nil {
		return operation, record.Error
	}
	return operation, nil
}

func GetOperationSetByIDAndUserID(operationSetID uuid.UUID, userID uuid.UUID) (operationSet models.OperationSet, err error) {
	operationSet = models.OperationSet{}
	err = nil

	record := Instance.Where("`operation_sets`.enabled = ?", 1).
		Where("`operation_sets`.id = ?", operationSetID).
		Joins("JOIN `operations` on `operation_sets`.operation_id = `operations`.id").
		Where("`operations`.enabled = ?", 1).
		Joins("JOIN `exercises` on `operations`.exercise_id = `exercises`.id").
		Where("`exercises`.enabled = ?", 1).
		Joins("JOIN `exercise_days` on `exercises`.exercise_day_id = `exercise_days`.id").
		Where("`exercise_days`.enabled = ?", 1).
		Joins("JOIN `users` on `exercise_days`.user_id = `users`.id").
		Where("`users`.enabled = ?", 1).
		Where("`users`.id = ?", userID).
		Find(&operationSet)

	if record.Error != nil {
		return operationSet, record.Error
	}

	return
}

func UpdateOperationSetInDB(operationSet models.OperationSet) (models.OperationSet, error) {
	record := Instance.Save(&operationSet)
	if record.Error != nil {
		return operationSet, record.Error
	}
	return operationSet, nil
}

func GetAllEnabledActions() (actions []models.Action, err error) {
	actions = []models.Action{}
	err = nil

	record := Instance.Where("`actions`.enabled = ?", 1).
		Find(&actions)

	if record.Error != nil {
		return actions, record.Error
	}

	return
}

func GetActionByName(actionName string) (action models.Action, err error) {
	action = models.Action{}
	err = nil
	actionName = strings.ToLower(actionName)

	record := Instance.Where("`actions`.enabled = ?", 1).
		Where("LOWER(`actions`.name) = ? OR LOWER(`actions`.norwegian_name) = ?", actionName, actionName).
		Find(&action)

	if record.Error != nil {
		return action, record.Error
	} else if record.RowsAffected != 1 {
		return action, errors.New("Failed to find action.")
	}

	return
}

func GetActionByID(actionID uuid.UUID) (action models.Action, err error) {
	action = models.Action{}
	err = nil

	record := Instance.Where("`actions`.enabled = ?", 1).
		Where("`actions`.id = ?", actionID).
		Find(&action)

	if record.Error != nil {
		return action, record.Error
	} else if record.RowsAffected != 1 {
		return action, errors.New("Record not found.")
	}

	return
}

func CreateActionInDB(action models.Action) (models.Action, error) {
	record := Instance.Create(&action)
	if record.Error != nil {
		return action, record.Error
	}
	return action, nil
}

func GetActionByStravaName(stravaName string) (action *models.Action, err error) {
	action = nil
	err = nil
	stravaName = strings.ToLower(stravaName)

	record := Instance.Where("`actions`.enabled = ?", 1).
		Where("LOWER(`actions`.strava_name) = ?", stravaName).
		Find(&action)

	if record.Error != nil {
		return action, record.Error
	} else if record.RowsAffected != 1 {
		return nil, err
	}

	return
}

func GetOperationByStravaIDAndUserIDAndExerciseID(userID uuid.UUID, stravaID int, exerciseID uuid.UUID) (operation *models.Operation, err error) {
	operation = nil
	err = nil

	record := Instance.Where("`operations`.enabled = ?", 1).
		Where("`operations`.strava_id = ?", stravaID).
		Joins("JOIN `exercises` on `operations`.exercise_id = `exercises`.id").
		Where("`exercises`.enabled = ?", 1).
		Where("`exercises`.id = ?", exerciseID).
		Joins("JOIN `exercise_days` on `exercises`.exercise_day_id = `exercise_days`.id").
		Where("`exercise_days`.enabled = ?", 1).
		Joins("JOIN `users` on `exercise_days`.user_id = `users`.id").
		Where("`users`.enabled = ?", 1).
		Where("`users`.id = ?", userID).
		Find(&operation)

	if record.Error != nil {
		return nil, record.Error
	} else if record.RowsAffected != 1 {
		return nil, err
	}

	return
}

func GetOperationSetByStravaIDAndUserIDAndOperationID(userID uuid.UUID, stravaID int, operationID uuid.UUID) (operationSet *models.OperationSet, err error) {
	operationSet = nil
	err = nil

	record := Instance.Where("`operation_sets`.enabled = ?", 1).
		Where("`operation_sets`.strava_id = ?", stravaID).
		Joins("JOIN `operations` on `operation_sets`.operation_id = `operations`.id").
		Where("`operations`.enabled = ?", 1).
		Where("`operations`.id = ?", operationID).
		Joins("JOIN `exercises` on `operations`.exercise_id = `exercises`.id").
		Where("`exercises`.enabled = ?", 1).
		Joins("JOIN `exercise_days` on `exercises`.exercise_day_id = `exercise_days`.id").
		Where("`exercise_days`.enabled = ?", 1).
		Joins("JOIN `users` on `exercise_days`.user_id = `users`.id").
		Where("`users`.enabled = ?", 1).
		Where("`users`.id = ?", userID).
		Find(&operationSet)

	if record.Error != nil {
		return nil, record.Error
	} else if record.RowsAffected != 1 {
		return nil, err
	}

	return
}
