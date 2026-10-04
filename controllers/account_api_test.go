package controllers

import (
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/aunefyren/treningheten/files"
	"github.com/aunefyren/treningheten/internal/smtptest"
	"github.com/aunefyren/treningheten/models"
)

// withControllerSMTP turns SMTP on and points it at a fresh in-process server.
func withControllerSMTP(t *testing.T) *smtptest.Server {
	t.Helper()

	server := smtptest.Start(t)
	previous := files.ConfigFile
	files.ConfigFile.SMTPEnabled = true
	files.ConfigFile.SMTPHost = server.Host
	files.ConfigFile.SMTPPort = server.Port
	files.ConfigFile.SMTPUsername = ""
	files.ConfigFile.SMTPFrom = "noreply@test.local"
	files.ConfigFile.TreninghetenExternalURL = ""
	files.ConfigFile.TreninghetenEnvironment = "production"
	t.Cleanup(func() { files.ConfigFile = previous })
	return server
}

// lastMailMatching returns the first capture group of pattern in the newest mail, with
// quoted-printable soft breaks undone.
func lastMailMatching(t *testing.T, server *smtptest.Server, pattern string) string {
	t.Helper()

	messages := server.Messages()
	if len(messages) == 0 {
		t.Fatal("no mail was sent")
	}
	data := strings.ReplaceAll(messages[len(messages)-1].Data, "=\r\n", "")
	data = strings.ReplaceAll(data, "=3D", "=")
	match := regexp.MustCompile(pattern).FindStringSubmatch(data)
	if match == nil {
		t.Fatalf("no %q in mail:\n%s", pattern, data)
	}
	return match[1]
}

// passwordLogin runs the OAuth password grant and returns the decoded response.
func passwordLogin(h *apiHarness, email, password string) (int, map[string]any) {
	h.t.Helper()

	form := url.Values{"grant_type": {"password"}, "client_id": {models.FirstPartyClientID}, "username": {email}, "password": {password}}
	request := h.doForm("/api/oauth/token", form)
	return request.Code, decodeBody(h.t, request.Body.Bytes())
}

func TestRegistrationVerificationAndPasswordReset(t *testing.T) {
	h := newAPIHarness(t)
	server := withControllerSMTP(t)
	files.ConfigFile.TreninghetenExternalURL = "https://trening.test"
	_, adminToken := h.user("admin@account.test", true)

	invite := h.expect(http.StatusCreated, "POST", "/api/admin/invites", adminToken, nil)
	code, _ := field(t, invite, "invitation").(string)
	if code == "" {
		t.Fatalf("no invite code in %v", invite)
	}
	h.ok("GET", "/api/admin/invites", adminToken, nil)

	registration := models.UserCreationRequest{
		FirstName: "New", LastName: "Member", Email: "new@account.test",
		Password: "Password123", PasswordRepeat: "Password123", InviteCode: code,
	}

	weak := registration
	weak.Password, weak.PasswordRepeat = "short", "short"
	h.expect(http.StatusBadRequest, "POST", "/api/open/users", "", weak)
	mismatch := registration
	mismatch.PasswordRepeat = "Password124"
	h.expect(http.StatusBadRequest, "POST", "/api/open/users", "", mismatch)
	badInvite := registration
	badInvite.InviteCode = "NOPE"
	h.expect(http.StatusBadRequest, "POST", "/api/open/users", "", badInvite)

	h.expect(http.StatusCreated, "POST", "/api/open/users", "", registration)
	verificationCode := lastMailMatching(t, server, `this code: <b>([^<]+)</b>`)

	// The invite is spent.
	again := registration
	again.Email = "second@account.test"
	h.expect(http.StatusBadRequest, "POST", "/api/open/users", "", again)

	status, login := passwordLogin(h, "new@account.test", "Password123")
	if status != http.StatusOK {
		t.Fatalf("login: status = %d, body %v", status, login)
	}
	token, _ := login["access_token"].(string)

	// Unverified users are held at the auth gate but can verify and resend.
	h.expect(http.StatusForbidden, "GET", "/api/auth/news", token, nil)
	h.ok("POST", "/api/open/users/verification", token, nil)
	verificationCode = lastMailMatching(t, server, `this code: <b>([^<]+)</b>`)
	h.expect(http.StatusBadRequest, "POST", "/api/open/users/verify/WRONG", token, nil)
	h.ok("POST", "/api/open/users/verify/"+verificationCode, token, nil)
	h.expect(http.StatusCreated, "GET", "/api/auth/news", token, nil)

	// Wrong password and unknown user both fail the grant without saying which.
	if status, _ := passwordLogin(h, "new@account.test", "Wrong123"); status == http.StatusOK {
		t.Error("login with the wrong password succeeded")
	}

	// Password reset by mail link.
	h.ok("POST", "/api/open/users/reset", "", map[string]string{"email": "nobody@account.test"})
	h.ok("POST", "/api/open/users/reset", "", map[string]string{"email": "new@account.test"})
	resetCode := lastMailMatching(t, server, `reset_code=([A-Za-z0-9]+)`)

	verified := h.ok("GET", "/api/open/users/reset/"+resetCode, "", nil)
	if verified["expired"] != false {
		t.Errorf("fresh reset code reported expired: %v", verified)
	}
	if expired := h.ok("GET", "/api/open/users/reset/UNKNOWN", "", nil); expired["expired"] != true {
		t.Errorf("unknown reset code not reported expired: %v", expired)
	}

	h.expect(http.StatusBadRequest, "POST", "/api/open/users/password", "", models.UserUpdatePasswordRequest{ResetCode: resetCode, Password: "Password456", PasswordRepeat: "Password457"})
	h.expect(http.StatusBadRequest, "POST", "/api/open/users/password", "", models.UserUpdatePasswordRequest{ResetCode: "UNKNOWN", Password: "Password456", PasswordRepeat: "Password456"})
	h.ok("POST", "/api/open/users/password", "", models.UserUpdatePasswordRequest{ResetCode: resetCode, Password: "Password456", PasswordRepeat: "Password456"})

	if status, _ := passwordLogin(h, "new@account.test", "Password123"); status == http.StatusOK {
		t.Error("the old password still works after a reset")
	}
	if status, body := passwordLogin(h, "new@account.test", "Password456"); status != http.StatusOK {
		t.Errorf("the new password doesn't work: %d %v", status, body)
	}
}

func TestResetRefusedWithoutSMTP(t *testing.T) {
	h := newAPIHarness(t)
	h.expect(http.StatusBadRequest, "POST", "/api/open/users/reset", "", map[string]string{"email": "a@b.test"})
}

func TestAccountSettings(t *testing.T) {
	h := newAPIHarness(t)
	_, adminToken := h.user("admin@settings.test", true)

	// A real password is needed for UpdateUser, so register through the API.
	invite := h.expect(http.StatusCreated, "POST", "/api/admin/invites", adminToken, nil)
	code := field(t, invite, "invitation").(string)
	h.expect(http.StatusCreated, "POST", "/api/open/users", "", models.UserCreationRequest{
		FirstName: "Set", LastName: "Tings", Email: "me@settings.test", Password: "Password123", PasswordRepeat: "Password123", InviteCode: code,
	})
	_, login := passwordLogin(h, "me@settings.test", "Password123")
	token := login["access_token"].(string)

	me := h.ok("POST", "/api/auth/tokens/validate", token, nil)
	_ = me
	users := h.ok("GET", "/api/auth/users", token, nil)
	var myID string
	for _, user := range field(t, users, "users").([]any) {
		if field(t, user, "first_name") == "Set" {
			myID = idOf(t, user)
		}
	}
	if myID == "" {
		t.Fatal("registered user not in the user list")
	}

	self := h.ok("GET", "/api/auth/users/"+myID, token, nil)
	if email := field(t, self, "user", "email"); email != "me@settings.test" {
		t.Errorf("own profile email = %v", email)
	}
	// Someone else only sees the public shape (no e-mail).
	_, otherToken := h.user("other@settings.test", false)
	public := h.ok("GET", "/api/auth/users/"+myID, otherToken, nil)
	if _, leaked := field(t, public, "user").(map[string]any)["email"]; leaked {
		t.Error("another user's e-mail is exposed on the public profile")
	}
	h.expect(http.StatusBadRequest, "GET", "/api/auth/users/nope", token, nil)

	userPath := "/api/auth/users/" + myID

	h.expect(http.StatusUnauthorized, "POST", userPath, token, models.UserUpdateRequest{Email: "me@settings.test", OldPassword: "Wrong123"})
	for name, request := range map[string]models.UserUpdateRequest{
		"password mismatch": {Email: "me@settings.test", OldPassword: "Password123", Password: "Password999", PasswordRepeat: "Password998"},
		"weak password":     {Email: "me@settings.test", OldPassword: "Password123", Password: "weak", PasswordRepeat: "weak"},
		"e-mail in use":     {Email: "admin@settings.test", OldPassword: "Password123"},
	} {
		if code := h.do("POST", userPath, token, request).Code; code != http.StatusBadRequest {
			t.Errorf("%s: status = %d, want 400", name, code)
		}
	}

	updated := h.ok("POST", userPath, token, models.UserUpdateRequest{
		Email: "me@settings.test", OldPassword: "Password123", ShareActivities: boolPtr(true), ShareStatistics: boolPtr(false),
	})
	if _, found := updated["data"]; !found {
		t.Errorf("update did not return a fresh token set: %v", updated)
	}

	h.ok("PATCH", userPath, token, models.UserPartialUpdateRequest{
		SundayAlert: boolPtr(true), StravaPublic: boolPtr(true), HevyPublic: boolPtr(false),
		WheelColor: stringPtr("#112233"), WheelBorderColor: stringPtr("#445566"), WheelEmoji: stringPtr("🔥"),
	})
	h.expect(http.StatusBadRequest, "PATCH", userPath, token, models.UserPartialUpdateRequest{WheelColor: stringPtr("red; drop table")})
	// The path id is ignored: a PATCH always updates the caller, so another user's
	// request can't touch this account.
	h.ok("PATCH", userPath, otherToken, models.UserPartialUpdateRequest{SundayAlert: boolPtr(false)})
	if alert := field(t, h.ok("GET", userPath, token, nil), "user", "sunday_alert"); alert != true {
		t.Errorf("sunday_alert = %v after another user's PATCH, want it untouched (true)", alert)
	}
}

func TestAdminContentAndReads(t *testing.T) {
	h := newAPIHarness(t)
	_, adminToken := h.user("admin@content.test", true)
	user, token := h.user("reader@content.test", false)

	// News: admin-only writes, readable by everyone.
	h.expect(http.StatusForbidden, "POST", "/api/admin/news", token, models.NewsCreationRequest{Title: "x", Body: "y", Date: time.Now()})
	h.expect(http.StatusBadRequest, "POST", "/api/admin/news", adminToken, models.NewsCreationRequest{Title: "", Body: "y", Date: time.Now()})
	created := h.expect(http.StatusCreated, "POST", "/api/admin/news", adminToken, models.NewsCreationRequest{Title: "Hello", Body: "World", Date: time.Now()})
	newsID := idOf(t, field(t, created, "news").([]any)[0])
	h.expect(http.StatusCreated, "GET", "/api/auth/news", token, nil)
	h.expect(http.StatusCreated, "GET", "/api/auth/news/"+newsID, token, nil)
	h.expect(http.StatusBadRequest, "GET", "/api/auth/news/nope", token, nil)
	h.expect(http.StatusOK, "DELETE", "/api/admin/news/"+newsID, adminToken, nil)
	h.expect(http.StatusNotFound, "DELETE", "/api/admin/news/"+newsID, adminToken, nil)
	h.expect(http.StatusNotFound, "GET", "/api/auth/news/"+newsID, token, nil)
	h.expect(http.StatusBadRequest, "DELETE", "/api/admin/news/nope", adminToken, nil)
	h.expect(http.StatusForbidden, "DELETE", "/api/admin/news/"+newsID, token, nil)

	// Invites: create, list, delete.
	invite := h.expect(http.StatusCreated, "POST", "/api/admin/invites", adminToken, nil)
	inviteCode := field(t, invite, "invitation").(string)
	var inviteID string
	for _, listed := range field(t, invite, "invites").([]any) {
		if field(t, listed, "code") == inviteCode {
			inviteID = idOf(t, listed)
		}
	}
	if inviteID == "" {
		t.Fatalf("new invite %s not in the returned list", inviteCode)
	}
	h.expect(http.StatusOK, "DELETE", "/api/admin/invites/"+inviteID, adminToken, nil)
	h.expect(http.StatusNotFound, "DELETE", "/api/admin/invites/"+inviteID, adminToken, nil)
	h.expect(http.StatusBadRequest, "DELETE", "/api/admin/invites/nope", adminToken, nil)

	// Admin reads.
	server := h.ok("GET", "/api/admin/server-info", adminToken, nil)
	if _, found := field(t, server, "server").(map[string]any); !found {
		t.Errorf("server-info has no server object: %v", server)
	}
	h.ok("GET", "/api/admin/stats", adminToken, nil)
	exerciseDays := h.ok("GET", "/api/admin/exercise-days", adminToken, nil)
	if _, found := exerciseDays["exercise_days"].([]any); !found {
		t.Errorf("admin exercise-days has no exercise_days list: %v", exerciseDays)
	}
	h.expect(http.StatusForbidden, "GET", "/api/admin/stats", token, nil)

	// Weights.
	for _, invalid := range []float64{0, -1, maxBodyWeightKg + 0.01} {
		h.expect(http.StatusBadRequest, "POST", "/api/auth/weights", token, models.WeightValueCreationRequest{Date: time.Now(), Weight: invalid})
	}
	weight := h.expect(http.StatusCreated, "POST", "/api/auth/weights", token, models.WeightValueCreationRequest{Date: time.Now(), Weight: 80.5, UserID: user.ID})
	weightID := idOf(t, weight, "weight")
	h.ok("GET", "/api/auth/weights", token, nil)
	h.ok("GET", "/api/auth/weights/"+weightID, token, nil)
	_, otherToken := h.user("other@content.test", false)
	if code := h.do("GET", "/api/auth/weights/"+weightID, otherToken, nil).Code; code < 400 {
		t.Errorf("reading another user's weight: status = %d, want an error", code)
	}
	if code := h.do("DELETE", "/api/auth/weights/"+weightID, otherToken, nil).Code; code < 400 {
		t.Errorf("deleting another user's weight: status = %d, want an error", code)
	}
	h.ok("DELETE", "/api/auth/weights/"+weightID, token, nil)

	// Personal access tokens: create, use, list, revoke.
	h.expect(http.StatusBadRequest, "POST", "/api/auth/pats", token, models.PATCreationRequest{Name: "ci", Scope: "api:read", ExpiresInDays: 0})
	h.expect(http.StatusBadRequest, "POST", "/api/auth/pats", token, models.PATCreationRequest{Name: "ci", Scope: "api:everything", ExpiresInDays: 30})
	h.expect(http.StatusForbidden, "POST", "/api/auth/pats", token, models.PATCreationRequest{Name: "ci", Scope: "api:read", Admin: true, ExpiresInDays: 30})
	pat := h.expect(http.StatusCreated, "POST", "/api/auth/pats", token, models.PATCreationRequest{Name: "ci", Scope: "api:read", ExpiresInDays: 30})
	patToken, _ := field(t, pat, "data", "token").(string)
	patID := idOf(t, pat, "data", "pat")

	h.ok("GET", "/api/auth/weights", patToken, nil)
	// A read-only PAT cannot write.
	h.expect(http.StatusForbidden, "POST", "/api/auth/weights", patToken, models.WeightValueCreationRequest{Date: time.Now(), Weight: 70})

	pats := h.ok("GET", "/api/auth/pats", token, nil)
	if got := len(field(t, pats, "data").([]any)); got != 1 {
		t.Errorf("pats = %d, want 1", got)
	}
	h.ok("DELETE", "/api/auth/pats/"+patID, token, nil)
	h.expect(http.StatusUnauthorized, "GET", "/api/auth/weights", patToken, nil)
}
