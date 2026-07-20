package database

import (
	"errors"
	"time"

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

// GetExerciseForUserWithHevyWorkoutID finds the exercise a Hevy workout was imported
// into. The Hevy workout id is stored directly on the exercise, so this only scopes to
// the user via the exercise day (no operation/set joins like the Strava lookup needs).
func GetExerciseForUserWithHevyWorkoutID(userID uuid.UUID, hevyWorkoutID string) (exercise *models.Exercise, err error) {
	exercise = nil
	err = nil

	exerciseRecord := Instance.Model(exercise).
		Where("`exercises`.enabled = ?", 1).
		Where("`exercises`.hevy_workout_id = ?", hevyWorkoutID).
		Joins("JOIN `exercise_days` on `exercises`.exercise_day_id = `exercise_days`.id").
		Where("`exercise_days`.enabled = ?", 1).
		Where("`exercise_days`.user_id = ?", userID).
		Find(&exercise)

	if exerciseRecord.Error != nil {
		return exercise, exerciseRecord.Error
	} else if exerciseRecord.RowsAffected != 1 {
		return nil, err
	}

	return exercise, err
}

// GetHevyExerciseForUserNearTime returns an enabled Hevy-imported exercise whose start
// time falls within ±window of start, for the given user (used to skip a Strava activity
// that duplicates a Hevy workout). Returns nil when there is no match.
func GetHevyExerciseForUserNearTime(userID uuid.UUID, start time.Time, window time.Duration) (*models.Exercise, error) {
	var exercises []models.Exercise

	lower := start.Add(-window).UTC().Format("2006-01-02 15:04:05")
	upper := start.Add(window).UTC().Format("2006-01-02 15:04:05")

	record := Instance.
		Where("`exercises`.enabled = ?", 1).
		Where("`exercises`.hevy_workout_id IS NOT NULL").
		Where("`exercises`.`time` >= ?", lower).
		Where("`exercises`.`time` <= ?", upper).
		Joins("JOIN `exercise_days` on `exercises`.exercise_day_id = `exercise_days`.id").
		Where("`exercise_days`.enabled = ?", 1).
		Where("`exercise_days`.user_id = ?", userID).
		Order("`exercises`.`time` ASC").
		Limit(1).
		Find(&exercises)

	if record.Error != nil {
		return nil, record.Error
	} else if len(exercises) == 0 {
		return nil, nil
	}

	return &exercises[0], nil
}

// GetStravaExerciseForUserNearTime returns an enabled Strava-sourced exercise (one with a
// set carrying a Strava id, and not itself a Hevy import) whose start time falls within
// ±window of start, for the given user (used so a Hevy workout can supersede an
// already-imported Strava duplicate). Returns nil when there is no match.
func GetStravaExerciseForUserNearTime(userID uuid.UUID, start time.Time, window time.Duration) (*models.Exercise, error) {
	var exercises []models.Exercise

	lower := start.Add(-window).UTC().Format("2006-01-02 15:04:05")
	upper := start.Add(window).UTC().Format("2006-01-02 15:04:05")

	record := Instance.
		Where("`exercises`.enabled = ?", 1).
		Where("`exercises`.hevy_workout_id IS NULL").
		Where("`exercises`.`time` >= ?", lower).
		Where("`exercises`.`time` <= ?", upper).
		Joins("JOIN `operations` on `operations`.exercise_id = `exercises`.id").
		Where("`operations`.enabled = ?", 1).
		Joins("JOIN `operation_sets` on `operation_sets`.operation_id = `operations`.id").
		Where("`operation_sets`.enabled = ?", 1).
		Where("`operation_sets`.strava_id IS NOT NULL").
		Joins("JOIN `exercise_days` on `exercises`.exercise_day_id = `exercise_days`.id").
		Where("`exercise_days`.enabled = ?", 1).
		Where("`exercise_days`.user_id = ?", userID).
		Group("`exercises`.id").
		Order("`exercises`.`time` ASC").
		Limit(1).
		Find(&exercises)

	if record.Error != nil {
		return nil, record.Error
	} else if len(exercises) == 0 {
		return nil, nil
	}

	return &exercises[0], nil
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
