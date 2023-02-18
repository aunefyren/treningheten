package controllers

import (
	"aunefyren/treningheten/database"
	"aunefyren/treningheten/middlewares"
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

// Get current leaderboard from ongoing season
func APIGetCurrentSeasonLeaderboard(context *gin.Context) {

	// Get current time
	now := time.Now()

	// Verify user membership to group
	season, err := GetOngoingSeasonFromDB(now)
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

	// Get user ID
	userID, err := middlewares.GetAuthUsername(context.GetHeader("Authorization"))
	if err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		context.Abort()
		return
	}

	// Verify goal exists within season
	goal, err := database.GetGoalFromUserWithinSeason(int(season.ID), userID)
	if err != nil {
		log.Println("Failed to verify goal status. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to verify goal status."})
		context.Abort()
		return
	}

	// Convert goal to GoalObject
	goalObject, err := ConvertGoalToGoalObject(goal)
	if err != nil {
		log.Println("Failed to verify goal status. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to verify goal status."})
		context.Abort()
		return
	}

	seasonLeaderboard := models.SeasonLeaderboard{
		UserGoal: goalObject,
		Season:   seasonObject,
	}

	seasonLeaderboard.Weeks, err = RetrieveWeeksFromSeason(now, seasonObject)
	if err != nil {
		log.Println("Failed to retrieve weeks for season. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to retrieve weeks for season."})
		context.Abort()
		return
	}

	if len(seasonLeaderboard.Weeks) > 0 {
		userFound := false
		userIndex := 0
		for index, user := range seasonLeaderboard.Weeks[0].UserWeekResults {
			if user.User.ID == uint(userID) {
				userFound = true
				userIndex = index
				break
			}
		}
		if userFound {
			if seasonLeaderboard.Weeks[0].UserWeekResults[userIndex].WeekCompletion >= 1 {
				seasonLeaderboard.CurrentStreak = seasonLeaderboard.Weeks[0].UserWeekResults[userIndex].CurrentStreak + 1
			} else {
				seasonLeaderboard.CurrentStreak = 0
			}
		}
	}

	exercisesThisWeek, err := GetExercisesForWeekUsingGoal(now, int(goalObject.ID))
	if err != nil {
		log.Println("Failed to retrieve exercises for this week. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to retrieve exercises for this week."})
		context.Abort()
		return
	}

	// Define exercise sum
	exerciseSum := 0

	// Sum all exercises
	for _, day := range exercisesThisWeek.Days {

		exerciseSum += day.ExerciseInterval

	}

	seasonLeaderboard.CurrentCompletion = float64(exerciseSum) / float64(goalObject.ExerciseInterval)

	// Return group with owner and success message
	context.JSON(http.StatusOK, gin.H{"leaderboard": seasonLeaderboard, "message": "Season leaderboard retrieved."})

}

func RetrieveWeeksFromSeason(pointInTime time.Time, season models.SeasonObject) ([]models.WeekResults, error) {

	var weeksResults []models.WeekResults

	// Season has not started, return zero weeks
	if pointInTime.Before(season.Start) {
		return []models.WeekResults{}, nil
	}

	goals, err := database.GetGoalsFromWithinSeason(int(season.ID))
	if err != nil {
		return []models.WeekResults{}, err
	}

	type UserStreak struct {
		UserID int `json:"user_id"`
		Streak int `json:"streak"`
	}

	currentTime := season.Start
	finished := false
	userStreaks := []UserStreak{}
	for finished == false {

		// New week
		weekResult := models.WeekResults{}

		// Add weel details
		weekResult.WeekYear, weekResult.WeekNumber = currentTime.ISOWeek()

		// Go through all goals
		for _, goal := range goals {

			// Weel result for goal
			newResult := models.UserWeekResults{}

			// Get the exercises from the week
			week, err := GetExercisesForWeekUsingGoal(currentTime, int(goal.ID))
			if err != nil {
				return []models.WeekResults{}, err
			}

			// Define exercise sum
			exerciseSum := 0

			// Sum all exercises
			for _, day := range week.Days {

				exerciseSum += day.ExerciseInterval

			}

			// Get goal object
			goalObject, err := ConvertGoalToGoalObject(goal)
			if err != nil {
				return []models.WeekResults{}, err
			}

			// Add details to week result for goal
			newResult.User = goalObject.User
			newResult.WeekCompletion = float64(exerciseSum) / float64(goal.ExerciseInterval)
			newResult.CurrentStreak = 0

			// Find user in streak dict
			userFound := false
			userIndex := 0
			for index, userStreak := range userStreaks {
				if userStreak.UserID == int(goalObject.User.ID) {
					userFound = true
					userIndex = index
					break
				}
			}

			if !userFound {
				// Not found in dict, current streak is 0
				newResult.CurrentStreak = 0
				userStreak := UserStreak{
					UserID: int(goalObject.User.ID),
					Streak: 0,
				}
				userStreaks = append(userStreaks, userStreak)
				// Find new index
				userFound = false
				userIndex = 0
				for index, userStreak := range userStreaks {
					if userStreak.UserID == int(goalObject.User.ID) {
						userFound = true
						userIndex = index
						break
					}
				}
			}

			if !userFound {
				return []models.WeekResults{}, errors.New("Failed to process streak.")
			}

			// Found in streak, retrieve current streak
			if newResult.WeekCompletion >= 1 {
				newResult.CurrentStreak = userStreaks[userIndex].Streak
				userStreaks[userIndex].Streak = userStreaks[userIndex].Streak + 1
			} else {
				newResult.CurrentStreak = userStreaks[userIndex].Streak
				userStreaks[userIndex].Streak = 0
			}

			// Append to array
			weekResult.UserWeekResults = append(weekResult.UserWeekResults, newResult)

		}

		currentTime = currentTime.AddDate(0, 0, 7)
		pointInTimeYear, pointInTimeWeek := pointInTime.ISOWeek()
		_, currentTimeWeek := currentTime.ISOWeek()

		if currentTime.After(season.End) || (currentTime.Year() > pointInTimeYear || (currentTimeWeek >= pointInTimeWeek && currentTime.Year() == pointInTimeYear)) {
			finished = true
		}

		weeksResults = append(weeksResults, weekResult)

	}

	weeksResults = ReverseWeeksArray(weeksResults)

	return weeksResults, nil

}

func ReverseWeeksArray(input []models.WeekResults) []models.WeekResults {
	if len(input) == 0 {
		return input
	}
	return append(ReverseWeeksArray(input[1:]), input[0])
}
