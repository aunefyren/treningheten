package database

import (
	"aunefyren/treningheten/models"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/thanhpk/randstr"
)

// Generate a random invite code an return ut
func GenerateRandomInvite() (string, error) {
	var invite models.Invite

	randomString := randstr.String(16)
	invite.Code = strings.ToUpper(randomString)
	invite.ID = uuid.New()

	record := Instance.Create(&invite)
	if record.Error != nil {
		return "", record.Error
	}

	return invite.Code, nil
}

// Verify unused invite code exists
func VerifyUnusedUserInviteCode(providedCode string) (bool, error) {
	var invitestruct models.Invite
	inviterecords := Instance.Where("`invites`.enabled = ?", 1).Where("`invites`.used = ?", 0).Where("`invites`.code = ?", providedCode).Find(&invitestruct)
	if inviterecords.Error != nil {
		return false, inviterecords.Error
	}
	if inviterecords.RowsAffected != 1 {
		return false, nil
	}
	return true, nil
}

// Set invite code to used
func SetUsedUserInviteCode(providedCode string, userIDClaimer uuid.UUID) error {
	var invitestruct models.Invite
	inviterecords := Instance.Model(invitestruct).Where("`invites`.code = ?", providedCode).Update("used", 1)
	if inviterecords.Error != nil {
		return inviterecords.Error
	}
	if inviterecords.RowsAffected != 1 {
		return errors.New("Code not changed in database.")
	}

	inviterecords = Instance.Model(invitestruct).Where("`invites`.code= ?", providedCode).Update("recipient_id", userIDClaimer)
	if inviterecords.Error != nil {
		return inviterecords.Error
	}
	if inviterecords.RowsAffected != 1 {
		return errors.New("Recipient not changed in database.")
	}

	return nil
}

// Set invite code to used
func GetAllEnabledInvites() ([]models.Invite, error) {
	var invitestruct []models.Invite
	inviterecords := Instance.Where("`invites`.enabled = ?", 1).Find(&invitestruct)
	if inviterecords.Error != nil {
		return []models.Invite{}, inviterecords.Error
	}
	if inviterecords.RowsAffected == 0 {
		return []models.Invite{}, nil
	}
	return invitestruct, nil
}

// Get invite using ID
func GetInviteByID(inviteID uuid.UUID) (models.Invite, error) {
	var invitestruct models.Invite
	inviterecords := Instance.Where("`invites`.enabled = ?", 1).Where("`invites`.ID = ?", inviteID).Find(&invitestruct)
	if inviterecords.Error != nil {
		return models.Invite{}, inviterecords.Error
	}
	if inviterecords.RowsAffected != 1 {
		return models.Invite{}, errors.New("Invite not found.")
	}
	return invitestruct, nil
}

// Set invite to disabled by ID
func DeleteInviteByID(inviteID uuid.UUID) error {
	var invitestruct models.Invite
	inviterecords := Instance.Model(invitestruct).Where("`invites`.ID = ?", inviteID).Update("enabled", 0)
	if inviterecords.Error != nil {
		return inviterecords.Error
	}
	if inviterecords.RowsAffected != 1 {
		return errors.New("Code not changed in database.")
	}

	return nil
}
