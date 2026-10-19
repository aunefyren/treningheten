package database

import (
	"github.com/aunefyren/treningheten/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// GetSportMaxHeartratesForUserID returns every per-sport max heart rate a user has set.
// Sports without a row follow the user's global anchors.
func GetSportMaxHeartratesForUserID(userID uuid.UUID) (rows []models.UserSportMaxHeartrate, err error) {
	rows = []models.UserSportMaxHeartrate{}
	record := Instance.Where("user_sport_max_heartrates.user_id = ?", userID).Find(&rows)
	if record.Error != nil {
		return []models.UserSportMaxHeartrate{}, record.Error
	}
	return rows, nil
}

// SaveTrainingProfileInDB writes the user row and replaces their per-sport max heart rates
// with sportMax, in one transaction so a failure can't leave half a profile saved. The old
// rows are hard-deleted: they carry no history, and a soft-deleted row would still hold the
// (user, action) unique index.
func SaveTrainingProfileInDB(user models.User, sportMax []models.SportMaxHeartrate) error {
	return Instance.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(&user).Error; err != nil {
			return err
		}
		if err := tx.Unscoped().Where("user_id = ?", user.ID).Delete(&models.UserSportMaxHeartrate{}).Error; err != nil {
			return err
		}
		for _, entry := range sportMax {
			row := models.UserSportMaxHeartrate{UserID: user.ID, ActionID: entry.ActionID, MaxHeartrate: entry.MaxHeartrate}
			row.ID = uuid.New()
			if err := tx.Create(&row).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// GetObservedMaxHeartratesByActionForUserID returns, per sport (action), the highest heart
// rate recorded across the user's enabled activities. It reads the per-operation
// max_heartrate rollup (already capped against sensor spikes, see models.ObservedMaxHeartrate),
// so no stream is parsed and a deleted activity drops out on its own. Sports with no
// heart-rate data are absent from the map.
func GetObservedMaxHeartratesByActionForUserID(userID uuid.UUID) (map[uuid.UUID]int, error) {
	var rows []struct {
		ActionID     string
		MaxHeartrate int
	}
	record := Instance.Table("operations AS o").
		Select("o.action_id AS action_id, MAX(o.max_heartrate) AS max_heartrate").
		Joins("JOIN exercises AS e ON o.exercise_id = e.id AND e.enabled = true").
		Joins("JOIN exercise_days AS d ON e.exercise_day_id = d.id AND d.enabled = true").
		Where("o.enabled = true").
		Where("e.is_on = true").
		Where("d.user_id = ?", userID).
		Where("o.action_id IS NOT NULL").
		Where("o.max_heartrate IS NOT NULL").
		Group("o.action_id").
		Scan(&rows)
	if record.Error != nil {
		return nil, record.Error
	}

	observed := map[uuid.UUID]int{}
	for _, row := range rows {
		actionID, err := uuid.Parse(row.ActionID)
		if err != nil || row.MaxHeartrate <= 0 {
			continue
		}
		observed[actionID] = row.MaxHeartrate
	}
	return observed, nil
}
