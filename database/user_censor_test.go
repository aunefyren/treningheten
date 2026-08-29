package database

import (
	"testing"

	"github.com/aunefyren/treningheten/models"
)

// The list getters censor in a loop. They used to range by value and assign to the copy,
// which left the returned slice fully uncensored — every user's bcrypt hash, e-mail, live
// password-reset code and Strava credential were served by GET /api/auth/users. These tests
// pin the behaviour of the censored getters and of their explicit uncensored siblings.

func seedSensitiveUser(t *testing.T, email string, sundayAlert bool) models.User {
	t.Helper()

	return makeTestUser(t, email, func(u *models.User) {
		reset := "RESETCODE123"
		verification := "VERIFYCODE123"
		strava := "r:strava-refresh-token"
		u.ResetCode = &reset
		u.VerificationCode = &verification
		u.StravaCode = &strava
		u.SundayAlert = sundayAlert
	})
}

func assertCensored(t *testing.T, users []models.User, context string) {
	t.Helper()

	if len(users) == 0 {
		t.Fatalf("%s returned no users", context)
	}
	for _, user := range users {
		if user.Password != "REDACTED" {
			t.Errorf("%s: Password = %q, want it redacted", context, user.Password)
		}
		if user.Email != "REDACTED" {
			t.Errorf("%s: Email = %q, want it redacted", context, user.Email)
		}
		if user.ResetCode != nil {
			t.Errorf("%s: ResetCode = %v, want nil", context, *user.ResetCode)
		}
		if user.VerificationCode != nil {
			t.Errorf("%s: VerificationCode = %v, want nil", context, *user.VerificationCode)
		}
		if user.StravaCode != nil {
			t.Errorf("%s: StravaCode = %v, want nil", context, *user.StravaCode)
		}
	}
}

func TestGetUsersInformationCensorsEveryRow(t *testing.T) {
	newTestDB(t)

	seedSensitiveUser(t, "one@censor.test", false)
	seedSensitiveUser(t, "two@censor.test", false)

	users, err := GetUsersInformation()
	if err != nil {
		t.Fatalf("GetUsersInformation() error: %v", err)
	}
	if len(users) != 2 {
		t.Fatalf("GetUsersInformation() returned %d users, want 2", len(users))
	}
	assertCensored(t, users, "GetUsersInformation")
}

func TestGetAllUsersWithSundayAlertsEnabledCensorsEveryRow(t *testing.T) {
	newTestDB(t)

	seedSensitiveUser(t, "alert@censor.test", true)
	seedSensitiveUser(t, "quiet@censor.test", false)

	users, err := GetAllUsersWithSundayAlertsEnabled()
	if err != nil {
		t.Fatalf("GetAllUsersWithSundayAlertsEnabled() error: %v", err)
	}
	if len(users) != 1 {
		t.Fatalf("returned %d users, want only the one with alerts enabled", len(users))
	}
	assertCensored(t, users, "GetAllUsersWithSundayAlertsEnabled")
}

func TestUncensoredGettersKeepCredentials(t *testing.T) {
	newTestDB(t)

	seedSensitiveUser(t, "job@censor.test", true)

	// The Strava sync selects on StravaCode, so its getter must not censor.
	all, err := GetAllUsersUncensored()
	if err != nil {
		t.Fatalf("GetAllUsersUncensored() error: %v", err)
	}
	if len(all) != 1 {
		t.Fatalf("GetAllUsersUncensored() returned %d users, want 1", len(all))
	}
	if all[0].StravaCode == nil || *all[0].StravaCode != "r:strava-refresh-token" {
		t.Errorf("GetAllUsersUncensored() dropped StravaCode: %v", all[0].StravaCode)
	}

	// The Sunday reminder is an e-mail, so its getter must not censor the address.
	alerts, err := GetAllUsersWithSundayAlertsEnabledUncensored()
	if err != nil {
		t.Fatalf("GetAllUsersWithSundayAlertsEnabledUncensored() error: %v", err)
	}
	if len(alerts) != 1 {
		t.Fatalf("returned %d users, want 1", len(alerts))
	}
	if alerts[0].Email != "job@censor.test" {
		t.Errorf("Email = %q, want it preserved", alerts[0].Email)
	}
}
