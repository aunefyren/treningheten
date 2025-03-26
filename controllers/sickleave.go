package controllers

import (
	"aunefyren/treningheten/database"
	"aunefyren/treningheten/middlewares"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Get full workout calender for the week from the user, sanitize and update database
func APIRegisterSickleave(context *gin.Context) {
	var seasonID = context.Param("season_id")

	seasonIDUUIID, err := uuid.Parse(seasonID)
	if err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse ID."})
		context.Abort()
		return
	}

	// Get user ID
	userID, err := middlewares.GetAuthUsername(context.GetHeader("Authorization"))
	if err != nil {
		log.Info("Failed to verify user ID. Error: " + "Failed to verify user ID.")
		context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		context.Abort()
		return
	}

	// Get current season
	season, err := database.GetSeasonByID(seasonIDUUIID)
	if err != nil {
		log.Info("Failed to verify current season status. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to verify current season status."})
		context.Abort()
		return
	} else if season == nil {
		log.Info("Failed to verify current season status. Error: No active or future seasons found.")
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to verify current season status."})
		context.Abort()
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
	goalStatus, goalID, err := database.VerifyUserGoalInSeason(userID, season.ID)
	if err != nil {
		log.Info("Failed to verify goal status. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to verify goal status."})
		context.Abort()
		return
	} else if !goalStatus {
		log.Info("User does not have a goal for season: " + season.ID.String())
		context.JSON(http.StatusBadRequest, gin.H{"error": "You don't have a goal set for this season."})
		context.Abort()
		return
	}

	// Check for sickleave left
	sickleaveArray, sickleaveFound, err := database.GetUnusedSickleaveForGoalWithinWeek(goalID)
	if err != nil {
		log.Info("Failed to verify sick leave. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify sick leave."})
		context.Abort()
		return
	} else if !sickleaveFound {
		context.JSON(http.StatusBadRequest, gin.H{"error": "You have no unused sickleave."})
		context.Abort()
		return
	}

	// Check if week is already sickleave
	sickLeave, err := database.GetUsedSickleaveForGoalWithinWeek(now, goalID)
	if err != nil {
		log.Info("Failed to verify sick leave. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify sick leave."})
		context.Abort()
		return
	} else if sickLeave != nil && sickLeave.Used {
		context.JSON(http.StatusBadRequest, gin.H{"error": "This week is already marked as sick leave."})
		context.Abort()
		return
	}

	err = database.SetSickleaveToUsedByID(sickleaveArray[0].ID)
	if err != nil {
		log.Info("Failed to update sick leave. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update sick leave."})
		context.Abort()
		return
	}

	// Give achievement to user
	err = GiveUserAnAchievement(userID, uuid.MustParse("420b020c-2cad-4898-bb94-d86dc0031203"), now)
	if err != nil {
		log.Info("Failed to give achievement for user '" + userID.String() + "'. Ignoring. Error: " + err.Error())
	}

	context.JSON(http.StatusOK, gin.H{"message": "Sick leave used."})

}
