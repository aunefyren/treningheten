package controllers

import (
	"net/http"
	"testing"
	"time"

	"github.com/aunefyren/treningheten/auth"
	"github.com/aunefyren/treningheten/database"
	"github.com/aunefyren/treningheten/files"
	"github.com/aunefyren/treningheten/models"
	"github.com/google/uuid"
)

func TestValidateUserUpdate(t *testing.T) {
	now := time.Date(2026, 10, 3, 12, 0, 0, 0, time.UTC)
	adult := now.AddDate(-30, 0, 0)
	child := now.AddDate(-5, 0, 0)
	maxHR, restHR, tooHigh, tooLow := 190, 50, 300, 10

	tests := []struct {
		name    string
		request models.UserUpdateRequest
		wantOK  bool
	}{
		{"empty request", models.UserUpdateRequest{}, true},
		{"strong password", models.UserUpdateRequest{Password: "Password123", PasswordRepeat: "Password123"}, true},
		{"password mismatch", models.UserUpdateRequest{Password: "Password123", PasswordRepeat: "Password124"}, false},
		{"weak password", models.UserUpdateRequest{Password: "weak", PasswordRepeat: "weak"}, false},
		{"adult birth date", models.UserUpdateRequest{BirthDate: &adult}, true},
		{"too young", models.UserUpdateRequest{BirthDate: &child}, false},
		{"valid heart rates", models.UserUpdateRequest{MaxHeartrate: &maxHR, RestingHeartrate: &restHR}, true},
		{"max HR too high", models.UserUpdateRequest{MaxHeartrate: &tooHigh}, false},
		{"resting HR too low", models.UserUpdateRequest{RestingHeartrate: &tooLow}, false},
		{"resting above max", models.UserUpdateRequest{MaxHeartrate: intPtr(110), RestingHeartrate: intPtr(115)}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			message, err := validateUserUpdate(tt.request, now)
			if err != nil {
				t.Fatalf("validateUserUpdate() error: %v", err)
			}
			if gotOK := message == ""; gotOK != tt.wantOK {
				t.Errorf("validateUserUpdate() message = %q, want valid = %v", message, tt.wantOK)
			}
		})
	}
}

// userWithPassword creates a verified user with a real password hash, so the account
// update endpoint (which demands the current password) can be exercised.
func userWithPassword(h *apiHarness, email, password string) (models.User, string) {
	h.t.Helper()

	admin := false
	user := models.User{FirstName: "Mail", LastName: "Changer", Email: email, Enabled: true, Verified: true, Admin: &admin}
	user.ID = uuid.New()
	if err := user.HashPassword(password); err != nil {
		h.t.Fatalf("failed to hash password: %v", err)
	}
	created, err := database.RegisterUserInDB(user)
	if err != nil {
		h.t.Fatalf("failed to create user %s: %v", email, err)
	}
	token, _, err := auth.GenerateAccessToken(created.ID, false, auth.ScopeForUser(false), models.FirstPartyClientID)
	if err != nil {
		h.t.Fatalf("failed to mint access token: %v", err)
	}
	return created, token
}

func TestUpdateUserEmailChange(t *testing.T) {
	tests := []struct {
		name         string
		smtp         bool
		newEmail     string
		wantVerified bool
		wantMail     bool
	}{
		{"same e-mail with SMTP on stays verified", true, "mail@change.test", true, false},
		{"changed e-mail with SMTP on is re-verified", true, "new@change.test", false, true},
		{"changed e-mail with SMTP off stays verified", false, "new@change.test", true, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newAPIHarness(t)
			user, token := userWithPassword(h, "mail@change.test", "Password123")
			if !tt.smtp {
				previous := files.ConfigFile
				files.ConfigFile.SMTPEnabled = false
				t.Cleanup(func() { files.ConfigFile = previous })
			}
			var mailCount func() int
			if tt.smtp {
				server := withControllerSMTP(t)
				mailCount = func() int { return len(server.Messages()) }
			}

			response := h.ok("POST", "/api/auth/users/"+user.ID.String(), token, models.UserUpdateRequest{Email: tt.newEmail, OldPassword: "Password123"})
			if verified := field(t, response, "verified"); verified != tt.wantVerified {
				t.Errorf("response verified = %v, want %v", verified, tt.wantVerified)
			}

			stored, err := database.GetAllUserInformation(user.ID)
			if err != nil {
				t.Fatalf("GetAllUserInformation() error: %v", err)
			}
			if stored.Email != tt.newEmail {
				t.Errorf("stored e-mail = %q, want %q", stored.Email, tt.newEmail)
			}
			if stored.Verified != tt.wantVerified {
				t.Errorf("stored verified = %v, want %v", stored.Verified, tt.wantVerified)
			}
			if tt.smtp {
				if sent := mailCount() > 0; sent != tt.wantMail {
					t.Errorf("verification mail sent = %v, want %v", sent, tt.wantMail)
				}
				if tt.wantMail && stored.VerificationCode == nil {
					t.Error("re-verified account has no verification code")
				}
			}
		})
	}

	t.Run("e-mail in use is rejected without changes", func(t *testing.T) {
		h := newAPIHarness(t)
		h.user("taken@change.test", false)
		user, token := userWithPassword(h, "mail@change.test", "Password123")
		withControllerSMTP(t)

		h.expect(http.StatusBadRequest, "POST", "/api/auth/users/"+user.ID.String(), token, models.UserUpdateRequest{Email: "taken@change.test", OldPassword: "Password123"})
		stored, err := database.GetAllUserInformation(user.ID)
		if err != nil {
			t.Fatalf("GetAllUserInformation() error: %v", err)
		}
		if stored.Email != "mail@change.test" || !stored.Verified {
			t.Errorf("rejected update changed the account: email = %q, verified = %v", stored.Email, stored.Verified)
		}
	})
}
