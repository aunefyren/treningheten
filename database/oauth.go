package database

import (
	"errors"
	"time"

	"github.com/aunefyren/treningheten/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// --- OAuth clients ---

// CreateOAuthClient persists a newly registered client.
func CreateOAuthClient(client *models.OAuthClient) error {
	if client.ID == uuid.Nil {
		client.ID = uuid.New()
	}
	record := Instance.Create(client)
	return record.Error
}

// GetOAuthClientByClientID returns a client by its public client_id.
func GetOAuthClientByClientID(clientID string) (models.OAuthClient, error) {
	var client models.OAuthClient
	record := Instance.Where("client_id = ?", clientID).Find(&client)
	if record.Error != nil {
		return models.OAuthClient{}, record.Error
	}
	if record.RowsAffected != 1 {
		return models.OAuthClient{}, errors.New("client not found")
	}
	return client, nil
}

// --- Authorization codes ---

// CreateAuthorizationCode persists a single-use authorization code.
func CreateAuthorizationCode(code *models.OAuthAuthorizationCode) error {
	if code.ID == uuid.Nil {
		code.ID = uuid.New()
	}
	record := Instance.Create(code)
	return record.Error
}

// GetAuthorizationCodeByHash returns an authorization code by its hash.
func GetAuthorizationCodeByHash(codeHash string) (models.OAuthAuthorizationCode, error) {
	var code models.OAuthAuthorizationCode
	record := Instance.Where("code_hash = ?", codeHash).Find(&code)
	if record.Error != nil {
		return models.OAuthAuthorizationCode{}, record.Error
	}
	if record.RowsAffected != 1 {
		return models.OAuthAuthorizationCode{}, errors.New("authorization code not found")
	}
	return code, nil
}

// ConsumeAuthorizationCode marks a code as used. It returns an error if the code
// was already consumed, enforcing single-use semantics atomically.
func ConsumeAuthorizationCode(id uuid.UUID) error {
	now := time.Now()
	record := Instance.Model(&models.OAuthAuthorizationCode{}).
		Where("id = ? AND consumed_at IS NULL", id).
		Update("consumed_at", now)
	if record.Error != nil {
		return record.Error
	}
	if record.RowsAffected != 1 {
		return errors.New("authorization code already used")
	}
	return nil
}

// --- Refresh tokens ---

// CreateRefreshToken persists a hashed refresh token.
func CreateRefreshToken(token *models.OAuthRefreshToken) error {
	if token.ID == uuid.Nil {
		token.ID = uuid.New()
	}
	record := Instance.Create(token)
	return record.Error
}

// GetRefreshTokenByHash returns a refresh token by its hash.
func GetRefreshTokenByHash(tokenHash string) (models.OAuthRefreshToken, error) {
	var token models.OAuthRefreshToken
	record := Instance.Where("token_hash = ?", tokenHash).Find(&token)
	if record.Error != nil {
		return models.OAuthRefreshToken{}, record.Error
	}
	if record.RowsAffected != 1 {
		return models.OAuthRefreshToken{}, errors.New("refresh token not found")
	}
	return token, nil
}

// RotateRefreshToken marks an old refresh token as revoked and points it at its
// replacement, in a single transaction with the new token's creation.
func RotateRefreshToken(oldID uuid.UUID, newToken *models.OAuthRefreshToken) error {
	if newToken.ID == uuid.Nil {
		newToken.ID = uuid.New()
	}
	now := time.Now()
	return Instance.Transaction(func(tx *gorm.DB) error {
		// Atomically revoke the old token only if it is still active.
		record := tx.Model(&models.OAuthRefreshToken{}).
			Where("id = ? AND revoked_at IS NULL", oldID).
			Updates(map[string]interface{}{"revoked_at": now, "rotated_to": newToken.ID})
		if record.Error != nil {
			return record.Error
		}
		if record.RowsAffected != 1 {
			return errors.New("refresh token already rotated or revoked")
		}
		if err := tx.Create(newToken).Error; err != nil {
			return err
		}
		return nil
	})
}

// RevokeRefreshToken revokes a single refresh token by hash. It is idempotent.
func RevokeRefreshToken(tokenHash string) error {
	now := time.Now()
	record := Instance.Model(&models.OAuthRefreshToken{}).
		Where("token_hash = ? AND revoked_at IS NULL", tokenHash).
		Update("revoked_at", now)
	return record.Error
}

// RevokeRefreshTokenChain revokes a token and every token reachable via the
// rotation chain. Used for refresh-token reuse detection.
func RevokeRefreshTokenChain(startID uuid.UUID) error {
	now := time.Now()
	currentID := &startID
	for currentID != nil {
		var token models.OAuthRefreshToken
		record := Instance.Where("id = ?", *currentID).Find(&token)
		if record.Error != nil {
			return record.Error
		}
		if record.RowsAffected != 1 {
			break
		}
		if token.RevokedAt == nil {
			if err := Instance.Model(&models.OAuthRefreshToken{}).
				Where("id = ?", token.ID).
				Update("revoked_at", now).Error; err != nil {
				return err
			}
		}
		currentID = token.RotatedTo
	}
	return nil
}

// TouchRefreshToken records the last time a refresh token was used.
func TouchRefreshToken(id uuid.UUID) error {
	now := time.Now()
	return Instance.Model(&models.OAuthRefreshToken{}).
		Where("id = ?", id).
		Update("last_used_at", now).Error
}
