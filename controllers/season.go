package controllers

import (
	"aunefyren/treningheten/config"
	"aunefyren/treningheten/database"
	"aunefyren/treningheten/middlewares"
	"aunefyren/treningheten/models"
	"errors"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// GetOngoingSeason retrieves the currently active season, or the next upcoming.
func APIGetOngoingSeason(context *gin.Context) {

	// Verify user membership to group
	season, seasonFound, err := GetOngoingSeasonFromDB(time.Now())
	if err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		context.Abort()
		return
	} else if !seasonFound {
		context.JSON(http.StatusBadRequest, gin.H{"error": "No active or future seasons found."})
		context.Abort()
		return
	}

	seasonObject, err := ConvertSeasonToSeasonObject(season)
	if err != nil {
		log.Println("Failed to convert season to season object. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to convert season to season object."})
		context.Abort()
		return
	}

	// Censor goals
	for i := 0; i < len(seasonObject.Goals); i++ {
		seasonObject.Goals[i].ExerciseInterval = 0
	}

	// Get configuration
	config, err := config.GetConfig()
	if err != nil {
		log.Println("Failed to get config. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		context.Abort()
		return
	}

	// Return group with owner and success message
	context.JSON(http.StatusOK, gin.H{"season": seasonObject, "message": "Season retrieved.", "timezone": config.Timezone})
}

func GetOngoingSeasonFromDB(givenTime time.Time) (models.Season, bool, error) {

	current_time := givenTime
	chosen_season := models.Season{}
	change := false

	seasons, err := database.GetAllEnabledSeasons()
	if err != nil {
		return models.Season{}, false, err
	}

	if len(seasons) == 0 {
		return models.Season{}, false, nil
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
		return models.Season{}, false, nil
	}

	return chosen_season, true, nil

}

func ConvertSeasonToSeasonObject(season models.Season) (models.SeasonObject, error) {

	seasonObject := models.SeasonObject{
		Goals: []models.GoalObject{},
	}

	goals, err := database.GetGoalsFromWithinSeason(season.ID)
	if err != nil {
		log.Println("Failed to get goals within season. Ignoring.")
		return models.SeasonObject{}, err
	}

	for _, goal := range goals {

		goalObject, err := ConvertGoalToGoalObject(goal)
		if err != nil {
			log.Println("Failed to convert goal to goal object. Error: " + err.Error() + ". Skipping goal...")
			continue
		}

		seasonObject.Goals = append(seasonObject.Goals, goalObject)

	}

	prize, _, err := database.GetPrizeByID(season.PrizeID)
	if err != nil {
		log.Println("Failed to find prize by ID. Error: " + err.Error() + ". Returning.")
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

	seasonLocation, err := time.LoadLocation(season.TimeZone)
	if err != nil {
		log.Println("Failed to parse time zone. Error: " + err.Error())
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

	// Verify season overlap
	seasonOnGoing, seasonFound, err := GetOngoingSeasonFromDB(season.Start)
	if err != nil {
		log.Println("Failed to check season. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check season."})
		context.Abort()
		return
	} else if season.Start.After(seasonOnGoing.Start) && season.Start.Before(seasonOnGoing.End) && seasonFound {
		context.JSON(http.StatusBadRequest, gin.H{"error": "Season starts within season '" + seasonOnGoing.Name + "'."})
		context.Abort()
		return
	} else if season.End.After(seasonOnGoing.Start) && season.End.Before(seasonOnGoing.End) && seasonFound {
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

	// Verify prize ID
	_, prizeFound, err := database.GetPrizeByID(season.Prize)
	if err != nil {
		log.Println("Failed to verify prize ID. Error: " + err.Error())
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
		log.Println("Failed to verify create season in database. Error: " + err.Error())
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

	// Get current time
	now := time.Now()

	// Verify user membership to group
	season, seasonFound, err := GetOngoingSeasonFromDB(now)
	if err != nil {
		log.Println("Failed to check ongoing season. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		context.Abort()
		return
	} else if !seasonFound {
		context.JSON(http.StatusBadRequest, gin.H{"error": "No active or future seasons found."})
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
	goal, err := database.GetGoalFromUserWithinSeason(season.ID, userID)
	if err != nil {
		log.Println("Failed to verify goal status. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify goal status."})
		context.Abort()
		return
	}

	// Convert goal to GoalObject
	goalObject, err := ConvertGoalToGoalObject(goal)
	if err != nil {
		log.Println("Failed to verify goal status. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify goal status."})
		context.Abort()
		return
	}

	seasonLeaderboard := models.SeasonLeaderboard{
		UserGoal: goalObject,
		Season:   seasonObject,
	}

	seasonLeaderboard.PastWeeks, err = RetrieveWeekResultsFromSeasonWithinTimeframe(seasonObject.Start, now.AddDate(0, 0, -7), seasonObject)
	if err != nil {
		log.Println("Failed to retrieve past weeks for season. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to retrieve weeks for season."})
		context.Abort()
		return
	}

	thisWeek, err := RetrieveWeekResultsFromSeasonWithinTimeframe(now.AddDate(0, 0, -7), now, seasonObject)
	if err != nil {
		log.Println("Failed to retrieve current week for season. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve current week for season."})
		context.Abort()
		return
	} else if len(thisWeek) != 1 {
		log.Println("Got more than one week for current week.")
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Got more than one week for current week."})
		context.Abort()
		return
	} else {
		seasonLeaderboard.CurrentWeek = thisWeek[0]
	}

	// Return group with owner and success message
	context.JSON(http.StatusOK, gin.H{"leaderboard": seasonLeaderboard, "message": "Season leaderboard retrieved."})

}

func RetrieveWeekResultsFromSeasonWithinTimeframe(firstPointInTime time.Time, lastPointInTime time.Time, season models.SeasonObject) ([]models.WeekResults, error) {

	var weeksResults []models.WeekResults

	// Season has not started, return zero weeks
	if lastPointInTime.Before(firstPointInTime) {
		log.Println("lastPointInTime is before firstPointInTime. Returning nothing.")
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

		// Go through all goals
		for _, goal := range season.Goals {

			// Get Week result for goal
			weekResultForGoal, newUserStreaks, err := GetWeekResultForGoal(goal, currentTime, userStreaks)
			if err != nil {
				log.Println("Failed to get week results for user. Goal: " + goal.ID.String() + ". Error: " + err.Error() + ". Creating blank user.")
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
	newWeeksResults := []models.WeekResults{}
	for _, weeksResult := range weeksResults {
		if weeksResult.WeekDate.Before(firstPointInTime) || weeksResult.WeekDate.After(lastPointInTime) {
			continue
		}

		newWeeksResults = append(newWeeksResults, weeksResult)
	}

	// Reverse array
	newWeeksResults = ReverseWeeksArray(newWeeksResults)

	return newWeeksResults, nil
}

func GetWeekResultForGoal(goal models.GoalObject, currentTime time.Time, userStreaksInput []models.UserStreak) (newResult models.UserWeekResults, userStreaks []models.UserStreak, err error) {
	userStreaks = userStreaksInput
	newResult = models.UserWeekResults{}
	err = nil

	// Get the exercises from the week
	exercises, err := GetExercisesForWeekUsingGoal(currentTime, goal.ID)
	if err != nil {
		return models.UserWeekResults{}, userStreaks, err
	}

	// Define exercise sum
	exerciseSum := len(exercises)

	// Add details to week result for goal
	newResult.UserID = goal.User.ID
	newResult.WeekCompletion = (float64(exerciseSum) / float64(goal.ExerciseInterval))
	newResult.CurrentStreak = 0
	newResult.Competing = goal.Competing
	newResult.GoalID = goal.ID
	newResult.FullWeekParticipation = true

	currentTimeYear, currentTimeWeek := currentTime.ISOWeek()
	joinYear, joinWeek := goal.CreatedAt.ISOWeek()

	if joinYear == currentTimeYear && joinWeek >= currentTimeWeek || joinYear > currentTimeYear {
		newResult.FullWeekParticipation = false
	}

	// Check for debt for week
	debt, debtFound, err := database.GetDebtForWeekForUser(currentTime, goal.User.ID)
	if err != nil {
		log.Println("Failed to check for debt for user '" + goal.User.ID.String() + "'. Debt will be null.")
	} else if debtFound {
		debtObject, err := ConvertDebtToDebtObject(debt)
		if err != nil {
			log.Println("Failed to convert debt to debt object for user '" + goal.User.ID.String() + "'. Debt will be null.")
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

	sickleave, sickleaveFound, err := database.GetUsedSickleaveForGoalWithinWeek(currentTime, goal.ID)
	if err != nil {
		log.Println("Failed to process sickleave. Returning.")
		return models.UserWeekResults{}, userStreaks, errors.New("Failed to process sick leave.")
	}

	// Found in streak, retrieve current streak
	if newResult.FullWeekParticipation && newResult.WeekCompletion < 1 {
		newResult.CurrentStreak = userStreaks[userIndex].Streak
		userStreaks[userIndex].Streak = 0
	} else if sickleaveFound && sickleave.Used {
		newResult.CurrentStreak = userStreaks[userIndex].Streak
		newResult.SickLeave = true
	} else if newResult.WeekCompletion >= 1 {
		newResult.CurrentStreak = userStreaks[userIndex].Streak
		userStreaks[userIndex].Streak = userStreaks[userIndex].Streak + 1
	} else {
		newResult.CurrentStreak = userStreaks[userIndex].Streak
		userStreaks[userIndex].Streak = 0
	}

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

	seasonObjects := []models.SeasonObject{}

	seasons, err := database.GetAllEnabledSeasons()
	if err != nil {
		log.Println("Failed to get seasons from database. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get seasons from database."})
		context.Abort()
		return
	}

	for _, season := range seasons {
		seasonObject, err := ConvertSeasonToSeasonObject(season)
		if err != nil {
			log.Println("Failed process season. Error: " + err.Error())
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
		log.Println("Failed to get season from database. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get season from database."})
		context.Abort()
		return
	}

	seasonObject, err := ConvertSeasonToSeasonObject(season)
	if err != nil {
		log.Println("Failed process season. Error: " + err.Error())
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

// Get all enabled seasons
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
		log.Println("Failed to get season from database. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get season from database."})
		context.Abort()
		return
	}

	seasonObject, err := ConvertSeasonToSeasonObject(season)
	if err != nil {
		log.Println("Failed process season. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed process season."})
		context.Abort()
		return
	}

	now := time.Now()

	weekResults, err := RetrieveWeekResultsFromSeasonWithinTimeframe(season.Start, now, seasonObject)

	// Return seasons
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
		log.Println("Failed to get user ID. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to get user ID."})
		context.Abort()
		return
	}

	// Verify goal exists within season
	goal, err := database.GetGoalFromUserWithinSeason(seasonIDInt, userID)
	if err != nil {
		log.Println("Failed to verify goal status. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify goal status."})
		context.Abort()
		return
	}

	season, err := database.GetSeasonByID(seasonIDInt)
	if err != nil {
		log.Println("Failed to get season from database. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get season from database."})
		context.Abort()
		return
	}

	seasonObject, err := ConvertSeasonToSeasonObject(season)
	if err != nil {
		log.Println("Failed process season. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed process season."})
		context.Abort()
		return
	}

	now := time.Now()

	weekResults, err := RetrieveWeekResultsFromSeasonWithinTimeframe(season.Start, now, seasonObject)
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
				week, err := GetExerciseDaysForWeekUsingGoal(weekResult.WeekDate, goal.ID)
				if err != nil {
					log.Println("Failed to get exercise. Using empty week.")
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
		log.Println("Failed process wheel spins. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed process wheel spins."})
		context.Abort()
		return
	} else {
		wheelSpins = len(debts)
	}

	wins, _, err := database.GetDebtInSeasonWonByUserID(seasonIDInt, userID)
	if err != nil {
		log.Println("Failed process wheel spin wins. Error: " + err.Error())
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
