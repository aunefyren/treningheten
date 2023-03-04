package controllers

import (
	"aunefyren/treningheten/database"
	"aunefyren/treningheten/middlewares"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// Get full workout calender for the week from the user, sanitize and update database
func APIRegisterSickleave(context *gin.Context) {

	// Get user ID
	userID, err := middlewares.GetAuthUsername(context.GetHeader("Authorization"))
	if err != nil {
		log.Println("Failed to verify user ID. Error: " + "Failed to verify user ID.")
		context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		context.Abort()
		return
	}

	// Get current season
	season, err := GetOngoingSeasonFromDB(time.Now())
	if err != nil {
		log.Println("Failed to verify current season status. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to verify current season status."})
		context.Abort()
		return
	}

	// Current time
	now := time.Now()

	// Check if within season
	if now.Before(season.Start) || now.After(season.End) {
		context.JSON(http.StatusBadRequest, gin.H{"error": "Season has not started yet."})
		context.Abort()
		return
	}

	// Verify goal doesn't exist within season
	goalStatus, goalID, err := database.VerifyUserGoalInSeason(userID, int(season.ID))
	if err != nil {
		log.Println("Failed to verify goal status. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to verify goal status."})
		context.Abort()
		return
	} else if !goalStatus {
		log.Println("User does not have a goal for season: " + strconv.Itoa(int(season.ID)))
		context.JSON(http.StatusBadRequest, gin.H{"error": "You don't have a goal set for this season."})
		context.Abort()
		return
	}

	// Check for sickleave left
	sickleaveArray, sickleaveFound, err := database.GetUnusedSickleaveForGoalWithinWeek(int(goalID))
	if err != nil {
		log.Println("Failed to verify sickleave. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify sickleave."})
		context.Abort()
		return
	} else if !sickleaveFound {
		context.JSON(http.StatusBadRequest, gin.H{"error": "You have no unused sickleave."})
		context.Abort()
		return
	}

	// Check if week is already sickleave
	sickleave, sickleaveFound, err := database.GetUsedSickleaveForGoalWithinWeek(now, int(goalID))
	if err != nil {
		log.Println("Failed to verify sickleave. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify sickleave."})
		context.Abort()
		return
	} else if sickleaveFound && sickleave.SickleaveUsed {
		context.JSON(http.StatusBadRequest, gin.H{"error": "This week is already marked as sickleave."})
		context.Abort()
		return
	}

	err = database.SetSickleaveToUsedByID(int(sickleaveArray[0].ID))
	if err != nil {
		log.Println("Failed to update sickleave. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update sickleave."})
		context.Abort()
		return
	}

	context.JSON(http.StatusOK, gin.H{"message": "Sickleave used."})

}
