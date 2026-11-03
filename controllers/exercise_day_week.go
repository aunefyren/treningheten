package controllers

import (
	"errors"
	"math"
	"net/http"
	"time"

	"github.com/aunefyren/treningheten/database"
	"github.com/aunefyren/treningheten/logger"
	"github.com/aunefyren/treningheten/middlewares"
	"github.com/aunefyren/treningheten/models"
	"github.com/aunefyren/treningheten/utilities"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// maxWorkoutsPerDay caps how many counted workouts one day can hold (APICreateExercise
// enforces it; the front-page calendar steps 0–3).
const maxWorkoutsPerDay = 3

const dayWorkoutLimitMessage = "A day can hold three workouts that count."

// dayLoggableError is the rule for adding a workout to a day: only in the current ISO week,
// and never on a day that hasn't happened yet. Shared by APICreateExercise and the day
// page's week strip so the UI can't offer what the API refuses.
func dayLoggableError(date time.Time, now time.Time) error {
	dateYear, dateWeek := date.ISOWeek()
	nowYear, nowWeek := now.ISOWeek()
	if dateYear != nowYear || dateWeek != nowWeek {
		return errors.New("You can only add workouts to days in the current week.")
	}

	if utilities.SetClockToMinimum(date).After(utilities.SetClockToMinimum(now)) {
		return errors.New("You can't log workouts on days in the future.")
	}

	return nil
}

// APIGetExerciseDayWeek returns the week around one of the caller's days, for the header
// of /exercises/:id. See docs/exercises.md, "Day header".
func APIGetExerciseDayWeek(context *gin.Context) {
	exerciseDayID, err := uuid.Parse(context.Param("exercise_day_id"))
	if err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse ID."})
		context.Abort()
		return
	}

	userID, err := middlewares.GetAuthUsername(context.GetHeader("Authorization"))
	if err != nil {
		logger.Log.Info("Failed to verify user ID. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to verify user ID."})
		context.Abort()
		return
	}

	exerciseDay, err := database.GetExerciseDayByIDAndUserID(exerciseDayID, userID)
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

	week, err := buildExerciseDayWeek(exerciseDay.Date, userID, time.Now())
	if err != nil {
		logger.Log.Info("Failed to build week for exercise day. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get week."})
		context.Abort()
		return
	}

	context.JSON(http.StatusOK, gin.H{"week": week, "message": "Week retrieved."})
}

// buildExerciseDayWeek assembles the week holding the given date: the seven days with
// their counted workouts, which of them can be logged as of now, and the week's result in
// every season the user had a goal in.
func buildExerciseDayWeek(date time.Time, userID uuid.UUID, now time.Time) (models.ExerciseDayWeek, error) {
	week := models.ExerciseDayWeek{
		WorkoutLimit: maxWorkoutsPerDay,
		Days:         []models.ExerciseDayWeekDay{},
		Seasons:      []models.ExerciseDayWeekSeason{},
	}

	calendar, err := GetExerciseDaysForWeekUsingUserID(date, userID)
	if err != nil {
		return models.ExerciseDayWeek{}, err
	}

	for _, day := range calendar.Days {
		weekDay := models.ExerciseDayWeekDay{
			Date:     day.Date,
			Workouts: day.ExerciseInterval,
			Loggable: dayLoggableError(day.Date, now) == nil,
		}
		if day.ID != uuid.Nil {
			id := day.ID
			weekDay.ID = &id
		}
		week.Workouts += day.ExerciseInterval
		week.Loggable = week.Loggable || weekDay.Loggable
		week.Days = append(week.Days, weekDay)
	}

	seasons, err := exerciseDayWeekSeasons(date, userID)
	if err != nil {
		return models.ExerciseDayWeek{}, err
	}
	week.Seasons = seasons

	return week, nil
}

// exerciseDayWeekSeasons computes the week's result in each season that overlaps it and
// that the user had joined by the week's end. The numbers come from the season walk —
// the leaderboard's own rule, sick leave and saved workouts included — rather than a second
// count.
func exerciseDayWeekSeasons(date time.Time, userID uuid.UUID) ([]models.ExerciseDayWeekSeason, error) {
	results := []models.ExerciseDayWeekSeason{}

	weekStart, err := utilities.FindEarlierMonday(date)
	if err != nil {
		return results, err
	}
	weekStart = utilities.SetClockToMinimum(weekStart)
	weekEnd, err := utilities.FindNextSunday(date)
	if err != nil {
		return results, err
	}
	weekEnd = utilities.SetClockToMaximum(weekEnd)

	seasons, err := database.GetAllEnabledSeasons()
	if err != nil {
		return results, err
	}

	for _, season := range seasons {
		if season.Start.After(weekEnd) || season.End.Before(weekStart) {
			continue
		}

		seasonObject, err := ConvertSeasonToSeasonObject(season)
		if err != nil {
			return results, err
		}

		var goal *models.GoalObject
		for i := range seasonObject.Goals {
			if seasonObject.Goals[i].User.ID == userID && seasonObject.Goals[i].Enabled {
				goal = &seasonObject.Goals[i]
				break
			}
		}
		if goal == nil || goal.ExerciseInterval <= 0 || goal.CreatedAt.After(weekEnd) {
			continue
		}

		// Walk the season up to this week, like the leaderboard does: the streak and the
		// saved workouts entering the week depend on every week before it.
		weeks, err := RetrieveWeekResultsFromSeasonWithinTimeframe(seasonObject.Start, weekEnd, seasonObject)
		if err != nil {
			return results, err
		}

		var result *models.UserWeekResults
		key := isoWeekKey(weekStart)
		for i := range weeks {
			if isoWeekKey(weeks[i].WeekDate) == key {
				result = userWeekResult(weeks[i], goal.ID)
				break
			}
		}
		if result == nil {
			continue
		}

		workouts := int(math.Round(result.WeekCompletion * float64(goal.ExerciseInterval)))

		results = append(results, models.ExerciseDayWeekSeason{
			ID:         season.ID,
			Name:       season.Name,
			Goal:       goal.ExerciseInterval,
			Workouts:   workouts,
			Completion: result.WeekCompletion,
			SickLeave:  result.SickLeave,
			GoalMet:    result.GoalMet(),
			CarryUsed:  result.CarryUsed,
		})
	}

	return results, nil
}
