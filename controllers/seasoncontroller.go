package controllers

import (
	"aunefyren/treningheten/database"
	"aunefyren/treningheten/models"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// GetOngoingSeason retrieves the currently active season, or the next upcoming.
func APIGetOngoingSeason(context *gin.Context) {

	// Verify user membership to group
	season, err := GetOngoingSeasonFromDB(time.Now())
	if err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		context.Abort()
		return
	}

	seasonObject, err := ConvertSeasonToSeasonObject(season)
	if err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		context.Abort()
		return
	}

	// Return group with owner and success message
	context.JSON(http.StatusOK, gin.H{"season": seasonObject, "message": "Season retrieved."})

}

func GetOngoingSeasonFromDB(giventime time.Time) (models.Season, error) {

	current_time := giventime
	chosen_season := models.Season{}
	change := false

	seasons, err := database.GetAllEnabledSeasons()
	if err != nil {
		return models.Season{}, err
	}

	if len(seasons) == 0 {
		return models.Season{}, errors.New("Zero active or future seasons found.")
	}

	for _, season := range seasons {

		if season.Start.Before(current_time) && season.End.After(current_time) {
			chosen_season = season
			change = true
			break
		}

		if season.Start.After(current_time) && (season.Start.Before(chosen_season.Start) || !change) {
			chosen_season = season
			change = true
		}

	}

	if !change {
		return models.Season{}, errors.New("No active or future seasons found.")
	}

	return chosen_season, nil

}

func ConvertSeasonToSeasonObject(season models.Season) (models.SeasonObject, error) {

	seasonObject := models.SeasonObject{
		Goals: []models.GoalObject{},
	}

	goals, err := database.GetGoalsFromWithinSeason(int(season.ID))
	if err != nil {
		return models.SeasonObject{}, err
	}

	for _, goal := range goals {

		goalObject, err := ConvertGoalToGoalObject(goal)
		if err != nil {
			return models.SeasonObject{}, err
		}

		seasonObject.Goals = append(seasonObject.Goals, goalObject)

	}

	seasonObject.CreatedAt = season.CreatedAt
	seasonObject.DeletedAt = season.DeletedAt
	seasonObject.Description = season.Description
	seasonObject.Enabled = season.Enabled
	seasonObject.End = season.End
	seasonObject.ID = season.ID
	seasonObject.Model = season.Model
	seasonObject.Name = season.Name
	seasonObject.Start = season.Start
	seasonObject.UpdatedAt = season.UpdatedAt

	return seasonObject, nil

}

func APIRegisterSeason(context *gin.Context) {

	// Create season request
	var season models.SeasonCreationRequest
	var seasonDB models.Season

	if err := context.ShouldBindJSON(&season); err != nil {
		log.Println("Failed to parse request. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse request."})
		context.Abort()
		return
	}

	// Verify season name
	if season.Name == "" || len(season.Name) < 5 {
		context.JSON(http.StatusBadRequest, gin.H{"error": "Your season name must be above five chars."})
		context.Abort()
		return
	}

	// Verify season start
	now := time.Now()
	weekday := int(season.Start.Weekday())
	if season.Start.Before(now) || weekday != 1 {
		context.JSON(http.StatusBadRequest, gin.H{"error": "Your season must be in the future and start on a monday."})
		context.Abort()
		return
	}

	// Verify season end
	weekdaytwo := int(season.End.Weekday())
	if season.End.Before(season.Start) || weekdaytwo != 0 {
		context.JSON(http.StatusBadRequest, gin.H{"error": "Your season must end after the start and end on a sunday."})
		context.Abort()
		return
	}

	// Verify season start
	seasonOnGoing, err := GetOngoingSeasonFromDB(season.Start)
	if err == nil && (season.Start.After(seasonOnGoing.Start) && season.Start.Before(seasonOnGoing.End)) {
		context.JSON(http.StatusBadRequest, gin.H{"error": "Season starts within season '" + seasonOnGoing.Name + "'."})
		context.Abort()
		return
	}
	if err == nil && (season.End.After(seasonOnGoing.Start) && season.End.Before(seasonOnGoing.End)) {
		context.JSON(http.StatusBadRequest, gin.H{"error": "Season ends within season '" + seasonOnGoing.Name + "'."})
		context.Abort()
		return
	}

	// Verify unique season name
	uniqueSeasonName, err := database.VerifyUniqueSeasonName(season.Name)
	if err != nil {
		log.Println("Failed to verify unique season name. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify unique season name."})
		context.Abort()
		return
	} else if !uniqueSeasonName {
		context.JSON(http.StatusBadRequest, gin.H{"error": "Season name already in use."})
		context.Abort()
		return
	}

	// Create DB object
	seasonDB.Description = season.Description
	seasonDB.Name = season.Name
	seasonDB.Start = season.Start
	seasonDB.End = season.End

	// Register season in DB
	err = database.CreateSeasonInDB(seasonDB)
	if err != nil {
		log.Println("Failed to verify create season in database. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify create season in database."})
		context.Abort()
		return
	}

	context.JSON(http.StatusCreated, gin.H{"message": "Season created."})
}
