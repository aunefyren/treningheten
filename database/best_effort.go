package database

import (
	"github.com/aunefyren/treningheten/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ReplaceBestEffortsForOperation swaps an activity's best-effort rows for a freshly computed
// set, in one transaction so a reader never sees half of them.
func ReplaceBestEffortsForOperation(operationID uuid.UUID, userID uuid.UUID, efforts []models.BestEffortResult) error {
	return Instance.Transaction(func(tx *gorm.DB) error {
		if err := tx.Unscoped().Where("operation_id = ?", operationID).Delete(&models.BestEffort{}).Error; err != nil {
			return err
		}
		for _, effort := range efforts {
			row := models.BestEffort{
				UserID:         userID,
				OperationID:    operationID,
				DistanceM:      effort.DistanceM,
				ElapsedSeconds: effort.ElapsedSeconds,
				StartPoint:     effort.StartPoint,
				EndPoint:       effort.EndPoint,
			}
			row.ID = uuid.New()
			if err := tx.Create(&row).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// GetBestEffortRowsForUser returns every best effort on the user's visible workouts: enabled
// day, enabled and switched-on workout (so hidden duplicates and deleted workouts drop out),
// enabled activity with an activity type.
func GetBestEffortRowsForUser(userID uuid.UUID) ([]models.BestEffortRow, error) {
	rows := []models.BestEffortRow{}
	record := Instance.Table("best_efforts AS be").
		Select("be.operation_id AS operation_id, d.id AS exercise_day_id, o.action_id AS action_id, a.strava_name AS strava_name, "+
			"be.distance_m AS distance_m, be.elapsed_seconds AS elapsed_seconds, d.date AS date, e.time AS time").
		Joins("JOIN operations AS o ON o.id = be.operation_id AND o.enabled = true").
		Joins("JOIN actions AS a ON a.id = o.action_id").
		Joins("JOIN exercises AS e ON e.id = o.exercise_id AND e.enabled = true AND e.is_on = true").
		Joins("JOIN exercise_days AS d ON d.id = e.exercise_day_id AND d.enabled = true").
		Where("d.user_id = ?", userID).
		Where("be.user_id = ?", userID).
		Where("be.deleted_at IS NULL").
		Scan(&rows)
	return rows, record.Error
}

// GetOperationStreamsForUser returns the first stored stream of one of the user's activities
// and the Strava sport name of its activity type, or nil streams when it has none.
func GetOperationStreamsForUser(operationID uuid.UUID, userID uuid.UUID) (*models.StravaActivityStreams, string, error) {
	var row struct {
		Streams    *models.StravaStreamsJSON
		StravaName string
	}
	record := Instance.Table("operation_sets AS os").
		Select("os.strava_streams AS streams, COALESCE(a.strava_name, '') AS strava_name").
		Joins("JOIN operations AS o ON o.id = os.operation_id").
		Joins("LEFT JOIN actions AS a ON a.id = o.action_id").
		Joins("JOIN exercises AS e ON e.id = o.exercise_id").
		Joins("JOIN exercise_days AS d ON d.id = e.exercise_day_id").
		Where("os.operation_id = ?", operationID).
		Where("d.user_id = ?", userID).
		Where("os.strava_streams IS NOT NULL").
		Limit(1).
		Scan(&row)
	if record.Error != nil {
		return nil, "", record.Error
	}
	if row.Streams == nil {
		return nil, row.StravaName, nil
	}
	return &row.Streams.StravaActivityStreams, row.StravaName, nil
}
