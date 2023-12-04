package database

import (
	"aunefyren/treningheten/models"
	"errors"
	"time"

	"github.com/google/uuid"
)

// Create new exercise day within a season
func CreateExerciseDayForGoalInDatabase(exercise models.ExerciseDay) (uuid.UUID, error) {
	record := Instance.Create(&exercise)
	if record.Error != nil {
		return uuid.UUID{}, record.Error
	}
	return exercise.ID, nil
}

// Get exercise from goal on certain date
func GetExerciseDayByGoalAndDate(goalID uuid.UUID, date time.Time) (*models.ExerciseDay, error) {

	var exercise models.ExerciseDay

	startDayString := date.Format("2006-01-02") + " 00:00:00.000"
	endDayString := date.Format("2006-01-02") + " 23:59:59"

	goalrecord := Instance.Where("`exercise_days`.enabled = ?", 1).Where("`exercise_days`.goal_id = ?", goalID).Where("`exercise_days`.Date >= ?", startDayString).Where("`exercise_days`.Date <= ?", endDayString).Find(&exercise)
	if goalrecord.Error != nil {
		return nil, goalrecord.Error
	} else if goalrecord.RowsAffected == 0 {
		return nil, nil
	}

	return &exercise, nil
}

// Create new exercise within goal
func CreateExerciseDayInDB(exercise models.ExerciseDay) error {
	record := Instance.Create(&exercise)
	if record.Error != nil {
		return record.Error
	}
	return nil
}

// Update an exercise in the database
func UpdateExerciseDayInDatabase(exercise models.ExerciseDay) (err error) {

	err = nil

	startDayString := exercise.Date.Format("2006-01-02") + " 00:00:00.000"
	endDayString := exercise.Date.Format("2006-01-02") + " 23:59:59"

	err = UpdateExerciseDayNoteInDatabase(exercise.GoalID, startDayString, endDayString, exercise.Note)
	if err != nil {
		return err
	}

	err = UpdateExerciseDayIntervalInDatabase(exercise.GoalID, startDayString, endDayString, exercise.ExerciseInterval)
	if err != nil {
		return err
	}

	return err

}

func UpdateExerciseDayNoteInDatabase(goalID uuid.UUID, startDayString string, endDayString string, note string) (err error) {

	err = nil
	var exercisestruct models.ExerciseDay

	exerciserecord := Instance.Model(exercisestruct).Where("`exercise_days`.enabled = ?", 1).Where("`exercise_days`.goal_id = ?", goalID).Where("`exercise_days`.date >= ?", startDayString).Where("`exercise_days`.date <= ?", endDayString).Update("note", note)
	if exerciserecord.Error != nil {
		return exerciserecord.Error
	} else if exerciserecord.RowsAffected != 1 {
		return errors.New("Exercise note not updated in the database.")
	}

	return err
}

func UpdateExerciseDayIntervalInDatabase(goalID uuid.UUID, startDayString string, endDayString string, exerciseInterval int) (err error) {

	err = nil
	var exercisestruct models.ExerciseDay

	exerciserecordtwo := Instance.Model(exercisestruct).Where("`exercise_days`.enabled = ?", 1).Where("`exercise_days`.goal_id = ?", goalID).Where("`exercise_days`.date >= ?", startDayString).Where("`exercise_days`.date <= ?", endDayString).Update("exercise_interval", exerciseInterval)
	if exerciserecordtwo.Error != nil {
		return exerciserecordtwo.Error
	} else if exerciserecordtwo.RowsAffected != 1 {
		return errors.New("Exercise interval not updated in the database.")
	}

	return err

}

func GetExerciseDaysBetweenDatesUsingDates(goalID uuid.UUID, startDate time.Time, endDate time.Time) ([]models.ExerciseDay, error) {

	var exercises []models.ExerciseDay

	startDayString := startDate.Format("2006-01-02") + " 00:00:00.000"
	endDayString := endDate.Format("2006-01-02") + " 23:59:59"

	exerciserecord := Instance.Where("`exercise_days`.enabled = ?", 1).Where("`exercise_days`.goal_id = ?", goalID).Where("`exercise_days`.Date >= ?", startDayString).Where("`exercise_days`.Date <= ?", endDayString).Find(&exercises)
	if exerciserecord.Error != nil {
		return []models.ExerciseDay{}, exerciserecord.Error
	} else if exerciserecord.RowsAffected == 0 {
		return []models.ExerciseDay{}, nil
	}

	return exercises, nil

}

func GetExerciseDaysForUserUsingUserID(userID uuid.UUID) ([]models.ExerciseDay, error) {

	var exercises []models.ExerciseDay

	exerciserecord := Instance.Order("date desc").Where("`exercise_days`.enabled = ?", 1).Joins("JOIN goals on `exercise_days`.goal_id = `goals`.ID").Where("`goals`.user_id = ?", userID).Where("`goals`.enabled = ?", 1).Find(&exercises)
	if exerciserecord.Error != nil {
		return []models.ExerciseDay{}, exerciserecord.Error
	} else if exerciserecord.RowsAffected == 0 {
		return []models.ExerciseDay{}, nil
	}

	return exercises, nil

}

func GetExerciseDaysForUserUsingUserIDAndGoalID(userID uuid.UUID, goalID uuid.UUID) ([]models.ExerciseDay, error) {

	var exercises []models.ExerciseDay

	exerciserecord := Instance.Order("date desc").Where("`exercise_days`.enabled = ?", 1).Where("`exercise_days`.goal_id = ?", goalID).Joins("JOIN goals on `exercise_days`.goal_id = `goals`.ID").Where("`goals`.user_id = ?", userID).Where("`goals`.enabled = ?", 1).Find(&exercises)
	if exerciserecord.Error != nil {
		return []models.ExerciseDay{}, exerciserecord.Error
	} else if exerciserecord.RowsAffected == 0 {
		return []models.ExerciseDay{}, nil
	}

	return exercises, nil

}

func GetAllEnabledExerciseDays() ([]models.ExerciseDay, error) {

	var exercises []models.ExerciseDay

	exerciserecord := Instance.Order("date desc").Where("`exercise_days`.enabled = ?", 1).Find(&exercises)
	if exerciserecord.Error != nil {
		return []models.ExerciseDay{}, exerciserecord.Error
	}

	return exercises, nil

}
