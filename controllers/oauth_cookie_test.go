package controllers

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/aunefyren/treningheten/auth"
	"github.com/aunefyren/treningheten/database"
	"github.com/aunefyren/treningheten/files"
	"github.com/aunefyren/treningheten/models"
)

// findCookie returns the named cookie a response set, or nil.
func findCookie(recorder *httptest.ResponseRecorder, name string) *http.Cookie {
	for _, cookie := range recorder.Result().Cookies() {
		if cookie.Name == name {
			return cookie
		}
	}
	return nil
}

// cookieForm builds a token/revoke request that carries the refresh cookie.
func cookieForm(path string, form url.Values, cookieValue string) *http.Request {
	request := newFormRequest(path, form)
	if cookieValue != "" {
		request.AddCookie(&http.Cookie{Name: refreshCookieName, Value: cookieValue})
	}
	return request
}

func firstPartyRefresh() url.Values {
	return url.Values{"grant_type": {"refresh_token"}, "client_id": {models.FirstPartyClientID}}
}

func assertRefreshCookie(t *testing.T, cookie *http.Cookie, secure bool) {
	t.Helper()
	if cookie == nil || cookie.Value == "" {
		t.Fatal("no refresh cookie was set")
	}
	if !cookie.HttpOnly || cookie.SameSite != http.SameSiteStrictMode || cookie.Path != "/api/oauth" {
		t.Errorf("cookie = %+v, want HttpOnly, SameSite=Strict, Path=/api/oauth", cookie)
	}
	if cookie.Secure != secure {
		t.Errorf("Secure = %v, want %v", cookie.Secure, secure)
	}
	if cookie.MaxAge != int(auth.RefreshTokenTTL.Seconds()) {
		t.Errorf("Max-Age = %d, want the refresh token TTL", cookie.MaxAge)
	}
}

func assertCookieCleared(t *testing.T, recorder *httptest.ResponseRecorder) {
	t.Helper()
	if cleared := findCookie(recorder, refreshCookieName); cleared == nil || cleared.MaxAge >= 0 || cleared.Value != "" || !cleared.HttpOnly {
		t.Errorf("refresh cookie was not cleared: %+v", cleared)
	}
}

func TestFirstPartyRefreshTokenCookie(t *testing.T) {
	h := newAPIHarness(t)
	files.ConfigFile.TreninghetenExternalURL = ""
	userWithPassword(h, "cookie@oauth.test", "Password123")

	login := h.doForm("/api/oauth/token", url.Values{"grant_type": {"password"}, "client_id": {models.FirstPartyClientID}, "username": {"cookie@oauth.test"}, "password": {"Password123"}})
	body := decodeBody(t, login.Body.Bytes())
	if login.Code != http.StatusOK || body["access_token"] == nil {
		t.Fatalf("login: %d %v", login.Code, body)
	}
	if _, present := body["refresh_token"]; present {
		t.Error("the first-party response body carries the refresh token")
	}
	cookie := findCookie(login, refreshCookieName)
	assertRefreshCookie(t, cookie, false)

	// The refresh grant works from the cookie alone and rotates it.
	refreshed := serve(h, cookieForm("/api/oauth/token", firstPartyRefresh(), cookie.Value))
	refreshedBody := decodeBody(t, refreshed.Body.Bytes())
	if refreshed.Code != http.StatusOK || refreshedBody["access_token"] == nil {
		t.Fatalf("cookie refresh: %d %v", refreshed.Code, refreshedBody)
	}
	if _, present := refreshedBody["refresh_token"]; present {
		t.Error("the refresh response body carries the refresh token")
	}
	rotated := findCookie(refreshed, refreshCookieName)
	assertRefreshCookie(t, rotated, false)
	if rotated.Value == cookie.Value {
		t.Error("the refresh cookie was not rotated")
	}

	// A reused token is refused and clears the cookie.
	reused := serve(h, cookieForm("/api/oauth/token", firstPartyRefresh(), cookie.Value))
	if reused.Code != http.StatusBadRequest {
		t.Errorf("reused cookie: status = %d, want 400", reused.Code)
	}
	assertCookieCleared(t, reused)
	// Reuse revoked the whole chain.
	if chain := serve(h, cookieForm("/api/oauth/token", firstPartyRefresh(), rotated.Value)); chain.Code == http.StatusOK {
		t.Error("the chain survived reuse detection")
	}

	// No cookie and no field is a plain invalid request.
	if missing := serve(h, cookieForm("/api/oauth/token", firstPartyRefresh(), "")); missing.Code != http.StatusBadRequest {
		t.Errorf("no token: status = %d, want 400", missing.Code)
	}
}

func TestRefreshCookieSecureFollowsExternalURL(t *testing.T) {
	h := newAPIHarness(t)
	userWithPassword(h, "secure@oauth.test", "Password123")
	form := url.Values{"grant_type": {"password"}, "client_id": {models.FirstPartyClientID}, "username": {"secure@oauth.test"}, "password": {"Password123"}}

	previous := files.ConfigFile.TreninghetenExternalURL
	t.Cleanup(func() { files.ConfigFile.TreninghetenExternalURL = previous })
	for externalURL, want := range map[string]bool{"https://trening.test": true, "http://trening.test": false, "": false} {
		files.ConfigFile.TreninghetenExternalURL = externalURL
		assertRefreshCookie(t, findCookie(h.doForm("/api/oauth/token", form), refreshCookieName), want)
	}
}

func TestThirdPartyClientKeepsRefreshTokenInBody(t *testing.T) {
	h := newAPIHarness(t)
	setMCPEnabled(t, true)
	user, _ := h.user("third@oauth.test", false)
	registered := h.expect(http.StatusCreated, "POST", "/api/oauth/register", "", OAuthClientRegistrationRequest{RedirectURIs: []string{"http://127.0.0.1/cb"}})
	clientID := registered["client_id"].(string)

	set, err := auth.IssueTokenSet(user.ID, false, models.ScopeAPI, clientID)
	if err != nil {
		t.Fatal(err)
	}
	refresh := h.doForm("/api/oauth/token", url.Values{"grant_type": {"refresh_token"}, "client_id": {clientID}, "refresh_token": {set.RefreshToken}})
	body := decodeBody(t, refresh.Body.Bytes())
	if refresh.Code != http.StatusOK || body["refresh_token"] == nil {
		t.Fatalf("third-party refresh: %d %v, want the token in the body", refresh.Code, body)
	}
	if cookie := findCookie(refresh, refreshCookieName); cookie != nil {
		t.Errorf("a third-party client was given the cookie: %+v", cookie)
	}

	// The cookie is never accepted for another client_id, even one holding a valid token.
	firstParty, err := auth.IssueTokenSet(user.ID, false, models.ScopeAPI, models.FirstPartyClientID)
	if err != nil {
		t.Fatal(err)
	}
	other := serve(h, cookieForm("/api/oauth/token", url.Values{"grant_type": {"refresh_token"}, "client_id": {clientID}}, firstParty.RefreshToken))
	if other.Code != http.StatusBadRequest {
		t.Errorf("cookie with a third-party client_id: status = %d, want 400", other.Code)
	}
	// Revoke ignores the cookie for them too.
	serve(h, cookieForm("/api/oauth/revoke", url.Values{"client_id": {clientID}}, firstParty.RefreshToken))
	if still := serve(h, cookieForm("/api/oauth/token", firstPartyRefresh(), firstParty.RefreshToken)); still.Code != http.StatusOK {
		t.Errorf("a third-party revoke killed the first-party token: status = %d", still.Code)
	}
}

func TestRevokeFallsBackToRefreshCookie(t *testing.T) {
	h := newAPIHarness(t)
	user, _ := h.user("revoke@oauth.test", false)
	set, err := auth.IssueTokenSet(user.ID, false, models.ScopeAPI, models.FirstPartyClientID)
	if err != nil {
		t.Fatal(err)
	}

	revoked := serve(h, cookieForm("/api/oauth/revoke", url.Values{"client_id": {models.FirstPartyClientID}}, set.RefreshToken))
	if revoked.Code != http.StatusOK {
		t.Fatalf("revoke: status = %d", revoked.Code)
	}
	assertCookieCleared(t, revoked)
	if after := serve(h, cookieForm("/api/oauth/token", firstPartyRefresh(), set.RefreshToken)); after.Code == http.StatusOK {
		t.Error("the token still refreshes after revoke")
	}
}

func TestUserIssuersSetRefreshCookie(t *testing.T) {
	h := newAPIHarness(t)
	files.ConfigFile.TreninghetenExternalURL = ""
	user, token := userWithPassword(h, "issuer@oauth.test", "Password123")

	// Account update re-issues a token set.
	update := h.do("POST", "/api/auth/users/"+user.ID.String(), token, models.UserUpdateRequest{Email: "issuer@oauth.test", OldPassword: "Password123"})
	body := decodeBody(t, update.Body.Bytes())
	if update.Code != http.StatusOK {
		t.Fatalf("update: %d %v", update.Code, body)
	}
	assertRefreshCookie(t, findCookie(update, refreshCookieName), false)
	if _, present := field(t, body, "data").(map[string]any)["refresh_token"]; present {
		t.Error("the update response body carries the refresh token")
	}

	// Verifying the account auto-logs-in the same way.
	code, err := database.GenerateRandomVerificationCodeForUser(user.ID)
	if err != nil {
		t.Fatal(err)
	}
	verify := h.do("POST", "/api/open/users/verify/"+code, token, nil)
	body = decodeBody(t, verify.Body.Bytes())
	if verify.Code != http.StatusOK {
		t.Fatalf("verify: %d %v", verify.Code, body)
	}
	assertRefreshCookie(t, findCookie(verify, refreshCookieName), false)
	if _, present := field(t, body, "data").(map[string]any)["refresh_token"]; present {
		t.Error("the verify response body carries the refresh token")
	}
}
