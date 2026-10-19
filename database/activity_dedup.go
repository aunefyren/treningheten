package database

import (
	"time"

	"github.com/aunefyren/treningheten/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// stravaActivityCountChunk caps the IN-list size when counting Strava activities per
// exercise, keeping well under every backend's bound-parameter limit.
const stravaActivityCountChunk = 500

// importedExercisesForUser builds the base query for duplicate detection: a user's enabled,
// on, timed sessions that came from an import — a Hevy workout id on the exercise, or an
// enabled set carrying a Strava id. Manual sessions never take part (see
// docs/activity-dedup.md).
func importedExercisesForUser(userID uuid.UUID) *gorm.DB {
	return Instance.Model(&models.Exercise{}).
		Where("exercises.enabled = ?", true).
		Where("exercises.is_on = ?", true).
		Where("exercises.time IS NOT NULL").
		Where("(exercises.hevy_workout_id IS NOT NULL OR EXISTS ("+
			"SELECT 1 FROM operations JOIN operation_sets ON operation_sets.operation_id = operations.id "+
			"WHERE operations.exercise_id = exercises.id AND operations.enabled = ? "+
			"AND operation_sets.enabled = ? AND operation_sets.strava_id IS NOT NULL))", true, true).
		Joins("JOIN exercise_days on exercises.exercise_day_id = exercise_days.id").
		Where("exercise_days.enabled = ?", true).
		Where("exercise_days.user_id = ?", userID)
}

// GetImportedExercisesForUserNearTime returns the user's imported sessions starting within
// ±window of start, the candidates an incoming import is compared against.
func GetImportedExercisesForUserNearTime(userID uuid.UUID, start time.Time, window time.Duration) ([]models.Exercise, error) {
	exercises := []models.Exercise{}
	record := importedExercisesForUser(userID).
		Where("exercises.time >= ?", instantParam(start.Add(-window))).
		Where("exercises.time <= ?", instantParam(start.Add(window))).
		Order("exercises.time ASC").
		Find(&exercises)
	return exercises, record.Error
}

// GetImportedExercisesForUser returns every imported session of the user, oldest first; the
// input to a full duplicate reconcile.
func GetImportedExercisesForUser(userID uuid.UUID) ([]models.Exercise, error) {
	exercises := []models.Exercise{}
	record := importedExercisesForUser(userID).
		Order("exercises.time ASC").
		Find(&exercises)
	return exercises, record.Error
}

// CountStravaActivitiesForExercises returns how many distinct Strava activities each of the
// given sessions holds (see CountStravaActivitiesInExercise). Sessions without any are
// absent from the map.
func CountStravaActivitiesForExercises(exerciseIDs []uuid.UUID) (map[uuid.UUID]int64, error) {
	type countRow struct {
		ExerciseID uuid.UUID
		Activities int64
	}

	counts := map[uuid.UUID]int64{}
	for start := 0; start < len(exerciseIDs); start += stravaActivityCountChunk {
		end := min(start+stravaActivityCountChunk, len(exerciseIDs))

		rows := []countRow{}
		err := Instance.Model(&models.OperationSet{}).
			Select("operations.exercise_id AS exercise_id, COUNT(DISTINCT operation_sets.strava_id) AS activities").
			Joins("JOIN operations on operation_sets.operation_id = operations.id").
			Where("operation_sets.enabled = ?", true).
			Where("operation_sets.strava_id IS NOT NULL").
			Where("operations.enabled = ?", true).
			Where("operations.exercise_id IN ?", exerciseIDs[start:end]).
			Group("operations.exercise_id").
			Scan(&rows).Error
		if err != nil {
			return nil, err
		}
		for _, row := range rows {
			counts[row.ExerciseID] = row.Activities
		}
	}

	return counts, nil
}

// GetSupersededExerciseForUserWithStravaID returns the hidden duplicate a Strava activity
// was imported into, or nil when the activity has no hidden session. The regular lookup
// (GetExerciseForUserWithStravaID) only sees enabled sessions; this one lets the sync leave
// a hidden copy alone instead of importing it again.
func GetSupersededExerciseForUserWithStravaID(userID uuid.UUID, stravaID string) (*models.Exercise, error) {
	exercises := []models.Exercise{}
	record := Instance.Model(&models.Exercise{}).
		Where("exercises.enabled = ?", false).
		Where("exercises.superseded_by_id IS NOT NULL").
		Joins("JOIN operations on operations.exercise_id = exercises.id").
		Where("operations.enabled = ?", true).
		Joins("JOIN operation_sets on operation_sets.operation_id = operations.id").
		Where("operation_sets.enabled = ?", true).
		Where("operation_sets.strava_id LIKE ?", "%"+stravaID+"%").
		Joins("JOIN exercise_days on exercises.exercise_day_id = exercise_days.id").
		Where("exercise_days.enabled = ?", true).
		Where("exercise_days.user_id = ?", userID).
		Limit(1).
		Find(&exercises)
	if record.Error != nil || len(exercises) == 0 {
		return nil, record.Error
	}
	return &exercises[0], nil
}

// GetSupersededExerciseForUserWithHevyWorkoutID is the Hevy counterpart of
// GetSupersededExerciseForUserWithStravaID.
func GetSupersededExerciseForUserWithHevyWorkoutID(userID uuid.UUID, hevyWorkoutID string) (*models.Exercise, error) {
	exercises := []models.Exercise{}
	record := Instance.Model(&models.Exercise{}).
		Where("exercises.enabled = ?", false).
		Where("exercises.superseded_by_id IS NOT NULL").
		Where("exercises.hevy_workout_id = ?", hevyWorkoutID).
		Joins("JOIN exercise_days on exercises.exercise_day_id = exercise_days.id").
		Where("exercise_days.enabled = ?", true).
		Where("exercise_days.user_id = ?", userID).
		Limit(1).
		Find(&exercises)
	if record.Error != nil || len(exercises) == 0 {
		return nil, record.Error
	}
	return &exercises[0], nil
}

// GetSupersededExercisesForUser returns every session of the user hidden as a duplicate.
func GetSupersededExercisesForUser(userID uuid.UUID) ([]models.Exercise, error) {
	exercises := []models.Exercise{}
	record := Instance.Model(&models.Exercise{}).
		Where("exercises.superseded_by_id IS NOT NULL").
		Joins("JOIN exercise_days on exercises.exercise_day_id = exercise_days.id").
		Where("exercise_days.user_id = ?", userID).
		Find(&exercises)
	return exercises, record.Error
}

// GetExercisesSupersededBy returns the sessions hidden as duplicates of the given session.
func GetExercisesSupersededBy(exerciseID uuid.UUID) ([]models.Exercise, error) {
	exercises := []models.Exercise{}
	record := Instance.Model(&models.Exercise{}).
		Where("exercises.superseded_by_id = ?", exerciseID).
		Find(&exercises)
	return exercises, record.Error
}

// SetExerciseSupersededBy hides a session as a duplicate of supersededByID, or restores it
// when supersededByID is nil. Only the session row changes: every read path filters on
// exercises.enabled, so its operations and sets stay as they are and come back intact.
// Written as an explicit column update, since GORM's Save would skip a false `enabled`.
func SetExerciseSupersededBy(exerciseID uuid.UUID, supersededByID *uuid.UUID) error {
	return Instance.Model(&models.Exercise{}).
		Where("id = ?", exerciseID).
		Updates(map[string]interface{}{
			"enabled":          supersededByID == nil,
			"superseded_by_id": supersededByID,
		}).Error
}

// RepointSupersededExercises moves every duplicate hidden behind fromID to toID; used when
// a winning session is merged into another one.
func RepointSupersededExercises(fromID uuid.UUID, toID uuid.UUID) error {
	return Instance.Model(&models.Exercise{}).
		Where("superseded_by_id = ?", fromID).
		Update("superseded_by_id", toID).Error
}
