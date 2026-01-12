package controllers

import (
	"aunefyren/treningheten/database"
	"aunefyren/treningheten/logger"
	"aunefyren/treningheten/middlewares"
	"aunefyren/treningheten/models"
	"aunefyren/treningheten/utilities"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/mroth/weightedrand/v2"
)

// Calculate a time set one week in the past and generates the debt for that week.
func ProcessLastWeek() {
	now := time.Now()

	// Get a date time in last week
	lastWeek := now.AddDate(0, 0, -7)
	logger.Log.Debug("process time is: " + lastWeek.String())
	lastWeekYear, lastWeekWeek := lastWeek.ISOWeek()
	logger.Log.Debug("last week was:" + strconv.Itoa(lastWeekWeek) + " " + strconv.Itoa(lastWeekYear))

	// Get ongoing season last week
	seasons, err := GetOngoingSeasonsFromDB(lastWeek)
	logger.Log.Debug("Found number of seasons active last week: " + strconv.Itoa(len(seasons)))

	if err != nil {
		logger.Log.Error("Returned error getting last weeks season: " + err.Error())
	} else {
		for _, season := range seasons {
			err = ProcessWeekOfSeason(season, lastWeek, true, true, nil)
			if err != nil {
				logger.Log.Error("Returned error processing week for season. Error: " + err.Error())
			}
		}
	}

	// Get current week and check for season
	// Send reminder if season started this week
	seasonsNow, err := GetOngoingSeasonsFromDB(now)
	if err != nil {
		logger.Log.Info("Returned error getting this weeks season: " + err.Error())
	} else {
		for _, seasonNow := range seasonsNow {
			seasonNowYear, seasonNowWeek := seasonNow.Start.ISOWeek()
			nowYear, nowWeek := now.ISOWeek()
			if seasonNowYear == nowYear && seasonNowWeek == nowWeek {

				seasonObject, err := ConvertSeasonToSeasonObject(seasonNow)
				if err != nil {
					logger.Log.Error("Returned error converting season to season object: " + err.Error())
				} else {
					err = utilities.SendSMTPSeasonStartEmail(seasonObject)
					if err != nil {
						logger.Log.Error("Returned error sending season start e-mail: " + err.Error())
					}
				}
			}
		}
	}

	logger.Log.Info("Done generating results for last week.")
}

func ProcessWeekOfSeason(season models.Season, pointInTime time.Time, generateDebt bool, generateAchievements bool, targetUser *uuid.UUID) (err error) {
	err = nil
	logger.Log.Debug("Processing week of season: " + season.Name)
	logger.Log.Debug("Point in time: " + pointInTime.String())

	// Get results for time given
	if generateDebt {
		logger.Log.Trace("generating debt for week results for point in time: " + pointInTime.String())
		weekResults, err := GenerateDebtForWeek(pointInTime, season, targetUser)
		if err != nil {
			logger.Log.Error("Got error generating last weeks debt. Error: " + err.Error())
			return errors.New("Got error generating last weeks debt.")
		}

		// Generate week achievements for point in time
		if generateAchievements {
			logger.Log.Debug("generating achievements for week")
			err := GenerateAchievementsForWeek(weekResults, targetUser)
			if err != nil {
				logger.Log.Error("Got error generating weeks achievements. Error: " + err.Error())
				return errors.New("Got error generating weeks achievements.")
			}
		}
	}

	seasonEndYear, seasonEndWeek := season.End.ISOWeek()
	pointInTimeYear, pointInTimeWeek := pointInTime.ISOWeek()

	if pointInTimeWeek == seasonEndWeek && pointInTimeYear == seasonEndYear {
		logger.Log.Info("season over, checking for achievements")

		seasonObject, err := ConvertSeasonToSeasonObject(season)
		if err != nil {
			logger.Log.Error("Got error converting season to season object. Error: " + err.Error())
			return errors.New("Got error converting season to season object.")
		} else {

			logger.Log.Trace("generating week results for point in time: " + pointInTime.String())
			pastWeeks, err := RetrieveWeekResultsFromSeasonWithinTimeframe(seasonObject.Start, pointInTime, seasonObject)
			if err != nil {
				logger.Log.Error("Got error getting season results. Error: " + err.Error())
				return errors.New("Got error getting season results.")
			} else {
				logger.Log.Debug("generating achievements for season")
				err = GenerateAchievementsForSeason(pastWeeks, targetUser)
				if err != nil {
					logger.Log.Error("Got error generating weeks achievements. Error: " + err.Error())
					return errors.New("Got error generating weeks achievements.")
				}
			}
		}
	}

	return
}

// Receives a time and generates resulting debts based on the results of that week. Should be run on weeks after the results are gathered.
func GenerateDebtForWeek(givenTime time.Time, season models.Season, targetUser *uuid.UUID) (models.WeekResults, error) {
	// Stop if not within season
	if season.Start.After(givenTime) || season.End.Before(givenTime) {
		logger.Log.Info("Not in the middle of a season. Returning.")
		return models.WeekResults{}, errors.New("Not in the middle of a season.")
	}

	seasonObject, err := ConvertSeasonToSeasonObject(season)
	if err != nil {
		logger.Log.Info("Failed to convert season to season object. Returning. Error: " + err.Error())
		return models.WeekResults{}, errors.New("Failed to convert season to season object.")
	}

	givenTimeMonday, err := utilities.FindEarlierMonday(givenTime)
	if err != nil {
		logger.Log.Info("Failed to find earliest point in the week. Error: " + err.Error())
		return models.WeekResults{}, errors.New("Failed to find earliest point in the week.")
	}

	givenTimeSunday, err := utilities.FindNextSunday(givenTime)
	if err != nil {
		logger.Log.Info("Failed to find latest point in the week. Error: " + err.Error())
		return models.WeekResults{}, errors.New("Failed to find latest point in the week.")
	}

	lastWeekArray, err := RetrieveWeekResultsFromSeasonWithinTimeframe(givenTimeMonday, givenTimeSunday, seasonObject)
	if err != nil {
		logger.Log.Info("Failed to retrieve last week for season. Returning. Error: " + err.Error())
		return models.WeekResults{}, errors.New("Failed to retrieve last week for season.")
	} else if len(lastWeekArray) != 1 {
		logger.Log.Info("Failed to retrieve ONE week for season. Returning.")
		logger.Log.Info("Got this: ")
		logger.Log.Info(lastWeekArray)
		logger.Log.Info("________________")
		logger.Log.Info("Given time Monday: " + givenTimeMonday.String())
		logger.Log.Info("Given time Sunday: " + givenTimeSunday.String())
		return models.WeekResults{}, errors.New("Failed to retrieve ONE week for season.")
	}

	lastWeek := lastWeekArray[0]

	for _, userWeekResults := range lastWeek.UserWeekResults {
		logger.Log.Trace("week results for week: " + strconv.FormatFloat(userWeekResults.WeekCompletion, 'f', -1, 64) + " for user: " + userWeekResults.UserID.String())
	}

	winners := []uuid.UUID{}
	losers := []uuid.UUID{}

	// Debug line
	logger.Log.Info("Timeframe week start: " + givenTimeMonday.String())

	// Find losers and winners
	for _, user := range lastWeek.UserWeekResults {
		// Debug log message
		logger.Log.Info("Potential winner/loser: ")
		logger.Log.Info(user)

		if user.Competing && user.WeekCompletion < 1 && !user.SickLeave && user.FullWeekParticipation {
			losers = append(losers, user.UserID)
		} else if user.Competing && user.WeekCompletion >= 1 && !user.SickLeave {
			winners = append(winners, user.UserID)
		}
	}

	winner := &uuid.UUID{}
	winner = nil

	if len(losers) == 0 {
		logger.Log.Info("No losers this week. Returning.")
		return lastWeek, nil
	}

	if len(winners) == 0 {
		logger.Log.Info("No winners this week. Returning.")
		return lastWeek, nil
	} else if len(winners) == 1 {
		winner = &winners[0]
	}

	_, weekNumber := givenTime.ISOWeek()

	for _, user := range losers {
		if targetUser != nil && user != *targetUser {
			logger.Log.Info("Function was called with a target user. This weeks loser is not the target user. Skipping...")
			continue
		}

		_, debtFound, err := database.GetDebtForWeekForUserInSeasonID(givenTime, user, seasonObject.ID)
		if err != nil {
			logger.Log.Info("Failed check for debt for '" + user.String() + "'. Skipping.")
			continue
		} else if debtFound {
			logger.Log.Info("Debt found for '" + user.String() + "'. Skipping.")
			continue
		}

		debt := models.Debt{}
		debt.Date = utilities.SetClockToMinimum(givenTime)
		debt.LoserID = user
		debt.WinnerID = winner
		debt.SeasonID = season.ID
		debt.ID = uuid.New()

		logger.Log.Info("Creating debt for '" + user.String() + "'.")

		debtObject, err := database.RegisterDebtInDB(debt)
		if err != nil {
			logger.Log.Info("Failed to log debt for '" + user.String() + "'. Skipping.")
			continue
		}

		if len(winners) == 1 {

			nextSunday, err := utilities.FindNextSunday(givenTime)
			if err != nil {
				logger.Log.Info("Failed to find next Sunday for date. Skipping.")
			} else {

				// Give achievement to winner for winning
				err = GiveUserAnAchievement(*winner, uuid.MustParse("bb964360-6413-47c2-8400-ee87b40365a7"), nextSunday)
				if err != nil {
					logger.Log.Info("Failed to give achievement for user '" + winner.String() + "'. Ignoring. Error: " + err.Error())
				}

				// Give achievement to loser for spinning wheel
				err = GiveUserAnAchievement(user, uuid.MustParse("d415fffc-ea99-4b27-8929-aeb02ae44da3"), nextSunday)
				if err != nil {
					logger.Log.Info("Failed to give achievement for user '" + user.String() + "'. Ignoring. Error: " + err.Error())
				}

				// Get loser object
				loserObject, err := database.GetAllUserInformation(user)
				if err != nil {
					logger.Log.Info("Failed to get object for user '" + user.String() + "'. Ignoring. Error: " + err.Error())
				} else {

					// Notify loser by e-mail
					err = utilities.SendSMTPForWeekLost(loserObject, weekNumber)
					if err != nil {
						logger.Log.Info("Failed to notify user '" + user.String() + "' by e-mail. Ignoring. Error: " + err.Error())
					}

				}

				// Notify loser by push
				err = PushNotificationsForWeekLost(user)
				if err != nil {
					logger.Log.Info("Failed to notify user '" + user.String() + "' by push. Ignoring. Error: " + err.Error())
				}

				// Get winner object
				winnerObject, err := database.GetAllUserInformation(*winner)
				if err != nil {
					logger.Log.Info("Failed to get object for user '" + user.String() + "'. Ignoring. Error: " + err.Error())
				} else {

					// Notify winner by e-mail
					err = utilities.SendSMTPForWheelSpinWin(winnerObject, weekNumber)
					if err != nil {
						logger.Log.Info("Failed to notify user '" + user.String() + "' by e-mail. Ignoring. Error: " + err.Error())
					}

				}

				// Notify winner by push
				err = PushNotificationsForWheelSpinWin(*winner, debt)
				if err != nil {
					logger.Log.Info("Failed to notify user '" + user.String() + "' by push. Ignoring. Error: " + err.Error())
				}

			}

		} else {

			// Get loser object
			loserObject, err := database.GetAllUserInformation(user)
			if err != nil {
				logger.Log.Info("Failed to get object for user '" + user.String() + "'. Ignoring. Error: " + err.Error())
			} else {

				// Notify loser by e-mail
				err = utilities.SendSMTPForWheelSpin(loserObject, weekNumber)
				if err != nil {
					logger.Log.Info("Failed to notify user '" + user.String() + "' by e-mail. Ignoring. Error: " + err.Error())
				}

			}

			// Notify loser by push
			err = PushNotificationsForWheelSpin(user, debtObject)
			if err != nil {
				logger.Log.Info("Failed to notify user '" + user.String() + "' by push. Ignoring. Error: " + err.Error())
			}

		}
	}

	logger.Log.Info("Done logging debt. Returning.")
	return lastWeek, nil
}

func APIGetUnchosenDebt(context *gin.Context) {

	// Get user ID
	userID, err := middlewares.GetAuthUsername(context.GetHeader("Authorization"))
	if err != nil {
		logger.Log.Info("Failed to parse session details. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse session details."})
		context.Abort()
		return
	}

	debts, debtFound, err := database.GetUnchosenDebtForUserByUserID(userID)
	if err != nil {
		logger.Log.Info("Failed to check for debt. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check for debt."})
		context.Abort()
		return
	}

	debtObjects, err := ConvertDebtsToDebtObjects(debts)
	if err != nil {
		logger.Log.Info("Failed to convert debt to debt objects. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to convert debt to debt objects."})
		context.Abort()
		return
	}

	if !debtFound {
		context.JSON(http.StatusOK, gin.H{"message": "No unchosen debt found."})
		return
	}

	context.JSON(http.StatusOK, gin.H{"message": "Unchosen debt found.", "debt": debtObjects})

}

func ConvertDebtToDebtObject(debt models.Debt) (models.DebtObject, error) {

	var debtObject models.DebtObject

	if debt.WinnerID != nil {
		user, err := database.GetUserInformation(*debt.WinnerID)
		if err != nil {
			logger.Log.Info("Failed to get user information for user '" + debt.Winner.ID.String() + "'. Creating blank user. Error: " + err.Error())
			user = models.User{
				FirstName: "Deleted",
				LastName:  "Deleted",
				Email:     "Deleted",
			}
		}

		debtObject.Winner = &user
	} else {
		debtObject.Winner = nil
	}

	user, err := database.GetUserInformation(debt.LoserID)
	if err != nil {
		logger.Log.Info("Failed to get user information for user '" + debt.Loser.ID.String() + "'. Creating blank user. Error: " + err.Error())
		user = models.User{
			FirstName: "Deleted",
			LastName:  "Deleted",
			Email:     "Deleted",
		}
	}

	debtObject.Loser = user

	season, err := database.GetSeasonByID(debt.SeasonID)
	if err != nil {
		logger.Log.Info("Failed to get season '" + debt.Season.ID.String() + "' in database. Returning. Error: " + err.Error())
		return models.DebtObject{}, err
	} else if season == nil {
		logger.Log.Info("Failed to find season '" + debt.Season.ID.String() + "' in database. Returning. Error: " + err.Error())
		return models.DebtObject{}, err
	}

	seasonObject, err := ConvertSeasonToSeasonObject(*season)
	if err != nil {
		logger.Log.Info("Failed to convert season '" + debt.Season.ID.String() + "' to season object. Returning. Error: " + err.Error())
		return models.DebtObject{}, err
	}
	debtObject.Season = seasonObject

	debtObject.CreatedAt = debt.CreatedAt
	debtObject.Date = debt.Date
	debtObject.DeletedAt = debt.DeletedAt
	debtObject.Enabled = debt.Enabled
	debtObject.ID = debt.ID
	debtObject.Paid = debt.Paid
	debtObject.UpdatedAt = debt.UpdatedAt

	return debtObject, nil

}

func ConvertDebtsToDebtObjects(debts []models.Debt) ([]models.DebtObject, error) {

	var debtObjects []models.DebtObject

	for _, debt := range debts {
		debtObject, err := ConvertDebtToDebtObject(debt)
		if err != nil {
			logger.Log.Info("Failed to convert debt to debt object. Returning. Error: " + err.Error())
			return []models.DebtObject{}, err
		}
		debtObjects = append(debtObjects, debtObject)
	}

	return debtObjects, nil

}

func APIGetDebt(context *gin.Context) {

	// Create user request
	var debtID = context.Param("debt_id")

	// Parse group id
	debtIDInt, err := uuid.Parse(debtID)
	if err != nil {
		logger.Log.Info("Failed to parse debt ID. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse Debt ID."})
		context.Abort()
		return
	}

	// Get user ID
	userID, err := middlewares.GetAuthUsername(context.GetHeader("Authorization"))
	if err != nil {
		logger.Log.Info("Failed to parse session details. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse session details."})
		context.Abort()
		return
	}

	debt, debtFound, err := database.GetDebtByDebtID(debtIDInt)
	if err != nil {
		logger.Log.Info("Failed to get debt. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get debt."})
		context.Abort()
		return
	} else if !debtFound {
		context.JSON(http.StatusBadRequest, gin.H{"error": "Debt not found."})
		context.Abort()
		return
	}

	debtObject, err := ConvertDebtToDebtObject(debt)
	if err != nil {
		logger.Log.Info("Failed to convert debt to debt object. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to convert debt to debt object."})
		context.Abort()
		return
	}

	debtDateMonday, err := utilities.FindEarlierMonday(debtObject.Date)
	if err != nil {
		logger.Log.Info("Failed to find earlier Monday. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to find earlier Monday."})
		context.Abort()
		return
	}

	debtDateSunday, err := utilities.FindNextSunday(debtObject.Date)
	if err != nil {
		logger.Log.Info("Failed to find next Sunday. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to find next Sunday."})
		context.Abort()
		return
	}

	lastWeekArray, err := RetrieveWeekResultsFromSeasonWithinTimeframe(debtDateMonday, debtDateSunday, debtObject.Season)
	if err != nil {
		logger.Log.Info("Failed to retrieve week for debt. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve week for debt."})
		context.Abort()
		return
	} else if len(lastWeekArray) != 1 {
		logger.Log.Info("Failed to retrieve ONE week for debt. Got: '" + strconv.Itoa(len(lastWeekArray)) + "'.")
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve ONE week for debt."})
		context.Abort()
		return
	}

	lastWeek := lastWeekArray[0]

	logger.Log.Info("Week result for debt have the date: " + lastWeek.WeekDate.String())

	winners := []models.UserWithTickets{}

	for _, user := range lastWeek.UserWeekResults {

		if user.Competing && user.WeekCompletion >= 1.0 && !user.SickLeave {
			userObject, err := database.GetAllUserInformation(user.UserID)
			if err != nil {
				logger.Log.Info("Failed to get user object. Error: " + err.Error())
				context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user object."})
				context.Abort()
				return
			}

			userWithTickets := models.UserWithTickets{
				User:    userObject,
				Tickets: user.CurrentStreak + 1,
			}

			winners = append(winners, userWithTickets)
		}

	}

	// Check for wheelviews and mark them as viewed if matching
	if debtObject.Winner != nil {

		logger.Log.Info("Checking for debt views for user '" + userID.String() + "'.")

		wheelview, wheelviewFound, err := database.GetUnviewedWheelviewByDebtIDAndUserID(userID, debtObject.ID)
		if err != nil {
			logger.Log.Info("Failed to retrieve wheelview for user '" + userID.String() + "'. Continuing. Error: " + err.Error())
		} else if wheelviewFound {
			err = database.SetWheelviewToViewedByID(wheelview.ID)
			if err != nil {
				logger.Log.Info("Failed to update wheelview for user '" + userID.String() + "'. Continuing. Error: " + err.Error())
			}
			logger.Log.Info("Debt marked as viewed for user '" + userID.String() + "'.")

			// If a view was viewed and the viewer was the winner, give the winning achievement.
			if debtObject.Winner.ID == userID {
				// Give achievement to winner for winning
				err = GiveUserAnAchievement(userID, uuid.MustParse("bb964360-6413-47c2-8400-ee87b40365a7"), time.Now())
				if err != nil {
					logger.Log.Info("Failed to give achievement for user '" + userID.String() + "'. Ignoring. Error: " + err.Error())
				}
			}
		}
	}

	context.JSON(http.StatusOK, gin.H{"message": "Debt found.", "debt": debtObject, "winners": winners})

}

func APIChooseWinnerForDebt(context *gin.Context) {

	// Create user request
	var debtID = context.Param("debt_id")

	// Parse group id
	debtIDInt, err := uuid.Parse(debtID)
	if err != nil {
		logger.Log.Info("Failed to parse debt ID. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse Debt ID."})
		context.Abort()
		return
	}

	// Get user ID
	userID, err := middlewares.GetAuthUsername(context.GetHeader("Authorization"))
	if err != nil {
		logger.Log.Info("Failed to parse session details. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse session details."})
		context.Abort()
		return
	}

	debt, debtFound, err := database.GetDebtByDebtID(debtIDInt)
	if err != nil {
		logger.Log.Info("Failed to get debt. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get debt."})
		context.Abort()
		return
	} else if !debtFound {
		context.JSON(http.StatusBadRequest, gin.H{"error": "Debt not found."})
		context.Abort()
		return
	} else if debt.LoserID != userID {
		context.JSON(http.StatusUnauthorized, gin.H{"error": "No access."})
		context.Abort()
		return
	}

	if debt.Winner != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": "Winner has already been chosen."})
		context.Abort()
		return
	}

	// Convert to debt object
	debtObject, err := ConvertDebtToDebtObject(debt)
	if err != nil {
		logger.Log.Info("Failed to convert debt to debt object. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to convert debt to debt object."})
		context.Abort()
		return
	}

	debtDateMonday, err := utilities.FindEarlierMonday(debtObject.Date)
	if err != nil {
		logger.Log.Info("Failed to find earlier Monday. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to find earlier Monday."})
		context.Abort()
		return
	}

	debtDateSunday, err := utilities.FindNextSunday(debtObject.Date)
	if err != nil {
		logger.Log.Info("Failed to find next Sunday. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to find next Sunday."})
		context.Abort()
		return
	}

	// Get weeks results
	lastWeekArray, err := RetrieveWeekResultsFromSeasonWithinTimeframe(debtDateMonday, debtDateSunday, debtObject.Season)
	if err != nil {
		logger.Log.Info("Failed to retrieve last week for season. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed process results."})
		context.Abort()
		return
	} else if len(lastWeekArray) != 1 {
		logger.Log.Info("Failed to retrieve ONE week for season.")
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed process results."})
		context.Abort()
		return
	}

	sundayDate, err := utilities.FindNextSunday(debtObject.Date)
	if err != nil {
		logger.Log.Info("Failed to find next Sunday. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to find next Sunday."})
		context.Abort()
		return
	}

	lastWeek := lastWeekArray[0]
	winners := []models.UserWithTickets{}

	// Find weeks winners
	for _, user := range lastWeek.UserWeekResults {

		if user.Competing && user.WeekCompletion >= 1 && !user.SickLeave {
			userObject, err := database.GetAllUserInformation(user.UserID)
			if err != nil {
				logger.Log.Info("Failed to get user object. Error: " + err.Error())
				context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user object."})
				context.Abort()
				return
			}

			userWithTickets := models.UserWithTickets{
				User:    userObject,
				Tickets: user.CurrentStreak + 1,
			}
			winners = append(winners, userWithTickets)
			logger.Log.Info("Contestant '" + userObject.FirstName + " " + userObject.LastName + "' with '" + strconv.Itoa(user.CurrentStreak+1) + "' tickets added.")
		}

	}

	// Add winners as choices
	choices := []weightedrand.Choice[uuid.UUID, int]{}
	for _, user := range winners {
		choice := weightedrand.Choice[uuid.UUID, int]{
			Item:   user.User.ID,
			Weight: user.Tickets,
		}
		choices = append(choices, choice)
	}

	// Create chooser
	chooser, err := weightedrand.NewChooser(choices...)
	if err != nil {
		logger.Log.Info("Failed start randomizer. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed start randomizer."})
		context.Abort()
		return
	}

	// Pick winner
	winnerID := chooser.Pick()

	// Update winner in DB
	database.UpdateDebtWinner(debtIDInt, winnerID)

	// Give achievement to loser for losing
	err = GiveUserAnAchievement(userID, uuid.MustParse("d415fffc-ea99-4b27-8929-aeb02ae44da3"), sundayDate)
	if err != nil {
		logger.Log.Info("Failed to give achievement for user '" + userID.String() + "'. Ignoring. Error: " + err.Error())
	}

	// Get user object
	winnerUser, err := database.GetUserInformation(winnerID)

	// Create wheel views
	for _, user := range winners {
		wheelview := models.Wheelview{
			UserID: user.User.ID,
			DebtID: debtIDInt,
			Viewed: false,
		}
		wheelview.ID = uuid.New()
		err = database.CreateWheelview(wheelview)
		if err != nil {
			logger.Log.Info("Create wheelview for user '" + user.User.ID.String() + "'. Error: " + err.Error())
		}

		// Notify winner by e-mail
		winnerObject, err := database.GetAllUserInformation(user.User.ID)
		if err != nil {
			logger.Log.Info("Failed to get object for user '" + user.User.ID.String() + "'. Ignoring. Error: " + err.Error())
		} else {

			_, weekNumber := debtObject.Date.ISOWeek()

			// Notify winner by e-mail
			err = utilities.SendSMTPForWheelSpinCheck(winnerObject, weekNumber)
			if err != nil {
				logger.Log.Info("Failed to notify user '" + user.User.ID.String() + "' by e-mail. Ignoring. Error: " + err.Error())
			}

		}

		// Notify winner by push
		err = PushNotificationsForWheelSpinCheck(user.User.ID, debt)
		if err != nil {
			logger.Log.Info("Failed to notify user '" + user.User.ID.String() + "' by push. Ignoring. Error: " + err.Error())
		}
	}

	// Respond to API response
	context.JSON(http.StatusOK, gin.H{"message": "Winner chosen.", "debt": debtObject, "winner": winnerUser})

}

func APIGetDebtOverview(context *gin.Context) {

	debtOverview := models.DebtOverview{
		UnviewedDebt:      []models.WheelviewObject{},
		UnspunLostDebt:    []models.DebtObject{},
		UnreceivedWonDebt: []models.DebtObject{},
		UnpaidLostDebt:    []models.DebtObject{},
	}

	// Get user ID
	userID, err := middlewares.GetAuthUsername(context.GetHeader("Authorization"))
	if err != nil {
		logger.Log.Info("Failed to parse session details. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse session details."})
		context.Abort()
		return
	}

	wheelviews, wheelviewsFound, err := database.GetUnviewedWheelviewByUserID(userID)
	if err != nil {
		logger.Log.Info("Failed to get unviewed spins. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get unviewed spins."})
		context.Abort()
		return
	} else if wheelviewsFound {
		wheelviewObjects, err := ConvertWheelviewsToWheelviewObjects(wheelviews)
		if err != nil {
			logger.Log.Info("Failed to convert wheelviews to wheelview objects. Error: " + err.Error())
			context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to convert wheelviews to wheelview objects."})
			context.Abort()
			return
		}
		debtOverview.UnviewedDebt = wheelviewObjects
	}

	debts, debtsFound, err := database.GetUnreceivedDebtByUserID(userID)
	if err != nil {
		logger.Log.Info("Failed to get unreceived debts. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get unviewed spins."})
		context.Abort()
		return
	} else if debtsFound {
		debtObjects, err := ConvertDebtsToDebtObjects(debts)
		if err != nil {
			logger.Log.Info("Failed to convert debts to debt objects. Error: " + err.Error())
			context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to convert debts to debt objects."})
			context.Abort()
			return
		}

		// Check if viewed by reciever
		for _, debt := range debtObjects {
			wheelview, wheelviewFound, err := database.GetWheelviewByDebtIDAndUserID(userID, debt.ID)
			if err != nil {
				logger.Log.Info("Failed to get wheelview for debt. Error: " + err.Error())
				context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get wheelview for debt."})
				context.Abort()
				return
			} else if !wheelviewFound || wheelview.Viewed {
				debtOverview.UnreceivedWonDebt = append(debtOverview.UnreceivedWonDebt, debt)
			}
		}
	}

	debts, debtsFound, err = database.GetUnchosenDebtForUserByUserID(userID)
	if err != nil {
		logger.Log.Info("Failed to get unreceived debts. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get unpsun spins."})
		context.Abort()
		return
	} else if debtsFound {
		debtObjects, err := ConvertDebtsToDebtObjects(debts)
		if err != nil {
			logger.Log.Info("Failed to convert debts to debt objects. Error: " + err.Error())
			context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to convert debts to debt objects."})
			context.Abort()
			return
		}
		debtOverview.UnspunLostDebt = debtObjects
	}

	debts, debtsFound, err = database.GetUnpaidDebtForUser(userID)
	if err != nil {
		logger.Log.Info("Failed to get unreceived debts. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get unpaid debt."})
		context.Abort()
		return
	} else if debtsFound {
		debtObjects, err := ConvertDebtsToDebtObjects(debts)
		if err != nil {
			logger.Log.Info("Failed to convert debts to debt objects. Error: " + err.Error())
			context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to convert debts to debt objects."})
			context.Abort()
			return
		}
		debtOverview.UnpaidLostDebt = debtObjects
	}

	context.JSON(http.StatusOK, gin.H{"message": "Debt overview retrieved.", "overview": debtOverview})

}

func APISetPrizeReceived(context *gin.Context) {

	// Create debt request
	var debtID = context.Param("debt_id")

	// Parse group id
	debtIDInt, err := uuid.Parse(debtID)
	if err != nil {
		logger.Log.Info("Failed to parse debt ID. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse Debt ID."})
		context.Abort()
		return
	}

	// Get user ID
	userID, err := middlewares.GetAuthUsername(context.GetHeader("Authorization"))
	if err != nil {
		logger.Log.Info("Failed to parse session details. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse session details."})
		context.Abort()
		return
	}

	err = database.UpdateDebtPaidStatus(debtIDInt, userID)
	if err != nil {
		logger.Log.Info("Failed to update payment status. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update payment status."})
		context.Abort()
		return
	}

	context.JSON(http.StatusOK, gin.H{"message": "Prize payment status updated."})

}

func APIGenerateDebtForWeek(context *gin.Context) {
	var debtCreationRequest models.DebtCreationRequest

	// Bind the incoming request body to the GenerateDebtRequest model
	if err := context.ShouldBindJSON(&debtCreationRequest); err != nil {
		// If there is an error binding the request, return a Bad Request response
		logger.Log.Error("Failed to parse request. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse request."})
		context.Abort()
		return
	}

	logger.Log.Debug("process time is: " + debtCreationRequest.Date.String())

	seasons, err := GetOngoingSeasonsFromDB(debtCreationRequest.Date)
	if err != nil {
		logger.Log.Error("Got error getting seasons from timeframe. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Got error getting seasons from timeframe."})
		context.Abort()
		return
	} else {
		for _, season := range seasons {
			err = ProcessWeekOfSeason(season, debtCreationRequest.Date, true, true, debtCreationRequest.TargetUser)
			if err != nil {
				logger.Log.Error("Got error processing week for season. Error: " + err.Error())
				context.JSON(http.StatusInternalServerError, gin.H{"error": "Got error processing week for season."})
				context.Abort()
				return
			}
		}
	}

	context.JSON(http.StatusOK, gin.H{"message": "Debt generated."})
}
