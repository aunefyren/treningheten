package database

import (
	"aunefyren/treningheten/models"
	"errors"
)

// Create new exercise
func CreateExerciseForExerciseDayInDatabase(exercise models.Exercise) error {
	record := Instance.Create(&exercise)
	if record.Error != nil {
		return record.Error
	}
	return nil
}

// Get all exercise for exercise-day
func GetExerciseByExerciseDayID(exerciseDayID int) ([]models.Exercise, error) {

	var exercises []models.Exercise

	exerciseRecord := Instance.Where("`exercises`.enabled = ?", 1).Where("`exercises`.exercise_day = ?", exerciseDayID).Find(&exercises)
	if exerciseRecord.Error != nil {
		return []models.Exercise{}, exerciseRecord.Error
	}

	return exercises, nil

}

// Turn on exercise in dastabase
func UpdateExerciseByTurningOnByExerciseID(exerciseID int) error {

	var exercise models.Exercise

	exerciseRecord := Instance.Model(exercise).Where("`exercises`.enabled = ?", 1).Where("`exercises`.ID = ?", exerciseID).Update("on", 1)
	if exerciseRecord.Error != nil {
		return exerciseRecord.Error
	} else if exerciseRecord.RowsAffected != 1 {
		return errors.New("No exercise updated in the database.")
	}

	return nil

}

// Turn off exercise in dastabase
func UpdateExerciseByTurningOffByExerciseID(exerciseID int) error {

	var exercise models.Exercise

	exerciseRecord := Instance.Model(exercise).Where("`exercises`.enabled = ?", 1).Where("`exercises`.ID = ?", exerciseID).Update("on", 0)
	if exerciseRecord.Error != nil {
		return exerciseRecord.Error
	} else if exerciseRecord.RowsAffected != 1 {
		return errors.New("No exercise updated in the database.")
	}

	return nil

}
