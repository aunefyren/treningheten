package controllers

import (
	"aunefyren/treningheten/database"
	"aunefyren/treningheten/middlewares"
	"aunefyren/treningheten/models"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

func APIRegisterGoalToSeason(context *gin.Context) {

	// Create goal request
	var goal models.GoalCreationRequest
	var goalDB models.Goal

	if err := context.ShouldBindJSON(&goal); err != nil {
		log.Println("Failed to parse request. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse request."})
		context.Abort()
		return
	}

	// Get user ID
	userID, err := middlewares.GetAuthUsername(context.GetHeader("Authorization"))
	if err != nil {
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

	// Verify goal doesn't exist within season
	goalStatus, _, err := database.VerifyUserGoalInSeason(userID, int(season.ID))
	if err != nil {
		log.Println("Failed to verify goal status. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to verify goal status."})
		context.Abort()
		return
	} else if goalStatus {
		log.Println("User already has a goal for season: " + strconv.Itoa(int(season.ID)))
		context.JSON(http.StatusBadRequest, gin.H{"error": "You already have a goal this season."})
		context.Abort()
		return
	}

	// Verify exercise interval
	if goal.ExerciseInterval == 0 || goal.ExerciseInterval > 21 {
		context.JSON(http.StatusBadRequest, gin.H{"error": "Your exercise goal must be between 1 and 21."})
		context.Abort()
		return
	}

	// Finalize goal object
	goalDB.Competing = goal.Competing
	goalDB.ExerciseInterval = goal.ExerciseInterval
	goalDB.Season = int(season.ID)
	goalDB.User = userID

	// Create goal in DB
	err = database.CreateGoalInDB(goalDB)
	if err != nil {
		log.Println("Failed to create goal for season. Error: " + strconv.Itoa(int(season.ID)))
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to create goal for season."})
		context.Abort()
		return
	}

	context.JSON(http.StatusCreated, gin.H{"message": "Goal created."})
}

func ConvertGoalToGoalObject(goal models.Goal) (models.GoalObject, error) {

	var goalObject models.GoalObject

	user, err := database.GetUserInformation(goal.User)
	if err != nil {
		return models.GoalObject{}, err
	}

	goalObject.User = user

	goalObject.Competing = goal.Competing
	goalObject.CreatedAt = goal.CreatedAt
	goalObject.DeletedAt = goal.DeletedAt
	goalObject.Enabled = goal.Enabled
	goalObject.ExerciseInterval = goal.ExerciseInterval
	goalObject.ID = goal.ID
	goalObject.Model = goal.Model
	goalObject.Season = goal.Season
	goalObject.UpdatedAt = goal.UpdatedAt

	return goalObject, nil

}
