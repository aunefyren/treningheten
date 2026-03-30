package database

import (
	"errors"

	"github.com/aunefyren/treningheten/models"

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

	exerciseRecord := Instance.
		Where("`exercises`.enabled = ?", 1).
		Where("`exercises`.exercise_day_id = ?", exerciseDayID).
		Find(&exercises)

	if exerciseRecord.Error != nil {
		return []models.Exercise{}, exerciseRecord.Error
	}

	return exercises, nil

}

// Turn on exercise in database
func UpdateExerciseByTurningOnByExerciseID(exerciseID uuid.UUID) error {

	var exercise models.Exercise

	exerciseRecord := Instance.Model(exercise).Where("`exercises`.enabled = ?", 1).Where("`exercises`.ID = ?", exerciseID).Update("is_on", 1)
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

	exerciseRecord := Instance.Model(exercise).Where("`exercises`.enabled = ?", 1).Where("`exercises`.ID = ?", exerciseID).Update("is_on", 0)
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
		Where("`exercises`.is_on = ?", 1).
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

	exerciseRecord := Instance.Model(exercise).
		Where("`exercises`.enabled = ?", 1).
		Joins("JOIN `operations` on `operations`.exercise_id = `exercises`.id").
		Where("`operations`.enabled = ?", 1).
		Joins("JOIN `operation_sets` on `operation_sets`.operation_id = `operations`.id").
		Where("`operation_sets`.enabled = ?", 1).
		Where("`operation_sets`.strava_id LIKE ?", stravaIDString).
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

// Return exercise days where there are exercises that are enabled, is on, for a user
func GetAllExerciseDaysWithExerciseByUserID(userID uuid.UUID) ([]models.ExerciseDay, error) {
	var exerciseDays []models.ExerciseDay

	record := Instance.
		Where("`exercise_days`.enabled = ?", 1).
		Joins("JOIN `exercises` ON `exercises`.exercise_day_id = `exercise_days`.id").
		Where("`exercises`.enabled = ?", 1).
		Where("`exercises`.is_on = ?", 1).
		Joins("JOIN `users` ON `users`.id = `exercise_days`.user_id").
		Where("`users`.enabled = ?", 1).
		Where("`users`.id = ?", userID).
		Distinct().
		Find(&exerciseDays)

	if record.Error != nil {
		return []models.ExerciseDay{}, record.Error
	}

	return exerciseDays, nil
}

// get enabled exercises for user where strava is attached
func GetStravaExercisesByUserID(userID uuid.UUID) (exercises []models.Exercise, err error) {
	exercises = []models.Exercise{}
	err = nil

	record := Instance.
		Where("`exercises`.enabled = ?", 1).
		Where("`exercises`.is_on = ?", 1).
		Joins("JOIN `operations` on `operations`.exercise_id = `exercises`.id").
		Where("`operations`.enabled = ?", 1).
		Joins("JOIN `operation_sets` on `operation_sets`.operation_id = `operations`.id").
		Where("`operation_sets`.enabled = ?", 1).
		Where("`operation_sets`.strava_id IS NOT NULL").
		Joins("JOIN `exercise_days` on `exercises`.exercise_day_id = `exercise_days`.id").
		Where("`exercise_days`.enabled = ?", 1).
		Joins("JOIN `users` on `exercise_days`.user_id = `users`.id").
		Where("`users`.enabled = ?", 1).
		Where("`users`.id = ?", userID).
		Find(&exercises)

	if record.Error != nil {
		return exercises, record.Error
	}

	return
}
