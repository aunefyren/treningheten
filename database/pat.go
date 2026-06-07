package database

import (
	"errors"
	"time"

	"github.com/aunefyren/treningheten/models"

	"github.com/google/uuid"
)

// CreatePersonalAccessToken persists a new PAT.
func CreatePersonalAccessToken(pat *models.PersonalAccessToken) error {
	if pat.ID == uuid.Nil {
		pat.ID = uuid.New()
	}
	record := Instance.Create(pat)
	return record.Error
}

// GetPersonalAccessTokenByHash returns an active PAT by its token hash.
func GetPersonalAccessTokenByHash(tokenHash string) (models.PersonalAccessToken, error) {
	var pat models.PersonalAccessToken
	record := Instance.Where("token_hash = ?", tokenHash).Find(&pat)
	if record.Error != nil {
		return models.PersonalAccessToken{}, record.Error
	}
	if record.RowsAffected != 1 {
		return models.PersonalAccessToken{}, errors.New("personal access token not found")
	}
	return pat, nil
}

// GetPersonalAccessTokensByUser returns a user's non-revoked PATs.
func GetPersonalAccessTokensByUser(userID uuid.UUID) ([]models.PersonalAccessToken, error) {
	var pats []models.PersonalAccessToken
	record := Instance.Where("user_id = ? AND revoked_at IS NULL", userID).
		Order("created_at desc").Find(&pats)
	if record.Error != nil {
		return nil, record.Error
	}
	return pats, nil
}

// RevokePersonalAccessToken revokes a PAT owned by the given user. Scoping the
// update by user_id prevents revoking another user's token.
func RevokePersonalAccessToken(patID uuid.UUID, userID uuid.UUID) error {
	now := time.Now()
	record := Instance.Model(&models.PersonalAccessToken{}).
		Where("id = ? AND user_id = ? AND revoked_at IS NULL", patID, userID).
		Update("revoked_at", now)
	if record.Error != nil {
		return record.Error
	}
	if record.RowsAffected != 1 {
		return errors.New("token not found")
	}
	return nil
}

// TouchPersonalAccessToken records the last time a PAT was used.
func TouchPersonalAccessToken(patID uuid.UUID) error {
	now := time.Now()
	return Instance.Model(&models.PersonalAccessToken{}).
		Where("id = ?", patID).
		Update("last_used_at", now).Error
}
