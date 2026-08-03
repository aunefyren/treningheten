package controllers

import (
	"crypto/sha256"
	"encoding/base64"
	"net/url"
	"sort"
	"strings"
	"testing"

	"github.com/aunefyren/treningheten/models"
)

// s256Challenge builds the PKCE code challenge a client would send for a given verifier.
func s256Challenge(verifier string) string {
	sum := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

// TestValidateRedirectURI covers that a redirect URI is matched exactly against the
// registered list. Anything looser is an open-redirect: an attacker who can steer the
// authorization response to a URI they control walks away with the code.
func TestValidateRedirectURI(t *testing.T) {
	client := models.OAuthClient{
		RedirectURIs: "https://app.example/callback https://app.example/other",
	}

	tests := []struct {
		name        string
		redirectURI string
		want        bool
	}{
		{"registered", "https://app.example/callback", true},
		{"second registered entry", "https://app.example/other", true},
		{"unregistered", "https://evil.example/callback", false},
		{"suffix appended", "https://app.example/callback.evil.example", false},
		{"path appended", "https://app.example/callback/../evil", false},
		{"trailing slash added", "https://app.example/callback/", false},
		{"query appended", "https://app.example/callback?next=https://evil.example", false},
		{"prefix only", "https://app.example", false},
		{"scheme downgraded", "http://app.example/callback", false},
		{"empty", "", false},
		{"host is a prefix of the registered host", "https://app.example.evil.example/callback", false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := validateRedirectURI(client, test.redirectURI); got != test.want {
				t.Errorf("validateRedirectURI(%q) = %v, want %v", test.redirectURI, got, test.want)
			}
		})
	}

	// A client with no registered URIs can never be redirected to.
	if validateRedirectURI(models.OAuthClient{}, "https://app.example/callback") {
		t.Errorf("validateRedirectURI accepted a URI for a client with none registered")
	}
}

// TestNarrowScope covers that an authorization request can never be granted more than the
// client registered for, whatever it asks for.
func TestNarrowScope(t *testing.T) {
	client := models.OAuthClient{Scope: models.ScopeAPIRead + " " + models.ScopeAPIWrite}

	tests := []struct {
		name      string
		requested string
		want      string
	}{
		{"nothing requested falls back to the client scope", "", client.Scope},
		{"whitespace only falls back to the client scope", "   ", client.Scope},
		{"subset is granted as asked", models.ScopeAPIRead, models.ScopeAPIRead},
		{"full registered scope", client.Scope, client.Scope},
		{"admin escalation is dropped", models.ScopeAPIRead + " " + models.ScopeAdmin, models.ScopeAPIRead},
		{"unregistered scope is dropped", models.ScopeAPI, ""},
		{"entirely unknown scopes yield nothing", "openid profile", ""},
		{"extra whitespace is normalised", "  api:read   api:write ", client.Scope},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := narrowScope(client, test.requested); got != test.want {
				t.Errorf("narrowScope(%q) = %q, want %q", test.requested, got, test.want)
			}
		})
	}

	// A read-only client asking for write must not get it, however the request is spelled.
	readOnly := models.OAuthClient{Scope: models.ScopeAPIRead}
	for _, requested := range []string{models.ScopeAPIWrite, models.ScopeAdmin, models.ScopeAPI} {
		if got := narrowScope(readOnly, requested); got != "" {
			t.Errorf("narrowScope(read-only client, %q) = %q, want empty", requested, got)
		}
	}
}

// TestNarrowScopeToSupported covers the dynamic-registration path: a client may only
// register scopes this server actually advertises, and an unusable request falls back to
// the general API scope rather than to nothing.
func TestNarrowScopeToSupported(t *testing.T) {
	tests := []struct {
		name      string
		requested string
		want      string
	}{
		{"empty falls back", "", models.ScopeAPI},
		{"unknown scopes fall back", "openid profile email", models.ScopeAPI},
		{"supported scope is kept", models.ScopeAPIRead, models.ScopeAPIRead},
		{"admin is supported and kept", models.ScopeAdmin, models.ScopeAdmin},
		{"mixed request keeps only the supported half", models.ScopeAPIWrite + " openid", models.ScopeAPIWrite},
		{"order is preserved", models.ScopeAPIWrite + " " + models.ScopeAPIRead, models.ScopeAPIWrite + " " + models.ScopeAPIRead},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := narrowScopeToSupported(test.requested); got != test.want {
				t.Errorf("narrowScopeToSupported(%q) = %q, want %q", test.requested, got, test.want)
			}
		})
	}

	// Every advertised scope must survive the filter, or discovery would advertise a scope
	// no client can actually register for.
	for _, scope := range models.SupportedScopes {
		if got := narrowScopeToSupported(scope); got != scope {
			t.Errorf("narrowScopeToSupported(%q) = %q, but it is advertised as supported", scope, got)
		}
	}
}

// TestVerifyPKCE covers the proof-of-possession check that binds an authorization code to
// the client that started the flow.
func TestVerifyPKCE(t *testing.T) {
	verifier := "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
	challenge := s256Challenge(verifier)

	if !verifyPKCE(challenge, verifier) {
		t.Errorf("verifyPKCE rejected the matching verifier")
	}

	rejections := map[string][2]string{
		"wrong verifier":              {challenge, "some-other-verifier"},
		"empty verifier":              {challenge, ""},
		"empty challenge":             {"", verifier},
		"both empty":                  {"", ""},
		"plain verifier as challenge": {verifier, verifier},
		"truncated challenge":         {challenge[:len(challenge)-1], verifier},
		"standard base64 padding":     {base64.StdEncoding.EncodeToString([]byte(challenge)), verifier},
	}
	for name, pair := range rejections {
		t.Run(name, func(t *testing.T) {
			if verifyPKCE(pair[0], pair[1]) {
				t.Errorf("verifyPKCE accepted %s", name)
			}
		})
	}
}

// TestBuildRedirect covers assembling the authorization response URI: existing query
// parameters survive, empty values are omitted, and an unparseable URI is returned as-is.
func TestBuildRedirect(t *testing.T) {
	got := buildRedirect("https://app.example/callback?existing=1", map[string]string{
		"code":  "the-code",
		"state": "the-state",
		"error": "",
	})

	parsed, err := url.Parse(got)
	if err != nil {
		t.Fatalf("buildRedirect produced an unparseable URI %q: %v", got, err)
	}
	query := parsed.Query()

	if query.Get("existing") != "1" {
		t.Errorf("existing query parameter was dropped: %q", got)
	}
	if query.Get("code") != "the-code" || query.Get("state") != "the-state" {
		t.Errorf("parameters missing from %q", got)
	}
	if _, present := query["error"]; present {
		t.Errorf("empty parameter was written into %q", got)
	}
	if parsed.Scheme != "https" || parsed.Host != "app.example" || parsed.Path != "/callback" {
		t.Errorf("buildRedirect changed the redirect target: %q", got)
	}

	// A state value with URL-significant characters must come back out intact.
	roundTrip := buildRedirect("https://app.example/callback", map[string]string{"state": "a b&c=d?e"})
	parsed, err = url.Parse(roundTrip)
	if err != nil {
		t.Fatalf("buildRedirect produced an unparseable URI %q: %v", roundTrip, err)
	}
	if parsed.Query().Get("state") != "a b&c=d?e" {
		t.Errorf("state was mangled: %q", roundTrip)
	}

	// Unparseable input is passed through rather than turned into a different target.
	broken := "https://app.example/%zz"
	if got := buildRedirect(broken, map[string]string{"code": "x"}); got != broken {
		t.Errorf("buildRedirect(%q) = %q, want it returned unchanged", broken, got)
	}
}

// TestGenerateOpaqueAuthCode covers that issued codes are unique and URL-safe — they travel
// in a query string, so a code needing escaping would be a bug.
func TestGenerateOpaqueAuthCode(t *testing.T) {
	seen := map[string]bool{}
	for i := 0; i < 50; i++ {
		code := generateOpaqueAuthCode()
		if seen[code] {
			t.Fatalf("generateOpaqueAuthCode repeated a code")
		}
		seen[code] = true

		if len(code) < 32 {
			t.Errorf("code %q is too short", code)
		}
		if url.QueryEscape(code) != code {
			t.Errorf("code %q is not URL-safe", code)
		}
	}
}

// TestSupportedScopesAreSorted is a guard on the discovery document: the advertised scope
// list must contain exactly the four defined scopes, with no duplicates.
func TestSupportedScopesAreDistinct(t *testing.T) {
	unique := map[string]bool{}
	for _, scope := range models.SupportedScopes {
		if unique[scope] {
			t.Errorf("scope %q is advertised twice", scope)
		}
		unique[scope] = true
	}

	want := []string{models.ScopeAPI, models.ScopeAPIRead, models.ScopeAPIWrite, models.ScopeAdmin}
	got := append([]string{}, models.SupportedScopes...)
	sort.Strings(got)
	sort.Strings(want)
	if strings.Join(got, " ") != strings.Join(want, " ") {
		t.Errorf("SupportedScopes = %v, want %v", models.SupportedScopes, want)
	}
}
