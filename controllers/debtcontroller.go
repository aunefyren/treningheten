package controllers

import (
	"aunefyren/treningheten/database"
	"aunefyren/treningheten/middlewares"
	"aunefyren/treningheten/models"
	"aunefyren/treningheten/utilities"
	"errors"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/mroth/weightedrand/v2"
)

// Calcluates a time set one week in the past and generates the debt for that week.
func GenerateLastWeeksDebt() {

	// Get a date time in last week
	lastWeek := time.Now().AddDate(0, 0, -7)

	// Get results for last week
	weekResults, err := GenerateDebtForWeek(lastWeek)
	if err != nil {
		log.Println("Returned error generating last weeks debt: " + err.Error())
	}

	// Generate week achivements for last week
	err = GenerateAchivementsForWeek(weekResults)
	if err != nil {
		log.Println("Returned error generating weeks achivements: " + err.Error())
	}

	// Get ongoing season last week
	season, seasonFound, err := GetOngoingSeasonFromDB(lastWeek)
	if err != nil {
		log.Println("Returned error getting last weeks season: " + err.Error())
	} else if seasonFound {
		_, lastWeekWeek := lastWeek.ISOWeek()
		_, seasonEndWeek := season.End.ISOWeek()

		if lastWeekWeek == seasonEndWeek {

			log.Println("Season over, checking for achivements.")

			seasonObject, err := ConvertSeasonToSeasonObject(season)
			if err != nil {
				log.Println("Returned error converting season to season object: " + err.Error())
			} else {

				pastWeeks, err := RetrieveWeekResultsFromSeasonWithinTimeframe(seasonObject.Start, lastWeek, seasonObject)
				if err != nil {
					log.Println("Returned error getting season results: " + err.Error())
				} else {

					err = GenerateAchivementsForSeason(pastWeeks)
					if err != nil {
						log.Println("Returned error generating weeks achivements: " + err.Error())
					}

				}

			}
		}
	}

	// Get current week and check for season
	// Send reminder if season started this week
	now := time.Now()
	seasonNow, seasonNowFound, err := GetOngoingSeasonFromDB(now)
	if err != nil {
		log.Println("Returned error getting this weeks season: " + err.Error())
	} else if seasonNowFound {
		seasonNowYear, seasonNowWeek := seasonNow.Start.ISOWeek()
		nowYear, nowWeek := now.ISOWeek()
		if seasonNowYear == nowYear && seasonNowWeek == nowWeek {

			seasonObject, err := ConvertSeasonToSeasonObject(season)
			if err != nil {
				log.Println("Returned error converting season to season object: " + err.Error())
			} else {
				err = utilities.SendSMTPSeasonStartEmail(seasonObject)
				if err != nil {
					log.Println("Returned error sending season start e-mail: " + err.Error())
				}
			}

		}
	}

	return

}

// Recieves a time and generates resulting debts based on the results of that week. Should be run on weeks after the results are gathered.
func GenerateDebtForWeek(givenTime time.Time) (models.WeekResults, error) {

	// Get current season
	season, seasonFound, err := GetOngoingSeasonFromDB(givenTime)
	if err != nil {
		log.Println("Failed to verify current season status. Returning. Error: " + err.Error())
		return models.WeekResults{}, errors.New("Failed to verify current season status.")
	} else if !seasonFound {
		log.Println("Failed to verify current season status. Returning. Error: No active or future seasons found.")
		return models.WeekResults{}, errors.New("Failed to verify current season status.")
	}

	// Stop if not within season
	if season.Start.After(givenTime) || season.End.Before(givenTime) {
		log.Println("Not in the middle of a season. Returning.")
		return models.WeekResults{}, errors.New("Not in the middle of a season.")
	}

	seasonObject, err := ConvertSeasonToSeasonObject(season)
	if err != nil {
		log.Println("Failed to convert season to season object. Returning. Error: " + err.Error())
		return models.WeekResults{}, errors.New("Failed to convert season to season object.")
	}

	lastWeekArray, err := RetrieveWeekResultsFromSeasonWithinTimeframe(givenTime.AddDate(0, 0, -7), givenTime, seasonObject)
	if err != nil {
		log.Println("Failed to retrieve last week for season. Returning. Error: " + err.Error())
		return models.WeekResults{}, errors.New("Failed to retrieve last week for season.")
	} else if len(lastWeekArray) != 1 {
		log.Println("Failed to retrieve ONE week for season. Returning. Error: " + err.Error())
		return models.WeekResults{}, errors.New("Failed to retrieve ONE week for season.")
	}

	lastWeek := lastWeekArray[0]

	winners := []models.User{}
	losers := []models.User{}

	for _, user := range lastWeek.UserWeekResults {

		if user.Competing && user.WeekCompletion < 1 && !user.Sickleave {
			losers = append(losers, user.User)
		} else if user.Competing && user.WeekCompletion >= 1 && !user.Sickleave {
			winners = append(winners, user.User)
		}

	}

	winner := uuid.UUID{}

	if len(losers) == 0 {
		log.Println("No losers this week. Returning.")
		return lastWeek, nil
	}

	if len(winners) == 0 {
		log.Println("No winners this week. Returning.")
		return lastWeek, nil
	} else if len(winners) == 1 {
		winner = winners[0].ID
	}

	for _, user := range losers {

		_, debtFound, err := database.GetDebtForWeekForUserInSeasonID(givenTime, user.ID, seasonObject.ID)
		if err != nil {
			log.Println("Failed check for debt for '" + user.ID.String() + "'. Skipping.")
			continue
		} else if debtFound {
			log.Println("Debt found for '" + user.ID.String() + "'. Skipping.")
			continue
		}

		debt := models.Debt{}
		debt.Date = givenTime.Truncate(24 * time.Hour)
		debt.LoserID = user.ID
		debt.WinnerID = &winner
		debt.SeasonID = season.ID
		debt.ID = uuid.New()

		log.Println("Creating debt for '" + user.ID.String() + "'.")

		err = database.RegisterDebtInDB(debt)
		if err != nil {
			log.Println("Failed to log debt for '" + user.ID.String() + "'. Skipping.")
			continue
		}

		if len(winners) == 1 {

			nextSunday, err := utilities.FindNextSunday(givenTime)
			if err != nil {
				log.Println("Failed to find next Sunday for date. Skipping.")
			} else {

				// Give achivement to winner for winning
				err = GiveUserAnAchivement(winner, uuid.MustParse("bb964360-6413-47c2-8400-ee87b40365a7"), nextSunday)
				if err != nil {
					log.Println("Failed to give achivement for user '" + winner.String() + "'. Ignoring. Error: " + err.Error())
				}

				// Give achivement to loser for spinning wheel
				err = GiveUserAnAchivement(user.ID, uuid.MustParse("d415fffc-ea99-4b27-8929-aeb02ae44da3"), nextSunday)
				if err != nil {
					log.Println("Failed to give achivement for user '" + user.ID.String() + "'. Ignoring. Error: " + err.Error())
				}

				// Get loser object
				loserObject, err := database.GetAllUserInformation(user.ID)
				if err != nil {
					log.Println("Failed to get object for user '" + user.ID.String() + "'. Ignoring. Error: " + err.Error())
				} else {

					// Notify loser by e-mail
					err = utilities.SendSMTPForWeekLost(loserObject)
					if err != nil {
						log.Println("Failed to notify user '" + user.ID.String() + "' by e-mail. Ignoring. Error: " + err.Error())
					}

				}

				// Notify loser by push
				err = PushNotificationsForWeekLost(user.ID)
				if err != nil {
					log.Println("Failed to notify user '" + user.ID.String() + "' by push. Ignoring. Error: " + err.Error())
				}

				// Get winner object
				winnerObject, err := database.GetAllUserInformation(winner)
				if err != nil {
					log.Println("Failed to get object for user '" + user.ID.String() + "'. Ignoring. Error: " + err.Error())
				} else {

					// Notify winner by e-mail
					err = utilities.SendSMTPForWheelSpinWin(winnerObject)
					if err != nil {
						log.Println("Failed to notify user '" + user.ID.String() + "' by e-mail. Ignoring. Error: " + err.Error())
					}

				}

				// Notify winner by push
				err = PushNotificationsForWheelSpinWin(winner)
				if err != nil {
					log.Println("Failed to notify user '" + user.ID.String() + "' by push. Ignoring. Error: " + err.Error())
				}

			}

		} else {

			// Get loser object
			loserObject, err := database.GetAllUserInformation(user.ID)
			if err != nil {
				log.Println("Failed to get object for user '" + user.ID.String() + "'. Ignoring. Error: " + err.Error())
			} else {

				// Notify loser by e-mail
				err = utilities.SendSMTPForWheelSpin(loserObject)
				if err != nil {
					log.Println("Failed to notify user '" + user.ID.String() + "' by e-mail. Ignoring. Error: " + err.Error())
				}

			}

			// Notify loser by push
			err = PushNotificationsForWheelSpin(user.ID)
			if err != nil {
				log.Println("Failed to notify user '" + user.ID.String() + "' by push. Ignoring. Error: " + err.Error())
			}

		}
	}

	log.Println("Done logging debt. Returning.")
	return lastWeek, nil

}

func APIGetUnchosenDebt(context *gin.Context) {

	// Get user ID
	userID, err := middlewares.GetAuthUsername(context.GetHeader("Authorization"))
	if err != nil {
		log.Println("Failed to parse session details. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse session details."})
		context.Abort()
		return
	}

	debts, debtFound, err := database.GetUnchosenDebtForUserByUserID(userID)
	if err != nil {
		log.Println("Failed to check for debt. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check for debt."})
		context.Abort()
		return
	}

	debtObjects, err := ConvertDebtsToDebtObjects(debts)
	if err != nil {
		log.Println("Failed to convert debt to debt objects. Error: " + err.Error())
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
			log.Println("Failed to get user information for user '" + debt.Winner.ID.String() + "'. Creating blank user. Error: " + err.Error())
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
		log.Println("Failed to get user information for user '" + debt.Loser.ID.String() + "'. Creating blank user. Error: " + err.Error())
		user = models.User{
			FirstName: "Deleted",
			LastName:  "Deleted",
			Email:     "Deleted",
		}
	}

	debtObject.Loser = user

	season, err := database.GetSeasonByID(debt.SeasonID)
	if err != nil {
		log.Println("Failed to get season '" + debt.Season.ID.String() + "' in database. Returning. Error: " + err.Error())
		return models.DebtObject{}, err
	}
	seasonObject, err := ConvertSeasonToSeasonObject(season)
	if err != nil {
		log.Println("Failed to convert season '" + debt.Season.ID.String() + "' to season object. Returning. Error: " + err.Error())
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
			log.Println("Failed to convert debt to debt object. Returning. Error: " + err.Error())
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
		log.Println("Failed to parse debt ID. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse Debt ID."})
		context.Abort()
		return
	}

	// Get user ID
	userID, err := middlewares.GetAuthUsername(context.GetHeader("Authorization"))
	if err != nil {
		log.Println("Failed to parse session details. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse session details."})
		context.Abort()
		return
	}

	debt, debtFound, err := database.GetDebtByDebtID(debtIDInt)
	if err != nil {
		log.Println("Failed to get debt. Error: " + err.Error())
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
		log.Println("Failed to convert debt to debt object. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to convert debt to debt object."})
		context.Abort()
		return
	}

	debtDateMonday, err := utilities.FindEarlierMonday(debtObject.Date)
	if err != nil {
		log.Println("Failed to find earlier Monday. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to find earlier Monday."})
		context.Abort()
		return
	}

	debtDateFriday, err := utilities.FindNextSunday(debtObject.Date)
	if err != nil {
		log.Println("Failed to find next Sunday. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to find next Sunday."})
		context.Abort()
		return
	}

	lastWeekArray, err := RetrieveWeekResultsFromSeasonWithinTimeframe(debtDateMonday.AddDate(0, 0, -7), debtDateFriday, debtObject.Season)
	if err != nil {
		log.Println("Failed to retrieve week for debt. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve week for debt."})
		context.Abort()
		return
	} else if len(lastWeekArray) != 1 {
		log.Println("Failed to retrieve ONE week for debt. Got: '" + strconv.Itoa(len(lastWeekArray)) + "'.")
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve ONE week for debt."})
		context.Abort()
		return
	}

	lastWeek := lastWeekArray[0]

	log.Println("Week result for debt have the date: " + lastWeek.WeekDate.String())

	winners := []models.UserWithTickets{}

	for _, user := range lastWeek.UserWeekResults {

		if user.Competing && user.WeekCompletion >= 1.0 && !user.Sickleave {
			userWithTickets := models.UserWithTickets{
				User:    user.User,
				Tickets: user.CurrentStreak + 1,
			}
			winners = append(winners, userWithTickets)
		}

	}

	// Check for wheelviews and mark them as viewed if matching
	if debtObject.Winner != nil {

		log.Println("Checking for debt views for user '" + userID.String() + "'.")

		wheelview, wheelviewFound, err := database.GetUnviewedWheelviewByDebtIDAndUserID(userID, debtObject.ID)
		if err != nil {
			log.Println("Failed to retrieve wheelview for user '" + userID.String() + "'. Continuing. Error: " + err.Error())
		} else if wheelviewFound {
			err = database.SetWheelviewToViewedByID(wheelview.ID)
			if err != nil {
				log.Println("Failed to update wheelview for user '" + userID.String() + "'. Continuing. Error: " + err.Error())
			}
			log.Println("Debt marked as viewed for user '" + userID.String() + "'.")
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
		log.Println("Failed to parse debt ID. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse Debt ID."})
		context.Abort()
		return
	}

	// Get user ID
	userID, err := middlewares.GetAuthUsername(context.GetHeader("Authorization"))
	if err != nil {
		log.Println("Failed to parse session details. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse session details."})
		context.Abort()
		return
	}

	debt, debtFound, err := database.GetDebtByDebtID(debtIDInt)
	if err != nil {
		log.Println("Failed to get debt. Error: " + err.Error())
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
		log.Println("Failed to convert debt to debt object. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to convert debt to debt object."})
		context.Abort()
		return
	}

	// Get weeks results
	lastWeekArray, err := RetrieveWeekResultsFromSeasonWithinTimeframe(debtObject.Date.AddDate(0, 0, -7), debtObject.Date, debtObject.Season)
	if err != nil {
		log.Println("Failed to retrieve last week for season. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed process results."})
		context.Abort()
		return
	} else if len(lastWeekArray) != 1 {
		log.Println("Failed to retrieve ONE week for season. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed process results."})
		context.Abort()
		return
	}

	sundayDate, err := utilities.FindNextSunday(debtObject.Date.AddDate(0, 0, -7))
	if err != nil {
		log.Println("Failed to find next Sunday. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to find next Sunday."})
		context.Abort()
		return
	}

	lastWeek := lastWeekArray[0]
	winners := []models.UserWithTickets{}

	// Find weeks winners
	for _, user := range lastWeek.UserWeekResults {

		if user.Competing && user.WeekCompletion >= 1 && !user.Sickleave {
			userWithTickets := models.UserWithTickets{
				User:    user.User,
				Tickets: user.CurrentStreak + 1,
			}
			winners = append(winners, userWithTickets)
			log.Println("Contestant '" + user.User.FirstName + " " + user.User.LastName + "' with '" + strconv.Itoa(user.CurrentStreak+1) + "' tickets added.")
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
		log.Println("Failed start randomizer. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed start randomizer."})
		context.Abort()
		return
	}

	// Pick winner
	winnerID := chooser.Pick()

	// Update winner in DB
	database.UpdateDebtWinner(debtIDInt, winnerID)

	// Give achivement to winner for winning
	err = GiveUserAnAchivement(winnerID, uuid.MustParse("bb964360-6413-47c2-8400-ee87b40365a7"), sundayDate)
	if err != nil {
		log.Println("Failed to give achivement for user '" + winnerID.String() + "'. Ignoring. Error: " + err.Error())
	}

	// Give achivement to loser for losing
	err = GiveUserAnAchivement(userID, uuid.MustParse("d415fffc-ea99-4b27-8929-aeb02ae44da3"), sundayDate)
	if err != nil {
		log.Println("Failed to give achivement for user '" + userID.String() + "'. Ignoring. Error: " + err.Error())
	}

	// Get user object
	winnerUser, err := database.GetUserInformation(winnerID)

	// Create wheel views
	for _, user := range winners {
		wheelview := models.Wheelview{
			User:   user.User,
			DebtID: debtIDInt,
			Viewed: false,
		}
		wheelview.ID = uuid.New()
		err = database.CreateWheelview(wheelview)
		if err != nil {
			log.Println("Create wheelview for user '" + user.User.ID.String() + "'. Error: " + err.Error())
		}

		// Notify winner by e-mail
		winnerObject, err := database.GetAllUserInformation(user.User.ID)
		if err != nil {
			log.Println("Failed to get object for user '" + user.User.ID.String() + "'. Ignoring. Error: " + err.Error())
		} else {

			// Notify winner by e-mail
			err = utilities.SendSMTPForWheelSpinCheck(winnerObject)
			if err != nil {
				log.Println("Failed to notify user '" + user.User.ID.String() + "' by e-mail. Ignoring. Error: " + err.Error())
			}

		}

		// Notify winner by push
		err = PushNotificationsForWheelSpinCheck(user.User.ID)
		if err != nil {
			log.Println("Failed to notify user '" + user.User.ID.String() + "' by push. Ignoring. Error: " + err.Error())
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
		log.Println("Failed to parse session details. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse session details."})
		context.Abort()
		return
	}

	wheelviews, wheelviewsFound, err := database.GetUnviewedWheelviewByUserID(userID)
	if err != nil {
		log.Println("Failed to get unviewed spins. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get unviewed spins."})
		context.Abort()
		return
	} else if wheelviewsFound {
		wheelviewObjects, err := ConvertWheelviewsToWheelviewObjects(wheelviews)
		if err != nil {
			log.Println("Failed to convert wheelviews to wheelview objects. Error: " + err.Error())
			context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to convert wheelviews to wheelview objects."})
			context.Abort()
			return
		}
		debtOverview.UnviewedDebt = wheelviewObjects
	}

	debts, debtsFound, err := database.GetUnreceivedDebtByUserID(userID)
	if err != nil {
		log.Println("Failed to get unreceived debts. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get unviewed spins."})
		context.Abort()
		return
	} else if debtsFound {
		debtObjects, err := ConvertDebtsToDebtObjects(debts)
		if err != nil {
			log.Println("Failed to convert debts to debt objects. Error: " + err.Error())
			context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to convert debts to debt objects."})
			context.Abort()
			return
		}

		// Check if viewed by reciever
		for _, debt := range debtObjects {
			wheelview, wheelviewFound, err := database.GetWheelviewByDebtIDAndUserID(userID, debt.ID)
			if err != nil {
				log.Println("Failed to get wheelview for debt. Error: " + err.Error())
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
		log.Println("Failed to get unreceived debts. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get unpsun spins."})
		context.Abort()
		return
	} else if debtsFound {
		debtObjects, err := ConvertDebtsToDebtObjects(debts)
		if err != nil {
			log.Println("Failed to convert debts to debt objects. Error: " + err.Error())
			context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to convert debts to debt objects."})
			context.Abort()
			return
		}
		debtOverview.UnspunLostDebt = debtObjects
	}

	debts, debtsFound, err = database.GetUnpaidDebtForUser(userID)
	if err != nil {
		log.Println("Failed to get unreceived debts. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get unpaid debt."})
		context.Abort()
		return
	} else if debtsFound {
		debtObjects, err := ConvertDebtsToDebtObjects(debts)
		if err != nil {
			log.Println("Failed to convert debts to debt objects. Error: " + err.Error())
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
		log.Println("Failed to parse debt ID. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse Debt ID."})
		context.Abort()
		return
	}

	// Get user ID
	userID, err := middlewares.GetAuthUsername(context.GetHeader("Authorization"))
	if err != nil {
		log.Println("Failed to parse session details. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse session details."})
		context.Abort()
		return
	}

	err = database.UpdateDebtPaidStatus(debtIDInt, userID)
	if err != nil {
		log.Println("Failed to update payment status. Error: " + err.Error())
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
		log.Println("Failed to parse request. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse request."})
		context.Abort()
		return
	}

	weekResults, err := GenerateDebtForWeek(debtCreationRequest.Date)
	if err != nil {
		log.Println("Failed to generate debt. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate debt. Error: " + err.Error()})
		context.Abort()
		return
	}

	err = GenerateAchivementsForWeek(weekResults)
	if err != nil {
		log.Println("Returned error generating weeks achivements: " + err.Error())
	}

	season, seasonFound, err := GetOngoingSeasonFromDB(debtCreationRequest.Date)
	if err != nil {
		log.Println("Returned error getting last weeks season: " + err.Error())
	} else if seasonFound {
		_, lastWeekWeek := debtCreationRequest.Date.ISOWeek()
		_, seasonEndWeek := season.End.ISOWeek()

		if lastWeekWeek == seasonEndWeek {

			log.Println("Season over, checking for achivements.")

			seasonObject, err := ConvertSeasonToSeasonObject(season)
			if err != nil {
				log.Println("Returned error converting season to season object: " + err.Error())
			} else {

				pastWeeks, err := RetrieveWeekResultsFromSeasonWithinTimeframe(seasonObject.Start, debtCreationRequest.Date, seasonObject)
				if err != nil {
					log.Println("Returned error getting season results: " + err.Error())
				} else {

					err = GenerateAchivementsForSeason(pastWeeks)
					if err != nil {
						log.Println("Returned error generating weeks achivements: " + err.Error())
					}

				}

			}
		}
	}

	context.JSON(http.StatusOK, gin.H{"message": "Debt generated."})

}
