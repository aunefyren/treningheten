package controllers

import (
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/aunefyren/treningheten/auth"
	"github.com/aunefyren/treningheten/database"
	"github.com/aunefyren/treningheten/files"
	"github.com/aunefyren/treningheten/models"
)

// setMCPEnabled switches the MCP server (and with it dynamic client registration) on or
// off for one test and restores the previous setting afterwards.
func setMCPEnabled(t *testing.T, enabled bool) {
	t.Helper()
	previous := files.ConfigFile.MCPEnabled
	files.ConfigFile.MCPEnabled = &enabled
	t.Cleanup(func() { files.ConfigFile.MCPEnabled = previous })
}

func hasScope(scope string, want string) bool {
	for _, s := range strings.Fields(scope) {
		if s == want {
			return true
		}
	}
	return false
}

// A dynamically registered client can never hold admin, whichever path asks for it.
func TestAdminScopeReservedForFirstParty(t *testing.T) {
	h := newAPIHarness(t)
	setMCPEnabled(t, true)
	adminUser, adminToken := h.user("admin@reserved.test", true)
	redirect := "http://127.0.0.1:9999/callback"

	registered := h.expect(http.StatusCreated, "POST", "/api/oauth/register", "", OAuthClientRegistrationRequest{
		RedirectURIs: []string{redirect}, Scope: "admin api",
	})
	if registered["scope"] != "api" {
		t.Errorf("registered scope = %v, want admin dropped", registered["scope"])
	}
	clientID := registered["client_id"].(string)

	// A client whose stored scope already contains admin (registered before the fix).
	database.Instance.Model(&models.OAuthClient{}).Where("client_id = ?", clientID).Update("scope", "admin api")

	verifier, challenge := pkcePair()
	form := url.Values{"client_id": {clientID}, "redirect_uri": {redirect}, "scope": {"admin api"},
		"code_challenge": {challenge}, "code_challenge_method": {"S256"}, "approve": {"true"}}
	request := newFormRequest("/api/oauth/authorize/decision", form)
	request.Header.Set("Authorization", "Bearer "+adminToken)
	decision := decodeBody(t, serve(h, request).Body.Bytes())
	redirectURL, _ := url.Parse(decision["redirect"].(string))
	code := redirectURL.Query().Get("code")
	if code == "" {
		t.Fatalf("no code in %v", decision)
	}
	recorder := h.doForm("/api/oauth/token", url.Values{"grant_type": {"authorization_code"}, "client_id": {clientID},
		"code": {code}, "redirect_uri": {redirect}, "code_verifier": {verifier}})
	tokens := decodeBody(t, recorder.Body.Bytes())
	if recorder.Code != http.StatusOK || hasScope(tokens["scope"].(string), models.ScopeAdmin) {
		t.Fatalf("code exchange: %d %v, want a token without admin", recorder.Code, tokens)
	}

	// A refresh chain that already holds admin loses it at the next rotation.
	database.Instance.Model(&models.OAuthRefreshToken{}).Where("client_id = ?", clientID).Update("scope", "api admin")
	recorder = h.doForm("/api/oauth/token", url.Values{"grant_type": {"refresh_token"}, "client_id": {clientID},
		"refresh_token": {tokens["refresh_token"].(string)}})
	rotated := decodeBody(t, recorder.Body.Bytes())
	if recorder.Code != http.StatusOK || hasScope(rotated["scope"].(string), models.ScopeAdmin) || !hasScope(rotated["scope"].(string), models.ScopeAPI) {
		t.Fatalf("refresh: %d %v, want api without admin", recorder.Code, rotated)
	}

	// The first-party client still gets admin for an admin user.
	adminLogin, err := auth.IssueTokenSet(adminUser.ID, true, auth.ScopeForUser(true), models.FirstPartyClientID)
	if err != nil || !hasScope(adminLogin.Scope, models.ScopeAdmin) {
		t.Fatalf("first-party admin tokens = %v (err %v), want admin", adminLogin, err)
	}
	status, refreshed := func() (int, map[string]any) {
		r := h.doForm("/api/oauth/token", url.Values{"grant_type": {"refresh_token"}, "client_id": {models.FirstPartyClientID},
			"refresh_token": {adminLogin.RefreshToken}})
		return r.Code, decodeBody(t, r.Body.Bytes())
	}()
	if status != http.StatusOK || !hasScope(refreshed["scope"].(string), models.ScopeAdmin) {
		t.Errorf("first-party refresh: %d %v, want admin kept", status, refreshed)
	}

	// Discovery never advertises admin.
	for _, path := range []string{"/.well-known/oauth-authorization-server", "/.well-known/oauth-protected-resource"} {
		metadata := h.ok("GET", path, "", nil)
		for _, s := range metadata["scopes_supported"].([]any) {
			if s == models.ScopeAdmin {
				t.Errorf("%s advertises admin", path)
			}
		}
	}
}

func TestNarrowScopeDropsAdminForThirdParty(t *testing.T) {
	stored := models.OAuthClient{Scope: "api admin"}
	if got := narrowScope(stored, ""); got != "api" {
		t.Errorf("default scope = %q, want api", got)
	}
	if got := narrowScope(stored, "admin api"); got != "api" {
		t.Errorf("requested scope = %q, want api", got)
	}
	stored.FirstParty = true
	if got := narrowScope(stored, "admin api"); got != "admin api" {
		t.Errorf("first-party scope = %q, want admin api", got)
	}
}

// Dynamic client registration only exists while MCP is on.
func TestRegistrationRequiresMCP(t *testing.T) {
	h := newAPIHarness(t)
	body := OAuthClientRegistrationRequest{RedirectURIs: []string{"https://a.test/cb"}}

	setMCPEnabled(t, false)
	h.expect(http.StatusNotFound, "POST", "/api/oauth/register", "", body)
	if metadata := h.ok("GET", "/.well-known/oauth-authorization-server", "", nil); metadata["registration_endpoint"] != nil {
		t.Errorf("registration_endpoint advertised with MCP off: %v", metadata["registration_endpoint"])
	}

	setMCPEnabled(t, true)
	h.expect(http.StatusCreated, "POST", "/api/oauth/register", "", body)
	if metadata := h.ok("GET", "/.well-known/oauth-authorization-server", "", nil); metadata["registration_endpoint"] == nil {
		t.Error("registration_endpoint missing with MCP on")
	}
}
