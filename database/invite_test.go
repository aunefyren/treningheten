package database

import (
	"testing"

	"github.com/google/uuid"
)

func TestGenerateRandomInviteAndVerify(t *testing.T) {
	newTestDB(t)

	code, err := GenerateRandomInvite()
	if err != nil {
		t.Fatalf("GenerateRandomInvite returned error: %v", err)
	}
	if len(code) != 16 {
		t.Errorf("invite code length: got %d, want 16", len(code))
	}

	unused, err := VerifyUnusedUserInviteCode(code)
	if err != nil {
		t.Fatalf("VerifyUnusedUserInviteCode returned error: %v", err)
	}
	if !unused {
		t.Errorf("freshly generated invite should be unused")
	}

	missing, err := VerifyUnusedUserInviteCode("NOTACODE")
	if err != nil {
		t.Fatalf("VerifyUnusedUserInviteCode(missing) returned error: %v", err)
	}
	if missing {
		t.Errorf("unknown code should not verify as unused")
	}
}

func TestSetUsedUserInviteCode(t *testing.T) {
	newTestDB(t)

	code, err := GenerateRandomInvite()
	if err != nil {
		t.Fatalf("GenerateRandomInvite returned error: %v", err)
	}
	claimer := uuid.New()

	if err := SetUsedUserInviteCode(code, claimer); err != nil {
		t.Fatalf("SetUsedUserInviteCode returned error: %v", err)
	}

	unused, err := VerifyUnusedUserInviteCode(code)
	if err != nil {
		t.Fatalf("VerifyUnusedUserInviteCode returned error: %v", err)
	}
	if unused {
		t.Errorf("used invite should no longer verify as unused")
	}

	// A code that does not exist cannot be marked used.
	if err := SetUsedUserInviteCode("NOTACODE", claimer); err == nil {
		t.Errorf("expected error marking unknown code as used")
	}
}

func TestGetAllEnabledInvitesAndGetInviteByID(t *testing.T) {
	newTestDB(t)

	code, err := GenerateRandomInvite()
	if err != nil {
		t.Fatalf("GenerateRandomInvite returned error: %v", err)
	}

	invites, err := GetAllEnabledInvites()
	if err != nil {
		t.Fatalf("GetAllEnabledInvites returned error: %v", err)
	}
	if len(invites) != 1 {
		t.Fatalf("got %d invites, want 1", len(invites))
	}
	if invites[0].Code != code {
		t.Errorf("invite code: got %q, want %q", invites[0].Code, code)
	}

	inviteID := invites[0].ID
	found, err := GetInviteByID(inviteID)
	if err != nil {
		t.Fatalf("GetInviteByID returned error: %v", err)
	}
	if found.ID != inviteID {
		t.Errorf("GetInviteByID returned wrong invite")
	}

	if _, err := GetInviteByID(uuid.New()); err == nil {
		t.Errorf("expected error for unknown invite id")
	}
}

func TestDeleteInviteByID(t *testing.T) {
	newTestDB(t)

	if _, err := GenerateRandomInvite(); err != nil {
		t.Fatalf("GenerateRandomInvite returned error: %v", err)
	}
	invites, err := GetAllEnabledInvites()
	if err != nil || len(invites) != 1 {
		t.Fatalf("failed to seed invite: err=%v len=%d", err, len(invites))
	}
	inviteID := invites[0].ID

	if err := DeleteInviteByID(inviteID); err != nil {
		t.Fatalf("DeleteInviteByID returned error: %v", err)
	}

	remaining, err := GetAllEnabledInvites()
	if err != nil {
		t.Fatalf("GetAllEnabledInvites returned error: %v", err)
	}
	if len(remaining) != 0 {
		t.Errorf("disabled invite should be excluded, got %d", len(remaining))
	}

	if err := DeleteInviteByID(uuid.New()); err == nil {
		t.Errorf("expected error deleting unknown invite")
	}
}
