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
	season, seasonFound, err := GetOngoingSeasonFromDB(time.Now())
	if err != nil {
		log.Println("Failed to verify current season status. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to verify current season status."})
		context.Abort()
		return
	} else if !seasonFound {
		log.Println("Failed to verify current season status. Error: No active or future seasons found.")
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to verify current season status."})
		context.Abort()
		return
	}

	if season.Start.Before(time.Now()) {
		context.JSON(http.StatusBadRequest, gin.H{"error": "Season has already started."})
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
	goalID, err := database.CreateGoalInDB(goalDB)
	if err != nil {
		log.Println("Failed to create goal for season. Error: " + strconv.Itoa(int(season.ID)))
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to create goal for season."})
		context.Abort()
		return
	}

	// Create unused sickleave for goal
	for i := 0; i < season.Sickleave; i++ {
		sickleave := models.Sickleave{
			Goal: int(goalID),
		}
		database.CreateSickleave(sickleave)
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

	sickleaveArray, sickleaveFound, err := database.GetUnusedSickleaveForGoalWithinWeek(int(goal.ID))
	if err != nil {
		log.Println("Failed to process sickleave. Setting to 0.")
		goalObject.SickleaveLeft = 0
	} else if !sickleaveFound {
		goalObject.SickleaveLeft = 0
	} else {
		goalObject.SickleaveLeft = len(sickleaveArray)
	}

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

func APIDeleteGoalToSeason(context *gin.Context) {

	// Get user ID
	userID, err := middlewares.GetAuthUsername(context.GetHeader("Authorization"))
	if err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		context.Abort()
		return
	}

	// Get current season
	season, seasonFound, err := GetOngoingSeasonFromDB(time.Now())
	if err != nil {
		log.Println("Failed to verify current season status. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to verify current season status."})
		context.Abort()
		return
	} else if !seasonFound {
		log.Println("Failed to verify current season status. Error: No active or future seasons found.")
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to verify current season status."})
		context.Abort()
		return
	}

	if season.Start.Before(time.Now()) {
		context.JSON(http.StatusBadRequest, gin.H{"error": "Season has already started."})
		context.Abort()
		return
	}

	// Verify goal exists within season
	goalStatus, goalID, err := database.VerifyUserGoalInSeason(userID, int(season.ID))
	if err != nil {
		log.Println("Failed to verify goal status. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to verify goal status."})
		context.Abort()
		return
	} else if !goalStatus {
		log.Println("User does not have a goal for season: " + strconv.Itoa(int(season.ID)))
		context.JSON(http.StatusBadRequest, gin.H{"error": "You don't have a goal this season."})
		context.Abort()
		return
	}

	err = database.DisableGoalInDBUsingGoalID(goalID)
	if err != nil {
		log.Println("Failed to disable goal in database. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to disable goal in database."})
		context.Abort()
		return
	}

	context.JSON(http.StatusCreated, gin.H{"message": "Goal deleted."})
}
