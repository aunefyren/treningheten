package database

import (
	"errors"
	"time"

	"github.com/aunefyren/treningheten/models"

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

// Get exercise or user on certain date
func GetExerciseDayByUserIDAndDate(userID uuid.UUID, date time.Time) (*models.ExerciseDay, error) {

	var exercise models.ExerciseDay

	startDayString := date.Format("2006-01-02") + " 00:00:00.000"
	endDayString := date.Format("2006-01-02") + " 23:59:59"

	records := Instance.Where("`exercise_days`.enabled = ?", 1).Where("`exercise_days`.user_id = ?", userID).Where("`exercise_days`.Date >= ?", startDayString).Where("`exercise_days`.Date <= ?", endDayString).Find(&exercise)
	if records.Error != nil {
		return nil, records.Error
	} else if records.RowsAffected == 0 {
		return nil, nil
	}

	return &exercise, nil
}

// Get exercise days
func GetAllExerciseDays() ([]models.ExerciseDay, error) {
	var exerciseDays []models.ExerciseDay

	records := Instance.Where("`exercise_days`.enabled = ?", 1).
		Find(&exerciseDays)

	if records.Error != nil {
		return nil, records.Error
	} else if records.RowsAffected == 0 {
		return exerciseDays, nil
	}

	return exerciseDays, nil
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
func UpdateExerciseDayInDatabase(exercise models.ExerciseDay) (models.ExerciseDay, error) {
	record := Instance.Save(&exercise)
	if record.Error != nil {
		return exercise, record.Error
	}
	return exercise, nil
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

func GetExerciseDaysBetweenDatesUsingDatesAndUserID(userID uuid.UUID, startDate time.Time, endDate time.Time) ([]models.ExerciseDay, error) {

	var exercises []models.ExerciseDay

	startDayString := startDate.Format("2006-01-02") + " 00:00:00.000"
	endDayString := endDate.Format("2006-01-02") + " 23:59:59"

	exerciserecord := Instance.Where("`exercise_days`.enabled = ?", 1).Where("`exercise_days`.user_id = ?", userID).Where("`exercise_days`.Date >= ?", startDayString).Where("`exercise_days`.Date <= ?", endDayString).Find(&exercises)
	if exerciserecord.Error != nil {
		return []models.ExerciseDay{}, exerciserecord.Error
	} else if exerciserecord.RowsAffected == 0 {
		return []models.ExerciseDay{}, nil
	}

	return exercises, nil

}

func GetExerciseDaysForUserUsingUserID(userID uuid.UUID) ([]models.ExerciseDay, error) {

	var exercises []models.ExerciseDay

	exerciserecord := Instance.Order("date desc").
		Where("`exercise_days`.enabled = ?", 1).
		Where("`exercise_days`.user_id = ?", userID).
		Find(&exercises)

	if exerciserecord.Error != nil {
		return []models.ExerciseDay{}, exerciserecord.Error
	} else if exerciserecord.RowsAffected == 0 {
		return []models.ExerciseDay{}, nil
	}

	return exercises, nil

}

func GetExerciseDaysForUserUsingUserIDAndGoalID(userID uuid.UUID, goalID uuid.UUID) ([]models.ExerciseDay, error) {

	var exercises []models.ExerciseDay

	exerciserecord := Instance.Order("date desc").
		Where("`exercise_days`.enabled = ?", 1).
		Where("`exercise_days`.goal_id = ?", goalID).
		Joins("JOIN goals on `exercise_days`.goal_id = `goals`.ID").
		Where("`goals`.user_id = ?", userID).
		Where("`goals`.enabled = ?", 1).
		Find(&exercises)

	if exerciserecord.Error != nil {
		return []models.ExerciseDay{}, exerciserecord.Error
	} else if exerciserecord.RowsAffected == 0 {
		return []models.ExerciseDay{}, nil
	}

	return exercises, nil

}

// Get enabled exercises where the season and goals are enabled too.
func GetAllEnabledExerciseDays() ([]models.ExerciseDay, error) {
	var exercises []models.ExerciseDay

	exerciserecord := Instance.Order("date desc").
		Where("`exercise_days`.enabled = ?", 1).
		Joins("JOIN `goals` on `exercise_days`.goal_id = `goals`.id").
		Where("`goals`.enabled = ?", 1).
		Joins("JOIN `seasons` on `goals`.season_id = `seasons`.id").
		Where("`seasons`.enabled = ?", 1).
		Find(&exercises)
	if exerciserecord.Error != nil {
		return []models.ExerciseDay{}, exerciserecord.Error
	}

	return exercises, nil
}

func GetExerciseDayByID(exerciseDayID uuid.UUID) (*models.ExerciseDay, error) {
	var exerciseDay *models.ExerciseDay

	exerciserecord := Instance.Where("`exercise_days`.enabled = ?", 1).Where("`exercise_days`.id = ?", exerciseDayID).Find(&exerciseDay)
	if exerciserecord.Error != nil {
		return exerciseDay, exerciserecord.Error
	} else if exerciserecord.RowsAffected == 0 {
		return exerciseDay, nil
	}

	return exerciseDay, nil
}

func GetExerciseDayByIDAndUserID(exerciseDayID uuid.UUID, userID uuid.UUID) (exerciseDay *models.ExerciseDay, err error) {
	exerciseDay = &models.ExerciseDay{}
	err = nil

	exerciserecord := Instance.Where("`exercise_days`.enabled = ?", 1).
		Where("`exercise_days`.id = ?", exerciseDayID).
		Joins("JOIN `users` on `exercise_days`.user_id = `users`.id").
		Where("`users`.enabled = ?", 1).
		Where("`users`.id = ?", userID).
		Find(&exerciseDay)

	if exerciserecord.Error != nil {
		return nil, exerciserecord.Error
	} else if exerciserecord.RowsAffected != 1 {
		return nil, nil
	}

	return exerciseDay, nil
}

func UpdateExerciseDayInDB(exerciseDay models.ExerciseDay) (models.ExerciseDay, error) {
	record := Instance.Save(&exerciseDay)
	if record.Error != nil {
		return exerciseDay, record.Error
	}
	return exerciseDay, nil
}

func GetValidExercisesBetweenDatesUsingDatesByUserID(userID uuid.UUID, startDate time.Time, endDate time.Time) ([]models.Exercise, error) {
	var exercises []models.Exercise

	startDayString := startDate.Format("2006-01-02") + " 00:00:00.000"
	endDayString := endDate.Format("2006-01-02") + " 23:59:59"

	exerciserecord := Instance.
		Where("`exercises`.enabled = ?", 1).
		Where("`exercises`.is_on = ?", 1).
		Joins("JOIN `exercise_days` on `exercises`.exercise_day_id = `exercise_days`.id").
		Where("`exercise_days`.enabled = ?", 1).
		Where("`exercise_days`.user_id = ?", userID).
		Where("`exercise_days`.Date >= ?", startDayString).
		Where("`exercise_days`.Date <= ?", endDayString).
		Where("`exercise_days`.Date <= ?", endDayString).
		Find(&exercises)

	if exerciserecord.Error != nil {
		return []models.Exercise{}, exerciserecord.Error
	}

	return exercises, nil
}

func GetExerciseDayByDateAndGoal(goalID uuid.UUID, date time.Time) (*models.ExerciseDay, error) {
	var exercise models.ExerciseDay
	var err error

	startDayString := date.Format("2006-01-02") + " 00:00:00.000"
	endDayString := date.Format("2006-01-02") + " 23:59:59"

	goalrecord := Instance.Where("`exercise_days`.enabled = ?", 1).
		Where("`exercise_days`.goal_id = ?", goalID).
		Where("`exercise_days`.Date >= ?", startDayString).
		Where("`exercise_days`.Date <= ?", endDayString).
		Find(&exercise)

	if goalrecord.Error != nil {
		return nil, goalrecord.Error
	} else if goalrecord.RowsAffected != 1 {
		return nil, err
	}

	return &exercise, err
}

func GetExerciseDayByDateAndUserID(userID uuid.UUID, date time.Time) (*models.ExerciseDay, error) {
	var exercise models.ExerciseDay
	var err error

	startDayString := date.Format("2006-01-02") + " 00:00:00.000"
	endDayString := date.Format("2006-01-02") + " 23:59:59"

	goalrecord := Instance.Where("`exercise_days`.enabled = ?", 1).
		Where("`exercise_days`.user_id = ?", userID).
		Where("`exercise_days`.Date >= ?", startDayString).
		Where("`exercise_days`.Date <= ?", endDayString).
		Find(&exercise)

	if goalrecord.Error != nil {
		return nil, goalrecord.Error
	} else if goalrecord.RowsAffected != 1 {
		return nil, err
	}

	return &exercise, err
}

func GetExerciseDaysForSharingUsersUsingDates(startDate time.Time, endDate time.Time) ([]models.ExerciseDay, error) {
	var exercises []models.ExerciseDay

	startDayString := startDate.Format("2006-01-02") + " 00:00:00.000"
	endDayString := endDate.Format("2006-01-02") + " 23:59:59"

	exerciserecord := Instance.
		Where("`exercise_days`.enabled = ?", 1).
		Where("`exercise_days`.Date >= ?", startDayString).
		Where("`exercise_days`.Date <= ?", endDayString).
		Joins("JOIN `users` on `exercise_days`.user_id = `users`.id").
		Where("`users`.enabled = ?", 1).
		Where("`users`.share_activities = ?", 1).
		Find(&exercises)

	if exerciserecord.Error != nil {
		return []models.ExerciseDay{}, exerciserecord.Error
	} else if exerciserecord.RowsAffected == 0 {
		return []models.ExerciseDay{}, nil
	}

	return exercises, nil
}
