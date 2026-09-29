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

// GetExerciseByIDAndUserID returns the enabled, switched-on exercise with this id belonging to
// this user, or **nil** when there is none — a wrong id, another user's, or a disabled/off
// session. Callers treat a nil result as "not found / not yours", so the miss MUST return nil:
// GORM's Find allocates the struct whether or not a row matched, and returning that zero-value
// pointer made every `exercise == nil` check in the callers dead code (one caller had no check
// at all, which let an activity be attached to another user's session). Mirrors
// GetExerciseDayByIDAndUserID.
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
		return nil, record.Error
	} else if record.RowsAffected != 1 {
		return nil, nil
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
		return models.Exercise{}, ErrExerciseNotFound
	}

	return exercise, nil
}

// ErrExerciseNotFound is returned when a session doesn't exist or isn't the caller's, so
// handlers can answer 404 rather than treating a miss as a server error.
var ErrExerciseNotFound = errors.New("No exercise found.")

func UpdateExerciseInDB(exercise models.Exercise) (models.Exercise, error) {
	record := Instance.Save(&exercise)
	if record.Error != nil {
		return exercise, record.Error
	}
	return exercise, nil
}

// SetExerciseCountsTowardGoal writes the goal-counting flag with an explicit column update.
// It exists because the field carries a `default:true` tag, so GORM omits a false zero value
// from an INSERT and the DB default silently wins — importers that decide the flag while
// creating the session must persist it through here (same reason as
// UpsertActivityGoalSettingInDB).
// SetExerciseIsOn writes is_on explicitly. Like counts_toward_goal it is tagged
// default:true, so a false value never survives a plain Create.
func SetExerciseIsOn(exerciseID uuid.UUID, isOn bool) error {
	return Instance.Model(&models.Exercise{}).
		Where("`exercises`.id = ?", exerciseID).
		Update("is_on", isOn).Error
}

func SetExerciseCountsTowardGoal(exerciseID uuid.UUID, countsTowardGoal bool) error {
	return Instance.Model(&models.Exercise{}).
		Where("`exercises`.id = ?", exerciseID).
		Update("counts_toward_goal", countsTowardGoal).Error
}

// CountStravaActivitiesInExercise returns how many distinct Strava activities the session
// holds. The Strava sync uses it to spot a *combined* session (see APIStravaCombine): with
// several activities merged into one session, no single activity's privacy setting may
// speak for the whole thing.
func CountStravaActivitiesInExercise(exerciseID uuid.UUID) (int64, error) {
	var count int64
	err := Instance.Model(&models.OperationSet{}).
		Where("`operation_sets`.enabled = ?", 1).
		Where("`operation_sets`.strava_id IS NOT NULL").
		Joins("JOIN `operations` on `operation_sets`.operation_id = `operations`.id").
		Where("`operations`.enabled = ?", 1).
		Where("`operations`.exercise_id = ?", exerciseID).
		Distinct("`operation_sets`.strava_id").
		Count(&count).Error
	return count, err
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
