package controllers

import (
	"aunefyren/treningheten/config"
	"aunefyren/treningheten/database"
	"aunefyren/treningheten/logger"
	"aunefyren/treningheten/middlewares"
	"aunefyren/treningheten/models"
	"aunefyren/treningheten/utilities"
	"errors"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// GetOngoingSeason retrieves the currently active season, or the next upcoming.
func APIGetOngoingSeasons(context *gin.Context) {
	// Get user ID
	userID, err := middlewares.GetAuthUsername(context.GetHeader("Authorization"))
	if err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		context.Abort()
		return
	}

	// Verify user membership to group
	seasons, err := GetOngoingSeasonsFromDBForUserID(time.Now(), userID)
	if err != nil {
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get ongoing seasons for user."})
		context.Abort()
		return
	}

	seasonObjects, err := ConvertSeasonsToSeasonObjects(seasons)
	if err != nil {
		logger.Log.Info("Failed to convert season to season object. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to convert season to season object."})
		context.Abort()
		return
	}

	// Censor goals
	for i := 0; i < len(seasonObjects); i++ {
		for j := 0; j < len(seasonObjects[i].Goals); j++ {
			if seasonObjects[i].Goals[j].User.ID != userID {
				seasonObjects[i].Goals[j].ExerciseInterval = 0
			}
		}
	}

	// Get configuration
	config, err := config.GetConfig()
	if err != nil {
		logger.Log.Info("Failed to get config. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		context.Abort()
		return
	}

	// Return group with owner and success message
	context.JSON(http.StatusOK, gin.H{"seasons": seasonObjects, "message": "Seasons retrieved.", "timezone": config.Timezone})
}

func GetOngoingSeasonsFromDB(givenTime time.Time) (currentSeasons []models.Season, err error) {
	currentTime := givenTime
	currentSeasons = []models.Season{}

	seasons, err := database.GetAllEnabledSeasons()
	if err != nil {
		return
	}

	for _, season := range seasons {
		if season.End.After(currentTime) && season.Start.Before(currentTime) {
			currentSeasons = append(currentSeasons, season)
		}
	}

	return
}

func GetOngoingSeasonsFromDBForUserID(givenTime time.Time, userID uuid.UUID) (currentSeasons []models.Season, err error) {
	currentTime := givenTime
	currentSeasons = []models.Season{}
	tempSeasons := []models.Season{}

	seasons, err := database.GetAllEnabledSeasons()
	if err != nil {
		return
	}

	for _, season := range seasons {
		if season.End.After(currentTime) && season.Start.Before(currentTime) {
			tempSeasons = append(tempSeasons, season)
		}
	}

	for _, season := range tempSeasons {
		goalFound, _, err := database.VerifyUserGoalInSeason(userID, season.ID)
		if err != nil {
			logger.Log.Info("Failed to verify if goal in season. Skipping...")
			continue
		} else if goalFound {
			currentSeasons = append(currentSeasons, season)
		}
	}

	return
}

func ConvertSeasonToSeasonObject(season models.Season) (models.SeasonObject, error) {
	seasonObject := models.SeasonObject{
		Goals: []models.GoalObject{},
	}

	goals, err := database.GetGoalsFromWithinSeason(season.ID)
	if err != nil {
		logger.Log.Info("Failed to get goals within season. Ignoring.")
		return models.SeasonObject{}, err
	}

	for _, goal := range goals {

		goalObject, err := ConvertGoalToGoalObject(goal)
		if err != nil {
			logger.Log.Info("Failed to convert goal to goal object. Error: " + err.Error() + ". Skipping goal...")
			continue
		}

		seasonObject.Goals = append(seasonObject.Goals, goalObject)

	}

	prize, _, err := database.GetPrizeByID(season.PrizeID)
	if err != nil {
		logger.Log.Info("Failed to find prize by ID. Error: " + err.Error() + ". Returning.")
		return models.SeasonObject{}, errors.New("Failed to find prize by ID.")
	}

	seasonObject.Prize = prize

	seasonObject.CreatedAt = season.CreatedAt
	seasonObject.DeletedAt = season.DeletedAt
	seasonObject.Description = season.Description
	seasonObject.Enabled = season.Enabled
	seasonObject.End = season.End
	seasonObject.ID = season.ID
	seasonObject.Name = season.Name
	seasonObject.Start = season.Start
	seasonObject.UpdatedAt = season.UpdatedAt
	seasonObject.Sickleave = season.Sickleave
	seasonObject.JoinAnytime = season.JoinAnytime

	return seasonObject, nil
}

func ConvertSeasonsToSeasonObjects(seasons []models.Season) (seasonObjects []models.SeasonObject, err error) {
	err = nil
	seasonObjects = []models.SeasonObject{}

	for _, season := range seasons {
		seasonObject, err := ConvertSeasonToSeasonObject(season)
		if err != nil {
			logger.Log.Info("Failed to convert season to season object. Returning. Error: " + err.Error())
			return []models.SeasonObject{}, err
		}
		seasonObjects = append(seasonObjects, seasonObject)
	}

	return
}

func APIRegisterSeason(context *gin.Context) {

	// Create season request
	var season models.SeasonCreationRequest
	var seasonDB models.Season

	if err := context.ShouldBindJSON(&season); err != nil {
		logger.Log.Info("Failed to parse request. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse request."})
		context.Abort()
		return
	}

	seasonLocation, err := time.LoadLocation(season.TimeZone)
	if err != nil {
		logger.Log.Info("Failed to parse time zone. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse time zone."})
		context.Abort()
		return
	}

	season.Name = strings.TrimSpace(season.Name)
	season.Description = strings.TrimSpace(season.Description)
	season.Start = time.Date(season.Start.Year(), season.Start.Month(), season.Start.Day(), 00, 00, 00, 00, seasonLocation)
	season.End = time.Date(season.End.Year(), season.End.Month(), season.End.Day(), 23, 59, 59, 59, seasonLocation)

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
	weekdayTwo := int(season.End.Weekday())
	if season.End.Before(season.Start) || weekdayTwo != 0 {
		context.JSON(http.StatusBadRequest, gin.H{"error": "Your season must end after the start and end on a sunday."})
		context.Abort()
		return
	}

	// Verify season sickleave
	if season.Sickleave < 0 || season.Sickleave > 99 {
		context.JSON(http.StatusBadRequest, gin.H{"error": "The season sick leave must be between 0 and 99."})
		context.Abort()
		return
	}

	// Verify unique season name
	uniqueSeasonName, err := database.VerifyUniqueSeasonName(season.Name)
	if err != nil {
		logger.Log.Info("Failed to verify unique season name. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify unique season name."})
		context.Abort()
		return
	} else if !uniqueSeasonName {
		context.JSON(http.StatusBadRequest, gin.H{"error": "Season name already in use."})
		context.Abort()
		return
	}

	// Verify prize ID
	_, prizeFound, err := database.GetPrizeByID(season.Prize)
	if err != nil {
		logger.Log.Info("Failed to verify prize ID. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify prize ID."})
		context.Abort()
		return
	} else if !prizeFound {
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to find prize ID."})
		context.Abort()
		return
	}

	// Create DB object
	seasonDB.Description = season.Description
	seasonDB.Name = season.Name
	seasonDB.Start = season.Start
	seasonDB.End = season.End
	seasonDB.JoinAnytime = &season.JoinAnytime
	seasonDB.PrizeID = season.Prize
	seasonDB.Sickleave = season.Sickleave
	seasonDB.ID = uuid.New()

	// Register season in DB
	err = database.CreateSeasonInDB(seasonDB)
	if err != nil {
		logger.Log.Info("Failed to verify create season in database. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify create season in database."})
		context.Abort()
		return
	}

	context.JSON(http.StatusCreated, gin.H{"message": "Season created."})
}

func compareTimes(t1, t2 time.Time) int {
	return int(t1.Sub(t2))
}

// Get current leaderboard from ongoing season
func APIGetCurrentSeasonLeaderboard(context *gin.Context) {
	// Create user request
	var seasonID = context.Param("season_id")
	seasonIDParsed, err := uuid.Parse(seasonID)
	if err != nil {
		logger.Log.Info("Failed to parse season ID. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse season ID."})
		context.Abort()
		return
	}

	// Get current time
	now := time.Now()

	// Verify user membership to group
	season, err := database.GetSeasonByID(seasonIDParsed)
	if err != nil {
		logger.Log.Info("Failed to check ongoing season. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		context.Abort()
		return
	} else if season == nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": "Season not found."})
		context.Abort()
		return
	}

	seasonObject, err := ConvertSeasonToSeasonObject(*season)
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

	var goalObject *models.GoalObject
	// Verify goal exists within season
	goal, err := database.GetGoalFromUserWithinSeason(season.ID, userID)
	if err != nil {
		logger.Log.Info("Failed to verify goal status. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify goal status."})
		context.Abort()
		return
	} else if goal != nil {
		// Convert goal to GoalObject
		newGoalObject, err := ConvertGoalToGoalObject(*goal)
		if err != nil {
			logger.Log.Info("Failed to verify goal status. Error: " + err.Error())
			context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify goal status."})
			context.Abort()
			return
		}
		goalObject = &newGoalObject
	}

	seasonLeaderboard := models.SeasonLeaderboard{
		UserGoal: goalObject,
		Season:   seasonObject,
	}

	seasonLeaderboard.PastWeeks, err = RetrieveWeekResultsFromSeasonWithinTimeframe(seasonObject.Start, now.AddDate(0, 0, -7), seasonObject)
	if err != nil {
		logger.Log.Info("Failed to retrieve past weeks for season. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to retrieve weeks for season."})
		context.Abort()
		return
	}

	if now.Before(season.End) && now.After(season.Start) {
		thisWeek, err := RetrieveWeekResultsFromSeasonWithinTimeframe(now.AddDate(0, 0, -7), now, seasonObject)
		if err != nil {
			logger.Log.Info("Failed to retrieve current week for season. Error: " + err.Error())
			context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve current week for season."})
			context.Abort()
			return
		} else if len(thisWeek) != 1 {
			logger.Log.Info("Got more than one week for current week.")
			context.JSON(http.StatusInternalServerError, gin.H{"error": "Got more than one week for current week."})
			context.Abort()
			return
		} else {
			seasonLeaderboard.CurrentWeek = &thisWeek[0]
		}
	}

	// Return group with owner and success message
	context.JSON(http.StatusOK, gin.H{"leaderboard": seasonLeaderboard, "message": "Season leaderboard retrieved."})

}

func RetrieveWeekResultsFromSeasonWithinTimeframe(firstPointInTime time.Time, lastPointInTime time.Time, season models.SeasonObject) ([]models.WeekResults, error) {
	var weeksResults []models.WeekResults
	logger.Log.Debug("Retrieving week results from season within timeframe.")
	logger.Log.Debug("First point in time: " + firstPointInTime.String())
	logger.Log.Debug("Last point in time: " + lastPointInTime.String())

	// Season has not started, return zero weeks
	if lastPointInTime.Before(firstPointInTime) {
		logger.Log.Info("lastPointInTime is before firstPointInTime. Returning nothing.")
		return []models.WeekResults{}, nil
	}

	currentTime := season.Start
	finished := false
	userStreaks := []models.UserStreak{}
	for !finished {
		// New week
		weekResult := models.WeekResults{
			UserWeekResults: []models.UserWeekResults{},
		}

		// Add week details
		weekResult.WeekYear, weekResult.WeekNumber = currentTime.ISOWeek()
		weekResult.WeekDate = currentTime

		logger.Log.Debug("Processing new week: " + currentTime.String())
		logger.Log.Debug("Starting on goals...")

		// Go through all goals
		for _, goal := range season.Goals {
			logger.Log.Debug("Processing new goal '" + goal.ID.String() + "' for user '" + goal.User.FirstName + " " + goal.User.LastName + "'")

			// Get Week result for goal
			weekResultForGoal, newUserStreaks, err := GetWeekResultForGoal(goal, currentTime, userStreaks)
			if err != nil {
				logger.Log.Warn("Failed to get week results for user. Goal: " + goal.ID.String() + ". Error: " + err.Error() + ". Creating blank user.")
				continue
			}

			userStreaks = newUserStreaks

			// Append to array
			weekResult.UserWeekResults = append(weekResult.UserWeekResults, weekResultForGoal)
		}

		weeksResults = append(weeksResults, weekResult)

		currentTime = currentTime.AddDate(0, 0, 7)

		if currentTime.After(season.End) || currentTime.After(lastPointInTime) {
			finished = true
		}
	}

	// Remove weeks from before selected starting point
	logger.Log.Info("Removing weeks outside of given time frame.")
	newWeeksResults := []models.WeekResults{}
	for _, weeksResult := range weeksResults {
		if weeksResult.WeekDate.Before(firstPointInTime) || weeksResult.WeekDate.After(lastPointInTime) {
			logger.Log.Info("Removing week: " + weeksResult.WeekDate.String())
			continue
		}

		newWeeksResults = append(newWeeksResults, weeksResult)
	}

	// Reverse array
	newWeeksResults = ReverseWeeksArray(newWeeksResults)

	logger.Log.Info("Returning week results from season within timeframe.")

	return newWeeksResults, nil
}

func GetWeekResultForGoal(goal models.GoalObject, currentTime time.Time, userStreaksInput []models.UserStreak) (newResult models.UserWeekResults, userStreaks []models.UserStreak, err error) {
	userStreaks = userStreaksInput
	newResult = models.UserWeekResults{}
	err = nil

	// Get the exercises from the week
	exercises, err := GetExercisesForWeekUsingUserID(currentTime, goal.User.ID)
	if err != nil {
		return models.UserWeekResults{}, userStreaks, err
	}

	// Define exercise sum
	exerciseSum := len(exercises)

	logger.Log.Trace("processing exercises for point in time: " + currentTime.String() + ". Exercises: " + strconv.Itoa(len(exercises)) + ". Goal exercise: " + strconv.Itoa(goal.ExerciseInterval))

	// Add details to week result for goal
	newResult.UserID = goal.User.ID
	newResult.WeekCompletion = (float64(exerciseSum) / float64(goal.ExerciseInterval))
	newResult.CurrentStreak = 0
	newResult.Competing = goal.Competing
	newResult.GoalID = goal.ID
	newResult.FullWeekParticipation = true

	logger.Log.Trace("week completion: " + strconv.FormatFloat(newResult.WeekCompletion, 'f', -1, 64))

	currentTimeYear, currentTimeWeek := currentTime.ISOWeek()
	joinYear, joinWeek := goal.CreatedAt.ISOWeek()

	if joinYear == currentTimeYear && joinWeek >= currentTimeWeek || joinYear > currentTimeYear {
		newResult.FullWeekParticipation = false
	}

	// Check for debt for week
	debt, debtFound, err := database.GetDebtForWeekForUser(currentTime, goal.User.ID)
	if err != nil {
		logger.Log.Info("Failed to check for debt for user '" + goal.User.ID.String() + "'. Debt will be null.")
	} else if debtFound {
		debtObject, err := ConvertDebtToDebtObject(debt)
		if err != nil {
			logger.Log.Info("Failed to convert debt to debt object for user '" + goal.User.ID.String() + "'. Debt will be null.")
		} else {
			newResult.Debt = &debtObject
		}
	}

	// Find user in streak dict
	userFound := false
	userIndex := 0
	for index, userStreak := range userStreaks {
		if userStreak.UserID == goal.User.ID {
			userFound = true
			userIndex = index
			break
		}
	}

	if !userFound {
		// Not found in dict, current streak is 0
		newResult.CurrentStreak = 0
		userStreak := models.UserStreak{
			UserID: goal.User.ID,
			Streak: 0,
		}
		userStreaks = append(userStreaks, userStreak)
		// Find new index
		userFound = false
		userIndex = 0
		for index, userStreak := range userStreaks {
			if userStreak.UserID == goal.User.ID {
				userFound = true
				userIndex = index
				break
			}
		}
	}

	if !userFound {
		return models.UserWeekResults{}, userStreaks, errors.New("Failed to process streak.")
	}

	sickLeave, err := database.GetUsedSickleaveForGoalWithinWeek(currentTime, goal.ID)
	if err != nil {
		logger.Log.Info("Failed to process sickleave. Returning.")
		return models.UserWeekResults{}, userStreaks, errors.New("Failed to process sick leave.")
	}

	// Found in streak, retrieve current streak
	if newResult.FullWeekParticipation && newResult.WeekCompletion < 1 && (sickLeave == nil || !sickLeave.Used) {
		newResult.CurrentStreak = userStreaks[userIndex].Streak
		userStreaks[userIndex].Streak = 0
	} else if sickLeave != nil && sickLeave.Used {
		newResult.CurrentStreak = userStreaks[userIndex].Streak
		newResult.SickLeave = true
	} else if newResult.WeekCompletion >= 1 {
		newResult.CurrentStreak = userStreaks[userIndex].Streak
		userStreaks[userIndex].Streak = userStreaks[userIndex].Streak + 1
	} else {
		newResult.CurrentStreak = userStreaks[userIndex].Streak
		userStreaks[userIndex].Streak = 0
	}

	logger.Log.Trace("streak is now: " + strconv.Itoa(userStreaks[userIndex].Streak))
	logger.Log.Trace("streak is now on week results: " + strconv.Itoa(newResult.CurrentStreak))

	return newResult, userStreaks, nil
}

func ReverseWeeksArray(input []models.WeekResults) []models.WeekResults {
	if len(input) == 0 {
		return input
	}
	return append(ReverseWeeksArray(input[1:]), input[0])
}

// Get all enabled seasons
func APIGetSeasons(context *gin.Context) {
	seasons := []models.Season{}
	seasonObjects := []models.SeasonObject{}

	// Get user ID
	userID, err := middlewares.GetAuthUsername(context.GetHeader("Authorization"))
	if err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to get user ID."})
		context.Abort()
		return
	}

	potentialSeasons, err := database.GetAllEnabledSeasons()
	if err != nil {
		logger.Log.Info("Failed to get seasons from database. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get seasons from database."})
		context.Abort()
		return
	}

	// Remove non-potential seasons
	potentialBoolean, okay := context.GetQuery("potential")
	countdownBoolean, okayTwo := context.GetQuery("countdown")
	if okay && potentialBoolean == "true" {
		newSeasons := []models.Season{}

		for _, season := range potentialSeasons {
			goal, err := database.GetGoalFromUserWithinSeason(season.ID, userID)
			if err != nil {
				logger.Log.Info("Failed to check for goal within seasons. Error: " + err.Error())
				context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check for goal within seasons."})
				context.Abort()
				return
			} else if goal != nil {
				continue
			}

			now := time.Now()
			if now.After(season.Start) && (season.JoinAnytime == nil || *season.JoinAnytime == false) {
				logger.Log.Info(season.JoinAnytime)
				continue
			}

			if now.After(season.End) {
				continue
			}

			newSeasons = append(newSeasons, season)
		}

		seasons = newSeasons
	} else if okayTwo && countdownBoolean == "true" {
		newSeasons := []models.Season{}

		for _, season := range potentialSeasons {
			goal, err := database.GetGoalFromUserWithinSeason(season.ID, userID)
			if err != nil {
				logger.Log.Info("Failed to check for goal within seasons. Error: " + err.Error())
				context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check for goal within seasons."})
				context.Abort()
				return
			} else if goal == nil {
				continue
			}

			now := time.Now()
			if now.After(season.Start) {
				continue
			}

			newSeasons = append(newSeasons, season)
		}

		seasons = newSeasons
	} else {
		seasons = potentialSeasons
	}

	for _, season := range seasons {
		seasonObject, err := ConvertSeasonToSeasonObject(season)
		if err != nil {
			logger.Log.Info("Failed process season. Error: " + err.Error())
			context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed process season."})
			context.Abort()
			return
		}
		seasonObjects = append(seasonObjects, seasonObject)
	}

	// Return seasons
	context.JSON(http.StatusOK, gin.H{"seasons": seasonObjects, "message": "Seasons retrieved."})

}

// Get one enabled seasons
func APIGetSeason(context *gin.Context) {
	var seasonID = context.Param("season_id")

	seasonIDUUIID, err := uuid.Parse(seasonID)
	if err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse ID."})
		context.Abort()
		return
	}

	season, err := database.GetSeasonByID(seasonIDUUIID)
	if err != nil {
		logger.Log.Info("Failed to get season from database. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get season from database."})
		context.Abort()
		return
	} else if season == nil {
		logger.Log.Info("Failed to find season.")
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to find season."})
		context.Abort()
		return
	}

	seasonObject, err := ConvertSeasonToSeasonObject(*season)
	if err != nil {
		logger.Log.Info("Failed process season. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed process season."})
		context.Abort()
		return
	}

	// Censor goals
	for i := 0; i < len(seasonObject.Goals); i++ {
		seasonObject.Goals[i].ExerciseInterval = 0
	}

	// Return season
	context.JSON(http.StatusOK, gin.H{"season": seasonObject, "message": "Season retrieved."})

}

// Get weeks from season
func APIGetSeasonWeeks(context *gin.Context) {

	// Create user request
	var seasonID = context.Param("season_id")

	// Parse group id
	seasonIDInt, err := uuid.Parse(seasonID)
	if err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		context.Abort()
		return
	}

	season, err := database.GetSeasonByID(seasonIDInt)
	if err != nil {
		logger.Log.Info("Failed to get season from database. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get season from database."})
		context.Abort()
		return
	} else if season == nil {
		logger.Log.Info("Failed to find season. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to find season."})
		context.Abort()
		return
	}

	seasonObject, err := ConvertSeasonToSeasonObject(*season)
	if err != nil {
		logger.Log.Info("Failed process season. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed process season."})
		context.Abort()
		return
	}

	now := time.Now()

	weekResults, err := RetrieveWeekResultsFromSeasonWithinTimeframe(season.Start, now, seasonObject)
	if err != nil {
		logger.Log.Info("Failed to retrieve results. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve results."})
		context.Abort()
		return
	}

	// Return season weeks
	context.JSON(http.StatusOK, gin.H{"leaderboard": weekResults, "message": "Season leaderboard retrieved."})
}

// Get all weeks from within season with actual exercise intervals
func APIGetSeasonWeeksPersonal(context *gin.Context) {

	// Create user request
	var seasonID = context.Param("season_id")

	// Parse group id
	seasonIDInt, err := uuid.Parse(seasonID)
	if err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		context.Abort()
		return
	}

	// Get user ID
	userID, err := middlewares.GetAuthUsername(context.GetHeader("Authorization"))
	if err != nil {
		logger.Log.Info("Failed to get user ID. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to get user ID."})
		context.Abort()
		return
	}

	// Verify goal exists within season
	goal, err := database.GetGoalFromUserWithinSeason(seasonIDInt, userID)
	if err != nil {
		logger.Log.Info("Failed to verify goal status. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify goal status."})
		context.Abort()
		return
	}

	season, err := database.GetSeasonByID(seasonIDInt)
	if err != nil {
		logger.Log.Info("Failed to get season from database. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get season from database."})
		context.Abort()
		return
	} else if season == nil {
		logger.Log.Info("Failed to find season.")
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to find season."})
		context.Abort()
		return
	}

	seasonObject, err := ConvertSeasonToSeasonObject(*season)
	if err != nil {
		logger.Log.Info("Failed process season. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed process season."})
		context.Abort()
		return
	}

	now := time.Now()

	weekResults, err := RetrieveWeekResultsFromSeasonWithinTimeframe(season.Start, now, seasonObject)
	if err != nil {
		logger.Log.Info("Failed get week results. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed get week results."})
		context.Abort()
		return
	}

	weekResultsPersonal := []models.WeekResultsPersonal{}
	weekDaysPersonal := models.WeekFrequency{}

	for _, weekResult := range weekResults {

		weekResultPersonal := models.WeekResultsPersonal{
			WeekNumber: weekResult.WeekNumber,
			WeekDate:   weekResult.WeekDate,
			WeekYear:   weekResult.WeekYear,
		}

		userFound := false

		for _, result := range weekResult.UserWeekResults {

			if result.UserID == userID {
				userWeekResultPersonal := models.UserWeekResultPersonal{
					CurrentStreak: result.CurrentStreak,
					UserID:        result.UserID,
					SickLeave:     result.SickLeave,
					Competing:     result.Competing,
					Debt:          result.Debt,
					ExerciseGoal:  goal.ExerciseInterval,
				}

				userWeekResultPersonal.WeekCompletionInterval = int(float64(result.WeekCompletion) * float64(goal.ExerciseInterval))

				// Get the exercises from the week
				week, err := GetExerciseDaysForWeekUsingUserID(weekResult.WeekDate, goal.UserID)
				if err != nil {
					logger.Log.Info("Failed to get exercise. Using empty week.")
					week = models.Week{
						Days: []models.ExerciseDayObject{},
					}
				}

				for _, day := range week.Days {
					weekday := day.Date.Weekday()

					if weekday == 0 {
						weekDaysPersonal.Sunday = weekDaysPersonal.Sunday + day.ExerciseInterval
					} else if weekday == 1 {
						weekDaysPersonal.Monday = weekDaysPersonal.Monday + day.ExerciseInterval
					} else if weekday == 2 {
						weekDaysPersonal.Tuesday = weekDaysPersonal.Tuesday + day.ExerciseInterval
					} else if weekday == 3 {
						weekDaysPersonal.Wednesday = weekDaysPersonal.Wednesday + day.ExerciseInterval
					} else if weekday == 4 {
						weekDaysPersonal.Thursday = weekDaysPersonal.Thursday + day.ExerciseInterval
					} else if weekday == 5 {
						weekDaysPersonal.Friday = weekDaysPersonal.Friday + day.ExerciseInterval
					} else if weekday == 6 {
						weekDaysPersonal.Saturday = weekDaysPersonal.Saturday + day.ExerciseInterval
					}
				}

				weekResultPersonal.UserWeekResults = userWeekResultPersonal
				userFound = true
				break

			}

		}

		if !userFound {
			context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create statistics for user."})
			context.Abort()
			return
		}

		weekResultsPersonal = append(weekResultsPersonal, weekResultPersonal)

	}

	wheelSpins := 0
	wheelsWon := 0

	debts, _, err := database.GetDebtInSeasonLostByUserID(seasonIDInt, userID)
	if err != nil {
		logger.Log.Info("Failed process wheel spins. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed process wheel spins."})
		context.Abort()
		return
	} else {
		wheelSpins = len(debts)
	}

	wins, _, err := database.GetDebtInSeasonWonByUserID(seasonIDInt, userID)
	if err != nil {
		logger.Log.Info("Failed process wheel spin wins. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed process wheel spin wins."})
		context.Abort()
		return
	} else {
		wheelsWon = len(wins)
	}

	type wheelStatistics struct {
		WheelSpins int `json:"wheel_spins"`
		WheelWon   int `json:"wheels_won"`
	}

	wheelStatisticsStruct := wheelStatistics{
		WheelSpins: wheelSpins,
		WheelWon:   wheelsWon,
	}

	// Return seasons
	context.JSON(http.StatusOK, gin.H{"leaderboard": weekResultsPersonal, "weekdays": weekDaysPersonal, "message": "Season weeks retrieved.", "wheel_statistics": wheelStatisticsStruct})
}

func APIGetCurrentSeasonActivities(context *gin.Context) {
	// Create user request
	var seasonID = context.Param("season_id")
	seasonIDParsed, err := uuid.Parse(seasonID)
	if err != nil {
		logger.Log.Info("Failed to parse season ID. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse season ID."})
		context.Abort()
		return
	}

	// Get current time
	now := time.Now()
	mondayStart, err := utilities.FindEarlierMonday(now)
	if err != nil {
		logger.Log.Info("Failed to find Monday. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to find Monday."})
		context.Abort()
		return
	}

	sundayEnd, err := utilities.FindNextSunday(now)
	if err != nil {
		logger.Log.Info("Failed to find Sunday. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to find Sunday."})
		context.Abort()
		return
	}

	// Verify user membership to group
	season, err := database.GetSeasonByID(seasonIDParsed)
	if err != nil {
		logger.Log.Info("Failed to check ongoing season. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check ongoing season."})
		context.Abort()
		return
	} else if season == nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": "Season not found."})
		context.Abort()
		return
	}

	// Get user ID
	userID, err := middlewares.GetAuthUsername(context.GetHeader("Authorization"))
	if err != nil {
		logger.Log.Info("Failed to be user from header. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to be user from header."})
		context.Abort()
		return
	}

	// Verify goal exists within season
	goal, err := database.GetGoalFromUserWithinSeason(season.ID, userID)
	if err != nil {
		logger.Log.Info("Failed to verify goal status. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify goal status."})
		context.Abort()
		return
	} else if goal == nil {
		context.JSON(http.StatusInternalServerError, gin.H{"error": "You must be a part of the season to see the activities."})
		context.Abort()
		return
	}

	allExerciseDays, err := database.GetExerciseDaysForSharingUsersUsingDates(mondayStart, sundayEnd)
	if err != nil {
		logger.Log.Info("Failed to get exercise days from time frame. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to get exercise days from time frame."})
		context.Abort()
		return
	}

	filteredExerciseDays := []models.ExerciseDay{}
	validatedUsers := []uuid.UUID{}
	for _, exerciseDay := range allExerciseDays {
		if exerciseDay.UserID == nil {
			continue
		}

		foundInCache := false
		for _, validatedUserID := range validatedUsers {
			if validatedUserID == *exerciseDay.UserID {
				foundInCache = true
				break
			}
		}

		if foundInCache {
			filteredExerciseDays = append(filteredExerciseDays, exerciseDay)
			continue
		} else {
			// Verify goal exists within season
			goal, err := database.GetGoalFromUserWithinSeason(season.ID, *exerciseDay.UserID)
			if err != nil {
				logger.Log.Info("Failed to verify goal status for '" + exerciseDay.UserID.String() + "'. Error: " + err.Error())
				context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify goal status."})
				context.Abort()
				return
			}

			if goal != nil {
				filteredExerciseDays = append(filteredExerciseDays, exerciseDay)
				validatedUsers = append(validatedUsers, *exerciseDay.UserID)
				continue
			}
		}
	}

	exerciseDayObjects, err := ConvertExerciseDaysToExerciseDayObjects(filteredExerciseDays)
	if err != nil {
		logger.Log.Info("Failed to convert exercise day to exercise day objects. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to convert exercise day to exercise day objects."})
		context.Abort()
		return
	}

	allActivities := []models.Activity{}
	for _, exerciseDayObject := range exerciseDayObjects {
		for _, exercise := range exerciseDayObject.Exercises {
			if exercise.On && exercise.Enabled {
				newActivity := models.Activity{}
				newActivity.ExerciseID = exercise.ID
				newActivity.User = exerciseDayObject.User
				newActivity.Time = exercise.Time
				newActivity.Actions = []models.Action{}

				if exerciseDayObject.User.StravaPublic != nil && *exerciseDayObject.User.StravaPublic {
					newActivity.StravaIDs = exercise.StravaID
				} else {
					newActivity.StravaIDs = []string{}
				}

				for _, operation := range exercise.Operations {
					if operation.Action != nil {
						newActivity.Actions = append(newActivity.Actions, *operation.Action)
					}
				}

				allActivities = append(allActivities, newActivity)
			}
		}
	}

	sort.Slice(allActivities, func(i, j int) bool {
		return allActivities[j].Time.Before(allActivities[i].Time)
	})

	// Return activities
	context.JSON(http.StatusOK, gin.H{"activities": allActivities})
}
