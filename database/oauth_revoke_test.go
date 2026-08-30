package database

import (
	"testing"
	"time"

	"github.com/aunefyren/treningheten/models"

	"github.com/google/uuid"
)

// insertRefreshToken writes a live refresh token for a user and returns its hash.
func insertRefreshToken(t *testing.T, userID uuid.UUID, hash string) string {
	t.Helper()

	token := models.OAuthRefreshToken{
		TokenHash: hash,
		UserID:    userID,
		ClientID:  models.FirstPartyClientID,
		Scope:     models.ScopeAPI,
		ExpiresAt: time.Now().Add(30 * 24 * time.Hour),
	}
	if err := CreateRefreshToken(&token); err != nil {
		t.Fatalf("failed to create refresh token: %v", err)
	}
	return hash
}

// A password change must end every session the user has, not just one — that is the
// whole point of changing a password after a compromise.
func TestRevokeAllRefreshTokensForUser(t *testing.T) {
	newTestDB(t)

	victim := makeTestUser(t, "victim@example.com", nil)
	other := makeTestUser(t, "other@example.com", nil)

	insertRefreshToken(t, victim.ID, "victim-session-one")
	insertRefreshToken(t, victim.ID, "victim-session-two")
	insertRefreshToken(t, other.ID, "other-session")

	if err := RevokeAllRefreshTokensForUser(victim.ID); err != nil {
		t.Fatalf("failed to revoke refresh tokens: %v", err)
	}

	for _, hash := range []string{"victim-session-one", "victim-session-two"} {
		token, err := GetRefreshTokenByHash(hash)
		if err != nil {
			t.Fatalf("failed to read back %q: %v", hash, err)
		}
		if token.RevokedAt == nil {
			t.Fatalf("%q should have been revoked", hash)
		}
	}

	// Another user's sessions are untouched.
	untouched, err := GetRefreshTokenByHash("other-session")
	if err != nil {
		t.Fatalf("failed to read back the other user's token: %v", err)
	}
	if untouched.RevokedAt != nil {
		t.Fatal("revocation must be scoped to one user")
	}
}

// Revoking twice must not error or move an existing revocation timestamp.
func TestRevokeAllRefreshTokensForUserIsIdempotent(t *testing.T) {
	newTestDB(t)

	user := makeTestUser(t, "idempotent@example.com", nil)
	insertRefreshToken(t, user.ID, "session")

	if err := RevokeAllRefreshTokensForUser(user.ID); err != nil {
		t.Fatalf("first revoke failed: %v", err)
	}
	first, err := GetRefreshTokenByHash("session")
	if err != nil {
		t.Fatalf("failed to read back token: %v", err)
	}

	if err := RevokeAllRefreshTokensForUser(user.ID); err != nil {
		t.Fatalf("second revoke failed: %v", err)
	}
	second, err := GetRefreshTokenByHash("session")
	if err != nil {
		t.Fatalf("failed to read back token: %v", err)
	}

	if !first.RevokedAt.Equal(*second.RevokedAt) {
		t.Fatal("a second revoke should not move the revocation timestamp")
	}
}

// A user with no sessions is not an error case.
func TestRevokeAllRefreshTokensForUserWithNoTokens(t *testing.T) {
	newTestDB(t)

	user := makeTestUser(t, "sessionless@example.com", nil)
	if err := RevokeAllRefreshTokensForUser(user.ID); err != nil {
		t.Fatalf("revoking nothing should succeed, got: %v", err)
	}
}
