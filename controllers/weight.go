package controllers

import (
	"net/http"
	"sort"

	"github.com/aunefyren/treningheten/database"
	"github.com/aunefyren/treningheten/logger"
	"github.com/aunefyren/treningheten/middlewares"
	"github.com/aunefyren/treningheten/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// maxBodyWeightKg is the upper bound accepted for a weight entry. Weights carry no unit
// and are implicitly kilograms; the bound only rejects obvious typos.
const maxBodyWeightKg = 500

// validBodyWeight reports whether weight is a plausible body weight in kilograms.
func validBodyWeight(weight float64) bool {
	return weight > 0 && weight <= maxBodyWeightKg
}

func APIGetWeightsForUser(context *gin.Context) {
	// Get user ID
	userID, err := middlewares.GetAuthUsername(context.GetHeader("Authorization"))
	if err != nil {
		logger.Log.Error("Failed to verify user ID. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to verify user ID."})
		context.Abort()
		return
	}

	weights, err := database.GetEnabledWeightsForUser(userID)
	if err != nil {
		logger.Log.Error("Failed to get weights for user. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get weights for user."})
		context.Abort()
		return
	}

	sort.Slice(weights, func(i, j int) bool {
		return weights[j].Date.Before(weights[i].Date)
	})

	context.JSON(http.StatusOK, gin.H{"message": "Weights retrieved.", "weights": weights})
}

func APIGetWeightForUser(context *gin.Context) {
	// Get user ID
	userID, err := middlewares.GetAuthUsername(context.GetHeader("Authorization"))
	if err != nil {
		logger.Log.Error("Failed to verify user ID. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to verify user ID."})
		context.Abort()
		return
	}

	var weightIDString = context.Param("weight_id")
	weightID, err := uuid.Parse(weightIDString)
	if err != nil {
		logger.Log.Error("Failed to verify weight ID. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to verify weight ID."})
		context.Abort()
		return
	}

	weight, err := database.GetEnabledWeightsByWeightIDAndUserID(userID, weightID)
	if err != nil {
		logger.Log.Warn("Failed to get weight. Error: " + err.Error())
		context.JSON(http.StatusNotFound, gin.H{"error": "Failed to get weight."})
		context.Abort()
		return
	}

	context.JSON(http.StatusOK, gin.H{"message": "Weight retrieved.", "weight": weight})
}

func APICreateWeightForUser(context *gin.Context) {
	// Initialize variables
	var weightCreationRequest models.WeightValueCreationRequest
	var weight models.WeightValue

	// Get user ID
	userID, err := middlewares.GetAuthUsername(context.GetHeader("Authorization"))
	if err != nil {
		logger.Log.Error("Failed to verify user ID. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to verify user ID."})
		context.Abort()
		return
	}

	// Parse creation request
	if err := context.ShouldBindJSON(&weightCreationRequest); err != nil {
		logger.Log.Error("Failed to parse creation request. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse creation request."})
		context.Abort()
		return
	}

	if !validBodyWeight(weightCreationRequest.Weight) {
		context.JSON(http.StatusBadRequest, gin.H{"error": "Weight must be above 0 and at most 500 kg."})
		context.Abort()
		return
	}

	weight.ID = uuid.New()
	weight.UserID = userID
	weight.Date = weightCreationRequest.Date
	weight.Weight = weightCreationRequest.Weight

	weight, err = database.CreateWeightInDB(weight)
	if err != nil {
		logger.Log.Error("Failed to create weight in DB. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create weight in DB."})
		context.Abort()
		return
	}

	context.JSON(http.StatusCreated, gin.H{"message": "Weight created.", "weight": weight})
}

func APIDeleteWeightForUser(context *gin.Context) {
	// Get user ID
	userID, err := middlewares.GetAuthUsername(context.GetHeader("Authorization"))
	if err != nil {
		logger.Log.Error("Failed to verify user ID. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to verify user ID."})
		context.Abort()
		return
	}

	var weightIDString = context.Param("weight_id")
	weightID, err := uuid.Parse(weightIDString)
	if err != nil {
		logger.Log.Error("Failed to verify weight ID. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to verify weight ID."})
		context.Abort()
		return
	}

	weight, err := database.GetEnabledWeightsByWeightIDAndUserID(userID, weightID)
	if err != nil {
		logger.Log.Warn("Failed to get weight. Error: " + err.Error())
		context.JSON(http.StatusNotFound, gin.H{"error": "Failed to get weight."})
		context.Abort()
		return
	}

	weight.Enabled = false

	_, err = database.UpdateWeightInDB(weight)
	if err != nil {
		logger.Log.Error("Failed to update weight in DB. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update weight in DB."})
		context.Abort()
		return
	}

	context.JSON(http.StatusOK, gin.H{"message": "Weight deleted."})
}
