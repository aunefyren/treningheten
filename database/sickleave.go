package database

import (
	"aunefyren/treningheten/models"
	"errors"
	"time"

	"github.com/google/uuid"
)

// Retrieves a sickleave for a chosen week for a chosen goal
func GetUsedSickleaveForGoalWithinWeek(time time.Time, goalID uuid.UUID) (sickLeave *models.Sickleave, err error) {
	sickLeave = nil
	err = nil

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

	sickLeaveRecord := Instance.Where("`sickleaves`.enabled = ?", 1).Where("`sickleaves`.goal_id = ?", goalID).Where("`sickleaves`.date >= ?", startDayString).Where("`sickleaves`.date <= ?", endDayString).Find(&sickLeave)

	if sickLeaveRecord.Error != nil {
		return nil, sickLeaveRecord.Error
	} else if sickLeaveRecord.RowsAffected != 1 {
		return nil, err
	}

	return sickLeave, err
}

// Retrieve a unused sickleave for chosen goal
func GetUnusedSickleaveForGoalWithinWeek(goalID uuid.UUID) ([]models.Sickleave, bool, error) {

	var sickleavestruct []models.Sickleave

	sickleaverecord := Instance.Where("`sickleaves`.enabled = ?", 1).Where("`sickleaves`.goal_id = ?", goalID).Where("`sickleaves`.used = ?", 0).Find(&sickleavestruct)
	if sickleaverecord.Error != nil {
		return []models.Sickleave{}, false, sickleaverecord.Error
	} else if sickleaverecord.RowsAffected == 0 {
		return []models.Sickleave{}, false, nil
	}

	return sickleavestruct, true, nil

}

// Update sickleave in database to used and date to now by sickleave ID
func SetSickleaveToUsedByID(sickleaveID uuid.UUID) error {

	var sickleavestruct models.Sickleave

	now := time.Now()
	Date := now.Format("2006-01-02") + " 00:00:00.000"

	sickleaverecord := Instance.Model(sickleavestruct).Where("`sickleaves`.enabled = ?", 1).Where("`sickleaves`.ID = ?", sickleaveID).Update("used", 1)
	if sickleaverecord.Error != nil {
		return sickleaverecord.Error
	} else if sickleaverecord.RowsAffected != 1 {
		return errors.New("No sickleave updated in the database.")
	}

	sickleaverecordtwo := Instance.Model(sickleavestruct).Where("`sickleaves`.enabled = ?", 1).Where("`sickleaves`.ID = ?", sickleaveID).Update("date", Date)
	if sickleaverecordtwo.Error != nil {
		return sickleaverecordtwo.Error
	} else if sickleaverecordtwo.RowsAffected != 1 {
		return errors.New("No sickleave updated in the database.")
	}

	return nil

}

// Register unused sickleave for goal
func CreateSickleave(sickleave models.Sickleave) error {
	record := Instance.Create(&sickleave)
	if record.Error != nil {
		return record.Error
	}
	return nil
}
