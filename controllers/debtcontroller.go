package controllers

import (
	"aunefyren/treningheten/database"
	"aunefyren/treningheten/middlewares"
	"aunefyren/treningheten/models"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mroth/weightedrand/v2"
)

func GenerateLastWeeksDebt() {

	now := time.Now()

	// Get current season
	season, err := GetOngoingSeasonFromDB(now)
	if err != nil {
		log.Println("Failed to verify current season status. Returning. Error: " + err.Error())
		return
	}

	// Stop if not within season
	if season.Start.After(now) || season.End.Before(now) {
		log.Println("Not in the middle of a season. Returning.")
		return
	}

	seasonObject, err := ConvertSeasonToSeasonObject(season)
	if err != nil {
		log.Println("Failed to convert season to season object. Returning. Error: " + err.Error())
		return
	}

	lastWeekArray, err := RetrieveWeekResultsFromSeasonWithinTimeframe(now.AddDate(0, 0, -14), now.AddDate(0, 0, -7), seasonObject)
	if err != nil {
		log.Println("Failed to retrieve last week for season. Returning. Error: " + err.Error())
		return
	} else if len(lastWeekArray) != 1 {
		log.Println("Failed to retrieve ONE week for season. Returning. Error: " + err.Error())
		return
	}

	lastWeek := lastWeekArray[0]

	winners := []models.User{}
	losers := []models.User{}

	for _, user := range lastWeek.UserWeekResults {

		if user.Competing && user.WeekCompletion < 1 {
			losers = append(losers, user.User)
		} else if user.Competing && user.WeekCompletion >= 1 && !user.Sickleave {
			winners = append(winners, user.User)
		}

	}

	winner := 0

	if len(losers) == 0 {
		log.Println("No losers this week. Returning.")
		return
	}

	if len(winners) == 0 {
		log.Println("No winners this week. Returning.")
		return
	} else if len(winners) == 1 {
		winner = int(winners[0].ID)
	}

	for _, user := range losers {

		_, debtFound, err := database.GetDebtForWeekForUserInSeasonID(now.AddDate(0, 0, -7), int(user.ID), int(seasonObject.ID))
		if err != nil {
			log.Println("Failed check for debt for '" + strconv.Itoa(int(user.ID)) + "'. Skipping.")
			continue
		} else if debtFound {
			log.Println("Debt found for '" + strconv.Itoa(int(user.ID)) + "'. Skipping.")
			continue
		}

		debt := models.Debt{}
		debt.Date = now.AddDate(0, 0, -7).Truncate(24 * time.Hour)
		debt.Loser = int(user.ID)
		debt.Winner = winner
		debt.Season = int(season.ID)

		log.Println("Creating debt for '" + strconv.Itoa(int(user.ID)) + "'.")

		err = database.RegisterDebtInDB(debt)
		if err != nil {
			log.Println("Failed to log debt for '" + strconv.Itoa(int(user.ID)) + "'. Skipping.")
			continue
		}
	}

	log.Println("Done logging debt. Returning.")

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

	if debt.Winner != 0 {
		user, err := database.GetUserInformation(debt.Winner)
		if err != nil {
			log.Println("Failed to get user information for user '" + strconv.Itoa(debt.Winner) + "'. Returning. Error: " + err.Error())
			return models.DebtObject{}, err
		}

		debtObject.Winner = user
	} else {
		debtObject.Winner = models.User{}
	}

	if debt.Loser != 0 {
		user, err := database.GetUserInformation(debt.Loser)
		if err != nil {
			log.Println("Failed to get user information for user '" + strconv.Itoa(debt.Loser) + "'. Returning. Error: " + err.Error())
			return models.DebtObject{}, err
		}

		debtObject.Loser = user
	} else {
		debtObject.Loser = models.User{}
	}

	season, err := database.GetSeasonByID(int(debt.Season))
	if err != nil {
		log.Println("Failed to get season '" + strconv.Itoa(debt.Season) + "' in database. Returning. Error: " + err.Error())
		return models.DebtObject{}, err
	}
	seasonObject, err := ConvertSeasonToSeasonObject(season)
	if err != nil {
		log.Println("Failed to convert season '" + strconv.Itoa(debt.Season) + "' to season object. Returning. Error: " + err.Error())
		return models.DebtObject{}, err
	}
	debtObject.Season = seasonObject

	debtObject.CreatedAt = debt.CreatedAt
	debtObject.Date = debt.Date
	debtObject.DeletedAt = debt.DeletedAt
	debtObject.Enabled = debt.Enabled
	debtObject.ID = debt.ID
	debtObject.Model = debt.Model
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
	debtIDInt, err := strconv.Atoi(debtID)
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

	lastWeekArray, err := RetrieveWeekResultsFromSeasonWithinTimeframe(debtObject.Date.AddDate(0, 0, -7), debtObject.Date, debtObject.Season)
	if err != nil {
		log.Println("Failed to retrieve last week for season. Returning. Error: " + err.Error())
		return
	} else if len(lastWeekArray) != 1 {
		log.Println("Failed to retrieve ONE week for season. Returning. Error: " + err.Error())
		return
	}

	lastWeek := lastWeekArray[0]

	winners := []models.UserWithTickets{}

	for _, user := range lastWeek.UserWeekResults {

		if user.Competing && user.WeekCompletion >= 1 && !user.Sickleave {
			userWithTickets := models.UserWithTickets{
				User:    user.User,
				Tickets: user.CurrentStreak + 1,
			}
			winners = append(winners, userWithTickets)
		}

	}

	// Check for wheelviews and mark them as viewed if matching
	if debtObject.Winner.ID != 0 {

		log.Println("Checking for debt views for user '" + strconv.Itoa(userID) + "'.")

		wheelview, wheelviewFound, err := database.GetUnviewedWheelviewByDebtIDAndUserID(userID, int(debtObject.ID))
		if err != nil {
			log.Println("Failed to retrieve wheelview for user '" + strconv.Itoa(userID) + "'. Continuing. Error: " + err.Error())
		} else if wheelviewFound {
			err = database.SetWheelviewToViewedByID(int(wheelview.ID))
			if err != nil {
				log.Println("Failed to update wheelview for user '" + strconv.Itoa(userID) + "'. Continuing. Error: " + err.Error())
			}
			log.Println("Debt marked as viewed for user '" + strconv.Itoa(userID) + "'.")
		}
	}

	context.JSON(http.StatusOK, gin.H{"message": "Debt found.", "debt": debtObject, "winners": winners})

}

func APIChooseWinnerForDebt(context *gin.Context) {

	// Create user request
	var debtID = context.Param("debt_id")

	// Parse group id
	debtIDInt, err := strconv.Atoi(debtID)
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
	} else if debt.Loser != userID {
		context.JSON(http.StatusUnauthorized, gin.H{"error": "No access."})
		context.Abort()
		return
	}

	if debt.Winner != 0 {
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
	lastWeekArray, err := RetrieveWeekResultsFromSeasonWithinTimeframe(debtObject.Date.AddDate(0, 0, -14), debtObject.Date.AddDate(0, 0, -7), debtObject.Season)
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
	choices := []weightedrand.Choice[int, int]{}
	for _, user := range winners {
		choice := weightedrand.Choice[int, int]{
			Item:   int(user.User.ID),
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

	// Get user object
	winnerUser, err := database.GetUserInformation(winnerID)

	// Create wheel views
	for _, user := range winners {
		wheelview := models.Wheelview{
			User:   int(user.User.ID),
			Debt:   debtIDInt,
			Viewed: false,
		}
		err = database.CreateWheelview(wheelview)
		if err != nil {
			log.Println("Create wheelview for user '" + strconv.Itoa(int(user.User.ID)) + "'. Error: " + err.Error())
		}
	}

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
			wheelview, wheelviewFound, err := database.GetWheelviewByDebtIDAndUserID(userID, int(debt.ID))
			log.Println(debt.ID)
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
	debtIDInt, err := strconv.Atoi(debtID)
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
