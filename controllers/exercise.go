package controllers

import (
	"aunefyren/treningheten/database"
	"aunefyren/treningheten/middlewares"
	"aunefyren/treningheten/models"
	"aunefyren/treningheten/utilities"
	"errors"
	"html"
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

	requestLocation, err := time.LoadLocation(week.TimeZone)
	if err != nil {
		log.Println("Failed to parse time zone. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse time zone."})
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

		if len(strings.TrimSpace(day.Note)) > 255 {
			context.JSON(http.StatusBadRequest, gin.H{"error": "The note is too long."})
			context.Abort()
			return
		}

		if day.Date.After(now) && day.ExerciseInterval > 0 {
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

	// Declare variable for later user
	newExercise := models.Exercise{}

	// Process each day for database
	for _, day := range week.Days {

		exerciseDay, err := database.GetExerciseDayByGoalAndDate(goalID, day.Date)
		if err != nil {
			log.Println("Failed to verify exercise status. Error: " + err.Error())
			context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify exercise status."})
			context.Abort()
			return
		}

		// If exercise day exists in DB, update it
		if exerciseDay != nil {

			// If no changes, don't update it
			if exerciseDay.ExerciseInterval != day.ExerciseInterval || exerciseDay.Note != html.EscapeString(strings.TrimSpace(day.Note)) {

				exerciseDay.ExerciseInterval = day.ExerciseInterval
				exerciseDay.Note = html.EscapeString(strings.TrimSpace(day.Note))

				err = database.UpdateExerciseDayInDatabase(*exerciseDay)
				if err != nil {
					log.Println("Failed to update exercise-day in database. Error: " + err.Error())
					context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update exercise-day in database."})
					context.Abort()
					return
				}

				// If the exercise-interval has changed, correlate the exercise table
				err = CorrelateExerciseWithExerciseDay(int(exerciseDay.ID), exerciseDay.ExerciseInterval)
				if err != nil {
					log.Println("Failed to update exercises for exercise-day. Error: " + err.Error())
					context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update exercises for exercise-day."})
					context.Abort()
					return
				}

			}

		} else {

			// Create new exercise day
			exerciseDay := models.ExerciseDay{
				Date:             time.Date(day.Date.Year(), day.Date.Month(), day.Date.Day(), 00, 00, 00, 00, requestLocation),
				Note:             day.Note,
				ExerciseInterval: day.ExerciseInterval,
				Goal:             goalID,
			}

			exerciseDayID, err := database.CreateExerciseDayForGoalInDatabase(exerciseDay)
			if err != nil {
				log.Println("Failed to save exercise-day in database. Error: " + err.Error())
				context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save exercise-day in database."})
				context.Abort()
				return
			}

			for i := 0; i < exerciseDay.ExerciseInterval; i++ {
				newExercise = models.Exercise{
					ExerciseDay: int(exerciseDayID),
				}
				err = database.CreateExerciseForExerciseDayInDatabase(newExercise)
				if err != nil {
					log.Println("Failed to update exercise in database. Error: " + err.Error())
					context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update exercise in database."})
					context.Abort()
					return
				}
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

	context.JSON(http.StatusCreated, gin.H{"message": "Week saved.", "week": weekReturn})

}

// Get full workout calender for the week from the database, and return to user
func APIGetWeek(context *gin.Context) {

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

	exercises, err := database.GetExerciseDaysBetweenDatesUsingDates(goalID, startTime, endTime)
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
			newExercise := models.ExerciseDay{
				Date: currentDate.Truncate(24 * time.Hour),
			}
			week.Days = append(week.Days, newExercise)
		}

	}

	return week, nil

}

// Get full workout calender for the week from the database, and return to user
func APIGetAllExercise(context *gin.Context) {

	// Get user ID
	userID, err := middlewares.GetAuthUsername(context.GetHeader("Authorization"))
	if err != nil {
		log.Println("Failed to verify user ID. Error: " + "Failed to verify user ID.")
		context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		context.Abort()
		return
	}

	// Get exercises from user
	exercise, err := database.GetExerciseDaysForUserUsingUserID(userID)
	if err != nil {
		log.Println("Failed to get exercise. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get exercise."})
		context.Abort()
		return
	}

	context.JSON(http.StatusOK, gin.H{"message": "Exercise retrieved.", "exercise": exercise})

}

// Get full workout calender for the week from the database, and return to user
func APIGetExercise(context *gin.Context) {

	// Create user request
	var goalID = context.Param("goal_id")

	// Parse group id
	goalIDInt, err := strconv.Atoi(goalID)
	if err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
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

	// Get exercises from user
	exercise, err := database.GetExerciseDaysForUserUsingUserIDAndGoalID(userID, goalIDInt)
	if err != nil {
		log.Println("Failed to get exercise. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get exercise."})
		context.Abort()
		return
	}

	context.JSON(http.StatusOK, gin.H{"message": "Exercise retrieved.", "exercise": exercise})

}

func CorrelateExerciseWithExerciseDay(exerciseDayID int, exerciseDayExerciseInterval int) error {

	exercises, err := database.GetExerciseByExerciseDayID(exerciseDayID)
	if err != nil {
		log.Println("Failed to get exercises for exercise-day. Error: " + err.Error())
		return errors.New("Failed to get exercises for exercise-day.")
	}

	onExercisesSum := 0

	for _, exercise := range exercises {
		if exercise.On {
			onExercisesSum += 1
		}
	}

	if onExercisesSum == exerciseDayExerciseInterval {
		return nil
	} else if onExercisesSum > exerciseDayExerciseInterval {

		differential := onExercisesSum - exerciseDayExerciseInterval
		changed, err := TurnOffExercisesForExerciseDayByAmount(differential, exerciseDayID)
		if err != nil {
			log.Println("Failed to turn off exercises for exercise day. Error: " + err.Error())
			return errors.New("Failed to turn off exercises for exercise day.")
		}

		newDifferential := differential - changed

		if newDifferential == 0 {
			return nil
		} else {
			return errors.New("Failed to turn off enough exercises for exercise-day. Differential: " + strconv.Itoa(newDifferential))
		}

	} else {

		differential := exerciseDayExerciseInterval - onExercisesSum
		changed, err := TurnOnExercisesForExerciseDayByAmount(differential, exerciseDayID)
		if err != nil {
			log.Println("Failed to turn off exercises for exercise day. Error: " + err.Error())
			return errors.New("Failed to turn off exercises for exercise day.")
		}

		newDifferential := differential - changed

		if newDifferential == 0 {
			return nil
		}

		for i := 0; i < newDifferential; i++ {
			newExercise := models.Exercise{
				ExerciseDay: exerciseDayID,
			}
			err = database.CreateExerciseForExerciseDayInDatabase(newExercise)
			if err != nil {
				log.Println("Failed to create exercise in database. Error: " + err.Error())
				return errors.New("Failed to create exercise in database.")
			}
		}

	}

	return nil

}

func TurnOnExercisesForExerciseDayByAmount(amount int, exerciseDayID int) (int, error) {

	if amount == 0 {
		return 0, errors.New("Amount must be more than 0.")
	}

	turnedOn := 0

	exercises, err := database.GetExerciseByExerciseDayID(exerciseDayID)
	if err != nil {
		log.Println("Failed to get exercises for exercise-day. Error: " + err.Error())
		return 0, errors.New("Failed to get exercises for exercise-day.")
	}

	for _, exercise := range exercises {

		if !exercise.On {
			err = database.UpdateExerciseByTurningOnByExerciseID(int(exercise.ID))
			if err != nil {
				log.Println("Failed to turn on exercise. Error: " + err.Error())
				return 0, errors.New("Failed to turn on exercise.")
			}

			turnedOn += 1
		}

		if turnedOn >= amount {
			break
		}

	}

	return turnedOn, nil

}

func TurnOffExercisesForExerciseDayByAmount(amount int, exerciseDayID int) (int, error) {

	if amount == 0 {
		return 0, errors.New("Amount must be more than 0.")
	}

	turnedOff := 0

	exercises, err := database.GetExerciseByExerciseDayID(exerciseDayID)
	if err != nil {
		log.Println("Failed to get exercises for exercise-day. Error: " + err.Error())
		return 0, errors.New("Failed to get exercises for exercise-day.")
	}

	for _, exercise := range exercises {

		if exercise.On {
			err = database.UpdateExerciseByTurningOffByExerciseID(int(exercise.ID))
			if err != nil {
				log.Println("Failed to turn on exercise. Error: " + err.Error())
				return 0, errors.New("Failed to turn on exercise.")
			}

			turnedOff += 1
		}

		if turnedOff >= amount {
			break
		}

	}

	return turnedOff, nil

}

func CorrelateAllExercises() error {

	exerciseDays, err := database.GetAllEnabledExerciseDays()
	if err != nil {
		log.Println("Failed to get enabled exercises. Error: " + err.Error())
		return errors.New("Failed to get enabled exercises.")
	}

	for _, exerciseDay := range exerciseDays {
		err = CorrelateExerciseWithExerciseDay(int(exerciseDay.ID), exerciseDay.ExerciseInterval)
		if err != nil {
			log.Println("Failed to correlate exercise. Error: " + err.Error())
			return errors.New("Failed to correlate exercise.")
		}
	}

	return nil

}

//
func APICorrelateAllExercises(context *gin.Context) {

	err := CorrelateAllExercises()
	if err != nil {
		log.Println("Failed to correlate all exercises. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to correlate all exercises."})
		context.Abort()
		return
	}

	context.JSON(http.StatusOK, gin.H{"message": "All exercise correlated."})

}
