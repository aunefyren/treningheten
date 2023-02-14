package database

import (
	"aunefyren/treningheten/models"
	"errors"
	"strings"

	"github.com/thanhpk/randstr"
)

// Genrate a random invite code an return ut
func GenrateRandomInvite() (string, error) {
	var invite models.Invite

	randomString := randstr.String(16)
	invite.InviteCode = strings.ToUpper(randomString)

	record := Instance.Create(&invite)
	if record.Error != nil {
		return "", record.Error
	}

	return invite.InviteCode, nil
}

// Verify unsued invite code exists
func VerifyUnusedUserInviteCode(providedCode string) (bool, error) {
	var invitestruct models.Invite
	inviterecords := Instance.Where("`invites`.invite_enabled = ?", 1).Where("`invites`.invite_used= ?", 0).Where("`invites`.invite_code = ?", providedCode).Find(&invitestruct)
	if inviterecords.Error != nil {
		return false, inviterecords.Error
	}
	if inviterecords.RowsAffected != 1 {
		return false, nil
	}
	return true, nil
}

// Set invite code to used
func SetUsedUserInviteCode(providedCode string, userIDClaimer int) error {
	var invitestruct models.Invite
	inviterecords := Instance.Model(invitestruct).Where("`invites`.invite_code= ?", providedCode).Update("invite_used", 1)
	if inviterecords.Error != nil {
		return inviterecords.Error
	}
	if inviterecords.RowsAffected != 1 {
		return errors.New("Code not changed in database.")
	}

	inviterecords = Instance.Model(invitestruct).Where("`invites`.invite_code= ?", providedCode).Update("invite_recipient", userIDClaimer)
	if inviterecords.Error != nil {
		return inviterecords.Error
	}
	if inviterecords.RowsAffected != 1 {
		return errors.New("Recipient not changed in database.")
	}

	return nil
}