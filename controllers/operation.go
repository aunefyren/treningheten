package controllers

import (
	"aunefyren/treningheten/database"
	"aunefyren/treningheten/models"
	"errors"
	"log"
)

func ConvertOperationToOperationObject(operation models.Operation) (operationObject models.OperationObject, err error) {
	operationObject = models.OperationObject{}
	err = nil

	operationSets, err := database.GetOperationSetsByOperationID(operation.ID)
	if err != nil {
		log.Println("Failed to get operation sets using operation ID. Error: " + err.Error())
		return operationObject, errors.New("Failed to get operation sets using operation ID.")
	}

	operationSetObjects, err := ConvertOperationSetsToOperationSetObjects(operationSets)
	if err != nil {
		log.Println("Failed to convert operation sets to operation set objects. Error: " + err.Error())
		return operationObject, errors.New("Failed to convert operation sets to operation set objects.")
	}

	operationObject.OperationSets = operationSetObjects

	operationObject.CreatedAt = operation.CreatedAt
	operationObject.DeletedAt = operation.DeletedAt
	operationObject.Enabled = operation.Enabled
	operationObject.Exercise = operation.ExerciseID
	operationObject.ID = operation.ID
	operationObject.UpdatedAt = operation.UpdatedAt

	return
}

func ConvertOperationsToOperationObjects(operations []models.Operation) (operationObjects []models.OperationObject, err error) {
	operationObjects = []models.OperationObject{}
	err = nil

	for _, operation := range operations {
		operationObject, err := ConvertOperationToOperationObject(operation)
		if err != nil {
			log.Println("Failed to convert operation to operation object. Ignoring... Error: " + err.Error())
			continue
		}
		operationObjects = append(operationObjects, operationObject)
	}

	return
}

func ConvertOperationSetToOperationSetObject(operationSet models.OperationSet) (operationSetObject models.OperationSetObject, err error) {
	operationSetObject = models.OperationSetObject{}
	err = nil

	operationSetObject.CreatedAt = operationSet.CreatedAt
	operationSetObject.DeletedAt = operationSet.DeletedAt
	operationSetObject.Distance = operationSet.Distance
	operationSetObject.DistanceUnit = operationSet.DistanceUnit
	operationSetObject.Enabled = operationSet.Enabled
	operationSetObject.ID = operationSet.ID
	operationSetObject.Operation = operationSet.OperationID
	operationSetObject.Repetitions = operationSet.Repetitions
	operationSetObject.Time = operationSet.Time
	operationSetObject.Type = operationSet.Type
	operationSetObject.UpdatedAt = operationSet.UpdatedAt
	operationSetObject.Weight = operationSet.Weight
	operationSetObject.WeightUnit = operationSet.WeightUnit
	operationSetObject.Action = operationSet.Action

	return
}

func ConvertOperationSetsToOperationSetObjects(operationSets []models.OperationSet) (operationSetObjects []models.OperationSetObject, err error) {
	operationSetObjects = []models.OperationSetObject{}
	err = nil

	for _, operationSet := range operationSets {
		operationSetObject, err := ConvertOperationSetToOperationSetObject(operationSet)
		if err != nil {
			log.Println("Failed to convert operation set to operation set object. Ignoring... Error: " + err.Error())
			continue
		}
		operationSetObjects = append(operationSetObjects, operationSetObject)
	}

	return
}
