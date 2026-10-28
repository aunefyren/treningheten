package controllers

import (
	"errors"
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/aunefyren/treningheten/database"
	"github.com/aunefyren/treningheten/logger"
	"github.com/aunefyren/treningheten/middlewares"
	"github.com/aunefyren/treningheten/models"
	"github.com/aunefyren/treningheten/utilities"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Saved workouts ("carry-over" in the code): workouts beyond the weekly goal are saved and
// can make up a later short week. The UI only ever says "saved workouts". See
// docs/flexible-workouts.md.

// The bounds an admin can pick for a season's rule (APIRegisterSeason).
const (
	maxCarryOverMax   = 10
	maxCarryOverWeeks = 12
)

// carryOverSettings is a season's saved-workouts rule.
type carryOverSettings struct {
	max   int // most saved workouts held at once; 0 = off
	weeks int // how many weeks a saved workout lasts
}

func carryOverSettingsForSeason(season models.SeasonObject) carryOverSettings {
	return carryOverSettings{max: season.CarryOverMax, weeks: season.CarryOverWeeks}
}

func (settings carryOverSettings) enabled() bool {
	return settings.max > 0 && settings.weeks > 0
}

// carryBank is the saved workouts a participant holds, one entry per workout: the index of
// the last season week it can be used in. Entries are earned in week order and all last the
// same number of weeks, so the slice is always oldest (soonest to expire) first.
type carryBank []int

// live drops the saved workouts that have expired before the given week.
func (bank carryBank) live(week int) carryBank {
	live := carryBank{}
	for _, lastWeek := range bank {
		if lastWeek >= week {
			live = append(live, lastWeek)
		}
	}
	return live
}

// nextCarryBank is the saved-workouts rule for one week: given the bank entering the week
// (expired workouts are dropped first), the week's workouts and goal, whether the
// participant chose "Use saved workouts" for it and whether it is sick leave, it returns how
// many saved workouts the week uses and earns and the bank leaving it.
//
//   - Sick leave comes first: the week neither uses nor earns, but the bank keeps ageing.
//   - A short week uses saved workouts only when opted in, and only when they cover the
//     whole shortfall — then exactly the shortfall, oldest first. Otherwise none.
//   - A week beyond its goal saves the extra workouts, up to settings.max held in total;
//     each lasts settings.weeks weeks after this one.
//
// Shared by the week results (GetWeekResultForGoal) and, through UserWeekResults.GoalMet,
// by debts, the wheel and the achievements, so they can't judge a saved week differently.
func nextCarryBank(bank carryBank, workouts int, goal int, optedIn bool, sickLeave bool, week int, settings carryOverSettings) (used int, earned int, next carryBank) {
	if !settings.enabled() || goal <= 0 {
		return 0, 0, carryBank{}
	}

	next = bank.live(week)
	if sickLeave {
		return 0, 0, next
	}

	if shortfall := goal - workouts; shortfall > 0 {
		if optedIn && shortfall <= len(next) {
			used = shortfall
			next = append(carryBank{}, next[used:]...)
		}
		return used, 0, next
	}

	earned = workouts - goal
	if room := settings.max - len(next); earned > room {
		earned = room
	}
	if earned < 0 {
		earned = 0
	}
	for i := 0; i < earned; i++ {
		next = append(next, week+settings.weeks)
	}
	return 0, earned, next
}

// savedWorkoutsRule is the season's rule as one plain sentence, or "" when the season
// doesn't save workouts. The UI words it the same way (savedWorkoutsRule in functions.js).
func savedWorkoutsRule(max int, weeks int) string {
	if max <= 0 || weeks <= 0 {
		return ""
	}
	lasts := "1 week"
	if weeks != 1 {
		lasts = strconv.Itoa(weeks) + " weeks"
	}
	if max == 1 {
		return "1 extra workout can be saved, for " + lasts + "."
	}
	return "Up to " + strconv.Itoa(max) + " extra workouts can be saved, each for " + lasts + "."
}

// sundayReminderSavedWorkoutsNote is the Sunday reminder's line about saved workouts for one
// participant's running week, or "" when there is nothing useful to say: the goal is already
// met by workouts, the week is sick leave, or the saved workouts can't cover the gap.
func sundayReminderSavedWorkoutsNote(result models.UserWeekResults, goal int) string {
	if goal <= 0 || result.SickLeave || result.WeekCompletion >= 1 {
		return ""
	}
	if result.CarryUsed > 0 {
		return "Your saved workouts make up the rest of this week, so your goal counts as met."
	}

	workouts := int(math.Round(result.WeekCompletion * float64(goal)))
	shortfall := goal - workouts
	if result.CarryOptIn || shortfall <= 0 || result.CarriedIn < shortfall {
		return ""
	}
	return "You're " + pluralize(shortfall, "workout", "workouts") + " short of your goal, and you have " +
		pluralize(result.CarriedIn, "saved workout", "saved workouts") + " that can make up the difference. Choose \"Use saved workouts\" on the front page before the week ends."
}

// pluralize writes a count with the right form of its noun: "1 workout", "2 workouts".
func pluralize(count int, singular string, plural string) string {
	if count == 1 {
		return "1 " + singular
	}
	return strconv.Itoa(count) + " " + plural
}

// currentWeekResultsForSeason judges the running week of a season for every participant,
// walking the season from its first week so streaks and saved workouts are right.
func currentWeekResultsForSeason(seasonObject models.SeasonObject, now time.Time) (models.WeekResults, error) {
	monday, err := utilities.FindEarlierMonday(now)
	if err != nil {
		return models.WeekResults{}, err
	}
	sunday, err := utilities.FindNextSunday(now)
	if err != nil {
		return models.WeekResults{}, err
	}

	weeks, err := RetrieveWeekResultsFromSeasonWithinTimeframe(seasonObject.Start, utilities.SetClockToMaximum(sunday), seasonObject)
	if err != nil {
		return models.WeekResults{}, err
	}

	key := isoWeekKey(monday)
	for _, week := range weeks {
		if isoWeekKey(week.WeekDate) == key {
			return week, nil
		}
	}
	return models.WeekResults{}, errors.New("The season has no week running now.")
}

// userWeekResult picks one participant's result out of a week.
func userWeekResult(week models.WeekResults, goalID uuid.UUID) *models.UserWeekResults {
	for i := range week.UserWeekResults {
		if week.UserWeekResults[i].GoalID == goalID {
			return &week.UserWeekResults[i]
		}
	}
	return nil
}

// carryOverRequestContext validates what both "Use saved workouts" endpoints need: a
// season running now with the feature on, and a goal in it for the caller.
func carryOverRequestContext(context *gin.Context, now time.Time) (*models.Season, *models.Goal, bool) {
	seasonID, err := uuid.Parse(context.Param("season_id"))
	if err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse season ID."})
		context.Abort()
		return nil, nil, false
	}

	userID, err := middlewares.GetAuthUsername(context.GetHeader("Authorization"))
	if err != nil {
		logger.Log.Info("Failed to verify user ID. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to verify user ID."})
		context.Abort()
		return nil, nil, false
	}

	season, err := database.GetSeasonByID(seasonID)
	if err != nil {
		logger.Log.Info("Failed to get season. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get season."})
		context.Abort()
		return nil, nil, false
	} else if season == nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": "Season not found."})
		context.Abort()
		return nil, nil, false
	}

	if now.Before(season.Start) || now.After(season.End) {
		context.JSON(http.StatusBadRequest, gin.H{"error": "Saved workouts can only be used in a week of a running season."})
		context.Abort()
		return nil, nil, false
	}

	if season.CarryOverMax <= 0 {
		context.JSON(http.StatusBadRequest, gin.H{"error": "This season doesn't save extra workouts."})
		context.Abort()
		return nil, nil, false
	}

	goal, err := database.GetGoalFromUserWithinSeason(season.ID, userID)
	if err != nil {
		logger.Log.Info("Failed to verify goal status. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify goal status."})
		context.Abort()
		return nil, nil, false
	} else if goal == nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": "You don't have a goal set for this season."})
		context.Abort()
		return nil, nil, false
	}

	return season, goal, true
}

// APIRegisterCarryOverUse turns on "Use saved workouts" for the caller's running week in a
// season: if the week ends short, saved workouts make up the difference — but only if they
// cover all of it. Rejected without saved workouts, in a sick-leave week, or twice.
func APIRegisterCarryOverUse(context *gin.Context) {
	now := time.Now()

	season, goal, ok := carryOverRequestContext(context, now)
	if !ok {
		return
	}

	sickLeave, err := database.GetUsedSickleaveForGoalWithinWeek(now, goal.ID)
	if err != nil {
		logger.Log.Info("Failed to verify sick leave. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify sick leave."})
		context.Abort()
		return
	} else if sickLeave != nil && sickLeave.Used {
		context.JSON(http.StatusBadRequest, gin.H{"error": "This week is sick leave, so it can't use saved workouts."})
		context.Abort()
		return
	}

	existing, err := database.GetCarryOverUseForGoalWithinWeek(now, goal.ID)
	if err != nil {
		logger.Log.Info("Failed to check saved workouts for the week. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check saved workouts for the week."})
		context.Abort()
		return
	} else if existing != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": "Saved workouts are already on for this week."})
		context.Abort()
		return
	}

	seasonObject, err := ConvertSeasonToSeasonObject(*season)
	if err != nil {
		logger.Log.Info("Failed to convert season to season object. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get season."})
		context.Abort()
		return
	}

	week, err := currentWeekResultsForSeason(seasonObject, now)
	if err != nil {
		logger.Log.Info("Failed to work out the running week. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to count your saved workouts."})
		context.Abort()
		return
	}
	result := userWeekResult(week, goal.ID)
	if result == nil || result.CarriedIn < 1 {
		context.JSON(http.StatusBadRequest, gin.H{"error": "You have no saved workouts."})
		context.Abort()
		return
	}

	use := models.CarryOverUse{
		Enabled: true,
		GoalID:  goal.ID,
		Date:    utilities.SetClockToMinimum(now),
	}
	use.ID = uuid.New()

	if err := database.CreateCarryOverUseInDB(use); err != nil {
		logger.Log.Info("Failed to save the choice to use saved workouts. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to turn on saved workouts."})
		context.Abort()
		return
	}

	context.JSON(http.StatusOK, gin.H{"message": "Saved workouts will make up this week if they cover what's missing."})
}

// APIDeleteCarryOverUse undoes "Use saved workouts" for the caller's running week. Past
// weeks are settled and can't be changed.
func APIDeleteCarryOverUse(context *gin.Context) {
	now := time.Now()

	_, goal, ok := carryOverRequestContext(context, now)
	if !ok {
		return
	}

	disabled, err := database.DisableCarryOverUsesForGoalWithinWeek(now, goal.ID)
	if err != nil {
		logger.Log.Info("Failed to turn off saved workouts. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to turn off saved workouts."})
		context.Abort()
		return
	} else if disabled == 0 {
		context.JSON(http.StatusBadRequest, gin.H{"error": "Saved workouts aren't on for this week."})
		context.Abort()
		return
	}

	context.JSON(http.StatusOK, gin.H{"message": "Saved workouts won't be used this week."})
}
