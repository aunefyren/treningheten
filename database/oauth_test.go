package database

import (
	"testing"
	"time"

	"github.com/aunefyren/treningheten/models"

	"github.com/google/uuid"
)

// newRefreshToken builds a refresh token struct with the required non-null fields set.
func newRefreshToken(userID uuid.UUID, hash string) *models.OAuthRefreshToken {
	return &models.OAuthRefreshToken{
		TokenHash: hash,
		UserID:    userID,
		ClientID:  "test-client",
		Scope:     models.ScopeAPI,
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}
}

func TestCreateAndGetOAuthClient(t *testing.T) {
	newTestDB(t)

	client := &models.OAuthClient{
		ClientID:                "abc-client",
		ClientName:              "Test App",
		RedirectURIs:            "https://example.com/cb",
		GrantTypes:              "authorization_code",
		ResponseTypes:           "code",
		Scope:                   models.ScopeAPI,
		TokenEndpointAuthMethod: "none",
	}
	if err := CreateOAuthClient(client); err != nil {
		t.Fatalf("CreateOAuthClient returned error: %v", err)
	}
	if client.ID == uuid.Nil {
		t.Fatalf("CreateOAuthClient did not assign an ID")
	}

	found, err := GetOAuthClientByClientID("abc-client")
	if err != nil {
		t.Fatalf("GetOAuthClientByClientID returned error: %v", err)
	}
	if found.ClientName != "Test App" {
		t.Errorf("client name: got %q, want %q", found.ClientName, "Test App")
	}

	if _, err := GetOAuthClientByClientID("unknown"); err == nil {
		t.Errorf("expected error for unknown client id")
	}
}

func TestAuthorizationCodeLifecycle(t *testing.T) {
	newTestDB(t)

	user := makeTestUser(t, "authcode@example.com", nil)
	code := &models.OAuthAuthorizationCode{
		CodeHash:            "code-hash",
		ClientID:            "test-client",
		UserID:              user.ID,
		RedirectURI:         "https://example.com/cb",
		Scope:               models.ScopeAPI,
		CodeChallenge:       "challenge",
		CodeChallengeMethod: "S256",
		ExpiresAt:           time.Now().Add(10 * time.Minute),
	}
	if err := CreateAuthorizationCode(code); err != nil {
		t.Fatalf("CreateAuthorizationCode returned error: %v", err)
	}

	found, err := GetAuthorizationCodeByHash("code-hash")
	if err != nil {
		t.Fatalf("GetAuthorizationCodeByHash returned error: %v", err)
	}
	if found.UserID != user.ID {
		t.Errorf("auth code user: got %v, want %v", found.UserID, user.ID)
	}

	// First consume succeeds; second consume fails (single-use).
	if err := ConsumeAuthorizationCode(found.ID); err != nil {
		t.Fatalf("first ConsumeAuthorizationCode returned error: %v", err)
	}
	if err := ConsumeAuthorizationCode(found.ID); err == nil {
		t.Errorf("expected error consuming an already-used code")
	}

	if _, err := GetAuthorizationCodeByHash("missing"); err == nil {
		t.Errorf("expected error for unknown code hash")
	}
}

func TestCreateGetAndTouchRefreshToken(t *testing.T) {
	newTestDB(t)

	user := makeTestUser(t, "refresh@example.com", nil)
	token := newRefreshToken(user.ID, "refresh-hash")
	if err := CreateRefreshToken(token); err != nil {
		t.Fatalf("CreateRefreshToken returned error: %v", err)
	}

	found, err := GetRefreshTokenByHash("refresh-hash")
	if err != nil {
		t.Fatalf("GetRefreshTokenByHash returned error: %v", err)
	}
	if found.LastUsedAt != nil {
		t.Errorf("new token should have nil LastUsedAt")
	}

	if err := TouchRefreshToken(found.ID); err != nil {
		t.Fatalf("TouchRefreshToken returned error: %v", err)
	}
	reloaded, err := GetRefreshTokenByHash("refresh-hash")
	if err != nil {
		t.Fatalf("GetRefreshTokenByHash returned error: %v", err)
	}
	if reloaded.LastUsedAt == nil {
		t.Errorf("LastUsedAt should be set after touch")
	}

	if _, err := GetRefreshTokenByHash("gone"); err == nil {
		t.Errorf("expected error for unknown refresh token hash")
	}
}

func TestRotateRefreshToken(t *testing.T) {
	newTestDB(t)

	user := makeTestUser(t, "rotate@example.com", nil)
	old := newRefreshToken(user.ID, "old-hash")
	if err := CreateRefreshToken(old); err != nil {
		t.Fatalf("CreateRefreshToken returned error: %v", err)
	}

	replacement := newRefreshToken(user.ID, "new-hash")
	if err := RotateRefreshToken(old.ID, replacement); err != nil {
		t.Fatalf("RotateRefreshToken returned error: %v", err)
	}

	// Old token is revoked and points at the replacement.
	oldReloaded, err := GetRefreshTokenByHash("old-hash")
	if err != nil {
		t.Fatalf("GetRefreshTokenByHash(old) returned error: %v", err)
	}
	if oldReloaded.RevokedAt == nil {
		t.Errorf("rotated old token should be revoked")
	}
	if oldReloaded.RotatedTo == nil || *oldReloaded.RotatedTo != replacement.ID {
		t.Errorf("old token should point at replacement %v, got %v", replacement.ID, oldReloaded.RotatedTo)
	}

	// The replacement exists and is active.
	newReloaded, err := GetRefreshTokenByHash("new-hash")
	if err != nil {
		t.Fatalf("GetRefreshTokenByHash(new) returned error: %v", err)
	}
	if newReloaded.RevokedAt != nil {
		t.Errorf("replacement token should be active")
	}

	// Rotating the already-rotated token again fails.
	if err := RotateRefreshToken(old.ID, newRefreshToken(user.ID, "third-hash")); err == nil {
		t.Errorf("expected error rotating an already-rotated token")
	}
}

func TestRevokeRefreshToken(t *testing.T) {
	newTestDB(t)

	user := makeTestUser(t, "revoke@example.com", nil)
	token := newRefreshToken(user.ID, "revoke-hash")
	if err := CreateRefreshToken(token); err != nil {
		t.Fatalf("CreateRefreshToken returned error: %v", err)
	}

	if err := RevokeRefreshToken("revoke-hash"); err != nil {
		t.Fatalf("RevokeRefreshToken returned error: %v", err)
	}
	reloaded, err := GetRefreshTokenByHash("revoke-hash")
	if err != nil {
		t.Fatalf("GetRefreshTokenByHash returned error: %v", err)
	}
	if reloaded.RevokedAt == nil {
		t.Errorf("token should be revoked")
	}

	// Idempotent: revoking again (or an unknown hash) is not an error.
	if err := RevokeRefreshToken("revoke-hash"); err != nil {
		t.Errorf("second RevokeRefreshToken should be a no-op, got %v", err)
	}
	if err := RevokeRefreshToken("never-existed"); err != nil {
		t.Errorf("RevokeRefreshToken(unknown) should be a no-op, got %v", err)
	}
}

func TestRevokeRefreshTokenChain(t *testing.T) {
	newTestDB(t)

	user := makeTestUser(t, "chain@example.com", nil)

	// Build a rotation chain A → B → C. Rotation revokes A and B; C stays active.
	tokenA := newRefreshToken(user.ID, "chain-a")
	if err := CreateRefreshToken(tokenA); err != nil {
		t.Fatalf("CreateRefreshToken(A) returned error: %v", err)
	}
	tokenB := newRefreshToken(user.ID, "chain-b")
	if err := RotateRefreshToken(tokenA.ID, tokenB); err != nil {
		t.Fatalf("RotateRefreshToken(A→B) returned error: %v", err)
	}
	tokenC := newRefreshToken(user.ID, "chain-c")
	if err := RotateRefreshToken(tokenB.ID, tokenC); err != nil {
		t.Fatalf("RotateRefreshToken(B→C) returned error: %v", err)
	}

	// Revoking the chain from the head must revoke the still-active tail (C).
	if err := RevokeRefreshTokenChain(tokenA.ID); err != nil {
		t.Fatalf("RevokeRefreshTokenChain returned error: %v", err)
	}

	tail, err := GetRefreshTokenByHash("chain-c")
	if err != nil {
		t.Fatalf("GetRefreshTokenByHash(C) returned error: %v", err)
	}
	if tail.RevokedAt == nil {
		t.Errorf("tail token C should be revoked after chain revocation")
	}
}
