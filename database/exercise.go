package database

import (
	"aunefyren/treningheten/models"
	"errors"
	"time"
)

// Create new goal within a season
func CreateExerciseForGoalInDatabase(exercise models.Exercise) error {
	record := Instance.Create(&exercise)
	if record.Error != nil {
		return record.Error
	}
	return nil
}

// Get exercise from goal on certain date
func GetExerciseByGoalAndDate(goalID int, date time.Time) (*models.Exercise, error) {

	var exercise models.Exercise

	startDayString := date.Format("2006-01-02") + " 00:00:00.000"
	endDayString := date.Format("2006-01-02") + " 23:59:59"

	goalrecord := Instance.Where("`exercises`.enabled = ?", 1).Where("`exercises`.Goal = ?", goalID).Where("`exercises`.Date >= ?", startDayString).Where("`exercises`.Date <= ?", endDayString).Find(&exercise)
	if goalrecord.Error != nil {
		return nil, goalrecord.Error
	} else if goalrecord.RowsAffected == 0 {
		return nil, nil
	}

	return &exercise, nil
}

// Create new exercise within goal
func CreateExerciseInDB(exercise models.Exercise) error {
	record := Instance.Create(&exercise)
	if record.Error != nil {
		return record.Error
	}
	return nil
}

// Update an exercise in the database
func UpdateExerciseInDatabase(exercise models.Exercise) error {

	var exercisestruct models.Exercise

	startDayString := exercise.Date.Format("2006-01-02") + " 00:00:00.000"
	endDayString := exercise.Date.Format("2006-01-02") + " 23:59:59"

	exerciserecord := Instance.Model(exercisestruct).Where("`exercises`.enabled = ?", 1).Where("`exercises`.Goal = ?", exercise.Goal).Where("`exercises`.Date >= ?", startDayString).Where("`exercises`.Date <= ?", endDayString).Update("note", exercise.Note)
	if exerciserecord.Error != nil {
		return exerciserecord.Error
	} else if exerciserecord.RowsAffected != 1 {
		return errors.New("No exercise updated in the database.")
	}

	exerciserecordtwo := Instance.Model(exercisestruct).Where("`exercises`.enabled = ?", 1).Where("`exercises`.Goal = ?", exercise.Goal).Where("`exercises`.Date >= ?", startDayString).Where("`exercises`.Date <= ?", endDayString).Update("exercise_interval", exercise.ExerciseInterval)
	if exerciserecordtwo.Error != nil {
		return exerciserecordtwo.Error
	} else if exerciserecordtwo.RowsAffected != 1 {
		return errors.New("No exercise updated in the database.")
	}

	return nil

}

func GetExercisesBetweenDatesUsingDates(goalID int, startDate time.Time, endDate time.Time) ([]models.Exercise, error) {

	var exercises []models.Exercise

	startDayString := startDate.Format("2006-01-02") + " 00:00:00.000"
	endDayString := endDate.Format("2006-01-02") + " 23:59:59"

	exerciserecord := Instance.Where("`exercises`.enabled = ?", 1).Where("`exercises`.Goal = ?", goalID).Where("`exercises`.Date >= ?", startDayString).Where("`exercises`.Date <= ?", endDayString).Find(&exercises)
	if exerciserecord.Error != nil {
		return []models.Exercise{}, exerciserecord.Error
	} else if exerciserecord.RowsAffected == 0 {
		return []models.Exercise{}, nil
	}

	return exercises, nil

}

func GetExercisesForUserUsingUserID(userID int) ([]models.Exercise, error) {

	var exercises []models.Exercise

	exerciserecord := Instance.Order("date desc").Where("`exercises`.enabled = ?", 1).Joins("JOIN goals on `exercises`.goal = `goals`.ID").Where("`goals`.user = ?", userID).Where("`goals`.enabled = ?", 1).Find(&exercises)
	if exerciserecord.Error != nil {
		return []models.Exercise{}, exerciserecord.Error
	} else if exerciserecord.RowsAffected == 0 {
		return []models.Exercise{}, nil
	}

	return exercises, nil

}

func GetExercisesForUserUsingUserIDAndGoalID(userID int, goalID int) ([]models.Exercise, error) {

	var exercises []models.Exercise

	exerciserecord := Instance.Order("date desc").Where("`exercises`.enabled = ?", 1).Where("`exercises`.goal = ?", goalID).Joins("JOIN goals on `exercises`.goal = `goals`.ID").Where("`goals`.user = ?", userID).Where("`goals`.enabled = ?", 1).Find(&exercises)
	if exerciserecord.Error != nil {
		return []models.Exercise{}, exerciserecord.Error
	} else if exerciserecord.RowsAffected == 0 {
		return []models.Exercise{}, nil
	}

	return exercises, nil

}
