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
	"github.com/google/uuid"
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
	goalStatus, goalID, err := database.VerifyUserGoalInSeason(userID, season.ID)
	if err != nil {
		log.Println("Failed to verify goal status. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to verify goal status."})
		context.Abort()
		return
	} else if !goalStatus {
		log.Println("User does not have a goal for season: " + season.ID.String())
		context.JSON(http.StatusBadRequest, gin.H{"error": "You don't have a goal set for this season."})
		context.Abort()
		return
	}

	// Check if week is sickleave
	sickLeave, err := database.GetUsedSickleaveForGoalWithinWeek(now, goalID)
	if err != nil {
		log.Println("Failed to verify sickleave. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify sickleave."})
		context.Abort()
		return
	} else if sickLeave != nil && sickLeave.Used {
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

		exerciseDayDB, err := database.GetExerciseDayByGoalAndDate(goalID, day.Date)
		if err != nil {
			log.Println("Failed to verify exercise status. Error: " + err.Error())
			context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify exercise status."})
			context.Abort()
			return
		}

		// If exercise day exists in DB, update it
		if exerciseDayDB != nil {

			exerciseDay, err := ConvertExerciseDayToExerciseDayObject(*exerciseDayDB)
			if err != nil {
				log.Println("Failed to convert exercise day to exercise day object. Error: " + err.Error())
				context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to convert exercise day to exercise day object."})
				context.Abort()
				return
			}

			// If no changes, don't update it
			if exerciseDay.ExerciseInterval != day.ExerciseInterval || exerciseDay.Note != html.EscapeString(strings.TrimSpace(day.Note)) {

				exerciseDayDB.Note = html.EscapeString(strings.TrimSpace(day.Note))

				*exerciseDayDB, err = database.UpdateExerciseDayInDatabase(*exerciseDayDB)
				if err != nil {
					log.Println("Failed to update exercise-day in database. Error: " + err.Error())
					context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update exercise-day in database."})
					context.Abort()
					return
				}

				exerciseDay, err := ConvertExerciseDayToExerciseDayObject(*exerciseDayDB)
				if err != nil {
					log.Println("Failed to convert exercise day to exercise day object. Error: " + err.Error())
					context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to convert exercise day to exercise day object."})
					context.Abort()
					return
				}

				// If the exercise-interval has changed, correlate the exercise table
				err = CorrelateExerciseWithExerciseDay(exerciseDay.ID, day.ExerciseInterval)
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
				Date:   time.Date(day.Date.Year(), day.Date.Month(), day.Date.Day(), 00, 00, 00, 00, requestLocation),
				Note:   day.Note,
				GoalID: goalID,
			}
			exerciseDay.ID = uuid.New()

			exerciseDayID, err := database.CreateExerciseDayForGoalInDatabase(exerciseDay)
			if err != nil {
				log.Println("Failed to save exercise-day in database. Error: " + err.Error())
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
					log.Println("Failed to update exercise in database. Error: " + err.Error())
					context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update exercise in database."})
					context.Abort()
					return
				}
			}
		}
	}

	// Get week for goal using current time
	weekReturn, err := GetExerciseDaysForWeekUsingGoal(now, goalID)
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
	goalStatus, goalID, err := database.VerifyUserGoalInSeason(userID, season.ID)
	if err != nil {
		log.Println("Failed to verify goal status. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to verify goal status."})
		context.Abort()
		return
	} else if !goalStatus {
		log.Println("User does not have a goal for season: " + season.ID.String())
		context.JSON(http.StatusBadRequest, gin.H{"error": "You don't have a goal set for this season."})
		context.Abort()
		return
	}

	// Get week for goal using current time
	week, err := GetExerciseDaysForWeekUsingGoal(now, goalID)
	if err != nil {
		log.Println("Failed to get calendar. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get calender."})
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

				exerciseDayObject, err := ConvertExerciseDayToExerciseDayObject(exercise)
				if err != nil {
					log.Println("Failed to convert exercise day to exercise day object. Error: " + err.Error())
					return models.Week{}, errors.New("Failed to convert exercise day to exercise day object.")
				}

				week.Days = append(week.Days, exerciseDayObject)
				added = true
				break

			}

		}

		if !added {
			newExercise := models.ExerciseDayObject{
				Date: currentDate.Truncate(24 * time.Hour),
			}
			week.Days = append(week.Days, newExercise)
		}

	}

	return week, nil

}

// Get full workout calender for the week from the database, and return to user
func APIGetExerciseDays(context *gin.Context) {

	// Get user ID
	userID, err := middlewares.GetAuthUsername(context.GetHeader("Authorization"))
	if err != nil {
		log.Println("Failed to verify user ID. Error: " + "Failed to verify user ID.")
		context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		context.Abort()
		return
	}

	var exerciseDays = []models.ExerciseDay{}

	// Get exercises from user
	goal, okay := context.GetQuery("goal")
	if !okay {
		exerciseDays, err = database.GetExerciseDaysForUserUsingUserID(userID)
		if err != nil {
			log.Println("Failed to get exercise. Error: " + err.Error())
			context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get exercise."})
			context.Abort()
			return
		}
	} else {
		goalID, err := uuid.Parse(goal)
		if err != nil {
			context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse goal ID."})
			context.Abort()
			return
		}

		// Get exercises from user
		exerciseDays, err = database.GetExerciseDaysForUserUsingUserIDAndGoalID(userID, goalID)
		if err != nil {
			log.Println("Failed to get exercise. Error: " + err.Error())
			context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get exercise."})
			context.Abort()
			return
		}
	}

	exerciseDayObjects, err := ConvertExerciseDaysToExerciseDayObjects(exerciseDays)
	if err != nil {
		log.Println("Failed to get convert exercise days to exercise day objects. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get convert exercise days to exercise day objects."})
		context.Abort()
		return
	}

	context.JSON(http.StatusOK, gin.H{"message": "Exercise retrieved.", "exercise": exerciseDayObjects})
}

// Change exercises to correlate with exercise days
func CorrelateExerciseWithExerciseDay(exerciseDayID uuid.UUID, exerciseDayExerciseInterval int) error {

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
				ExerciseDayID: exerciseDayID,
			}
			newExercise.ID = uuid.New()

			err = database.CreateExerciseForExerciseDayInDatabase(newExercise)
			if err != nil {
				log.Println("Failed to create exercise in database. Error: " + err.Error())
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
		log.Println("Failed to get exercises for exercise-day. Error: " + err.Error())
		return 0, errors.New("Failed to get exercises for exercise-day.")
	}

	for _, exercise := range exercises {

		if !exercise.On {
			err = database.UpdateExerciseByTurningOnByExerciseID(exercise.ID)
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

func TurnOffExercisesForExerciseDayByAmount(amount int, exerciseDayID uuid.UUID) (int, error) {

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
			err = database.UpdateExerciseByTurningOffByExerciseID(exercise.ID)
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
		exerciseDayObject, err := ConvertExerciseDayToExerciseDayObject(exerciseDay)
		if err != nil {
			log.Println("Failed to convert exercise day to exercise day object. Error: " + err.Error())
			return errors.New("Failed to convert exercise day to exercise day object.")
		}

		err = CorrelateExerciseWithExerciseDay(exerciseDay.ID, exerciseDayObject.ExerciseInterval)
		if err != nil {
			log.Println("Failed to correlate exercise. Error: " + err.Error())
			return errors.New("Failed to correlate exercise.")
		}
	}

	return nil
}

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

func ConvertExerciseDayToExerciseDayObject(exerciseDay models.ExerciseDay) (exerciseDayObject models.ExerciseDayObject, err error) {
	exerciseDayObject = models.ExerciseDayObject{}
	err = nil

	goal, err := database.GetGoalUsingGoalID(exerciseDay.GoalID)
	if err != nil {
		log.Println("Failed to get goal using goal ID. Error: " + err.Error())
		return exerciseDayObject, errors.New("Failed to get goal using goal ID.")
	}

	goalObject, err := ConvertGoalToGoalObject(goal)
	if err != nil {
		log.Println("Failed to convert goal to goal object. Error: " + err.Error())
		return exerciseDayObject, errors.New("Failed to convert goal to goal object.")
	}

	exerciseDayObject.Goal = goalObject

	exercises, err := database.GetExerciseByExerciseDayID(exerciseDay.ID)
	if err != nil {
		log.Println("Failed to get exercises for day. Error: " + err.Error())
		return exerciseDayObject, errors.New("Failed to get exercises for day.")
	}

	exerciseObjects, err := ConvertExercisesToExerciseObjects(exercises)
	if err != nil {
		log.Println("Failed to convert exercises to exercise objects. Error: " + err.Error())
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
			log.Println("Failed to convert exercise day to exercise day object. Ignoring... Error: " + err.Error())
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
		log.Println("Failed to get operations using exercise ID. Error: " + err.Error())
		return exerciseObject, errors.New("Failed to get operations using exercise ID.")
	}

	operationObjects, err := ConvertOperationsToOperationObjects(operations)
	if err != nil {
		log.Println("Failed to convert operations to operation objects. Error: " + err.Error())
		return exerciseObject, errors.New("Failed to convert operations to operation objects.")
	}

	exerciseObject.Operations = operationObjects

	exerciseObject.CreatedAt = exercise.CreatedAt
	exerciseObject.DeletedAt = exercise.DeletedAt
	exerciseObject.Enabled = exercise.Enabled
	exerciseObject.ExerciseDay = exercise.ExerciseDayID
	exerciseObject.ID = exercise.ID
	exerciseObject.Note = exercise.Note
	exerciseObject.On = exercise.On
	exerciseObject.UpdatedAt = exercise.UpdatedAt
	exerciseObject.Duration = exercise.Duration
	exerciseObject.StravaID = exercise.StravaID

	return
}

func ConvertExercisesToExerciseObjects(exercises []models.Exercise) (exerciseObjects []models.ExerciseObject, err error) {
	exerciseObjects = []models.ExerciseObject{}
	err = nil

	for _, exercise := range exercises {
		exerciseObject, err := ConvertExerciseToExerciseObject(exercise)
		if err != nil {
			log.Println("Failed to convert exercise to exercise object. Ignoring... Error: " + err.Error())
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

	exerciseDay, err := database.GetExerciseDayByID(exerciseIDUUIID)
	if err != nil {
		log.Println("Failed to get exercise day. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get exercise day."})
		context.Abort()
		return
	}

	exerciseDayObject, err := ConvertExerciseDayToExerciseDayObject(exerciseDay)
	if err != nil {
		log.Println("Failed to get convert exercise day to exercise day object. Error: " + err.Error())
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
	var exerciseDay models.ExerciseDay
	var exerciseDayID = context.Param("exercise_day_id")

	// Parse creation request
	if err := context.ShouldBindJSON(&exerciseDayUpdateRequest); err != nil {
		log.Println("Failed to parse update request. Error: " + err.Error())
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
		log.Println("Failed to get exercise day. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get exercise day."})
		context.Abort()
		return
	}

	exerciseDay.Note = strings.TrimSpace(exerciseDayUpdateRequest.Note)

	exerciseDay, err = database.UpdateExerciseDayInDatabase(exerciseDay)
	if err != nil {
		log.Println("Failed to update exercise day. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update exercise day."})
		context.Abort()
		return
	}

	exerciseDayObject, err := ConvertExerciseDayToExerciseDayObject(exerciseDay)
	if err != nil {
		log.Println("Failed to convert exercise day to exercise day object. Error: " + err.Error())
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
		log.Println("Failed to verify user ID. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to verify user ID."})
		context.Abort()
		return
	}

	var exerciseID = context.Param("exercise_id")
	exerciseIDUUID, err := uuid.Parse(exerciseID)
	if err != nil {
		log.Println("Failed to verify exercise ID. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to verify exercise ID."})
		context.Abort()
		return
	}

	// Parse update request
	if err := context.ShouldBindJSON(&exerciseUpdateRequest); err != nil {
		log.Println("Failed to parse update request. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse update request."})
		context.Abort()
		return
	}

	exercise, err = database.GetExerciseByIDAndUserID(exerciseIDUUID, userID)
	if err != nil {
		log.Println("Failed to get exercise. Error: " + err.Error())
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
		log.Println("Failed to get exercise day. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get exercise day."})
		context.Abort()
		return
	}

	exerciseDayObject, err := ConvertExerciseDayToExerciseDayObject(exerciseDay)
	if err != nil {
		log.Println("Failed to convert exercise day to exercise day object. Error: " + err.Error())
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
		log.Println("Failed to update exercise. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update exercise."})
		context.Abort()
		return
	}

	exerciseObject, err := ConvertExerciseToExerciseObject(exercise)
	if err != nil {
		log.Println("Failed to get convert exercise to exercise object. Error: " + err.Error())
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
		log.Println("Failed to verify user ID. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to verify user ID."})
		context.Abort()
		return
	}

	// Parse creation request
	if err := context.ShouldBindJSON(&exerciseCreationRequest); err != nil {
		log.Println("Failed to parse creation request. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse creation request."})
		context.Abort()
		return
	}

	exerciseDay, err := database.GetExerciseDayByIDAndUserID(exerciseCreationRequest.ExerciseDayID, userID)
	if err != nil {
		log.Println("Failed to verify exercise day. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to verify exercise day."})
		context.Abort()
		return
	}

	exerciseDayObject, err := ConvertExerciseDayToExerciseDayObject(exerciseDay)
	if err != nil {
		log.Println("Failed to convert exercise day to exercise day object. Error: " + err.Error())
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
	if exerciseDayObject.Date.Round(0).After(now.Round(0)) {
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
		log.Println("Failed to create exercise. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create exercise."})
		context.Abort()
		return
	}

	exerciseObject, err := ConvertExerciseToExerciseObject(exercise)
	if err != nil {
		log.Println("Failed to get convert exercise to exercise object. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get convert exercise to exercise object."})
		context.Abort()
		return
	}

	context.JSON(http.StatusCreated, gin.H{"message": "Exercise created.", "exercise": exerciseObject})
}

func GetExercisesForWeekUsingGoal(timeReq time.Time, goalID uuid.UUID) (exercises []models.Exercise, err error) {
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
		log.Println("Failed to find earlier Monday for date. Error: " + err.Error())
		return exercises, errors.New("Failed to find earlier Monday for date.")
	}
	_, startTimeWeek = startTime.ISOWeek()

	// Find sunday
	endTime, err = utilities.FindNextSunday(timeReq)
	if err != nil {
		log.Println("Failed to find next Sunday for date. Error: " + err.Error())
		return exercises, errors.New("Failed to find next Sunday for date.")
	}
	_, endTimeWeek = endTime.ISOWeek()

	// Verify all dates are the same week
	if timeReqWeek != startTimeWeek || timeReqWeek != endTimeWeek {
		log.Println("Required time week: " + strconv.Itoa(timeReqWeek))
		log.Println("Start time week: " + strconv.Itoa(startTimeWeek))
		log.Println("End time week: " + strconv.Itoa(endTimeWeek))
		return exercises, errors.New("Managed to find dates outside of chosen week.")
	}

	exercises, err = database.GetValidExercisesBetweenDatesUsingDates(goalID, startTime, endTime)
	if err != nil {
		return exercises, err
	}

	return
}
