package database

import (
	"aunefyren/treningheten/models"
	"errors"
	"time"
)

// receive a user strcut and save it in the database
func RegisterDebtInDB(debt models.Debt) error {
	dbRecord := Instance.Create(&debt)

	if dbRecord.Error != nil {
		return dbRecord.Error
	} else if dbRecord.RowsAffected != 1 {
		return errors.New("Failed to update DB.")
	}

	return nil
}

// Retrieve debt for user for a week
func GetDebtForWeekForUser(time time.Time, userID int) (models.Debt, bool, error) {

	var debtStruct models.Debt

	timeWeekday := time.Weekday()
	startDayString := "Error"
	endDayString := "Error"
	finished := false
	timeTwo := time

	// Find monday
	if int(timeWeekday) == 1 {
		startDayString = time.Format("2006-01-02") + " 00:00:00.000"
	} else {
		finished = false
		timeTwo = time
		for finished == false {
			timeTwo = timeTwo.AddDate(0, 0, -1)
			timeTwoWeekday := timeTwo.Weekday()
			if int(timeTwoWeekday) == 1 {
				finished = true
				startDayString = timeTwo.Format("2006-01-02") + " 00:00:00.000"
			}
		}
	}

	// Find sunday
	if int(timeWeekday) == 0 {
		endDayString = time.Format("2006-01-02") + " 23:59:59"
	} else {
		finished = false
		timeTwo = time
		for finished == false {
			timeTwo = timeTwo.AddDate(0, 0, 1)
			timeTwoWeekday := timeTwo.Weekday()
			if int(timeTwoWeekday) == 0 {
				finished = true
				endDayString = timeTwo.Format("2006-01-02") + " 23:59:59"
			}
		}
	}

	debtRecord := Instance.Where("`debts`.enabled = ?", 1).Where("`debts`.Loser = ?", userID).Where("`debts`.Date >= ?", startDayString).Where("`debts`.Date <= ?", endDayString).Find(&debtStruct)
	if debtRecord.Error != nil {
		return models.Debt{}, false, debtRecord.Error
	} else if debtRecord.RowsAffected != 1 {
		return models.Debt{}, false, nil
	}

	return debtStruct, true, nil

}

// Retrieve debt for user for a week in a season
func GetDebtForWeekForUserInSeasonID(time time.Time, userID int, seasonID int) (models.Debt, bool, error) {

	var debtStruct models.Debt

	timeWeekday := time.Weekday()
	startDayString := "Error"
	endDayString := "Error"
	finished := false
	timeTwo := time

	// Find monday
	if int(timeWeekday) == 1 {
		startDayString = time.Format("2006-01-02") + " 00:00:00.000"
	} else {
		finished = false
		timeTwo = time
		for finished == false {
			timeTwo = timeTwo.AddDate(0, 0, -1)
			timeTwoWeekday := timeTwo.Weekday()
			if int(timeTwoWeekday) == 1 {
				finished = true
				startDayString = timeTwo.Format("2006-01-02") + " 00:00:00.000"
			}
		}
	}

	// Find sunday
	if int(timeWeekday) == 0 {
		endDayString = time.Format("2006-01-02") + " 23:59:59"
	} else {
		finished = false
		timeTwo = time
		for finished == false {
			timeTwo = timeTwo.AddDate(0, 0, 1)
			timeTwoWeekday := timeTwo.Weekday()
			if int(timeTwoWeekday) == 0 {
				finished = true
				endDayString = timeTwo.Format("2006-01-02") + " 23:59:59"
			}
		}
	}

	debtRecord := Instance.Where("`debts`.enabled = ?", 1).Where("`debts`.Loser = ?", userID).Where("`debts`.season = ?", seasonID).Where("`debts`.Date >= ?", startDayString).Where("`debts`.Date <= ?", endDayString).Find(&debtStruct)
	if debtRecord.Error != nil {
		return models.Debt{}, false, debtRecord.Error
	} else if debtRecord.RowsAffected != 1 {
		return models.Debt{}, false, nil
	}

	return debtStruct, true, nil

}

// Get all the debt for a user using user ID where a winner isn't chosen
func GetUnchosenDebtForUserByUserID(userID int) ([]models.Debt, bool, error) {

	var debtStruct []models.Debt

	debtRecord := Instance.Where("`debts`.enabled = ?", 1).Where("`debts`.loser = ?", userID).Where("`debts`.winner IS NULL").Find(&debtStruct)
	if debtRecord.Error != nil {
		return []models.Debt{}, false, debtRecord.Error
	} else if debtRecord.RowsAffected == 0 {
		return []models.Debt{}, false, nil
	}

	return debtStruct, true, nil

}

// Get debt by debt ID
func GetDebtByDebtID(debtID int) (models.Debt, bool, error) {

	var debtStruct models.Debt

	debtRecord := Instance.Where("`debts`.enabled = ?", 1).Where("`debts`.ID = ?", debtID).Find(&debtStruct)
	if debtRecord.Error != nil {
		return models.Debt{}, false, debtRecord.Error
	} else if debtRecord.RowsAffected != 1 {
		return models.Debt{}, false, nil
	}

	return debtStruct, true, nil

}

// Update debt winner
func UpdateDebtWinner(debtID int, winnerID int) error {

	var debt models.Debt

	debtRecords := Instance.Model(debt).Where("`debts`.enabled = ?", 1).Where("`debts`.ID = ?", debtID).Update("winner", winnerID)
	if debtRecords.Error != nil {
		return debtRecords.Error
	}
	if debtRecords.RowsAffected != 1 {
		return errors.New("Debt not changed in database.")
	}

	return nil
}

// Get all the debt for a user where the prize is not received
func GetUnreceivedDebtByUserID(userID int) ([]models.Debt, bool, error) {

	var debtStruct []models.Debt

	debtRecord := Instance.Where("`debts`.enabled = ?", 1).Where("`debts`.Winner = ?", userID).Where("`debts`.Paid = ?", 0).Find(&debtStruct)
	if debtRecord.Error != nil {
		return []models.Debt{}, false, debtRecord.Error
	} else if debtRecord.RowsAffected == 0 {
		return []models.Debt{}, false, nil
	}

	return debtStruct, true, nil

}

// Get all the debt for a user where the prize is not received and not unviewed
func GetUnpaidDebtForUser(userID int) ([]models.Debt, bool, error) {

	var debtStruct []models.Debt

	debtRecord := Instance.Where("`debts`.enabled = ?", 1).Where("`debts`.Loser = ?", userID).Where("`debts`.Paid = ?", 0).Find(&debtStruct)
	if debtRecord.Error != nil {
		return []models.Debt{}, false, debtRecord.Error
	} else if debtRecord.RowsAffected == 0 {
		return []models.Debt{}, false, nil
	}

	return debtStruct, true, nil

}

// Update debt paid status
func UpdateDebtPaidStatus(debtID int, userID int) error {

	var debt models.Debt

	debtRecords := Instance.Model(debt).Where("`debts`.enabled = ?", 1).Where("`debts`.ID = ?", debtID).Where("`debts`.winner = ?", userID).Update("paid", 1)
	if debtRecords.Error != nil {
		return debtRecords.Error
	}
	if debtRecords.RowsAffected != 1 {
		return errors.New("Paid status not changed in database.")
	}

	return nil
}

// Get all the debt for a season where the prize is received for a user ID
func GetDebtInSeasonWonByUserID(seasonID int, userID int) ([]models.Debt, bool, error) {

	var debtStruct []models.Debt

	debtRecord := Instance.Where("`debts`.enabled = ?", 1).Where("`debts`.season = ?", seasonID).Where("`debts`.winner = ?", userID).Find(&debtStruct)
	if debtRecord.Error != nil {
		return []models.Debt{}, false, debtRecord.Error
	} else if debtRecord.RowsAffected == 0 {
		return []models.Debt{}, false, nil
	}

	return debtStruct, true, nil

}

// Get all the debt for a season where the prize is lost by a user ID
func GetDebtInSeasonLostByUserID(seasonID int, userID int) ([]models.Debt, bool, error) {

	var debtStruct []models.Debt

	debtRecord := Instance.Where("`debts`.enabled = ?", 1).Where("`debts`.season = ?", seasonID).Where("`debts`.loser = ?", userID).Find(&debtStruct)
	if debtRecord.Error != nil {
		return []models.Debt{}, false, debtRecord.Error
	} else if debtRecord.RowsAffected == 0 {
		return []models.Debt{}, false, nil
	}

	return debtStruct, true, nil

}
