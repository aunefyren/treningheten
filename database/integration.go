package database

import (
	"github.com/aunefyren/treningheten/models"

	"github.com/google/uuid"
)

// GetIntegrationStatus returns the health row for a (user, provider) pair, or nil when
// there is none — which means the connection is healthy.
func GetIntegrationStatus(userID uuid.UUID, provider string) (*models.IntegrationStatus, error) {
	status := models.IntegrationStatus{}
	record := Instance.Where("integration_statuses.user_id = ?", userID).
		Where("integration_statuses.provider = ?", provider).
		Find(&status)

	if record.Error != nil {
		return nil, record.Error
	} else if record.RowsAffected != 1 {
		return nil, nil
	}

	return &status, nil
}

// SaveIntegrationStatus creates or updates a health row. The row carries no default-true
// bools, so the insert-drops-false trap does not apply.
func SaveIntegrationStatus(status models.IntegrationStatus) (models.IntegrationStatus, error) {
	if status.ID == uuid.Nil {
		status.ID = uuid.New()
	}
	record := Instance.Save(&status)
	if record.Error != nil {
		return status, record.Error
	}
	return status, nil
}

// DeleteIntegrationStatus removes the health row for a (user, provider) pair: the
// connection recovered or was disconnected. Hard delete, so the unique (user_id,
// provider) index stays free for the next breakage.
func DeleteIntegrationStatus(userID uuid.UUID, provider string) error {
	record := Instance.Unscoped().
		Where("integration_statuses.user_id = ?", userID).
		Where("integration_statuses.provider = ?", provider).
		Delete(&models.IntegrationStatus{})
	return record.Error
}
