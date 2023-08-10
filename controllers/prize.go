package controllers

import (
	"aunefyren/treningheten/database"
	"aunefyren/treningheten/models"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func APIGetPrizes(context *gin.Context) {

	prizes, _, err := database.GetPrizes()
	if err != nil {
		log.Println("Failed to load prizes. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load prizes."})
		context.Abort()
		return
	}

	// Return group with owner and success message
	context.JSON(http.StatusOK, gin.H{"message": "Prizes retrieved.", "prizes": prizes})

}

func APIRegisterPrize(context *gin.Context) {

	// Create week request
	var prize models.PrizeCreationRequest

	// Parse request
	if err := context.ShouldBindJSON(&prize); err != nil {
		log.Println("Failed to parse request. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse request."})
		context.Abort()
		return
	}

	prize.Name = strings.TrimSpace(prize.Name)

	if len(prize.Name) < 5 || prize.Name == "" {
		context.JSON(http.StatusBadRequest, gin.H{"error": "Name must be five characters or more."})
		context.Abort()
		return
	}

	// Verify unique prize name and quantity
	_, prizeFound, err := database.GetPrizeByNameAndQuantity(prize.Name, prize.Quanitity)
	if err != nil {
		log.Println("Failed to check prizes. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to check prizes."})
		context.Abort()
		return
	} else if prizeFound {
		context.JSON(http.StatusBadRequest, gin.H{"error": "Prize already exists."})
		context.Abort()
		return
	}

	prizeDB := models.Prize{
		Name:      strings.TrimSpace(prize.Name),
		Quanitity: prize.Quanitity,
	}

	// Create prize in DB
	err = database.CreatePrizeInDB(prizeDB)
	if err != nil {
		log.Println("Failed to register prize in database. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to register prize in database."})
		context.Abort()
		return
	}

	// Return group with owner and success message
	context.JSON(http.StatusOK, gin.H{"message": "Prize created.", "prize": prize})

}
