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

// Return exercises that are enabled and on
func GetExerciseByIDAndUserID(exerciseID uuid.UUID, userID uuid.UUID) (*models.Exercise, error) {
	var exercise *models.Exercise

	record := Instance.Where("`exercises`.enabled = ?", 1).
		Where("`exercises`.id = ?", exerciseID).
		Where("`exercises`.on = ?", 1).
		Joins("JOIN `exercise_days` on `exercises`.exercise_day_id = `exercise_days`.id").
		Where("`exercise_days`.enabled = ?", 1).
		Joins("JOIN `users` on `exercise_days`.user_id = `users`.id").
		Where("`users`.enabled = ?", 1).
		Where("`users`.id = ?", userID).
		Find(&exercise)

	if record.Error != nil {
		return exercise, record.Error
	} else if record.RowsAffected != 1 {
		return exercise, nil
	}

	return exercise, nil
}

// Return exercises that are enabled
func GetAllExerciseByIDAndUserID(exerciseID uuid.UUID, userID uuid.UUID) (models.Exercise, error) {
	var exercise models.Exercise

	record := Instance.Where("`exercises`.enabled = ?", 1).
		Where("`exercises`.id = ?", exerciseID).
		Joins("JOIN `exercise_days` on `exercises`.exercise_day_id = `exercise_days`.id").
		Where("`exercise_days`.enabled = ?", 1).
		Joins("JOIN `users` on `exercise_days`.user_id = `users`.id").
		Where("`users`.enabled = ?", 1).
		Where("`users`.id = ?", userID).
		Find(&exercise)

	if record.Error != nil {
		return models.Exercise{}, record.Error
	} else if record.RowsAffected != 1 {
		return models.Exercise{}, errors.New("No exercise found.")
	}

	return exercise, nil
}

func UpdateExerciseInDB(exercise models.Exercise) (models.Exercise, error) {
	record := Instance.Save(&exercise)
	if record.Error != nil {
		return exercise, record.Error
	}
	return exercise, nil
}

func CreateExerciseInDB(exercise models.Exercise) (models.Exercise, error) {
	record := Instance.Create(&exercise)
	if record.Error != nil {
		return exercise, record.Error
	}
	return exercise, nil
}

func GetExerciseForUserWithStravaID(userID uuid.UUID, stravaID string) (exercise *models.Exercise, err error) {
	exercise = nil
	err = nil

	stravaIDString := "%" + stravaID + "%"

	exerciseRecord := Instance.Model(exercise).Where("`exercises`.enabled = ?", 1).
		Where("`exercises`.strava_id LIKE ?", stravaIDString).
		Joins("JOIN `exercise_days` on `exercises`.exercise_day_id = `exercise_days`.id").
		Where("`exercise_days`.enabled = ?", 1).
		Joins("JOIN `users` on `exercise_days`.user_id = `users`.id").
		Where("`users`.enabled = ?", 1).
		Where("`users`.id = ?", userID).
		Find(&exercise)

	if exerciseRecord.Error != nil {
		return exercise, exerciseRecord.Error
	} else if exerciseRecord.RowsAffected != 1 {
		return nil, err
	}

	return exercise, err
}
