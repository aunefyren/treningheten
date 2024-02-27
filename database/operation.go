package database

import (
	"aunefyren/treningheten/models"

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
