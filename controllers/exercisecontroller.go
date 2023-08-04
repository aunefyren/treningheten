package controllers

import (
	"aunefyren/treningheten/auth"
	"aunefyren/treningheten/database"
	"aunefyren/treningheten/middlewares"
	"aunefyren/treningheten/models"
	"aunefyren/treningheten/utilities"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// Get full workout calender for the week from the user, sanitize and update database
func APIRegisterWeek(context *gin.Context) {

	// Create week request
	var week models.WeekCreationRequest

	// Parse request
	if err := context.ShouldBindJSON(&week); err != nil {
		log.Println("Failed to parse request. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse request."})
		context.Abort()
		return
	}

	// Get user ID
	userID, err := middlewares.GetAuthUsername(context.GetHeader("Authorization"))
	if err != nil {
		log.Println("Failed to verify user ID. Error: " + "Failed to verify user ID.")
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

	// Current time
	now := time.Now()
	isoYear, isoWeek := now.ISOWeek()

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

	// Check if week is sickleave
	sickleave, sickleaveFound, err := database.GetUsedSickleaveForGoalWithinWeek(now, int(goalID))
	if err != nil {
		log.Println("Failed to verify sickleave. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify sickleave."})
		context.Abort()
		return
	} else if sickleaveFound && sickleave.SickleaveUsed {
		context.JSON(http.StatusBadRequest, gin.H{"error": "This week is marked as sickleave."})
		context.Abort()
		return
	}

	// Check if any debt is unspun
	_, debtsFound, err := database.GetUnchosenDebtForUserByUserID(userID)
	if err != nil {
		log.Println("Failed to get unspun spins. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get unviewed spins."})
		context.Abort()
		return
	} else if debtsFound {
		context.JSON(http.StatusBadRequest, gin.H{"error": "You must spin the wheel first."})
		context.Abort()
		return
	}

	// Verify all weekdays are present
	if len(week.Days) != 7 {
		context.JSON(http.StatusBadRequest, gin.H{"error": "Week is not seven days."})
		context.Abort()
		return
	}

	// Array for processed weekdays
	var foundWeekDays []int

	// Verify dates in request
	for index, day := range week.Days {

		// Get year and week
		isoYearRequest, isoWeekRequest := day.Date.ISOWeek()
		weekInt := int(day.Date.Weekday())

		// Verify correct week and year
		if isoYear != isoYearRequest || isoWeek != isoWeekRequest {
			context.JSON(http.StatusBadRequest, gin.H{"error": "The registered week must contain dates from the current week."})
			context.Abort()
			return
		}

		// Verify order of dates
		if index+1 != int(weekInt) && !(index == 6 && weekInt == 0) {
			context.JSON(http.StatusBadRequest, gin.H{"error": "Week is not in the correct order."})
			context.Abort()
			return
		}

		// Validate exercise interval
		if day.ExerciseInterval > 3 {
			context.JSON(http.StatusBadRequest, gin.H{"error": "You can only exercise three times a day."})
			context.Abort()
			return
		}

		if len(day.Note) > 255 {
			context.JSON(http.StatusBadRequest, gin.H{"error": "The note is too long."})
			context.Abort()
			return
		}

		if day.Date.Truncate(24*time.Hour).After(now.Truncate(24*time.Hour)) && day.ExerciseInterval > 0 {
			context.JSON(http.StatusBadRequest, gin.H{"error": "You can't register exercises on days in the future."})
			context.Abort()
			return
		}

		// Look through found weekdays
		alreadyRegistered := false
		for _, weekDay := range foundWeekDays {
			if weekDay == int(weekInt) {
				alreadyRegistered = true
				break
			}
		}

		// If weekday has been found, stop the user
		// If not, add to found weekdays
		if alreadyRegistered {
			context.JSON(http.StatusBadRequest, gin.H{"error": "Duplicate weekdays in request."})
			context.Abort()
			return
		} else {
			foundWeekDays = append(foundWeekDays, weekInt)
		}

	}

	// Process each day for database
	for _, day := range week.Days {

		exercise, err := database.GetExerciseByGoalAndDate(goalID, day.Date)
		if err != nil {
			log.Println("Failed to verify exercise status. Error: " + err.Error())
			context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify exercise status."})
			context.Abort()
			return
		}

		if exercise != nil {

			exercise.ExerciseInterval = day.ExerciseInterval
			exercise.Note = strings.TrimSpace(day.Note)

			err = database.UpdateExerciseInDatabase(*exercise)
			if err != nil {
				log.Println("Failed to update exercise in database. Error: " + err.Error())
				context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update exercise in database."})
				context.Abort()
				return
			}

		} else {

			exercise := models.Exercise{
				Date:             day.Date,
				Note:             day.Note,
				ExerciseInterval: day.ExerciseInterval,
				Goal:             goalID,
			}

			err = database.CreateExerciseForGoalInDatabase(exercise)
			if err != nil {
				log.Println("Failed to save exercise in database. Error: " + err.Error())
				context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save exercise in database."})
				context.Abort()
				return
			}

		}
	}

	// Get week for goal using current time
	weekReturn, err := GetExercisesForWeekUsingGoal(now, goalID)
	if err != nil {
		log.Println("Failed to get calendar. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get calender."})
		context.Abort()
		return
	}

	// Get user object by ID
	token := ""
	user, err := database.GetAllUserInformation(userID)
	if err != nil {
		log.Println("Failed to get user object for user '" + strconv.Itoa(userID) + "'. Not returning token. Error: " + err.Error())
	} else {
		// Generate new token to refresh expiration time
		token, err = auth.GenerateJWT(user.FirstName, user.LastName, user.Email, int(user.ID), *user.Admin, user.Verified, user.SundayAlert)
		if err != nil {
			log.Println("Failed to create JWT token for user '" + strconv.Itoa(userID) + "'. Not returning token. Error: " + err.Error())
		}
	}

	context.JSON(http.StatusCreated, gin.H{"message": "Week saved.", "week": weekReturn, "token": token})

}

// Get full workout calender for the week from the database, and return to user
func APIRGetWeek(context *gin.Context) {

	// Get user ID
	userID, err := middlewares.GetAuthUsername(context.GetHeader("Authorization"))
	if err != nil {
		log.Println("Failed to verify user ID. Error: " + "Failed to verify user ID.")
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

	// Get week for goal using current time
	week, err := GetExercisesForWeekUsingGoal(now, goalID)
	if err != nil {
		log.Println("Failed to get calendar. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get calender."})
		context.Abort()
		return
	}

	context.JSON(http.StatusOK, gin.H{"message": "Week retrieved.", "week": week})

}

func GetExercisesForWeekUsingGoal(timeReq time.Time, goalID int) (models.Week, error) {

	week := models.Week{}
	var startTime time.Time
	startTimeWeek := 0
	var endTime time.Time
	endTimeWeek := 0
	_, timeReqWeek := timeReq.ISOWeek()

	// Find monday
	startTime, err := utilities.FindEarlierMonday(timeReq)
	if err != nil {
		log.Println("Failed to find earlier Monday for date. Error: " + err.Error())
		return models.Week{}, errors.New("Failed to find earlier Monday for date.")
	}
	_, startTimeWeek = startTime.ISOWeek()

	// Find sunday
	endTime, err = utilities.FindNextSunday(timeReq)
	if err != nil {
		log.Println("Failed to find next Sunday for date. Error: " + err.Error())
		return models.Week{}, errors.New("Failed to find next Sunday for date.")
	}
	_, endTimeWeek = endTime.ISOWeek()

	// Verify all dates are the same week
	if timeReqWeek != startTimeWeek || timeReqWeek != endTimeWeek {
		log.Println("Required time week: " + strconv.Itoa(timeReqWeek))
		log.Println("Start time week: " + strconv.Itoa(startTimeWeek))
		log.Println("End time week: " + strconv.Itoa(endTimeWeek))
		return models.Week{}, errors.New("Managed to find dates outside of chosen week.")
	}

	exercises, err := database.GetExercisesBetweenDatesUsingDates(goalID, startTime, endTime)
	if err != nil {
		return models.Week{}, err
	}

	for i := 0; i < 7; i++ {

		currentDate := startTime.AddDate(0, 0, i)
		added := false

		for _, exercise := range exercises {

			if currentDate.Format("2006-01-02") == exercise.Date.Format("2006-01-02") {

				week.Days = append(week.Days, exercise)
				added = true
				break

			}

		}

		if !added {
			newExercise := models.Exercise{
				Date: currentDate.Truncate(24 * time.Hour),
			}
			week.Days = append(week.Days, newExercise)
		}

	}

	return week, nil

}

// Get full workout calender for the week from the database, and return to user
func APIRGetExercise(context *gin.Context) {

	// Get user ID
	userID, err := middlewares.GetAuthUsername(context.GetHeader("Authorization"))
	if err != nil {
		log.Println("Failed to verify user ID. Error: " + "Failed to verify user ID.")
		context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		context.Abort()
		return
	}

	// Get exercises from user
	exercise, err := database.GetExercisesForUserUsingUserID(userID)
	if err != nil {
		log.Println("Failed to get exercise. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get exercise."})
		context.Abort()
		return
	}

	context.JSON(http.StatusOK, gin.H{"message": "Exercise retrieved.", "exercise": exercise})

}
