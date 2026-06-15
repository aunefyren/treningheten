package database

import (
	"testing"

	"github.com/aunefyren/treningheten/models"

	"github.com/google/uuid"
)

func TestRegisterAndGetUserRoundTrip(t *testing.T) {
	newTestDB(t)

	created := makeTestUser(t, "round@trip.test", nil)

	got, err := GetAllUserInformation(created.ID)
	if err != nil {
		t.Fatalf("GetAllUserInformation() error: %v", err)
	}
	if got.ID != created.ID {
		t.Errorf("ID = %v, want %v", got.ID, created.ID)
	}
	if got.Email != "round@trip.test" {
		t.Errorf("Email = %q, want %q", got.Email, "round@trip.test")
	}
	// GetAllUserInformation is the uncensored read; sensitive fields stay intact.
	if got.Password != "hashed-password" {
		t.Errorf("Password = %q, want it preserved", got.Password)
	}
}

func TestGetUserInformationCensors(t *testing.T) {
	newTestDB(t)

	created := makeTestUser(t, "censor@me.test", func(u *models.User) {
		code := "secret-verification"
		u.VerificationCode = &code
	})

	got, err := GetUserInformation(created.ID)
	if err != nil {
		t.Fatalf("GetUserInformation() error: %v", err)
	}
	if got.Password != "REDACTED" {
		t.Errorf("Password = %q, want REDACTED", got.Password)
	}
	if got.Email != "REDACTED" {
		t.Errorf("Email = %q, want REDACTED", got.Email)
	}
	if got.VerificationCode != nil {
		t.Errorf("VerificationCode = %v, want nil", got.VerificationCode)
	}
}

func TestGetUserInformationRequiresEnabled(t *testing.T) {
	newTestDB(t)

	created := makeTestUser(t, "disabled@me.test", func(u *models.User) {
		u.Enabled = false
	})

	if _, err := GetUserInformation(created.ID); err == nil {
		t.Error("expected error fetching a disabled user, got nil")
	}
}

func TestGetUserInformationMissing(t *testing.T) {
	newTestDB(t)

	if _, err := GetAllUserInformation(uuid.New()); err == nil {
		t.Error("expected error for non-existent user, got nil")
	}
}

func TestVerifyUniqueUserEmail(t *testing.T) {
	newTestDB(t)

	makeTestUser(t, "taken@unique.test", nil)

	tests := []struct {
		name       string
		email      string
		wantUnique bool
	}{
		{"unused email is unique", "free@unique.test", true},
		{"taken email is not unique", "taken@unique.test", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			unique, err := VerifyUniqueUserEmail(tt.email)
			if err != nil {
				t.Fatalf("VerifyUniqueUserEmail() error: %v", err)
			}
			if unique != tt.wantUnique {
				t.Errorf("VerifyUniqueUserEmail(%q) = %v, want %v", tt.email, unique, tt.wantUnique)
			}
		})
	}
}

func TestUpdateUserPersistsChanges(t *testing.T) {
	newTestDB(t)

	created := makeTestUser(t, "update@me.test", nil)

	created.FirstName = "Renamed"
	created.SundayAlert = true
	if _, err := UpdateUser(created); err != nil {
		t.Fatalf("UpdateUser() error: %v", err)
	}

	got, err := GetAllUserInformation(created.ID)
	if err != nil {
		t.Fatalf("GetAllUserInformation() error: %v", err)
	}
	if got.FirstName != "Renamed" {
		t.Errorf("FirstName = %q, want Renamed", got.FirstName)
	}
	if !got.SundayAlert {
		t.Error("SundayAlert = false, want true")
	}
}

// ClearStravaConnectionForUser must NULL strava_code and strava_id while leaving the
// rest of the row intact. This is the targeted-Updates path that the connect-flow bug
// fix relies on (a full Save of a stale struct is what was clobbering strava_code).
func TestClearStravaConnectionForUser(t *testing.T) {
	newTestDB(t)

	created := makeTestUser(t, "strava@me.test", func(u *models.User) {
		u.StravaCode = strPtr("r:encryptedtoken")
		u.StravaID = strPtr("123456")
	})

	if err := ClearStravaConnectionForUser(created.ID); err != nil {
		t.Fatalf("ClearStravaConnectionForUser() error: %v", err)
	}

	got, err := GetAllUserInformation(created.ID)
	if err != nil {
		t.Fatalf("GetAllUserInformation() error: %v", err)
	}
	if got.StravaCode != nil {
		t.Errorf("StravaCode = %v, want nil", *got.StravaCode)
	}
	if got.StravaID != nil {
		t.Errorf("StravaID = %v, want nil", *got.StravaID)
	}
	// Unrelated fields must be untouched.
	if got.Email != "strava@me.test" {
		t.Errorf("Email = %q, want preserved", got.Email)
	}
}

func TestGetStravaUsersOnlyReturnsConnected(t *testing.T) {
	newTestDB(t)

	connected := makeTestUser(t, "connected@strava.test", func(u *models.User) {
		u.StravaCode = strPtr("r:token")
	})
	makeTestUser(t, "unconnected@strava.test", nil)
	// Disabled users must be excluded even when they have a code.
	makeTestUser(t, "disabled@strava.test", func(u *models.User) {
		u.StravaCode = strPtr("r:token")
		u.Enabled = false
	})

	users, err := GetStravaUsers()
	if err != nil {
		t.Fatalf("GetStravaUsers() error: %v", err)
	}
	if len(users) != 1 {
		t.Fatalf("got %d strava users, want 1", len(users))
	}
	if users[0].ID != connected.ID {
		t.Errorf("returned user ID = %v, want %v", users[0].ID, connected.ID)
	}
}
