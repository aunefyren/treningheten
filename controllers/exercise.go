package controllers

import (
	"aunefyren/treningheten/database"
	"aunefyren/treningheten/logger"
	"aunefyren/treningheten/middlewares"
	"aunefyren/treningheten/models"
	"aunefyren/treningheten/utilities"
	"errors"
	"html"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Get full workout calender for the week from the user, sanitize and update database
func APIRegisterWeek(context *gin.Context) {

	// Create week request
	var week models.WeekCreationRequest

	// Parse request
	if err := context.ShouldBindJSON(&week); err != nil {
		logger.Log.Info("Failed to parse request. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse request."})
		context.Abort()
		return
	}

	// Get user ID
	userID, err := middlewares.GetAuthUsername(context.GetHeader("Authorization"))
	if err != nil {
		logger.Log.Info("Failed to verify user ID. Error: " + "Failed to verify user ID.")
		context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		context.Abort()
		return
	}

	requestLocation, err := time.LoadLocation(week.TimeZone)
	if err != nil {
		logger.Log.Info("Failed to parse time zone. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse time zone."})
		context.Abort()
		return
	}

	// Current time
	now := time.Now()
	isoYear, isoWeek := now.ISOWeek()

	// Check if any debt is unspun
	_, debtsFound, err := database.GetUnchosenDebtForUserByUserID(userID)
	if err != nil {
		logger.Log.Info("Failed to get unspun spins. Error: " + err.Error())
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

		exerciseDayDB, err := database.GetExerciseDayByUserIDAndDate(userID, day.Date)
		if err != nil {
			logger.Log.Info("Failed to verify exercise status. Error: " + err.Error())
			context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify exercise status."})
			context.Abort()
			return
		}

		// If exercise day exists in DB, update it
		if exerciseDayDB != nil {

			exerciseDay, err := ConvertExerciseDayToExerciseDayObject(*exerciseDayDB)
			if err != nil {
				logger.Log.Info("Failed to convert exercise day to exercise day object. Error: " + err.Error())
				context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to convert exercise day to exercise day object."})
				context.Abort()
				return
			}

			// If no changes, don't update it
			if exerciseDay.ExerciseInterval != day.ExerciseInterval || exerciseDay.Note != html.EscapeString(strings.TrimSpace(day.Note)) {

				exerciseDayDB.Note = html.EscapeString(strings.TrimSpace(day.Note))

				*exerciseDayDB, err = database.UpdateExerciseDayInDatabase(*exerciseDayDB)
				if err != nil {
					logger.Log.Info("Failed to update exercise-day in database. Error: " + err.Error())
					context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update exercise-day in database."})
					context.Abort()
					return
				}

				exerciseDay, err := ConvertExerciseDayToExerciseDayObject(*exerciseDayDB)
				if err != nil {
					logger.Log.Info("Failed to convert exercise day to exercise day object. Error: " + err.Error())
					context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to convert exercise day to exercise day object."})
					context.Abort()
					return
				}

				// If the exercise-interval has changed, correlate the exercise table
				err = CorrelateExerciseWithExerciseDay(exerciseDay.ID, day.ExerciseInterval)
				if err != nil {
					logger.Log.Info("Failed to update exercises for exercise-day. Error: " + err.Error())
					context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update exercises for exercise-day."})
					context.Abort()
					return
				}

			}

		} else {

			// Create new exercise day
			exerciseDay := models.ExerciseDay{
				Date:   time.Date(day.Date.Year(), day.Date.Month(), day.Date.Day(), 00, 00, 00, 00, requestLocation),
				Note:   day.Note,
				UserID: &userID,
			}
			exerciseDay.ID = uuid.New()

			exerciseDayID, err := database.CreateExerciseDayForGoalInDatabase(exerciseDay)
			if err != nil {
				logger.Log.Info("Failed to save exercise-day in database. Error: " + err.Error())
				context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save exercise-day in database."})
				context.Abort()
				return
			}

			for i := 0; i < day.ExerciseInterval; i++ {
				newExercise = models.Exercise{
					ExerciseDayID: exerciseDayID,
				}
				newExercise.ID = uuid.New()

				err = database.CreateExerciseForExerciseDayInDatabase(newExercise)
				if err != nil {
					logger.Log.Info("Failed to update exercise in database. Error: " + err.Error())
					context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update exercise in database."})
					context.Abort()
					return
				}
			}
		}
	}

	// Get week for goal using current time
	weekReturn, err := GetExerciseDaysForWeekUsingUserID(now, userID)
	if err != nil {
		logger.Log.Info("Failed to get calendar. Error: " + err.Error())
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
		logger.Log.Info("Failed to verify user ID. Error: " + "Failed to verify user ID.")
		context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		context.Abort()
		return
	}

	// Current time
	now := time.Now()

	// Get week for goal using current time
	week, err := GetExerciseDaysForWeekUsingUserID(now, userID)
	if err != nil {
		logger.Log.Info("Failed to get calendar. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get calendar."})
		context.Abort()
		return
	}

	context.JSON(http.StatusOK, gin.H{"message": "Week retrieved.", "week": week})

}

func GetExerciseDaysForWeekUsingGoal(timeReq time.Time, goalID uuid.UUID) (models.Week, error) {
	week := models.Week{}
	var startTime time.Time
	startTimeWeek := 0
	var endTime time.Time
	endTimeWeek := 0
	_, timeReqWeek := timeReq.ISOWeek()

	// Find monday
	startTime, err := utilities.FindEarlierMonday(timeReq)
	if err != nil {
		logger.Log.Info("Failed to find earlier Monday for date. Error: " + err.Error())
		return models.Week{}, errors.New("Failed to find earlier Monday for date.")
	}
	_, startTimeWeek = startTime.ISOWeek()

	// Find sunday
	endTime, err = utilities.FindNextSunday(timeReq)
	if err != nil {
		logger.Log.Info("Failed to find next Sunday for date. Error: " + err.Error())
		return models.Week{}, errors.New("Failed to find next Sunday for date.")
	}
	_, endTimeWeek = endTime.ISOWeek()

	// Verify all dates are the same week
	if timeReqWeek != startTimeWeek || timeReqWeek != endTimeWeek {
		logger.Log.Info("Required time week: " + strconv.Itoa(timeReqWeek))
		logger.Log.Info("Start time week: " + strconv.Itoa(startTimeWeek))
		logger.Log.Info("End time week: " + strconv.Itoa(endTimeWeek))
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

				exerciseDayObject, err := ConvertExerciseDayToExerciseDayObject(exercise)
				if err != nil {
					logger.Log.Info("Failed to convert exercise day to exercise day object. Error: " + err.Error())
					return models.Week{}, errors.New("Failed to convert exercise day to exercise day object.")
				}

				week.Days = append(week.Days, exerciseDayObject)
				added = true
				break

			}

		}

		if !added {
			newExercise := models.ExerciseDayObject{
				Date: utilities.SetClockToMinimum(currentDate),
			}
			week.Days = append(week.Days, newExercise)
		}

	}

	return week, nil
}

func GetExerciseDaysForWeekUsingUserID(timeReq time.Time, userID uuid.UUID) (models.Week, error) {
	week := models.Week{}
	var startTime time.Time
	startTimeWeek := 0
	var endTime time.Time
	endTimeWeek := 0
	_, timeReqWeek := timeReq.ISOWeek()

	// Find monday
	startTime, err := utilities.FindEarlierMonday(timeReq)
	if err != nil {
		logger.Log.Info("Failed to find earlier Monday for date. Error: " + err.Error())
		return models.Week{}, errors.New("Failed to find earlier Monday for date.")
	}
	_, startTimeWeek = startTime.ISOWeek()

	// Find sunday
	endTime, err = utilities.FindNextSunday(timeReq)
	if err != nil {
		logger.Log.Info("Failed to find next Sunday for date. Error: " + err.Error())
		return models.Week{}, errors.New("Failed to find next Sunday for date.")
	}
	_, endTimeWeek = endTime.ISOWeek()

	// Verify all dates are the same week
	if timeReqWeek != startTimeWeek || timeReqWeek != endTimeWeek {
		logger.Log.Info("Required time week: " + strconv.Itoa(timeReqWeek))
		logger.Log.Info("Start time week: " + strconv.Itoa(startTimeWeek))
		logger.Log.Info("End time week: " + strconv.Itoa(endTimeWeek))
		return models.Week{}, errors.New("Managed to find dates outside of chosen week.")
	}

	exercises, err := database.GetExerciseDaysBetweenDatesUsingDatesAndUserID(userID, startTime, endTime)
	if err != nil {
		return models.Week{}, err
	}

	for i := 0; i < 7; i++ {

		currentDate := startTime.AddDate(0, 0, i)
		added := false

		for _, exercise := range exercises {

			if currentDate.Format("2006-01-02") == exercise.Date.Format("2006-01-02") {

				exerciseDayObject, err := ConvertExerciseDayToExerciseDayObject(exercise)
				if err != nil {
					logger.Log.Info("Failed to convert exercise day to exercise day object. Error: " + err.Error())
					return models.Week{}, errors.New("Failed to convert exercise day to exercise day object.")
				}

				week.Days = append(week.Days, exerciseDayObject)
				added = true
				break

			}

		}

		if !added {
			newExercise := models.ExerciseDayObject{
				Date: utilities.SetClockToMinimum(currentDate),
			}
			week.Days = append(week.Days, newExercise)
		}

	}

	week.Goals = []models.Goal{}
	timeString := utilities.TimeToMySQLTimestamp(timeReq)
	activeGoals, err := database.GetActiveGoalsForUserIDAndDate(userID, timeString)
	if err != nil {
		logger.Log.Info("Failed to get active goals. Error: " + err.Error())
	} else {
		week.Goals = activeGoals
	}

	return week, nil
}

// Get full workout calender for the week from the database, and return to user
func APIGetExerciseDays(context *gin.Context) {
	var goalObject *models.GoalObject = nil

	// Get user ID
	userID, err := middlewares.GetAuthUsername(context.GetHeader("Authorization"))
	if err != nil {
		logger.Log.Info("Failed to verify user ID. Error: " + "Failed to verify user ID.")
		context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		context.Abort()
		return
	}

	var exerciseDays = []models.ExerciseDay{}

	// Get exercises from user
	goal, okay := context.GetQuery("goal")
	year, okayTwo := context.GetQuery("year")
	if okay {
		goalID, err := uuid.Parse(goal)
		if err != nil {
			context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse goal ID."})
			context.Abort()
			return
		}

		goalTMP, err := database.GetGoalUsingGoalID(goalID)
		if err != nil {
			logger.Log.Info("Failed to get goal. Error: " + err.Error())
			context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get goal."})
			context.Abort()
			return
		} else if goalTMP == nil {
			context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to find goal."})
			context.Abort()
			return
		} else if goalTMP.UserID != userID {
			context.JSON(http.StatusUnauthorized, gin.H{"error": "Not your goal."})
			context.Abort()
			return
		}

		// Get exercises from user
		exerciseDays, err = database.GetExerciseDaysForUserUsingUserIDAndGoalID(userID, goalID)
		if err != nil {
			logger.Log.Info("Failed to get exercise. Error: " + err.Error())
			context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get exercise."})
			context.Abort()
			return
		}

		goalObjectTMP, err := ConvertGoalToGoalObject(*goalTMP)
		if err != nil {
			logger.Log.Info("Failed to convert goal to goal object. Error: " + err.Error())
			context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to convert goal to goal object."})
			context.Abort()
			return
		}

		goalObject = &goalObjectTMP

	} else if okayTwo {
		yearInt, err := strconv.ParseInt(year, 10, 64)
		if err != nil {
			logger.Log.Info("Failed to convert year string to int. Error: " + err.Error())
			context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to convert year string to int."})
			context.Abort()
			return
		}

		newExerciseDays, err := database.GetExerciseDaysForUserUsingUserID(userID)
		if err != nil {
			logger.Log.Info("Failed to get exercise days. Error: " + err.Error())
			context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get exercise."})
			context.Abort()
			return
		}

		for _, exerciseDay := range newExerciseDays {
			if int64(exerciseDay.Date.Year()) == yearInt {
				exerciseDays = append(exerciseDays, exerciseDay)
			}
		}
	} else {
		exerciseDays, err = database.GetExerciseDaysForUserUsingUserID(userID)
		if err != nil {
			logger.Log.Info("Failed to get exercise. Error: " + err.Error())
			context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get exercise."})
			context.Abort()
			return
		}
	}

	exerciseDayObjects, err := ConvertExerciseDaysToExerciseDayObjects(exerciseDays)
	if err != nil {
		logger.Log.Info("Failed to get convert exercise days to exercise day objects. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get convert exercise days to exercise day objects."})
		context.Abort()
		return
	}

	context.JSON(http.StatusOK, gin.H{"message": "Exercise days retrieved.", "exercise": exerciseDayObjects, "goal": goalObject})
}

// Get full workout calender for the week from the database, and return to user
func APIAdminGetExerciseDays(context *gin.Context) {
	var exerciseDays = []models.ExerciseDay{}

	exerciseDays, err := database.GetAllExerciseDays()
	if err != nil {
		logger.Log.Info("Failed to get exercise days. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get exercise."})
		context.Abort()
		return
	}

	exerciseDayObjects, err := ConvertExerciseDaysToExerciseDayObjects(exerciseDays)
	if err != nil {
		logger.Log.Info("Failed to get convert exercise days to exercise day objects. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get convert exercise days to exercise day objects."})
		context.Abort()
		return
	}

	context.JSON(http.StatusOK, gin.H{"message": "Exercise days retrieved.", "exercise": exerciseDayObjects})
}

// Change exercises to correlate with exercise days
func CorrelateExerciseWithExerciseDay(exerciseDayID uuid.UUID, exerciseDayExerciseInterval int) error {

	exercises, err := database.GetExerciseByExerciseDayID(exerciseDayID)
	if err != nil {
		logger.Log.Info("Failed to get exercises for exercise-day. Error: " + err.Error())
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
			logger.Log.Info("Failed to turn off exercises for exercise day. Error: " + err.Error())
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
			logger.Log.Info("Failed to turn off exercises for exercise day. Error: " + err.Error())
			return errors.New("Failed to turn off exercises for exercise day.")
		}

		newDifferential := differential - changed

		if newDifferential == 0 {
			return nil
		}

		for i := 0; i < newDifferential; i++ {
			newExercise := models.Exercise{
				ExerciseDayID: exerciseDayID,
			}
			newExercise.ID = uuid.New()

			err = database.CreateExerciseForExerciseDayInDatabase(newExercise)
			if err != nil {
				logger.Log.Info("Failed to create exercise in database. Error: " + err.Error())
				return errors.New("Failed to create exercise in database.")
			}
		}

	}

	return nil

}

func TurnOnExercisesForExerciseDayByAmount(amount int, exerciseDayID uuid.UUID) (int, error) {

	if amount == 0 {
		return 0, errors.New("Amount must be more than 0.")
	}

	turnedOn := 0

	exercises, err := database.GetExerciseByExerciseDayID(exerciseDayID)
	if err != nil {
		logger.Log.Info("Failed to get exercises for exercise-day. Error: " + err.Error())
		return 0, errors.New("Failed to get exercises for exercise-day.")
	}

	for _, exercise := range exercises {

		if !exercise.On {
			err = database.UpdateExerciseByTurningOnByExerciseID(exercise.ID)
			if err != nil {
				logger.Log.Info("Failed to turn on exercise. Error: " + err.Error())
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

func TurnOffExercisesForExerciseDayByAmount(amount int, exerciseDayID uuid.UUID) (int, error) {

	if amount == 0 {
		return 0, errors.New("Amount must be more than 0.")
	}

	turnedOff := 0

	exercises, err := database.GetExerciseByExerciseDayID(exerciseDayID)
	if err != nil {
		logger.Log.Info("Failed to get exercises for exercise-day. Error: " + err.Error())
		return 0, errors.New("Failed to get exercises for exercise-day.")
	}

	for _, exercise := range exercises {

		if exercise.On {
			err = database.UpdateExerciseByTurningOffByExerciseID(exercise.ID)
			if err != nil {
				logger.Log.Info("Failed to turn on exercise. Error: " + err.Error())
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
		logger.Log.Info("Failed to get enabled exercises. Error: " + err.Error())
		return errors.New("Failed to get enabled exercises.")
	}

	for _, exerciseDay := range exerciseDays {
		exerciseDayObject, err := ConvertExerciseDayToExerciseDayObject(exerciseDay)
		if err != nil {
			logger.Log.Info("Failed to convert exercise day to exercise day object. Error: " + err.Error())
			return errors.New("Failed to convert exercise day to exercise day object.")
		}

		err = CorrelateExerciseWithExerciseDay(exerciseDay.ID, exerciseDayObject.ExerciseInterval)
		if err != nil {
			logger.Log.Info("Failed to correlate exercise. Error: " + err.Error())
			return errors.New("Failed to correlate exercise.")
		}
	}

	return nil
}

func APICorrelateAllExercises(context *gin.Context) {

	err := CorrelateAllExercises()
	if err != nil {
		logger.Log.Info("Failed to correlate all exercises. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to correlate all exercises."})
		context.Abort()
		return
	}

	context.JSON(http.StatusOK, gin.H{"message": "All exercise correlated."})

}

func ConvertExerciseDayToExerciseDayObject(exerciseDay models.ExerciseDay) (exerciseDayObject models.ExerciseDayObject, err error) {
	exerciseDayObject = models.ExerciseDayObject{}
	err = nil

	// Convert to new type of exercise day
	if exerciseDay.UserID == nil && exerciseDay.GoalID != nil {
		goal, err := database.GetGoalUsingGoalID(*exerciseDay.GoalID)
		if err != nil {
			logger.Log.Info("Failed to get goal using goal ID. Error: " + err.Error())
			return exerciseDayObject, errors.New("Failed to get goal using goal ID.")
		}
		exerciseDay.UserID = &goal.UserID
		exerciseDay, err = database.UpdateExerciseDayInDB(exerciseDay)
		if err != nil {
			logger.Log.Info("Failed to save new exercise day. Error: " + err.Error())
			return exerciseDayObject, errors.New("Failed to save new exercise day.")
		}
		logger.Log.Info("Converted exercise day to new format.")
	}

	if exerciseDay.UserID == nil {
		logger.Log.Info("Exercise day conversion error.")
		return exerciseDayObject, errors.New("Exercise day conversion error.")
	}

	user, err := database.GetUserInformation(*exerciseDay.UserID)
	if err != nil {
		logger.Log.Info("Failed to get user using user ID. Error: " + err.Error())
		return exerciseDayObject, errors.New("Failed to get user using user ID.")
	}

	exerciseDayObject.User = user

	exercises, err := database.GetExerciseByExerciseDayID(exerciseDay.ID)
	if err != nil {
		logger.Log.Info("Failed to get exercises for day. Error: " + err.Error())
		return exerciseDayObject, errors.New("Failed to get exercises for day.")
	}

	exerciseObjects, err := ConvertExercisesToExerciseObjects(exercises)
	if err != nil {
		logger.Log.Info("Failed to convert exercises to exercise objects. Error: " + err.Error())
		return exerciseDayObject, errors.New("Failed to convert exercises to exercise objects.")
	}

	exerciseDayObject.Exercises = exerciseObjects

	exerciseDayObject.ExerciseInterval = 0
	for _, exerciseObject := range exerciseObjects {
		if exerciseObject.On {
			exerciseDayObject.ExerciseInterval += 1
		}
	}

	exerciseDayObject.CreatedAt = exerciseDay.CreatedAt
	exerciseDayObject.Date = exerciseDay.Date
	exerciseDayObject.DeletedAt = exerciseDay.DeletedAt
	exerciseDayObject.Enabled = exerciseDay.Enabled
	exerciseDayObject.ID = exerciseDay.ID
	exerciseDayObject.Note = exerciseDay.Note
	exerciseDayObject.UpdatedAt = exerciseDay.UpdatedAt

	return
}

func ConvertExerciseDaysToExerciseDayObjects(exerciseDays []models.ExerciseDay) (exerciseDayObjects []models.ExerciseDayObject, err error) {
	exerciseDayObjects = []models.ExerciseDayObject{}
	err = nil

	for _, exerciseDay := range exerciseDays {
		exerciseDayObject, err := ConvertExerciseDayToExerciseDayObject(exerciseDay)
		if err != nil {
			logger.Log.Info("Failed to convert exercise day to exercise day object. Ignoring... Error: " + err.Error())
			continue
		}
		exerciseDayObjects = append(exerciseDayObjects, exerciseDayObject)
	}

	return
}

func ConvertExerciseToExerciseObject(exercise models.Exercise) (exerciseObject models.ExerciseObject, err error) {
	exerciseObject = models.ExerciseObject{}
	err = nil

	operations, err := database.GetOperationsByExerciseID(exercise.ID)
	if err != nil {
		logger.Log.Info("Failed to get operations using exercise ID. Error: " + err.Error())
		return exerciseObject, errors.New("Failed to get operations using exercise ID.")
	}

	operationObjects, err := ConvertOperationsToOperationObjects(operations)
	if err != nil {
		logger.Log.Info("Failed to convert operations to operation objects. Error: " + err.Error())
		return exerciseObject, errors.New("Failed to convert operations to operation objects.")
	}

	exerciseObject.Operations = operationObjects

	if exercise.StravaID != nil {
		idString := exercise.StravaID
		array := strings.Split(*idString, ";")
		newArray := []string{}

		for _, stravaID := range array {
			if stravaID != "" {
				newArray = append(newArray, stravaID)
			}
		}
		if len(newArray) < 1 {
			exerciseObject.StravaID = nil
		} else {
			exerciseObject.StravaID = newArray
		}
	} else {
		exerciseObject.StravaID = nil
	}

	exerciseObject.CreatedAt = exercise.CreatedAt
	exerciseObject.DeletedAt = exercise.DeletedAt
	exerciseObject.Enabled = exercise.Enabled
	exerciseObject.ExerciseDay = exercise.ExerciseDayID
	exerciseObject.ID = exercise.ID
	exerciseObject.Note = exercise.Note
	exerciseObject.On = exercise.On
	exerciseObject.UpdatedAt = exercise.UpdatedAt
	exerciseObject.Duration = exercise.Duration

	return
}

func ConvertExercisesToExerciseObjects(exercises []models.Exercise) (exerciseObjects []models.ExerciseObject, err error) {
	exerciseObjects = []models.ExerciseObject{}
	err = nil

	for _, exercise := range exercises {
		exerciseObject, err := ConvertExerciseToExerciseObject(exercise)
		if err != nil {
			logger.Log.Info("Failed to convert exercise to exercise object. Ignoring... Error: " + err.Error())
			continue
		}
		exerciseObjects = append(exerciseObjects, exerciseObject)
	}

	return
}

func APIGetExerciseDay(context *gin.Context) {
	var exerciseID = context.Param("exercise_day_id")

	exerciseIDUUIID, err := uuid.Parse(exerciseID)
	if err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse ID."})
		context.Abort()
		return
	}

	// Get user ID
	userID, err := middlewares.GetAuthUsername(context.GetHeader("Authorization"))
	if err != nil {
		logger.Log.Info("Failed to verify user ID. Error: " + "Failed to verify user ID.")
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to verify user ID."})
		context.Abort()
		return
	}

	exerciseDay, err := database.GetExerciseDayByIDAndUserID(exerciseIDUUIID, userID)
	if err != nil {
		logger.Log.Info("Failed to get exercise day. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get exercise day."})
		context.Abort()
		return
	} else if exerciseDay == nil {
		context.JSON(http.StatusNotFound, gin.H{"error": "Failed to find exercise day."})
		context.Abort()
		return
	}

	exerciseDayObject, err := ConvertExerciseDayToExerciseDayObject(*exerciseDay)
	if err != nil {
		logger.Log.Info("Failed to get convert exercise day to exercise day object. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get convert exercise day to exercise day object."})
		context.Abort()
		return
	}

	// Return a response with all news posts
	context.JSON(http.StatusCreated, gin.H{"message": "Exercise day retrieved.", "exercise_day": exerciseDayObject})
}

func APIUpdateExerciseDay(context *gin.Context) {
	// Initialize variables
	var exerciseDayUpdateRequest models.ExerciseDayUpdateRequest
	var exerciseDay *models.ExerciseDay
	var exerciseDayID = context.Param("exercise_day_id")

	// Parse creation request
	if err := context.ShouldBindJSON(&exerciseDayUpdateRequest); err != nil {
		logger.Log.Info("Failed to parse update request. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse update request."})
		context.Abort()
		return
	}

	if len(strings.TrimSpace(exerciseDayUpdateRequest.Note)) > 255 {
		context.JSON(http.StatusBadRequest, gin.H{"error": "The note is too long."})
		context.Abort()
		return
	}

	exerciseDayIDUUID, err := uuid.Parse(exerciseDayID)
	if err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse ID."})
		context.Abort()
		return
	}

	exerciseDay, err = database.GetExerciseDayByID(exerciseDayIDUUID)
	if err != nil {
		logger.Log.Info("Failed to get exercise day. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get exercise day."})
		context.Abort()
		return
	} else if exerciseDay == nil {
		logger.Log.Info("Failed to find exercise day. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to find exercise day."})
		context.Abort()
		return
	}

	exerciseDay.Note = strings.TrimSpace(exerciseDayUpdateRequest.Note)

	*exerciseDay, err = database.UpdateExerciseDayInDatabase(*exerciseDay)
	if err != nil {
		logger.Log.Info("Failed to update exercise day. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update exercise day."})
		context.Abort()
		return
	}

	exerciseDayObject, err := ConvertExerciseDayToExerciseDayObject(*exerciseDay)
	if err != nil {
		logger.Log.Info("Failed to convert exercise day to exercise day object. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to convert exercise day to exercise day object."})
		context.Abort()
		return
	}

	context.JSON(http.StatusCreated, gin.H{"message": "Exercise day update.", "exercise_day": exerciseDayObject})
}

func APIUpdateExercise(context *gin.Context) {
	// Initialize variables
	var exerciseUpdateRequest models.ExerciseUpdateRequest
	var exercise models.Exercise

	// Get user ID
	userID, err := middlewares.GetAuthUsername(context.GetHeader("Authorization"))
	if err != nil {
		logger.Log.Info("Failed to verify user ID. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to verify user ID."})
		context.Abort()
		return
	}

	var exerciseID = context.Param("exercise_id")
	exerciseIDUUID, err := uuid.Parse(exerciseID)
	if err != nil {
		logger.Log.Info("Failed to verify exercise ID. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to verify exercise ID."})
		context.Abort()
		return
	}

	// Parse update request
	if err := context.ShouldBindJSON(&exerciseUpdateRequest); err != nil {
		logger.Log.Info("Failed to parse update request. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse update request."})
		context.Abort()
		return
	}

	exercise, err = database.GetAllExerciseByIDAndUserID(exerciseIDUUID, userID)
	if err != nil {
		logger.Log.Info("Failed to get exercise. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get exercise."})
		context.Abort()
		return
	}

	turnedOn := false
	turnedOff := false
	if !exercise.On && exerciseUpdateRequest.On {
		turnedOn = true
	} else if exercise.On && !exerciseUpdateRequest.On {
		turnedOff = true
	}

	exerciseDay, err := database.GetExerciseDayByIDAndUserID(exercise.ExerciseDayID, userID)
	if err != nil {
		logger.Log.Info("Failed to get exercise day. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get exercise day."})
		context.Abort()
		return
	} else if exerciseDay == nil {
		context.JSON(http.StatusNotFound, gin.H{"error": "Failed to find exercise day."})
		context.Abort()
		return
	}

	exerciseDayObject, err := ConvertExerciseDayToExerciseDayObject(*exerciseDay)
	if err != nil {
		logger.Log.Info("Failed to convert exercise day to exercise day object. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to convert exercise day to exercise day object."})
		context.Abort()
		return
	}

	if turnedOn && exerciseDayObject.ExerciseInterval >= 3 {
		context.JSON(http.StatusBadRequest, gin.H{"error": "You can only exercise three times in a day."})
		context.Abort()
		return
	}

	exerciseYear, exerciseWeek := exerciseDayObject.Date.ISOWeek()
	nowYear, nowWeek := time.Now().ISOWeek()
	if turnedOff && (nowYear != exerciseYear || exerciseWeek != nowWeek) {
		context.JSON(http.StatusBadRequest, gin.H{"error": "You can't remove exercise sessions after the week has ended."})
		context.Abort()
		return
	}

	exercise.Note = strings.TrimSpace(exerciseUpdateRequest.Note)
	exercise.On = exerciseUpdateRequest.On
	exercise.Duration = exerciseUpdateRequest.Duration

	if len(exercise.Note) > 255 {
		context.JSON(http.StatusBadRequest, gin.H{"error": "Invalid note length."})
		context.Abort()
		return
	}

	exercise, err = database.UpdateExerciseInDB(exercise)
	if err != nil {
		logger.Log.Info("Failed to update exercise. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update exercise."})
		context.Abort()
		return
	}

	exerciseObject, err := ConvertExerciseToExerciseObject(exercise)
	if err != nil {
		logger.Log.Info("Failed to get convert exercise to exercise object. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get convert exercise to exercise object."})
		context.Abort()
		return
	}

	context.JSON(http.StatusOK, gin.H{"message": "Exercise updated.", "exercise": exerciseObject})
}

func APICreateExercise(context *gin.Context) {
	// Initialize variables
	var exerciseCreationRequest models.ExerciseCreationRequest
	var exercise models.Exercise

	// Get user ID
	userID, err := middlewares.GetAuthUsername(context.GetHeader("Authorization"))
	if err != nil {
		logger.Log.Info("Failed to verify user ID. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to verify user ID."})
		context.Abort()
		return
	}

	// Parse creation request
	if err := context.ShouldBindJSON(&exerciseCreationRequest); err != nil {
		logger.Log.Info("Failed to parse creation request. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse creation request."})
		context.Abort()
		return
	}

	exerciseDay, err := database.GetExerciseDayByIDAndUserID(exerciseCreationRequest.ExerciseDayID, userID)
	if err != nil {
		logger.Log.Info("Failed to verify exercise day. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to verify exercise day."})
		context.Abort()
		return
	} else if exerciseDay == nil {
		context.JSON(http.StatusNotFound, gin.H{"error": "Failed to find exercise day."})
		context.Abort()
		return
	}

	exerciseDayObject, err := ConvertExerciseDayToExerciseDayObject(*exerciseDay)
	if err != nil {
		logger.Log.Info("Failed to convert exercise day to exercise day object. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to convert exercise day to exercise day object."})
		context.Abort()
		return
	}

	if exerciseDayObject.ExerciseInterval >= 3 {
		context.JSON(http.StatusBadRequest, gin.H{"error": "You can only exercise three times in a day."})
		context.Abort()
		return
	}

	exerciseYear, exerciseWeek := exerciseDayObject.Date.ISOWeek()
	nowYear, nowWeek := time.Now().ISOWeek()
	if nowYear != exerciseYear || exerciseWeek != nowWeek {
		context.JSON(http.StatusBadRequest, gin.H{"error": "You can't add exercise sessions outside the week the day happened."})
		context.Abort()
		return
	}

	now := time.Now()
	if utilities.SetClockToMinimum(exerciseDayObject.Date).After(utilities.SetClockToMinimum(now)) {
		context.JSON(http.StatusBadRequest, gin.H{"error": "You can't create exercises on days in the future."})
		context.Abort()
		return
	}

	exercise.On = exerciseCreationRequest.On
	exercise.Duration = exerciseCreationRequest.Duration
	exercise.Note = strings.TrimSpace(exerciseCreationRequest.Note)
	exercise.ExerciseDayID = exerciseCreationRequest.ExerciseDayID
	exercise.ID = uuid.New()

	exercise, err = database.CreateExerciseInDB(exercise)
	if err != nil {
		logger.Log.Info("Failed to create exercise. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create exercise."})
		context.Abort()
		return
	}

	exerciseObject, err := ConvertExerciseToExerciseObject(exercise)
	if err != nil {
		logger.Log.Info("Failed to get convert exercise to exercise object. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get convert exercise to exercise object."})
		context.Abort()
		return
	}

	context.JSON(http.StatusCreated, gin.H{"message": "Exercise created.", "exercise": exerciseObject})
}

func GetExercisesForWeekUsingUserID(timeReq time.Time, userID uuid.UUID) (exercises []models.Exercise, err error) {
	err = nil
	exercises = []models.Exercise{}

	var startTime time.Time
	startTimeWeek := 0
	var endTime time.Time
	endTimeWeek := 0
	_, timeReqWeek := timeReq.ISOWeek()

	// Find monday
	startTime, err = utilities.FindEarlierMonday(timeReq)
	if err != nil {
		logger.Log.Info("Failed to find earlier Monday for date. Error: " + err.Error())
		return exercises, errors.New("Failed to find earlier Monday for date.")
	}
	_, startTimeWeek = startTime.ISOWeek()

	// Find sunday
	endTime, err = utilities.FindNextSunday(timeReq)
	if err != nil {
		logger.Log.Info("Failed to find next Sunday for date. Error: " + err.Error())
		return exercises, errors.New("Failed to find next Sunday for date.")
	}
	_, endTimeWeek = endTime.ISOWeek()

	// Verify all dates are the same week
	if timeReqWeek != startTimeWeek || timeReqWeek != endTimeWeek {
		logger.Log.Info("Required time week: " + strconv.Itoa(timeReqWeek))
		logger.Log.Info("Start time week: " + strconv.Itoa(startTimeWeek))
		logger.Log.Info("End time week: " + strconv.Itoa(endTimeWeek))
		return exercises, errors.New("Managed to find dates outside of chosen week.")
	}

	exercises, err = database.GetValidExercisesBetweenDatesUsingDatesByUserID(userID, startTime, endTime)
	if err != nil {
		return exercises, err
	}

	return
}

func APIStravaCombine(context *gin.Context) {
	// Create week request
	var stravaIDs []string
	var exercises []models.Exercise
	var exerciseDayID *uuid.UUID

	// Parse request
	if err := context.ShouldBindJSON(&stravaIDs); err != nil {
		logger.Log.Info("Failed to parse request. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse request."})
		context.Abort()
		return
	}

	// Get user ID
	userID, err := middlewares.GetAuthUsername(context.GetHeader("Authorization"))
	if err != nil {
		logger.Log.Info("Failed to verify user ID. Error: " + "Failed to verify user ID.")
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to verify user ID."})
		context.Abort()
		return
	}

	for _, stravaID := range stravaIDs {
		exercise, err := database.GetExerciseForUserWithStravaID(userID, stravaID)
		if err != nil {
			logger.Log.Info("Failed to get exercise. Error: " + err.Error())
			context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get exercise."})
			context.Abort()
			return
		} else if exercise == nil {
			logger.Log.Info("Failed to verify exercise. Error: " + err.Error())
			context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to verify exercise."})
			context.Abort()
			return
		}

		if exerciseDayID == nil {
			exerciseDayID = &exercise.ExerciseDayID
		} else if exerciseDayID.String() != exercise.ExerciseDayID.String() {
			context.JSON(http.StatusBadRequest, gin.H{"error": "Exercises are not on the same day."})
			context.Abort()
			return
		}

		exercises = append(exercises, *exercise)
	}

	exerciseDay, err := database.GetExerciseDayByID(*exerciseDayID)
	if err != nil {
		logger.Log.Info("Failed to get exercise day. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get exercise day."})
		context.Abort()
		return
	} else if exerciseDay == nil {
		logger.Log.Info("Failed to verify exercise day.")
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to verify exercise day."})
		context.Abort()
		return
	}

	nowYear, nowWeek := time.Now().ISOWeek()
	exerciseDayYear, exerciseDayWeek := exerciseDay.Date.ISOWeek()

	if (nowYear != exerciseDayYear) || (nowWeek != exerciseDayWeek) {
		context.JSON(http.StatusBadRequest, gin.H{"error": "You can no longer combine exercises for this day."})
		context.Abort()
		return
	}

	sort.Slice(exercises, func(i, j int) bool {
		return exercises[j].CreatedAt.After(exercises[i].CreatedAt)
	})

	// Build string for Strava IDs
	stravaIDsString := ""
	for index, stravaID := range stravaIDs {
		if index != 0 {
			stravaIDsString += ";"
		}
		stravaIDsString += stravaID
	}

	var masterExercise = models.Exercise{}
	for index, exercise := range exercises {
		if index == 0 {
			exercise.StravaID = &stravaIDsString
			masterExercise = exercise

			logger.Log.Info(masterExercise.ID)

			_, err := database.UpdateExerciseInDB(exercise)
			if err != nil {
				logger.Log.Info("Failed to update exercise. Error: " + err.Error())
				context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update exercise."})
				context.Abort()
				return
			}
		} else {
			operations, err := database.GetOperationsByExerciseID(exercise.ID)
			if err != nil {
				logger.Log.Info("Failed to get operations for exercise. Error: " + err.Error())
				context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get operations for exercise."})
				context.Abort()
				return
			}

			for _, operation := range operations {
				logger.Log.Info(masterExercise.ID)

				operation.ExerciseID = masterExercise.ID
				operation.Exercise = masterExercise

				logger.Log.Info(operation.ExerciseID)

				operation, err = database.UpdateOperationInDB(operation)
				if err != nil {
					logger.Log.Info("Failed to update operation in DB. Error: " + err.Error())
					context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update operation in DB."})
					context.Abort()
					return
				}

				logger.Log.Info(operation.ExerciseID)
			}

			exercise.Enabled = false
			_, err = database.UpdateExerciseInDB(exercise)
			if err != nil {
				logger.Log.Info("Failed to update exercise. Error: " + err.Error())
				context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update exercise."})
				context.Abort()
				return
			}
		}
	}

	context.JSON(http.StatusCreated, gin.H{"message": "Exercises combined."})
}

func APIStravaDivide(context *gin.Context) {
	var exerciseIDString = context.Param("exercise_id")
	exerciseID, err := uuid.Parse(exerciseIDString)
	if err != nil {
		logger.Log.Info("Failed to verify exercise ID. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to verify exercise ID."})
		context.Abort()
		return
	}

	// Get user ID
	userID, err := middlewares.GetAuthUsername(context.GetHeader("Authorization"))
	if err != nil {
		logger.Log.Info("Failed to verify user ID. Error: " + "Failed to verify user ID.")
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to verify user ID."})
		context.Abort()
		return
	}

	exercise, err := database.GetExerciseByIDAndUserID(exerciseID, userID)
	if err != nil {
		logger.Log.Info("Failed to get exercise. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get exercise."})
		context.Abort()
		return
	} else if exercise == nil {
		logger.Log.Info("Failed to verify exercise.")
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to verify exercise."})
		context.Abort()
		return
	}

	exerciseObject, err := ConvertExerciseToExerciseObject(*exercise)
	if err != nil {
		logger.Log.Info("Failed to get exercise object. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get exercise object."})
		context.Abort()
		return
	}

	exerciseDay, err := database.GetExerciseDayByID(exercise.ExerciseDayID)
	if err != nil {
		logger.Log.Info("Failed to get exercise day. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get exercise day."})
		context.Abort()
		return
	} else if exerciseDay == nil {
		logger.Log.Info("Failed to verify exercise day.")
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to verify exercise day."})
		context.Abort()
		return
	}

	now := time.Now()
	nowYear, nowWeek := now.ISOWeek()
	exerciseDayYear, exerciseDayWeek := exerciseDay.Date.ISOWeek()

	if (nowYear != exerciseDayYear) || (nowWeek != exerciseDayWeek) {
		context.JSON(http.StatusBadRequest, gin.H{"error": "You can no longer divide exercises for this day."})
		context.Abort()
		return
	}

	for index, stravaID := range exerciseObject.StravaID {
		var currentExercise *models.Exercise
		if index != 0 {
			currentExercise = &models.Exercise{}
			currentExercise.ID = uuid.New()
			currentExercise.CreatedAt = now
			currentExercise.UpdatedAt = now
			currentExercise.ExerciseDayID = exerciseDay.ID
		} else {
			currentExercise = exercise
		}

		currentExercise.StravaID = &stravaID

		if index != 0 {
			_, err := database.CreateExerciseInDB(*currentExercise)
			if err != nil {
				logger.Log.Info("Failed to create new exercises in DB. Error: " + err.Error())
				context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create new exercises in DB."})
				context.Abort()
				return
			}
		}

		for _, exerciseOperation := range exerciseObject.Operations {

			if exerciseOperation.StravaID != nil && string(*exerciseOperation.StravaID) == stravaID {
				logger.Log.Info(currentExercise.ID)

				operation, err := database.GetOperationByIDAndUserID(exerciseOperation.ID, userID)
				if err != nil {
					logger.Log.Info("Failed to get operation in DB. Error: " + err.Error())
					context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get operation in DB."})
					context.Abort()
					return
				}

				operation.ExerciseID = currentExercise.ID

				_, err = database.UpdateOperationInDB(operation)
				if err != nil {
					logger.Log.Info("Failed to update operation in DB. Error: " + err.Error())
					context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update operation in DB."})
					context.Abort()
					return
				}
			}
		}

		if index == 0 {
			_, err := database.UpdateExerciseInDB(*currentExercise)
			if err != nil {
				logger.Log.Info("Failed to update exercises in DB. Error: " + err.Error())
				context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update exercise in DB."})
				context.Abort()
				return
			}
		}
	}

	context.JSON(http.StatusCreated, gin.H{"message": "Exercises divided."})
}
