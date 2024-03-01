package database

import (
	"aunefyren/treningheten/models"
	"errors"

	"github.com/google/uuid"
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
func GetExerciseByExerciseDayID(exerciseDayID uuid.UUID) ([]models.Exercise, error) {

	var exercises []models.Exercise

	exerciseRecord := Instance.Where("`exercises`.enabled = ?", 1).Where("`exercises`.exercise_day_id = ?", exerciseDayID).Find(&exercises)
	if exerciseRecord.Error != nil {
		return []models.Exercise{}, exerciseRecord.Error
	}

	return exercises, nil

}

// Turn on exercise in dastabase
func UpdateExerciseByTurningOnByExerciseID(exerciseID uuid.UUID) error {

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
func UpdateExerciseByTurningOffByExerciseID(exerciseID uuid.UUID) error {

	var exercise models.Exercise

	exerciseRecord := Instance.Model(exercise).Where("`exercises`.enabled = ?", 1).Where("`exercises`.ID = ?", exerciseID).Update("on", 0)
	if exerciseRecord.Error != nil {
		return exerciseRecord.Error
	} else if exerciseRecord.RowsAffected != 1 {
		return errors.New("No exercise updated in the database.")
	}

	return nil

}

func GetExerciseByIDAndUserID(exerciseDayID uuid.UUID, userID uuid.UUID) (models.Exercise, error) {
	var exercise models.Exercise

	record := Instance.Where("`exercise_days`.enabled = ?", 1).
		Where("`exercise_days`.id = ?", exerciseDayID).
		Joins("JOIN `exercise_days` on `exercises`.exercise_day_id = `exercise_days`.id").
		Where("`exercise_days`.enabled = ?", 1).
		Joins("JOIN `goals` on `exercise_days`.goal_id = `goals`.id").
		Where("`goals`.enabled = ?", 1).
		Joins("JOIN `users` on `goals`.user_id = `users`.id").
		Where("`users`.enabled = ?", 1).
		Where("`users`.id = ?", userID).
		Find(&exercise)

	if record.Error != nil {
		return models.Exercise{}, record.Error
	} else if record.RowsAffected == 0 {
		return models.Exercise{}, nil
	}

	return exercise, nil
}
