package models

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

// Credentials and recovery codes must never reach a response body. CensorUserObject is the
// first line of defence; these `json:"-"` tags are the second, so that a getter that forgets
// to censor cannot leak them the way GET /api/auth/users once did.
func TestUserJSONOmitsCredentials(t *testing.T) {
	now := time.Now()
	reset := "RESETCODE123"
	verification := "VERIFYCODE123"
	strava := "r:strava-refresh-token"
	hevy := "hevy-api-key"

	user := User{
		FirstName:                  "Test",
		LastName:                   "User",
		Email:                      "test@example.com",
		Password:                   "$2a$14$abcdefghijklmnopqrstuv",
		ResetCode:                  &reset,
		ResetExpiration:            &now,
		VerificationCode:           &verification,
		VerificationCodeExpiration: &now,
		StravaCode:                 &strava,
		HevyAPIKey:                 &hevy,
	}

	encoded, err := json.Marshal(user)
	if err != nil {
		t.Fatalf("json.Marshal() error: %v", err)
	}
	payload := string(encoded)

	for _, secret := range []string{
		"$2a$14$abcdefghijklmnopqrstuv",
		reset,
		verification,
		strava,
		hevy,
	} {
		if strings.Contains(payload, secret) {
			t.Errorf("serialized User contains secret %q: %s", secret, payload)
		}
	}

	for _, field := range []string{
		`"password"`,
		`"reset_code"`,
		`"reset_expiration"`,
		`"verification_code"`,
		`"verification_code_expiration"`,
		`"strava_code"`,
	} {
		if strings.Contains(payload, field) {
			t.Errorf("serialized User still exposes field %s: %s", field, payload)
		}
	}
}

// Connection state is still reportable — just as a derived boolean, not the credential.
func TestUserJSONKeepsDerivedConnectionFlags(t *testing.T) {
	user := User{HevyConnected: true, StravaConnected: true}

	encoded, err := json.Marshal(user)
	if err != nil {
		t.Fatalf("json.Marshal() error: %v", err)
	}
	payload := string(encoded)

	for _, field := range []string{`"hevy_connected":true`, `"strava_connected":true`} {
		if !strings.Contains(payload, field) {
			t.Errorf("serialized User missing %s: %s", field, payload)
		}
	}
}
