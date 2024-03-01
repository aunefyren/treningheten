package controllers

import (
	"aunefyren/treningheten/database"
	"aunefyren/treningheten/middlewares"
	"aunefyren/treningheten/models"
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
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
	operationObject.Action = operation.Action
	operationObject.Type = operation.Type
	operationObject.WeightUnit = operation.WeightUnit
	operationObject.DistanceUnit = operation.DistanceUnit

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
	operationSetObject.Enabled = operationSet.Enabled
	operationSetObject.ID = operationSet.ID
	operationSetObject.Operation = operationSet.OperationID
	operationSetObject.Repetitions = operationSet.Repetitions
	operationSetObject.Time = operationSet.Time
	operationSetObject.UpdatedAt = operationSet.UpdatedAt
	operationSetObject.Weight = operationSet.Weight

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

func APIGetOperationsForUser(context *gin.Context) {
	// Get user ID
	userID, err := middlewares.GetAuthUsername(context.GetHeader("Authorization"))
	if err != nil {
		log.Println("Failed to verify user ID. Error: " + "Failed to verify user ID.")
		context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		context.Abort()
		return
	}

	operations, err := database.GetOperationsByUserID(userID)
	if err != nil {
		log.Println("Failed to get operations. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get operations."})
		context.Abort()
		return
	}

	operationObjects, err := ConvertOperationsToOperationObjects(operations)
	if err != nil {
		log.Println("Failed to get convert operations to operation objects. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get convert operations to operation objects."})
		context.Abort()
		return
	}

	context.JSON(http.StatusOK, gin.H{"message": "Operations retrieved.", "operations": operationObjects})
}

func APIGetOperation(context *gin.Context) {
	var operationID = context.Param("operation_id")

	operationIDUUID, err := uuid.Parse(operationID)
	if err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse ID."})
		context.Abort()
		return
	}

	// Get user ID
	userID, err := middlewares.GetAuthUsername(context.GetHeader("Authorization"))
	if err != nil {
		log.Println("Failed to verify user ID. Error: " + "Failed to verify user ID.")
		context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		context.Abort()
		return
	}

	operation, err := database.GetOperationByIDAndUserID(operationIDUUID, userID)
	if err != nil {
		log.Println("Failed to get operation. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get operation."})
		context.Abort()
		return
	}

	operationObject, err := ConvertOperationToOperationObject(operation)
	if err != nil {
		log.Println("Failed to get convert operation to operation object. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get convert operation to operation object."})
		context.Abort()
		return
	}

	context.JSON(http.StatusOK, gin.H{"message": "Operation retrieved.", "operation": operationObject})
}

func APIGetOperationSets(context *gin.Context) {
	operationSets := []models.OperationSet{}

	// Get user ID
	userID, err := middlewares.GetAuthUsername(context.GetHeader("Authorization"))
	if err != nil {
		log.Println("Failed to verify user ID. Error: " + "Failed to verify user ID.")
		context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		context.Abort()
		return
	}

	operationID, okay := context.GetQuery("operation_id")
	if !okay {
		operationSets, err = database.GetOperationSetsByUserID(userID)
		if err != nil {
			log.Println("Failed to get operation sets. Error: " + err.Error())
			context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get operation sets."})
			context.Abort()
			return
		}
	} else {
		// Parse id
		operationIDUUID, err := uuid.Parse(operationID)
		if err != nil {
			log.Println("Failed to parse operation ID. Error: " + err.Error())
			context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse operation ID."})
			context.Abort()
			return
		}

		operationSets, err = database.GetOperationSetsByOperationIDAndUserID(operationIDUUID, userID)
		if err != nil {
			log.Println("Failed to get operation sets. Error: " + err.Error())
			context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get operation sets."})
			context.Abort()
			return
		}
	}

	operationSetObjects, err := ConvertOperationSetsToOperationSetObjects(operationSets)
	if err != nil {
		log.Println("Failed to get convert operation sets to operation set objects. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get convert operation sets to operation set objects."})
		context.Abort()
		return
	}

	context.JSON(http.StatusOK, gin.H{"message": "Operation sets retrieved.", "operation_sets": operationSetObjects})
}

func APICreateOperationForUser(context *gin.Context) {
	// Initialize variables
	var operationCreationRequest models.OperationCreationRequest
	var operation models.Operation

	// Get user ID
	userID, err := middlewares.GetAuthUsername(context.GetHeader("Authorization"))
	if err != nil {
		log.Println("Failed to verify user ID. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to verify user ID."})
		context.Abort()
		return
	}

	// Parse creation request
	if err := context.ShouldBindJSON(&operationCreationRequest); err != nil {
		log.Println("Failed to parse creation request. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse creation request."})
		context.Abort()
		return
	}

	_, err = database.GetExerciseByIDAndUserID(operationCreationRequest.ExerciseID, userID)
	if err != nil {
		log.Println("Failed to verify exercise. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to verify exercise."})
		context.Abort()
		return
	}

	operation.Action = strings.TrimSpace(operationCreationRequest.Action)
	operation.DistanceUnit = strings.TrimSpace(operationCreationRequest.DistanceUnit)
	operation.ExerciseID = operationCreationRequest.ExerciseID
	operation.Type = strings.TrimSpace(operationCreationRequest.Type)
	operation.WeightUnit = strings.TrimSpace(operationCreationRequest.WeightUnit)
	operation.ID = uuid.New()

	if operation.Type != "lifting" && operation.Type != "moving" && operation.Type != "timing" {
		context.JSON(http.StatusBadRequest, gin.H{"error": "Invalid operation type."})
		context.Abort()
		return
	}

	if operation.DistanceUnit != "km" && operation.DistanceUnit != "miles" {
		context.JSON(http.StatusBadRequest, gin.H{"error": "Invalid operation distance unit."})
		context.Abort()
		return
	}

	if operation.WeightUnit != "kg" && operation.WeightUnit != "lb" {
		context.JSON(http.StatusBadRequest, gin.H{"error": "Invalid operation weight unit."})
		context.Abort()
		return
	}

	operation, err = database.CreateOperationInDB(operation)
	if err != nil {
		log.Println("Failed to create operation. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create operation."})
		context.Abort()
		return
	}

	operationObject, err := ConvertOperationToOperationObject(operation)
	if err != nil {
		log.Println("Failed to get convert operation to operation object. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get convert operation to operation object."})
		context.Abort()
		return
	}

	context.JSON(http.StatusCreated, gin.H{"message": "Operation created.", "operation": operationObject})
}

func APICreateOperationSetForUser(context *gin.Context) {
	// Initialize variables
	var operationSetCreationRequest models.OperationSetCreationRequest
	var operationSet models.OperationSet

	// Get user ID
	userID, err := middlewares.GetAuthUsername(context.GetHeader("Authorization"))
	if err != nil {
		log.Println("Failed to verify user ID. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to verify user ID."})
		context.Abort()
		return
	}

	// Parse creation request
	if err := context.ShouldBindJSON(&operationSetCreationRequest); err != nil {
		log.Println("Failed to parse creation request. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse creation request."})
		context.Abort()
		return
	}

	_, err = database.GetOperationByIDAndUserID(operationSetCreationRequest.OperationID, userID)
	if err != nil {
		log.Println("Failed to verify operation. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to verify operation."})
		context.Abort()
		return
	}

	operationSet.Distance = operationSetCreationRequest.Distance
	operationSet.Repetitions = operationSetCreationRequest.Repetitions
	operationSet.Time = operationSetCreationRequest.Time
	operationSet.Weight = operationSetCreationRequest.Weight
	operationSet.OperationID = operationSetCreationRequest.OperationID
	operationSet.ID = uuid.New()

	operationSet, err = database.CreateOperationSetInDB(operationSet)
	if err != nil {
		log.Println("Failed to create operation set. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create operation set."})
		context.Abort()
		return
	}

	operationSetObject, err := ConvertOperationSetToOperationSetObject(operationSet)
	if err != nil {
		log.Println("Failed to get convert operation set to operation set object. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get convert operation set to operation set object."})
		context.Abort()
		return
	}

	context.JSON(http.StatusCreated, gin.H{"message": "Operation set created.", "operation_set": operationSetObject})
}
