package controllers

import (
	"crypto/sha256"
	"encoding/base64"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/aunefyren/treningheten/database"
	"github.com/aunefyren/treningheten/models"
)

func pkcePair() (verifier string, challenge string) {
	verifier = "a-sufficiently-long-code-verifier-for-the-tests-0123456789"
	sum := sha256.Sum256([]byte(verifier))
	return verifier, base64.RawURLEncoding.EncodeToString(sum[:])
}

func TestOAuthAuthorizationCodeFlow(t *testing.T) {
	h := newAPIHarness(t)
	_, token := h.user("owner@oauth.test", false)
	redirect := "http://127.0.0.1:9999/callback"

	// Dynamic client registration (a public MCP-style client).
	h.expect(http.StatusBadRequest, "POST", "/api/oauth/register", "", "{broken")
	h.expect(http.StatusBadRequest, "POST", "/api/oauth/register", "", OAuthClientRegistrationRequest{ClientName: "No redirect"})
	registered := h.expect(http.StatusCreated, "POST", "/api/oauth/register", "", OAuthClientRegistrationRequest{
		ClientName: "  Test client  ", RedirectURIs: []string{redirect}, Scope: "api:read made-up-scope",
	})
	clientID := registered["client_id"].(string)
	if registered["scope"] != "api:read" {
		t.Errorf("granted scope = %v, want unsupported scopes dropped", registered["scope"])
	}
	if registered["client_secret"] != nil && registered["client_secret"] != "" {
		t.Error("a public client was issued a secret")
	}

	verifier, challenge := pkcePair()
	info := url.Values{"client_id": {clientID}, "redirect_uri": {redirect}, "response_type": {"code"},
		"code_challenge": {challenge}, "code_challenge_method": {"S256"}, "scope": {"api:read admin"}}
	consent := h.ok("GET", "/api/oauth/authorize?"+info.Encode(), "", nil)
	if consent["client_name"] != "Test client" || consent["scope"] != "api:read" {
		t.Errorf("consent = %v, want the trimmed name and scope narrowed to the client's", consent)
	}
	for name, mutate := range map[string]func(url.Values){
		"unknown client":   func(v url.Values) { v.Set("client_id", "nope") },
		"foreign redirect": func(v url.Values) { v.Set("redirect_uri", "https://evil.test/cb") },
		"implicit flow":    func(v url.Values) { v.Set("response_type", "token") },
		"no PKCE":          func(v url.Values) { v.Del("code_challenge") },
		"plain PKCE":       func(v url.Values) { v.Set("code_challenge_method", "plain") },
	} {
		values := url.Values{}
		for k, v := range info {
			values[k] = append([]string(nil), v...)
		}
		mutate(values)
		if code := h.do("GET", "/api/oauth/authorize?"+values.Encode(), "", nil).Code; code != http.StatusBadRequest {
			t.Errorf("%s: status = %d, want 400", name, code)
		}
	}

	decide := func(approve string, redirectURI string, withToken bool) (int, map[string]any) {
		form := url.Values{"client_id": {clientID}, "redirect_uri": {redirectURI}, "state": {"xyz"},
			"scope": {"api:read"}, "code_challenge": {challenge}, "code_challenge_method": {"S256"}, "approve": {approve}}
		request := newFormRequest("/api/oauth/authorize/decision", form)
		if withToken {
			request.Header.Set("Authorization", "Bearer "+token)
		}
		recorder := serve(h, request)
		return recorder.Code, decodeBody(t, recorder.Body.Bytes())
	}

	if code, _ := decide("true", redirect, false); code != http.StatusUnauthorized {
		t.Errorf("decision without login: status = %d, want 401", code)
	}
	if code, _ := decide("true", "https://evil.test/cb", true); code != http.StatusBadRequest {
		t.Errorf("decision to an unregistered redirect: status = %d, want 400", code)
	}
	_, denied := decide("false", redirect, true)
	if deniedURL, _ := url.Parse(denied["redirect"].(string)); deniedURL.Query().Get("error") != "access_denied" || deniedURL.Query().Get("state") != "xyz" {
		t.Errorf("denial redirect = %v", denied["redirect"])
	}

	_, approved := decide("true", redirect, true)
	approvedURL, _ := url.Parse(approved["redirect"].(string))
	code := approvedURL.Query().Get("code")
	if code == "" || approvedURL.Query().Get("state") != "xyz" {
		t.Fatalf("approval redirect = %v", approved["redirect"])
	}

	exchange := func(values url.Values) (int, map[string]any) {
		recorder := h.doForm("/api/oauth/token", values)
		return recorder.Code, decodeBody(t, recorder.Body.Bytes())
	}
	base := url.Values{"grant_type": {"authorization_code"}, "client_id": {clientID}, "code": {code}, "redirect_uri": {redirect}, "code_verifier": {verifier}}
	for name, mutate := range map[string]func(url.Values){
		"wrong verifier": func(v url.Values) { v.Set("code_verifier", "wrong-verifier") },
		"wrong redirect": func(v url.Values) { v.Set("redirect_uri", "http://127.0.0.1:9999/other") },
		"no verifier":    func(v url.Values) { v.Del("code_verifier") },
		"unknown code":   func(v url.Values) { v.Set("code", "not-a-code") },
		"unknown client": func(v url.Values) { v.Set("client_id", "nope") },
	} {
		values := url.Values{}
		for k, v := range base {
			values[k] = append([]string(nil), v...)
		}
		mutate(values)
		if status, _ := exchange(values); status == http.StatusOK {
			t.Errorf("%s: code exchange succeeded", name)
		}
	}

	status, tokens := exchange(base)
	if status != http.StatusOK || tokens["access_token"] == nil || tokens["refresh_token"] == nil {
		t.Fatalf("code exchange: %d %v", status, tokens)
	}
	// Single use.
	if status, _ := exchange(base); status == http.StatusOK {
		t.Error("an authorization code was accepted twice")
	}

	// The access token works against the API; read-only scope can't write.
	accessToken := tokens["access_token"].(string)
	h.ok("GET", "/api/auth/weights", accessToken, nil)
	h.expect(http.StatusForbidden, "POST", "/api/auth/weights", accessToken, map[string]any{"weight": 70})

	// Refresh rotates the token; replaying the old one is refused.
	refresh := tokens["refresh_token"].(string)
	status, rotated := exchange(url.Values{"grant_type": {"refresh_token"}, "client_id": {clientID}, "refresh_token": {refresh}})
	if status != http.StatusOK || rotated["refresh_token"] == refresh {
		t.Fatalf("refresh: %d %v", status, rotated)
	}
	if status, _ := exchange(url.Values{"grant_type": {"refresh_token"}, "client_id": {clientID}, "refresh_token": {refresh}}); status == http.StatusOK {
		t.Error("a rotated-out refresh token was accepted")
	}

	// Revocation always answers 200 and kills the refresh token.
	h.doForm("/api/oauth/revoke", url.Values{"token": {"unknown"}})
	h.doForm("/api/oauth/revoke", url.Values{"token": {rotated["refresh_token"].(string)}})
	if status, _ := exchange(url.Values{"grant_type": {"refresh_token"}, "client_id": {clientID}, "refresh_token": {rotated["refresh_token"].(string)}}); status == http.StatusOK {
		t.Error("a revoked refresh token was accepted")
	}

	// Grant-type dispatch errors.
	if status, _ := exchange(url.Values{"client_id": {clientID}}); status != http.StatusBadRequest {
		t.Errorf("missing grant_type: status = %d", status)
	}
	if status, _ := exchange(url.Values{"grant_type": {"client_credentials"}, "client_id": {clientID}}); status != http.StatusBadRequest {
		t.Errorf("unsupported grant_type: status = %d", status)
	}
	if status, _ := exchange(url.Values{"grant_type": {"password"}, "client_id": {"treningheten-web"}, "username": {"nobody@oauth.test"}, "password": {"x"}}); status == http.StatusOK {
		t.Error("password grant for an unknown user succeeded")
	}
	if status, _ := exchange(url.Values{"grant_type": {"password"}, "client_id": {"treningheten-web"}}); status != http.StatusBadRequest {
		t.Errorf("password grant without credentials: status = %d", status)
	}
}

func TestOAuthConfidentialClientRegistration(t *testing.T) {
	h := newAPIHarness(t)

	registered := h.expect(http.StatusCreated, "POST", "/api/oauth/register", "", OAuthClientRegistrationRequest{
		RedirectURIs: []string{"https://app.test/cb"}, TokenEndpointAuthMethod: "client_secret_basic",
	})
	secret, _ := registered["client_secret"].(string)
	if secret == "" {
		t.Fatal("a confidential client got no secret")
	}
	if registered["client_name"] != "Dynamically Registered Client" || registered["scope"] != "api" {
		t.Errorf("defaults = %v", registered)
	}

	// The token endpoint authenticates it with HTTP Basic; a wrong secret is refused.
	form := url.Values{"grant_type": {"refresh_token"}, "refresh_token": {"none"}}
	wrong := newFormRequest("/api/oauth/token", form)
	wrong.SetBasicAuth(registered["client_id"].(string), "wrong-secret")
	if recorder := serve(h, wrong); recorder.Code != http.StatusUnauthorized {
		t.Errorf("wrong client secret: status = %d, want 401", recorder.Code)
	}
	missing := newFormRequest("/api/oauth/token", url.Values{"grant_type": {"refresh_token"}, "refresh_token": {"none"}, "client_id": {registered["client_id"].(string)}})
	if recorder := serve(h, missing); recorder.Code != http.StatusUnauthorized {
		t.Errorf("no client secret: status = %d, want 401", recorder.Code)
	}
	// The right secret gets past client authentication (and fails on the bogus token).
	right := newFormRequest("/api/oauth/token", form)
	right.SetBasicAuth(registered["client_id"].(string), secret)
	if recorder := serve(h, right); recorder.Code != http.StatusBadRequest || !strings.Contains(recorder.Body.String(), "invalid_grant") {
		t.Errorf("right client secret: status = %d body %s, want 400 invalid_grant", recorder.Code, recorder.Body.String())
	}

	// Rows stored before the fix (public=true despite a secret) still demand the secret.
	database.Instance.Model(&models.OAuthClient{}).Where("client_id = ?", registered["client_id"]).Update("public", true)
	if recorder := serve(h, missing); recorder.Code != http.StatusUnauthorized {
		t.Errorf("legacy mis-stored client without secret: status = %d, want 401", recorder.Code)
	}

	metadata := h.ok("GET", "/.well-known/oauth-authorization-server", "", nil)
	if metadata["token_endpoint"] == nil || metadata["registration_endpoint"] == nil {
		t.Errorf("authorization server metadata = %v", metadata)
	}
	h.ok("GET", "/.well-known/oauth-protected-resource", "", nil)
}
