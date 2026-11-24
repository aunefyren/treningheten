package database

import (
	"errors"
	"time"

	"github.com/aunefyren/treningheten/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// receive a user strcut and save it in the database
func RegisterDebtInDB(debt models.Debt) (models.Debt, error) {
	dbRecord := Instance.Create(&debt)

	if dbRecord.Error != nil {
		return debt, dbRecord.Error
	} else if dbRecord.RowsAffected != 1 {
		return debt, errors.New("Failed to update DB.")
	}

	return debt, nil
}

// Retrieve debt for user for a week
func GetDebtForWeekForUser(time time.Time, userID uuid.UUID) (models.Debt, bool, error) {

	var debtStruct models.Debt

	timeWeekday := time.Weekday()
	startDayString := "Error"
	endDayString := "Error"
	finished := false
	timeTwo := time

	// Find monday
	if int(timeWeekday) == 1 {
		startDayString = time.Format("2006-01-02") + " 00:00:00"
	} else {
		finished = false
		timeTwo = time
		for finished == false {
			timeTwo = timeTwo.AddDate(0, 0, -1)
			timeTwoWeekday := timeTwo.Weekday()
			if int(timeTwoWeekday) == 1 {
				finished = true
				startDayString = timeTwo.Format("2006-01-02") + " 00:00:00"
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

	debtRecord := Instance.Where("debts.enabled = ?", true).Where("debts.loser_id = ?", userID).Where("debts.Date >= ?", startDayString).Where("debts.Date <= ?", endDayString).Find(&debtStruct)
	if debtRecord.Error != nil {
		return models.Debt{}, false, debtRecord.Error
	} else if debtRecord.RowsAffected != 1 {
		return models.Debt{}, false, nil
	}

	return debtStruct, true, nil

}

// GetDebtsForUserIDsBetweenDates returns enabled debts where the loser is one of the
// given users and the debt date falls between the two dates (inclusive), so callers can
// bucket debts by (user, week) without a query per week. Returns an empty slice when no
// user IDs are supplied.
func GetDebtsForUserIDsBetweenDates(userIDs []uuid.UUID, startDate time.Time, endDate time.Time) ([]models.Debt, error) {
	var debts []models.Debt

	if len(userIDs) == 0 {
		return []models.Debt{}, nil
	}

	startDayString := startDate.Format("2006-01-02") + " 00:00:00"
	endDayString := endDate.Format("2006-01-02") + " 23:59:59"

	debtRecord := Instance.
		Where("debts.enabled = ?", true).
		Where("debts.loser_id IN ?", userIDs).
		Where("debts.Date >= ?", startDayString).
		Where("debts.Date <= ?", endDayString).
		Find(&debts)

	if debtRecord.Error != nil {
		return []models.Debt{}, debtRecord.Error
	}

	return debts, nil
}

// Retrieve debt for user for a week in a season
func GetDebtForWeekForUserInSeasonID(time time.Time, userID uuid.UUID, seasonID uuid.UUID) (models.Debt, bool, error) {

	var debtStruct models.Debt

	timeWeekday := time.Weekday()
	startDayString := "Error"
	endDayString := "Error"
	finished := false
	timeTwo := time

	// Find monday
	if int(timeWeekday) == 1 {
		startDayString = time.Format("2006-01-02") + " 00:00:00"
	} else {
		finished = false
		timeTwo = time
		for finished == false {
			timeTwo = timeTwo.AddDate(0, 0, -1)
			timeTwoWeekday := timeTwo.Weekday()
			if int(timeTwoWeekday) == 1 {
				finished = true
				startDayString = timeTwo.Format("2006-01-02") + " 00:00:00"
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

	// Oldest first and only one: a week with duplicate debts (which the week fix collapses)
	// used to count as "no debt" because more than one row matched.
	debtRecord := Instance.Where("debts.enabled = ?", true).Where("debts.loser_id = ?", userID).Where("debts.season_id = ?", seasonID).Where("debts.Date >= ?", startDayString).Where("debts.Date <= ?", endDayString).Order("debts.created_at asc").Limit(1).Find(&debtStruct)
	if debtRecord.Error != nil {
		return models.Debt{}, false, debtRecord.Error
	} else if debtRecord.RowsAffected != 1 {
		return models.Debt{}, false, nil
	}

	return debtStruct, true, nil

}

// Get all the debt for a user using user ID where a winner isn't chosen
func GetUnchosenDebtForUserByUserID(userID uuid.UUID) ([]models.Debt, bool, error) {

	var debtStruct []models.Debt

	debtRecord := Instance.Where("debts.enabled = ?", true).Where("debts.loser_id = ?", userID).Where("debts.winner_id IS NULL").Find(&debtStruct)
	if debtRecord.Error != nil {
		return []models.Debt{}, false, debtRecord.Error
	} else if debtRecord.RowsAffected == 0 {
		return []models.Debt{}, false, nil
	}

	return debtStruct, true, nil

}

// Get debt by debt ID
func GetDebtByDebtID(debtID uuid.UUID) (models.Debt, bool, error) {

	var debtStruct models.Debt

	debtRecord := Instance.Where("debts.enabled = ?", true).Where("debts.ID = ?", debtID).Find(&debtStruct)
	if debtRecord.Error != nil {
		return models.Debt{}, false, debtRecord.Error
	} else if debtRecord.RowsAffected != 1 {
		return models.Debt{}, false, nil
	}

	return debtStruct, true, nil

}

// UpdateDebtWinner records the wheel's result, only while no winner is stored: the
// winner_id IS NULL condition makes the check-and-set atomic, so two concurrent spins of
// the same debt cannot both land — the second affects no row and errors.
func UpdateDebtWinner(debtID uuid.UUID, winnerID uuid.UUID) error {
	debtRecords := Instance.Model(&models.Debt{}).
		Where("debts.enabled = ?", true).
		Where("debts.ID = ?", debtID).
		Where("debts.winner_id IS NULL").
		Update("winner_id", winnerID)
	if debtRecords.Error != nil {
		return debtRecords.Error
	}
	if debtRecords.RowsAffected != 1 {
		return errors.New("Debt not changed in database.")
	}

	return nil
}

// Get all the debt for a user where the prize is not received
func GetUnreceivedDebtByUserID(userID uuid.UUID) ([]models.Debt, bool, error) {

	var debtStruct []models.Debt

	debtRecord := Instance.Where("debts.enabled = ?", true).Where("debts.winner_id = ?", userID).Where("debts.paid = ?", false).Find(&debtStruct)
	if debtRecord.Error != nil {
		return []models.Debt{}, false, debtRecord.Error
	} else if debtRecord.RowsAffected == 0 {
		return []models.Debt{}, false, nil
	}

	return debtStruct, true, nil

}

// Get all the debt for a user where the prize is not received and not unviewed
func GetUnpaidDebtForUser(userID uuid.UUID) ([]models.Debt, bool, error) {

	var debtStruct []models.Debt

	debtRecord := Instance.Where("debts.enabled = ?", true).Where("debts.loser_id = ?", userID).Where("debts.paid = ?", false).Find(&debtStruct)
	if debtRecord.Error != nil {
		return []models.Debt{}, false, debtRecord.Error
	} else if debtRecord.RowsAffected == 0 {
		return []models.Debt{}, false, nil
	}

	return debtStruct, true, nil

}

// Update debt paid status
func UpdateDebtPaidStatus(debtID uuid.UUID, userID uuid.UUID) error {

	var debt models.Debt

	debtRecords := Instance.Model(debt).Where("debts.enabled = ?", true).Where("debts.ID = ?", debtID).Where("debts.winner_id = ?", userID).Update("paid", true)
	if debtRecords.Error != nil {
		return debtRecords.Error
	}
	if debtRecords.RowsAffected != 1 {
		return errors.New("Paid status not changed in database.")
	}

	return nil
}

// Get all the debt for a season where the prize is received for a user ID
func GetDebtInSeasonWonByUserID(seasonID uuid.UUID, userID uuid.UUID) ([]models.Debt, bool, error) {
	var debtStruct = []models.Debt{}

	debtRecord := Instance.Where("debts.enabled = ?", true).Where("debts.season_id = ?", seasonID).Where("debts.winner_id = ?", userID).Find(&debtStruct)
	if debtRecord.Error != nil {
		return []models.Debt{}, false, debtRecord.Error
	} else if debtRecord.RowsAffected == 0 {
		return []models.Debt{}, false, nil
	}

	return debtStruct, true, nil

}

// GetDebtsForSeasonBetweenDates returns every enabled debt of the season dated between the two
// days (inclusive, whole days), oldest first. The week fix reconciles a season-week from it, so
// a week with duplicate debts for one loser comes back with all of them.
func GetDebtsForSeasonBetweenDates(seasonID uuid.UUID, startDate time.Time, endDate time.Time) ([]models.Debt, error) {
	debts := []models.Debt{}

	startDayString := startDate.Format("2006-01-02") + " 00:00:00"
	endDayString := endDate.Format("2006-01-02") + " 23:59:59"

	debtRecord := Instance.
		Where("debts.enabled = ?", true).
		Where("debts.season_id = ?", seasonID).
		Where("debts.Date >= ?", startDayString).
		Where("debts.Date <= ?", endDayString).
		Order("debts.created_at asc").
		Find(&debts)
	if debtRecord.Error != nil {
		return []models.Debt{}, debtRecord.Error
	}

	return debts, nil
}

// ErrDebtChanged means a conditional debt update found the debt no longer as the caller read
// it (spun, paid, disabled or re-assigned in the meantime) and changed nothing.
var ErrDebtChanged = errors.New("debt changed since it was read")

// debtStillAsSeen narrows an update to a debt that is still enabled, unpaid and holds the
// winner the caller read. A wheel spin (winner set from NULL), a payment or another fix that
// lands in between makes the update affect no row instead of overwriting it.
func debtStillAsSeen(debtID uuid.UUID, seenWinnerID *uuid.UUID) *gorm.DB {
	query := Instance.Model(&models.Debt{}).
		Where("debts.id = ?", debtID).
		Where("debts.enabled = ?", true).
		Where("debts.paid = ?", false)
	if seenWinnerID == nil {
		return query.Where("debts.winner_id IS NULL")
	}
	return query.Where("debts.winner_id = ?", *seenWinnerID)
}

// SetDebtWinnerIfUnchanged replaces an unpaid debt's winner (nil hands it back to the wheel),
// but only while the debt still holds seenWinnerID. Returns ErrDebtChanged otherwise.
func SetDebtWinnerIfUnchanged(debtID uuid.UUID, seenWinnerID *uuid.UUID, newWinnerID *uuid.UUID) error {
	var value any
	if newWinnerID != nil {
		value = *newWinnerID
	}
	record := debtStillAsSeen(debtID, seenWinnerID).Update("winner_id", value)
	if record.Error != nil {
		return record.Error
	}
	if record.RowsAffected != 1 {
		return ErrDebtChanged
	}
	return nil
}

// DisableDebtIfUnchanged switches an unpaid debt off, but only while it still holds
// seenWinnerID. Returns ErrDebtChanged otherwise.
func DisableDebtIfUnchanged(debtID uuid.UUID, seenWinnerID *uuid.UUID) error {
	record := debtStillAsSeen(debtID, seenWinnerID).Update("enabled", false)
	if record.Error != nil {
		return record.Error
	}
	if record.RowsAffected != 1 {
		return ErrDebtChanged
	}
	return nil
}

// Get all the debt for a season where the prize is lost by a user ID
func GetDebtInSeasonLostByUserID(seasonID uuid.UUID, userID uuid.UUID) ([]models.Debt, bool, error) {
	var debtStruct = []models.Debt{}

	debtRecord := Instance.Where("debts.enabled = ?", true).Where("debts.season_id = ?", seasonID).Where("debts.loser_id = ?", userID).Find(&debtStruct)
	if debtRecord.Error != nil {
		return []models.Debt{}, false, debtRecord.Error
	} else if debtRecord.RowsAffected == 0 {
		return []models.Debt{}, false, nil
	}

	return debtStruct, true, nil

}
