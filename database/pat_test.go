package database

import (
	"testing"
	"time"

	"github.com/aunefyren/treningheten/models"

	"github.com/google/uuid"
)

// makePAT persists a Personal Access Token for the given user and returns it.
func makePAT(t *testing.T, userID uuid.UUID, name, hash string) models.PersonalAccessToken {
	t.Helper()
	pat := &models.PersonalAccessToken{
		UserID:    userID,
		Name:      name,
		TokenHash: hash,
		Scope:     models.ScopeAPIRead,
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	if err := CreatePersonalAccessToken(pat); err != nil {
		t.Fatalf("failed to create PAT: %v", err)
	}
	if pat.ID == uuid.Nil {
		t.Fatalf("CreatePersonalAccessToken did not assign an ID")
	}
	return *pat
}

func TestCreateAndGetPersonalAccessTokenByHash(t *testing.T) {
	newTestDB(t)

	user := makeTestUser(t, "pat@example.com", nil)
	makePAT(t, user.ID, "laptop", "hash-abc")

	found, err := GetPersonalAccessTokenByHash("hash-abc")
	if err != nil {
		t.Fatalf("GetPersonalAccessTokenByHash returned error: %v", err)
	}
	if found.Name != "laptop" {
		t.Errorf("PAT name: got %q, want %q", found.Name, "laptop")
	}

	if _, err := GetPersonalAccessTokenByHash("nope"); err == nil {
		t.Errorf("expected error for unknown token hash")
	}
}

func TestGetPersonalAccessTokensByUser(t *testing.T) {
	newTestDB(t)

	user := makeTestUser(t, "patlist@example.com", nil)
	other := makeTestUser(t, "patother@example.com", nil)

	makePAT(t, user.ID, "one", "hash-1")
	revoked := makePAT(t, user.ID, "two", "hash-2")
	makePAT(t, other.ID, "elsewhere", "hash-3")

	// Revoke one of the user's tokens; it must drop out of the listing.
	if err := RevokePersonalAccessToken(revoked.ID, user.ID); err != nil {
		t.Fatalf("RevokePersonalAccessToken returned error: %v", err)
	}

	pats, err := GetPersonalAccessTokensByUser(user.ID)
	if err != nil {
		t.Fatalf("GetPersonalAccessTokensByUser returned error: %v", err)
	}
	if len(pats) != 1 {
		t.Fatalf("got %d active PATs, want 1", len(pats))
	}
	if pats[0].Name != "one" {
		t.Errorf("remaining PAT: got %q, want %q", pats[0].Name, "one")
	}
}

func TestRevokePersonalAccessTokenScoping(t *testing.T) {
	newTestDB(t)

	owner := makeTestUser(t, "patowner@example.com", nil)
	attacker := makeTestUser(t, "patattacker@example.com", nil)
	pat := makePAT(t, owner.ID, "victim", "hash-victim")

	// Another user cannot revoke a token they do not own.
	if err := RevokePersonalAccessToken(pat.ID, attacker.ID); err == nil {
		t.Errorf("expected error revoking another user's token")
	}

	// The owner can revoke it.
	if err := RevokePersonalAccessToken(pat.ID, owner.ID); err != nil {
		t.Fatalf("RevokePersonalAccessToken returned error: %v", err)
	}

	// Revoking again is a no-op that reports not found.
	if err := RevokePersonalAccessToken(pat.ID, owner.ID); err == nil {
		t.Errorf("expected error revoking an already-revoked token")
	}
}

func TestTouchPersonalAccessToken(t *testing.T) {
	newTestDB(t)

	user := makeTestUser(t, "pattouch@example.com", nil)
	pat := makePAT(t, user.ID, "touch", "hash-touch")

	if pat.LastUsedAt != nil {
		t.Fatalf("new PAT should have nil LastUsedAt")
	}

	if err := TouchPersonalAccessToken(pat.ID); err != nil {
		t.Fatalf("TouchPersonalAccessToken returned error: %v", err)
	}

	reloaded, err := GetPersonalAccessTokenByHash("hash-touch")
	if err != nil {
		t.Fatalf("GetPersonalAccessTokenByHash returned error: %v", err)
	}
	if reloaded.LastUsedAt == nil {
		t.Errorf("LastUsedAt should be set after touch")
	}
}
