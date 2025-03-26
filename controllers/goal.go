package controllers

import (
	"aunefyren/treningheten/database"
	"aunefyren/treningheten/middlewares"
	"aunefyren/treningheten/models"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func APIRegisterGoalToSeason(context *gin.Context) {

	// Create goal request
	var goal models.GoalCreationRequest
	var goalDB models.Goal

	if err := context.ShouldBindJSON(&goal); err != nil {
		log.Info("Failed to parse request. Error: " + err.Error())
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

	// Verify exercise interval
	if goal.ExerciseInterval == 0 || goal.ExerciseInterval > 21 {
		context.JSON(http.StatusBadRequest, gin.H{"error": "Your exercise goal must be between 1 and 21."})
		context.Abort()
		return
	}

	// Finalize goal object
	goalDB.Competing = goal.Competing
	goalDB.ExerciseInterval = goal.ExerciseInterval
	goalDB.SeasonID = goal.SeasonID
	goalDB.UserID = userID
	goalDB.ID = uuid.New()

	season, err := database.GetSeasonByID(goal.SeasonID)
	if err != nil {
		log.Info("Failed to get season by ID. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get season by ID."})
		context.Abort()
		return
	} else if season == nil {
		log.Info("Failed to find season by ID.")
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to find season by ID."})
		context.Abort()
		return
	} else if season.Start.Before(time.Now()) && !*season.JoinAnytime {
		context.JSON(http.StatusBadRequest, gin.H{"error": "Season has already started."})
		context.Abort()
		return
	}

	// Verify goal doesn't exist within season
	goalStatus, _, err := database.VerifyUserGoalInSeason(userID, season.ID)
	if err != nil {
		log.Info("Failed to verify goal status. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to verify goal status."})
		context.Abort()
		return
	} else if goalStatus {
		log.Info("User already has a goal for season: " + season.ID.String())
		context.JSON(http.StatusBadRequest, gin.H{"error": "You already have a goal this season."})
		context.Abort()
		return
	}

	// Create goal in DB
	goalID, err := database.CreateGoalInDB(goalDB)
	if err != nil {
		log.Info("Failed to create goal for season. Error: " + season.ID.String())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to create goal for season."})
		context.Abort()
		return
	}

	// Create unused sickleave for goal
	for i := 0; i < season.Sickleave; i++ {
		sickleave := models.Sickleave{
			GoalID: goalID,
		}
		sickleave.ID = uuid.New()
		database.CreateSickleave(sickleave)
	}

	// Give achievement to user
	err = GiveUserAnAchievement(userID, uuid.MustParse("7f2d49ad-d056-415e-aa80-0ada6db7cc00"), time.Now())
	if err != nil {
		log.Info("Failed to give achievement for user '" + userID.String() + "'. Ignoring. Error: " + err.Error())
	}

	context.JSON(http.StatusCreated, gin.H{"message": "Goal created."})

}

func ConvertGoalToGoalObject(goal models.Goal) (models.GoalObject, error) {

	var goalObject models.GoalObject

	user, err := database.GetUserInformation(goal.UserID)
	if err != nil {
		log.Info("Failed to get information for user '" + goal.User.ID.String() + "'. Returning. Error: " + err.Error())
		return models.GoalObject{}, err
	}

	goalObject.User = user

	sickleaveArray, sickleaveFound, err := database.GetUnusedSickleaveForGoalWithinWeek(goal.ID)
	if err != nil {
		log.Info("Failed to process sickleave. Setting to 0.")
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
	goalObject.SeasonID = goal.SeasonID
	goalObject.UpdatedAt = goal.UpdatedAt

	return goalObject, nil

}

func ConvertGoalsToGoalObjects(goals []models.Goal) ([]models.GoalObject, error) {

	var goalObjects []models.GoalObject

	for _, goal := range goals {
		goalObject, err := ConvertGoalToGoalObject(goal)
		if err != nil {
			log.Info("Failed to convert goal to goal object. Returning. Error: " + err.Error())
			return []models.GoalObject{}, err
		}
		goalObjects = append(goalObjects, goalObject)
	}

	if len(goalObjects) == 0 {
		return []models.GoalObject{}, nil
	}

	return goalObjects, nil

}

func APIDeleteGoalToSeason(context *gin.Context) {
	var goalID = context.Param("goal_id")

	goalIDUUIID, err := uuid.Parse(goalID)
	if err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse ID."})
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

	goal, err := database.GetGoalUsingGoalID(goalIDUUIID)
	if err != nil {
		log.Info("Failed to get goal by ID. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get goal by ID."})
		context.Abort()
		return
	} else if goal == nil {
		log.Info("Failed to find goal by ID. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to find goal by ID."})
		context.Abort()
		return
	} else if goal.UserID != userID {
		context.JSON(http.StatusUnauthorized, gin.H{"error": "Not your goal."})
		context.Abort()
		return
	}

	// Get current season
	season, err := database.GetSeasonByID(goal.SeasonID)
	if err != nil {
		log.Info("Failed to verify current season status. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to verify current season status."})
		context.Abort()
		return
	} else if season == nil {
		log.Info("Failed to verify current season status. Error: No active or future seasons found.")
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to verify current season status."})
		context.Abort()
		return
	}

	if season.Start.Before(time.Now()) {
		context.JSON(http.StatusBadRequest, gin.H{"error": "Season has already started."})
		context.Abort()
		return
	}

	err = database.DisableGoalInDBUsingGoalID(goal.ID)
	if err != nil {
		log.Info("Failed to disable goal in database. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to disable goal in database."})
		context.Abort()
		return
	}

	context.JSON(http.StatusCreated, gin.H{"message": "Goal deleted."})
}

func APIGetGoals(context *gin.Context) {

	// Get user ID
	userID, err := middlewares.GetAuthUsername(context.GetHeader("Authorization"))
	if err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		context.Abort()
		return
	}

	// Verify goal exists within season
	goals, err := database.GetGoalsForUserUsingUserID(userID)
	if err != nil {
		log.Info("Failed to get goals. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get goals."})
		context.Abort()
		return
	}

	goalObject, err := ConvertGoalsToGoalObjects(goals)
	if err != nil {
		log.Info("Failed to convert goals to goal objects. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to convert goals to goal objects."})
		context.Abort()
		return
	}

	context.JSON(http.StatusCreated, gin.H{"message": "Goals retrieved.", "goals": goalObject})
}
