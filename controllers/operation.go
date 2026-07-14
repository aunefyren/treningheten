package controllers

import (
	"errors"
	"net/http"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/aunefyren/treningheten/database"
	"github.com/aunefyren/treningheten/files"
	"github.com/aunefyren/treningheten/logger"
	"github.com/aunefyren/treningheten/middlewares"
	"github.com/aunefyren/treningheten/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

func ConvertOperationToOperationObject(operation models.Operation) (operationObject models.OperationObject, err error) {
	operationObject = models.OperationObject{}
	err = nil

	operationSets, err := database.GetOperationSetsByOperationID(operation.ID)
	if err != nil {
		logger.Log.Info("Failed to get operation sets using operation ID. Error: " + err.Error())
		return operationObject, errors.New("Failed to get operation sets using operation ID.")
	}

	operationSetObjects, err := ConvertOperationSetsToOperationSetObjects(operationSets)
	if err != nil {
		logger.Log.Info("Failed to convert operation sets to operation set objects. Error: " + err.Error())
		return operationObject, errors.New("Failed to convert operation sets to operation set objects.")
	}

	operationObject.OperationSets = operationSetObjects

	for _, operationSet := range operationSetObjects {
		if operationSet.StravaID != nil && *operationSet.StravaID != "" {
			operationObject.StravaID = operationSet.StravaID
			break
		}
	}

	if operation.ActionID != nil {
		action, err := database.GetActionByID(*operation.ActionID)
		if err != nil {
			logger.Log.Info("Failed to get action in database. Error: " + err.Error())
			return operationObject, errors.New("Failed to get action in database.")
		}

		operationObject.Action = &action
	} else {
		operationObject.Action = nil
	}

	// Flatten the linked gear's identity. Distance is left at zero here to avoid a
	// roll-up query per operation; the gear-list endpoint fills the computed total.
	if operation.GearID != nil {
		gear, err := database.GetGearByID(*operation.GearID)
		if err != nil {
			logger.Log.Info("Failed to get gear in database. Error: " + err.Error())
			return operationObject, errors.New("Failed to get gear in database.")
		} else if gear != nil {
			gearObject := ConvertGearToGearObject(*gear, 0)
			operationObject.Gear = &gearObject
		}
	}

	operationObject.Equipment = operation.Equipment
	operationObject.CreatedAt = operation.CreatedAt
	operationObject.DeletedAt = operation.DeletedAt
	operationObject.Enabled = operation.Enabled
	operationObject.Exercise = operation.ExerciseID
	operationObject.ID = operation.ID
	operationObject.UpdatedAt = operation.UpdatedAt
	operationObject.Type = operation.Type
	operationObject.WeightUnit = operation.WeightUnit
	operationObject.DistanceUnit = operation.DistanceUnit
	operationObject.Duration = operation.Duration
	operationObject.Note = operation.Note
	operationObject.Description = operation.Description
	operationObject.Tags = []string(operation.Tags)

	return
}

func ConvertOperationsToOperationObjects(operations []models.Operation) (operationObjects []models.OperationObject, err error) {
	operationObjects = []models.OperationObject{}
	err = nil

	for _, operation := range operations {
		operationObject, err := ConvertOperationToOperationObject(operation)
		if err != nil {
			logger.Log.Info("Failed to convert operation to operation object. Ignoring... Error: " + err.Error())
			continue
		}
		operationObjects = append(operationObjects, operationObject)
	}

	sort.Slice(operationObjects, func(i, j int) bool {
		return operationObjects[j].CreatedAt.After(operationObjects[i].CreatedAt)
	})

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
	operationSetObject.MovingTime = operationSet.MovingTime
	operationSetObject.UpdatedAt = operationSet.UpdatedAt
	operationSetObject.Weight = operationSet.Weight
	operationSetObject.StravaID = operationSet.StravaID
	operationSetObject.StravaStreams = operationSet.StravaStreams
	operationSetObject.StravaDataRetrievedAt = operationSet.StravaDataRetrievedAt

	return
}

func ConvertOperationSetsToOperationSetObjects(operationSets []models.OperationSet) (operationSetObjects []models.OperationSetObject, err error) {
	operationSetObjects = []models.OperationSetObject{}
	err = nil

	for _, operationSet := range operationSets {
		operationSetObject, err := ConvertOperationSetToOperationSetObject(operationSet)
		if err != nil {
			logger.Log.Info("Failed to convert operation set to operation set object. Ignoring... Error: " + err.Error())
			continue
		}
		operationSetObjects = append(operationSetObjects, operationSetObject)
	}

	sort.Slice(operationSetObjects, func(i, j int) bool {
		return operationSetObjects[j].CreatedAt.After(operationSetObjects[i].CreatedAt)
	})

	return
}

func APIGetOperationsForUser(context *gin.Context) {
	// Get user ID
	userID, err := middlewares.GetAuthUsername(context.GetHeader("Authorization"))
	if err != nil {
		logger.Log.Info("Failed to verify user ID. Error: " + "Failed to verify user ID.")
		context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		context.Abort()
		return
	}

	operations, err := database.GetOperationsByUserID(userID)
	if err != nil {
		logger.Log.Info("Failed to get operations. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get operations."})
		context.Abort()
		return
	}

	operationObjects, err := ConvertOperationsToOperationObjects(operations)
	if err != nil {
		logger.Log.Info("Failed to get convert operations to operation objects. Error: " + err.Error())
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
		logger.Log.Info("Failed to verify user ID. Error: " + "Failed to verify user ID.")
		context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		context.Abort()
		return
	}

	operation, err := database.GetOperationByIDAndUserID(operationIDUUID, userID)
	if err != nil {
		logger.Log.Info("Failed to get operation. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get operation."})
		context.Abort()
		return
	}

	operationObject, err := ConvertOperationToOperationObject(operation)
	if err != nil {
		logger.Log.Info("Failed to get convert operation to operation object. Error: " + err.Error())
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
		logger.Log.Info("Failed to verify user ID. Error: " + "Failed to verify user ID.")
		context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		context.Abort()
		return
	}

	operationID, okay := context.GetQuery("operation_id")
	if !okay {
		operationSets, err = database.GetOperationSetsByUserID(userID)
		if err != nil {
			logger.Log.Info("Failed to get operation sets. Error: " + err.Error())
			context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get operation sets."})
			context.Abort()
			return
		}
	} else {
		// Parse id
		operationIDUUID, err := uuid.Parse(operationID)
		if err != nil {
			logger.Log.Info("Failed to parse operation ID. Error: " + err.Error())
			context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse operation ID."})
			context.Abort()
			return
		}

		operationSets, err = database.GetOperationSetsByOperationIDAndUserID(operationIDUUID, userID)
		if err != nil {
			logger.Log.Info("Failed to get operation sets. Error: " + err.Error())
			context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get operation sets."})
			context.Abort()
			return
		}
	}

	operationSetObjects, err := ConvertOperationSetsToOperationSetObjects(operationSets)
	if err != nil {
		logger.Log.Info("Failed to get convert operation sets to operation set objects. Error: " + err.Error())
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
		logger.Log.Info("Failed to verify user ID. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to verify user ID."})
		context.Abort()
		return
	}

	// Parse creation request
	if err := context.ShouldBindJSON(&operationCreationRequest); err != nil {
		logger.Log.Info("Failed to parse creation request. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse creation request."})
		context.Abort()
		return
	}

	_, err = database.GetExerciseByIDAndUserID(operationCreationRequest.ExerciseID, userID)
	if err != nil {
		logger.Log.Info("Failed to verify exercise. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to verify exercise."})
		context.Abort()
		return
	}

	operation.ActionID = nil
	operation.DistanceUnit = strings.TrimSpace(operationCreationRequest.DistanceUnit)
	operation.ExerciseID = operationCreationRequest.ExerciseID
	operation.Type = strings.TrimSpace(operationCreationRequest.Type)
	operation.WeightUnit = strings.TrimSpace(operationCreationRequest.WeightUnit)
	operation.ID = uuid.New()

	if operationCreationRequest.Equipment != nil {
		trimmedEquipment := strings.TrimSpace(*operationCreationRequest.Equipment)

		if trimmedEquipment != "barbells" &&
			trimmedEquipment != "dumbbells" &&
			trimmedEquipment != "bands" &&
			trimmedEquipment != "rope" &&
			trimmedEquipment != "bench" &&
			trimmedEquipment != "treadmill" &&
			trimmedEquipment != "machine" {
			context.JSON(http.StatusBadRequest, gin.H{"error": "Invalid equipment type."})
			context.Abort()
			return
		}

		operation.Equipment = &trimmedEquipment
	} else {
		emptyString := ""
		operation.Equipment = &emptyString
	}

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
		logger.Log.Info("Failed to create operation. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create operation."})
		context.Abort()
		return
	}

	operationObject, err := ConvertOperationToOperationObject(operation)
	if err != nil {
		logger.Log.Info("Failed to get convert operation to operation object. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get convert operation to operation object."})
		context.Abort()
		return
	}

	// Give achievement to user for adding operation to exercise, ignore outcome
	go GiveUserAnAchievement(userID, uuid.MustParse("3d745d3a-b4b8-4194-bc72-653cfe4c351b"), time.Now(), 5)

	context.JSON(http.StatusCreated, gin.H{"message": "Operation created.", "operation": operationObject})
}

func APICreateOperationSetForUser(context *gin.Context) {
	// Initialize variables
	var operationSetCreationRequest models.OperationSetCreationRequest
	var operationSet models.OperationSet

	// Get user ID
	userID, err := middlewares.GetAuthUsername(context.GetHeader("Authorization"))
	if err != nil {
		logger.Log.Info("Failed to verify user ID. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to verify user ID."})
		context.Abort()
		return
	}

	// Parse creation request
	if err := context.ShouldBindJSON(&operationSetCreationRequest); err != nil {
		logger.Log.Info("Failed to parse creation request. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse creation request."})
		context.Abort()
		return
	}

	operation, err := database.GetOperationByIDAndUserID(operationSetCreationRequest.OperationID, userID)
	if err != nil {
		logger.Log.Info("Failed to verify operation. Error: " + err.Error())
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
		logger.Log.Info("Failed to create operation set. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create operation set."})
		context.Abort()
		return
	}

	operationSetObject, err := ConvertOperationSetToOperationSetObject(operationSet)
	if err != nil {
		logger.Log.Info("Failed to get convert operation set to operation set object. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get convert operation set to operation set object."})
		context.Abort()
		return
	}

	// ConvertOperationToOperationObject already loads every set from the DB,
	// including the one just created — so it must NOT be appended again (doing so
	// returns the new set twice).
	operationObject, err := ConvertOperationToOperationObject(operation)
	if err != nil {
		logger.Log.Info("Failed to get convert operation to operation object. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get convert operation to operation object."})
		context.Abort()
		return
	}

	context.JSON(http.StatusCreated, gin.H{"message": "Operation set created.", "operation_set": operationSetObject, "operation": operationObject})
}

func APIUpdateOperation(context *gin.Context) {
	// Initialize variables
	var operationUpdateRequest models.OperationUpdateRequest
	var operation models.Operation

	// Get user ID
	userID, err := middlewares.GetAuthUsername(context.GetHeader("Authorization"))
	if err != nil {
		logger.Log.Info("Failed to verify user ID. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to verify user ID."})
		context.Abort()
		return
	}

	var operationID = context.Param("operation_id")
	operationIDUUID, err := uuid.Parse(operationID)
	if err != nil {
		logger.Log.Info("Failed to verify operation ID. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to verify operation ID."})
		context.Abort()
		return
	}

	// Parse update request
	if err := context.ShouldBindJSON(&operationUpdateRequest); err != nil {
		logger.Log.Info("Failed to parse update request. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse update request."})
		context.Abort()
		return
	}

	operation, err = database.GetOperationByIDAndUserID(operationIDUUID, userID)
	if err != nil {
		logger.Log.Info("Failed to get operation. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get operation."})
		context.Abort()
		return
	}

	if operationUpdateRequest.Action != "" {
		action, err := database.GetActionByName(strings.TrimSpace(operationUpdateRequest.Action))
		if err != nil {
			logger.Log.Info("Failed to get action by name. Error: " + err.Error())
			context.JSON(http.StatusBadRequest, gin.H{"error": "Choose a valid exercise."})
			context.Abort()
			return
		}

		operation.ActionID = &action.ID
		operation.Action = &action
		operation.Type = action.Type
	} else {
		operation.ActionID = nil
		operation.Action = nil
		operation.Type = strings.TrimSpace(operationUpdateRequest.Type)
	}

	operation.DistanceUnit = strings.TrimSpace(operationUpdateRequest.DistanceUnit)
	operation.WeightUnit = strings.TrimSpace(operationUpdateRequest.WeightUnit)

	if operationUpdateRequest.Equipment != "" {
		trimmedEquipment := strings.TrimSpace(operationUpdateRequest.Equipment)

		if trimmedEquipment != "barbells" &&
			trimmedEquipment != "dumbbells" &&
			trimmedEquipment != "bands" &&
			trimmedEquipment != "rope" &&
			trimmedEquipment != "bench" &&
			trimmedEquipment != "treadmill" &&
			trimmedEquipment != "machine" {
			context.JSON(http.StatusBadRequest, gin.H{"error": "Invalid equipment type."})
			context.Abort()
			return
		}

		operation.Equipment = &trimmedEquipment
	} else {
		emptyString := ""
		operation.Equipment = &emptyString
	}

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

	// Tags are replaced only when explicitly provided; validate against the
	// controlled vocabulary and de-duplicate.
	if operationUpdateRequest.Tags != nil {
		validatedTags := []string{}
		for _, tag := range *operationUpdateRequest.Tags {
			trimmedTag := strings.TrimSpace(tag)
			if trimmedTag == "" {
				continue
			}
			if !models.IsValidTag(trimmedTag) {
				context.JSON(http.StatusBadRequest, gin.H{"error": "Invalid tag: " + trimmedTag})
				context.Abort()
				return
			}
			if !slices.Contains(validatedTags, trimmedTag) {
				validatedTags = append(validatedTags, trimmedTag)
			}
		}
		operation.Tags = models.TagList(validatedTags)
	}

	// Description is replaced only when explicitly provided; an empty value clears it.
	if operationUpdateRequest.Description != nil {
		trimmedDescription := strings.TrimSpace(*operationUpdateRequest.Description)
		if trimmedDescription == "" {
			operation.Description = nil
		} else {
			operation.Description = &trimmedDescription
		}
	}

	operation, err = database.UpdateOperationInDB(operation)
	if err != nil {
		logger.Log.Info("Failed to update operation. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update operation."})
		context.Abort()
		return
	}

	operationObject, err := ConvertOperationToOperationObject(operation)
	if err != nil {
		logger.Log.Info("Failed to get convert operation to operation object. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get convert operation to operation object."})
		context.Abort()
		return
	}

	context.JSON(http.StatusOK, gin.H{"message": "Operation updated.", "operation": operationObject})
}

func APIUpdateOperationSet(context *gin.Context) {
	// Initialize variables
	var operationSetUpdateRequest models.OperationSetUpdateRequest
	var operationSet models.OperationSet

	// Get user ID
	userID, err := middlewares.GetAuthUsername(context.GetHeader("Authorization"))
	if err != nil {
		logger.Log.Info("Failed to verify user ID. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to verify user ID."})
		context.Abort()
		return
	}

	var operationSetID = context.Param("operation_set_id")
	operationSetIDUUID, err := uuid.Parse(operationSetID)
	if err != nil {
		logger.Log.Info("Failed to verify operation set ID. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to verify operation set ID."})
		context.Abort()
		return
	}

	// Parse update request
	if err := context.ShouldBindJSON(&operationSetUpdateRequest); err != nil {
		logger.Log.Info("Failed to parse update request. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse update request."})
		context.Abort()
		return
	}

	operationSet, err = database.GetOperationSetByIDAndUserID(operationSetIDUUID, userID)
	if err != nil {
		logger.Log.Info("Failed to get operation set. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to get operation set."})
		context.Abort()
		return
	}

	operationSet.Distance = operationSetUpdateRequest.Distance
	operationSet.Repetitions = operationSetUpdateRequest.Repetitions
	operationSet.MovingTime = operationSetUpdateRequest.MovingTime
	operationSet.Weight = operationSetUpdateRequest.Weight

	operationSet, err = database.UpdateOperationSetInDB(operationSet)
	if err != nil {
		logger.Log.Info("Failed to update operation set. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update operation set."})
		context.Abort()
		return
	}

	operationSetObject, err := ConvertOperationSetToOperationSetObject(operationSet)
	if err != nil {
		logger.Log.Info("Failed to get convert operation set to operation set object. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get convert operation set to operation set object."})
		context.Abort()
		return
	}

	operation, err := database.GetOperationByIDAndUserID(operationSet.OperationID, userID)
	if err != nil {
		logger.Log.Info("Failed to get operation. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to get operation."})
		context.Abort()
		return
	}

	operationObject, err := ConvertOperationToOperationObject(operation)
	if err != nil {
		logger.Log.Info("Failed to get convert operation to operation object. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get convert operation to operation object."})
		context.Abort()
		return
	}

	context.JSON(http.StatusOK, gin.H{"message": "Operation set updated.", "operation_set": operationSetObject, "operation": operationObject})
}

func APIDeleteOperation(context *gin.Context) {
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
		logger.Log.Info("Failed to verify user ID. Error: " + "Failed to verify user ID.")
		context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		context.Abort()
		return
	}

	operation, err := database.GetOperationByIDAndUserID(operationIDUUID, userID)
	if err != nil {
		logger.Log.Info("Failed to get operation. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get operation."})
		context.Abort()
		return
	}

	operation.Enabled = false
	operation, err = database.UpdateOperationInDB(operation)
	if err != nil {
		logger.Log.Info("Failed to update operation in the database. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update operation in the database."})
		context.Abort()
		return
	}

	operationObject, err := ConvertOperationToOperationObject(operation)
	if err != nil {
		logger.Log.Info("Failed to get convert operation to operation object. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get convert operation to operation object."})
		context.Abort()
		return
	}

	context.JSON(http.StatusOK, gin.H{"message": "Operation deleted.", "operation": operationObject})
}

func APIDeleteOperationSet(context *gin.Context) {
	var operationSetID = context.Param("operation_set_id")

	operationSetIDUUID, err := uuid.Parse(operationSetID)
	if err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse ID."})
		context.Abort()
		return
	}

	// Get user ID
	userID, err := middlewares.GetAuthUsername(context.GetHeader("Authorization"))
	if err != nil {
		logger.Log.Info("Failed to verify user ID. Error: " + "Failed to verify user ID.")
		context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		context.Abort()
		return
	}

	operationSet, err := database.GetOperationSetByIDAndUserID(operationSetIDUUID, userID)
	if err != nil {
		logger.Log.Info("Failed to get operation set. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get operation set."})
		context.Abort()
		return
	}

	operationSet.Enabled = false
	operationSet, err = database.UpdateOperationSetInDB(operationSet)
	if err != nil {
		logger.Log.Info("Failed to update operation in the database. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update operation in the database."})
		context.Abort()
		return
	}

	operationSetObject, err := ConvertOperationSetToOperationSetObject(operationSet)
	if err != nil {
		logger.Log.Info("Failed to get convert operation set to operation set object. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get convert operation set to operation set object."})
		context.Abort()
		return
	}

	operation, err := database.GetOperationByIDAndUserID(operationSet.OperationID, userID)
	if err != nil {
		logger.Log.Info("Failed to get operation from the database. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get operation from the database."})
		context.Abort()
		return
	}

	operationObject, err := ConvertOperationToOperationObject(operation)
	if err != nil {
		logger.Log.Info("Failed to get convert operation to operation object. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get convert operation to operation object."})
		context.Abort()
		return
	}

	context.JSON(http.StatusOK, gin.H{"message": "Operation set deleted.", "operation_set": operationSetObject, "operation": operationObject})
}

func APIGetActions(context *gin.Context) {
	actions := []models.Action{}
	var err error

	// Get user ID
	userID, err := middlewares.GetAuthUsername(context.GetHeader("Authorization"))
	if err != nil {
		logger.Log.Info("Failed to verify user ID. Error: " + "Failed to verify user ID.")
		context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		context.Abort()
		return
	}

	experiencedString, okay := context.GetQuery("experienced")
	if !okay || strings.ToLower(experiencedString) != "true" {
		actions, err = database.GetAllEnabledActions()
		if err != nil {
			logger.Log.Info("Failed to get actions. Error: " + err.Error())
			context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get actions."})
			context.Abort()
			return
		}
	} else {
		actions, err = database.GetActionsDoneUsingUserID(userID)
		if err != nil {
			logger.Log.Info("Failed to get actions done by user. Error: " + err.Error())
			context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get actions done by user."})
			context.Abort()
			return
		}
	}

	context.JSON(http.StatusOK, gin.H{"message": "Actions retrieved.", "actions": actions})
}

func APICreateAction(context *gin.Context) {
	// Initialize variables
	var actionCreationRequest models.ActionCreationRequest
	var action models.Action

	// Parse creation request
	err := context.ShouldBindJSON(&actionCreationRequest)
	if err != nil {
		logger.Log.Info("Failed to parse creation request. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse creation request."})
		context.Abort()
		return
	}

	caserEng := cases.Title(language.English)
	caserNor := cases.Title(language.Norwegian)

	action.BodyPart = caserEng.String(strings.TrimSpace(actionCreationRequest.BodyPart))
	action.Name = caserEng.String(strings.TrimSpace(actionCreationRequest.Name))
	action.NorwegianName = caserNor.String(strings.TrimSpace(actionCreationRequest.NorwegianName))

	action.Type = strings.TrimSpace(actionCreationRequest.Type)
	action.Description = strings.TrimSpace(actionCreationRequest.Description)
	action.ID = uuid.New()

	if action.Type != "lifting" && action.Type != "moving" && action.Type != "timing" {
		context.JSON(http.StatusBadRequest, gin.H{"error": "Invalid exercise type."})
		context.Abort()
		return
	}

	if action.Name == "" && action.NorwegianName == "" {
		context.JSON(http.StatusBadRequest, gin.H{"error": "Both names can't be empty."})
		context.Abort()
		return
	}

	if len(action.Description) > 255 {
		context.JSON(http.StatusBadRequest, gin.H{"error": "Invalid description length."})
		context.Abort()
		return
	}

	action, err = database.CreateActionInDB(action)
	if err != nil {
		logger.Log.Info("Failed to create action. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create action."})
		context.Abort()
		return
	}

	context.JSON(http.StatusCreated, gin.H{"message": "Action created.", "action": action})
}

func APIGetActionStatistics(context *gin.Context) {
	var actionIDString = context.Param("action_id")
	layout := "2006-01-02T15:04:05Z"

	actionID, err := uuid.Parse(actionIDString)
	if err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse ID."})
		context.Abort()
		return
	}

	action, err := database.GetActionByID(actionID)
	if err != nil {
		logger.Log.Error("Failed to get action by ID. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to get action by ID."})
		context.Abort()
		return
	}

	// Get user ID
	userID, err := middlewares.GetAuthUsername(context.GetHeader("Authorization"))
	if err != nil {
		logger.Log.Error("Failed to verify user ID. Error: " + "Failed to verify user ID.")
		context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		context.Abort()
		return
	}

	startTimeString, okay := context.GetQuery("start")
	if !okay {
		context.JSON(http.StatusBadRequest, gin.H{"error": "Start query is missing."})
		context.Abort()
		return
	}
	startTime, err := time.Parse(layout, startTimeString)
	if err != nil {
		logger.Log.Error("Failed to parse start time string. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse start time string."})
		context.Abort()
		return
	}

	endTimeString, okay := context.GetQuery("end")
	if !okay {
		context.JSON(http.StatusBadRequest, gin.H{"error": "End query is missing."})
		context.Abort()
		return
	}
	endTime, err := time.Parse(layout, endTimeString)
	if err != nil {
		logger.Log.Error("Failed to parse end time string. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse end time string."})
		context.Abort()
		return
	}

	exerciseDays, err := database.GetExerciseDaysBetweenDatesUsingDatesAndUserID(userID, startTime, endTime)
	if err != nil {
		logger.Log.Error("Failed to get exercise days. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get exercise days."})
		context.Abort()
		return
	}

	finalOperationObjects := []models.OperationObject{}
	// Distinct sessions the matched operations belong to. The soundtrack is a
	// session-level fact, so media stats aggregate per session (counted once) rather
	// than per matching operation.
	matchedSessionIDs := map[uuid.UUID]bool{}
	actionStatistics := models.ActionStatistics{}
	actionStatisticsCompilation := models.StatisticsCompilation{}

	for _, exerciseDay := range exerciseDays {
		if exerciseDay.Enabled {
			finalExercises := []models.Exercise{}
			exercises, err := database.GetExerciseByExerciseDayID(exerciseDay.ID)
			if err != nil {
				logger.Log.Warn("Failed to get exercises for day. Error: " + err.Error())
				continue
			}

			for _, exercise := range exercises {
				if exercise.Enabled && exercise.IsOn {
					finalExercises = append(finalExercises, exercise)
				}
			}

			for _, exercise := range finalExercises {
				operations, err := database.GetOperationsByExerciseID(exercise.ID)
				if err != nil {
					logger.Log.Warn("Failed to get operations for exercise. Error: " + err.Error())
					continue
				}

				operationObjects, err := ConvertOperationsToOperationObjects(operations)
				if err != nil {
					logger.Log.Warn("Failed to convert operations to operation objects. Error: " + err.Error())
					continue
				}

				for _, operationObject := range operationObjects {
					if operationObject.Action != nil && operationObject.Action.ID == actionID {
						finalOperationObjects = append(finalOperationObjects, operationObject)
						matchedSessionIDs[operationObject.Exercise] = true
					}
				}

			}
		}
	}

	statisticsTops := models.StatisticsTopCompilation{}
	distanceTop := 0.0
	repetitionTop := 0.0
	timeTop := 0
	weightTop := 0.0

	statisticsSums := models.StatisticsSumCompilation{}
	statisticsSums.Distance = 0.0
	statisticsSums.Repetition = 0.0
	statisticsSums.Operations = 0
	statisticsSums.Time = 0
	statisticsSums.Weight = 0.0

	for _, operation := range finalOperationObjects {
		if !operation.Enabled {
			continue
		}

		statisticsSums.Operations += 1
		distance := 0.0
		repetition := 0.0
		timeSum := 0
		weight := 0.0

		for _, set := range operation.OperationSets {
			if !set.Enabled {
				continue
			}

			if set.Repetitions != nil {
				repetition += *set.Repetitions
			}
			if set.Time != nil {
				timeSum += int(*set.Time)
			}
			if set.Weight != nil {
				weight += *set.Weight
			}
			if set.Distance != nil {
				distance = *set.Distance
			}
		}

		statisticsSums.Repetition += repetition
		statisticsSums.Time += int64(timeSum)
		statisticsSums.Weight += weight
		statisticsSums.Distance += distance

		if (statisticsTops.Distance == nil || distance > distanceTop) && distance != 0.0 {
			statisticsTops.Distance = &operation
			distanceTop = distance
		}

		if (statisticsTops.Repetition == nil || repetition > repetitionTop) && repetition != 0.0 {
			statisticsTops.Repetition = &operation
			repetitionTop = repetition
		}

		if (statisticsTops.Time == nil || timeSum > timeTop) && timeSum != 0 {
			statisticsTops.Time = &operation
			timeTop = timeSum
		}

		if (statisticsTops.Weight == nil || weight > weightTop) && weight != 0.0 {
			statisticsTops.Weight = &operation
			weightTop = weight
		}
	}

	actionStatisticsCompilation.Sums = statisticsSums

	if statisticsSums.Operations > 0 {
		statisticsAverages := models.StatisticsAverageCompilation{
			Distance:   (statisticsSums.Distance / float64(statisticsSums.Operations)),
			Time:       int64(float64(statisticsSums.Time) / float64(statisticsSums.Operations)),
			Repetition: (statisticsSums.Repetition / float64(statisticsSums.Operations)),
			Weight:     (statisticsSums.Weight / float64(statisticsSums.Operations)),
		}
		actionStatisticsCompilation.Averages = statisticsAverages
	}

	actionStatisticsCompilation.Tops = statisticsTops

	actionStatistics.Statistics = actionStatisticsCompilation
	actionStatistics.Operations = finalOperationObjects
	actionStatistics.Action = action

	// Overlay soundtrack stats when media is enabled. Gather the distinct matched
	// sessions' playback rows and aggregate; a nil result (no rows) keeps the block
	// absent so the frontend renders it only when there is something to show.
	if files.ConfigFile.Media.Enabled {
		playback := []models.MediaPlaybackObject{}
		for sessionID := range matchedSessionIDs {
			rows, err := database.GetMediaPlaybackForExercise(sessionID)
			if err != nil {
				logger.Log.Warn("Failed to get media playback for session. Error: " + err.Error())
				continue
			}
			playback = append(playback, ConvertMediaPlaybackToObjects(rows)...)
		}
		actionStatistics.Media = computeActionMediaStatistics(playback)
	}

	context.JSON(http.StatusOK, gin.H{"message": "Action statistics retrieved.", "statistics": actionStatistics})
}

// computeActionMediaStatistics aggregates a period's matched-session playback rows into
// the soundtrack overlay. Songs drive the track/artist tallies and music listening time.
// Spoken audio is split by type: podcasts group by show (TopPodcast.Count = that show's
// episode count) and audiobooks group by book (Artist holds the author), each with its
// own listening time — so a spoken-audio soundtrack reads as more than one number.
// SpokenTime is kept as the podcast+audiobook total for compatibility. A play's span is
// its TrackLength when known, else the StartedAt→EndedAt gap. Returns nil when there are
// no rows so the caller leaves the block out entirely.
func computeActionMediaStatistics(playback []models.MediaPlaybackObject) *models.ActionMediaStatistics {
	if len(playback) == 0 {
		return nil
	}

	stats := models.ActionMediaStatistics{}
	trackCounts := map[string]*models.MediaCountItem{}
	artistCounts := map[string]*models.MediaCountItem{}
	podcastShows := map[string]*models.MediaCountItem{}
	audiobooks := map[string]*models.MediaCountItem{}
	podcastEpisodes := map[string]bool{}

	for _, row := range playback {
		span := int64(0)
		if row.TrackLength != nil && *row.TrackLength > 0 {
			span = *row.TrackLength
		} else if row.EndedAt != nil {
			if seconds := int64(row.EndedAt.Sub(row.StartedAt).Seconds()); seconds > 0 {
				span = seconds
			}
		}

		artist := ""
		if row.Artist != nil {
			artist = *row.Artist
		}
		album := ""
		if row.Album != nil {
			album = *row.Album
		}
		artwork := ""
		if row.ArtworkURL != nil {
			artwork = *row.ArtworkURL
		}

		switch row.MediaType {
		case models.MediaTypePodcast:
			stats.SpokenTime += span
			stats.PodcastTime += span
			// Group by show. Providers put the show in the artist (ABS DisplayAuthor,
			// Plex GrandparentTitle); fall back to album, then the episode title.
			show := firstNonEmpty(artist, album, row.Title)
			podcastEpisodes[show+"\x00"+row.Title] = true
			if item := podcastShows[show]; item != nil {
				item.Count += 1
				if item.Artwork == "" {
					item.Artwork = artwork
				}
			} else {
				podcastShows[show] = &models.MediaCountItem{Title: show, Count: 1, Artwork: artwork}
			}
			continue
		case models.MediaTypeAudiobook:
			stats.SpokenTime += span
			stats.AudiobookTime += span
			// Group by book title; the artist is the author.
			if item := audiobooks[row.Title]; item != nil {
				item.Count += 1
				if item.Artwork == "" {
					item.Artwork = artwork
				}
			} else {
				audiobooks[row.Title] = &models.MediaCountItem{Title: row.Title, Artist: artist, Count: 1, Artwork: artwork}
			}
			continue
		}

		// Song.
		stats.Songs += 1
		stats.ListeningTime += span

		trackKey := row.Title + "\x00" + artist
		if item := trackCounts[trackKey]; item != nil {
			item.Count += 1
			if item.Artwork == "" {
				item.Artwork = artwork
			}
		} else {
			trackCounts[trackKey] = &models.MediaCountItem{Title: row.Title, Artist: artist, Count: 1, Artwork: artwork}
		}

		if artist != "" {
			if item := artistCounts[artist]; item != nil {
				item.Count += 1
				if item.Artwork == "" {
					item.Artwork = artwork
				}
			} else {
				artistCounts[artist] = &models.MediaCountItem{Title: artist, Count: 1, Artwork: artwork}
			}
		}
	}

	stats.UniqueArtists = len(artistCounts)
	stats.TopTrack = mostPlayedItem(trackCounts)
	stats.TopArtist = mostPlayedItem(artistCounts)
	stats.PodcastEpisodes = len(podcastEpisodes)
	stats.Audiobooks = len(audiobooks)
	stats.TopPodcast = mostPlayedItem(podcastShows)
	stats.TopAudiobook = mostPlayedItem(audiobooks)

	return &stats
}

// firstNonEmpty returns the first non-empty (after trimming) value, or "" if all blank.
// Used to pick a podcast show label from the artist/album/title fallbacks.
func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

// mostPlayedItem returns the highest-count tally, breaking ties by Title for a stable
// result. Returns nil for an empty map (e.g. a spoken-audio-only period has no songs).
func mostPlayedItem(items map[string]*models.MediaCountItem) *models.MediaCountItem {
	var best *models.MediaCountItem
	for _, item := range items {
		if best == nil || item.Count > best.Count || (item.Count == best.Count && item.Title < best.Title) {
			best = item
		}
	}
	return best
}

func APISyncStravaOperationSet(context *gin.Context) {
	var operationSetID = context.Param("operation_set_id")

	if !files.ConfigFile.StravaEnabled {
		context.JSON(http.StatusBadRequest, gin.H{"error": "Strava is disabled on the tenant."})
		context.Abort()
		return
	}

	operationSetIDUUID, err := uuid.Parse(operationSetID)
	if err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse ID."})
		context.Abort()
		return
	}

	// Get user ID
	userID, err := middlewares.GetAuthUsername(context.GetHeader("Authorization"))
	if err != nil {
		logger.Log.Info("Failed to verify user ID. Error: " + "Failed to verify user ID.")
		context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		context.Abort()
		return
	}

	operationSet, err := database.GetOperationSetByIDAndUserID(operationSetIDUUID, userID)
	if err != nil {
		logger.Log.Info("Failed to get operation set. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get operation set."})
		context.Abort()
		return
	}

	if operationSet.StravaID == nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": "Operation set is not connected to a Strava activity."})
		context.Abort()
		return
	}

	user, err := database.GetAllUserInformation(userID)
	if err != nil {
		logger.Log.Info("Failed to get user object. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user object."})
		context.Abort()
		return
	}

	token, err := StravaGetAuthorizationForUser(user)
	if err != nil {
		logger.Log.Info("Failed to get authorize user toward Strava. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get authorize user toward Strava."})
		context.Abort()
		return
	}

	activity, err := StravaGetActivity(token, *operationSet.StravaID)
	if err != nil {
		logger.Log.Info("Failed to get activity from Strava. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get activity from Strava."})
		context.Abort()
		return
	}

	err = StravaSyncActivityForUser(activity, user, token, true)
	if err != nil {
		logger.Log.Info("Failed to update exercise with Strava data. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update exercise with Strava data."})
		context.Abort()
		return
	}

	operationSet, err = database.GetOperationSetByIDAndUserID(operationSetIDUUID, userID)
	if err != nil {
		logger.Log.Info("Failed to get operation set. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get operation set."})
		context.Abort()
		return
	}

	operationSetObject, err := ConvertOperationSetToOperationSetObject(operationSet)
	if err != nil {
		logger.Log.Info("Failed to get convert operation set to operation set object. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get convert operation set to operation set object."})
		context.Abort()
		return
	}

	operation, err := database.GetOperationByIDAndUserID(operationSet.OperationID, userID)
	if err != nil {
		logger.Log.Info("Failed to get operation. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to get operation."})
		context.Abort()
		return
	}

	operationObject, err := ConvertOperationToOperationObject(operation)
	if err != nil {
		logger.Log.Info("Failed to get convert operation to operation object. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get convert operation to operation object."})
		context.Abort()
		return
	}

	context.JSON(http.StatusOK, gin.H{"message": "Operation set synced.", "operation_set": operationSetObject, "operation": operationObject})
}
