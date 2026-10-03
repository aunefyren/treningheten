package controllers

import (
	"net/http"
	"testing"
	"time"

	"github.com/aunefyren/treningheten/database"
	"github.com/aunefyren/treningheten/files"
	"github.com/aunefyren/treningheten/models"
)

func TestFirstRegisteredUserBecomesAdmin(t *testing.T) {
	h := newAPIHarness(t)
	code, err := database.GenerateRandomInvite()
	if err != nil {
		t.Fatal(err)
	}
	h.expect(http.StatusCreated, "POST", "/api/open/users", "", models.UserCreationRequest{
		FirstName: "First", LastName: "Ever", Email: "first@edges.test", Password: "Password123", PasswordRepeat: "Password123", InviteCode: code,
	})
	first, _ := database.GetAllUserInformationByEmail("first@edges.test")
	if first.Admin == nil || !*first.Admin {
		t.Error("the first user on a fresh install is not an admin")
	}

	// A second registration with the same address is refused.
	second, _ := database.GenerateRandomInvite()
	h.expect(http.StatusBadRequest, "POST", "/api/open/users", "", models.UserCreationRequest{
		FirstName: "Copy", LastName: "Cat", Email: "first@edges.test", Password: "Password123", PasswordRepeat: "Password123", InviteCode: second,
	})
}

func TestAccountEdgeCases(t *testing.T) {
	h := newAPIHarness(t)
	server := withControllerSMTP(t)
	files.ConfigFile.TreninghetenExternalURL = "https://trening.test"
	_, adminToken := h.user("admin@edges.test", true)

	// Registration still succeeds if the verification mail can't be sent? It reports it.
	invite := field(t, h.expect(http.StatusCreated, "POST", "/api/admin/invites", adminToken, nil), "invitation").(string)
	files.ConfigFile.SMTPPort = 1
	if code := h.do("POST", "/api/open/users", "", models.UserCreationRequest{
		FirstName: "No", LastName: "Mail", Email: "nomail@edges.test", Password: "Password123", PasswordRepeat: "Password123", InviteCode: invite,
	}).Code; code < 400 {
		t.Errorf("registration with the mail server down: status = %d, want an error", code)
	}
	files.ConfigFile.SMTPPort = server.Port

	// An expired verification code is refused.
	user, token := h.user("verify@edges.test", false)
	database.Instance.Model(&models.User{}).Where("id = ?", user.ID).Update("verified", false)
	verification, err := database.GenerateRandomVerificationCodeForUser(user.ID)
	if err != nil {
		t.Fatal(err)
	}
	database.Instance.Model(&models.User{}).Where("id = ?", user.ID).Update("verification_code_expiration", time.Now().Add(-time.Hour))
	h.expect(http.StatusBadRequest, "POST", "/api/open/users/verify/"+verification, token, nil)

	// Password reset: no external URL, a malformed body, the mail server down, expired codes.
	files.ConfigFile.TreninghetenExternalURL = ""
	h.expect(http.StatusBadRequest, "POST", "/api/open/users/reset", "", map[string]string{"email": "verify@edges.test"})
	files.ConfigFile.TreninghetenExternalURL = "https://trening.test"
	h.expect(http.StatusBadRequest, "POST", "/api/open/users/reset", "", "{broken")
	files.ConfigFile.SMTPPort = 1
	if code := h.do("POST", "/api/open/users/reset", "", map[string]string{"email": "verify@edges.test"}).Code; code < 400 {
		t.Errorf("reset with the mail server down: status = %d, want an error", code)
	}
	files.ConfigFile.SMTPPort = server.Port

	resetCode, err := database.GenerateRandomResetCodeForUser(user.ID, true)
	if err != nil {
		t.Fatal(err)
	}
	database.Instance.Model(&models.User{}).Where("id = ?", user.ID).Update("reset_expiration", time.Now().Add(-time.Hour))
	if expired := h.ok("GET", "/api/open/users/reset/"+resetCode, "", nil); expired["expired"] != true {
		t.Errorf("expired reset code reported = %v", expired)
	}
	h.expect(http.StatusBadRequest, "POST", "/api/open/users/password", "", models.UserUpdatePasswordRequest{ResetCode: resetCode, Password: "Password456", PasswordRepeat: "Password456"})
	h.expect(http.StatusBadRequest, "POST", "/api/open/users/password", "", "{broken")

	// Wheel customisation must be a hex colour and a single emoji.
	userPath := "/api/auth/users/" + user.ID.String()
	database.Instance.Model(&models.User{}).Where("id = ?", user.ID).Update("verified", true)
	h.expect(http.StatusBadRequest, "PATCH", userPath, token, models.UserPartialUpdateRequest{WheelBorderColor: stringPtr("blue-ish")})
	h.expect(http.StatusBadRequest, "PATCH", userPath, token, models.UserPartialUpdateRequest{WheelEmoji: stringPtr("not an emoji")})
	h.expect(http.StatusBadRequest, "PATCH", userPath, token, "{broken")

	// Sharing off hides activities and statistics from others.
	database.Instance.Model(&models.User{}).Where("id = ?", user.ID).Updates(map[string]any{"share_activities": false, "share_statistics": false})
	h.expect(http.StatusForbidden, "GET", userPath+"/activities", adminToken, nil)
	h.expect(http.StatusForbidden, "GET", userPath+"/statistics", adminToken, nil)

	// No seasons: the Sunday job has nothing to do.
	SendSundayReminders()
}

func TestAccountUpdateEdgeCases(t *testing.T) {
	h := newAPIHarness(t)
	server := withControllerSMTP(t)
	_, adminToken := h.user("admin@update.test", true)
	invite := field(t, h.expect(http.StatusCreated, "POST", "/api/admin/invites", adminToken, nil), "invitation").(string)
	h.expect(http.StatusCreated, "POST", "/api/open/users", "", models.UserCreationRequest{
		FirstName: "Up", LastName: "Date", Email: "me@update.test", Password: "Password123", PasswordRepeat: "Password123", InviteCode: invite,
	})
	user, _ := database.GetAllUserInformationByEmail("me@update.test")
	database.Instance.Model(&models.User{}).Where("id = ?", user.ID).Update("verified", true)
	_, login := passwordLogin(h, "me@update.test", "Password123")
	token := login["access_token"].(string)
	userPath := "/api/auth/users/" + user.ID.String()

	maxHR, restHR := 150, 150
	h.expect(http.StatusBadRequest, "POST", userPath, token, models.UserUpdateRequest{Email: "me@update.test", OldPassword: "Password123", MaxHeartrate: &maxHR, RestingHeartrate: &restHR})
	// A bad image is the client's mistake: 400, with the reason.
	h.expect(http.StatusBadRequest, "POST", userPath, token, models.UserUpdateRequest{Email: "me@update.test", OldPassword: "Password123", ProfileImage: "data:image/jpeg;base64,bm90IGFuIGltYWdl"})

	// Characterizes current behaviour: changing the address keeps the account verified
	// and sends no verification mail (the re-verify branch only runs for accounts that
	// are already unverified, which the auth middleware keeps out). See docs/wip.md.
	before := len(server.Messages())
	updated := h.ok("POST", userPath, token, models.UserUpdateRequest{Email: "new@update.test", OldPassword: "Password123"})
	if updated["verified"] != true || len(server.Messages()) != before {
		t.Errorf("e-mail change: verified = %v, mails sent = %d", updated["verified"], len(server.Messages())-before)
	}
	if changed, _ := database.GetAllUserInformation(user.ID); changed.Email != "new@update.test" {
		t.Errorf("email = %q, want the new address", changed.Email)
	}
}
