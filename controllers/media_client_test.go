package controllers

import (
	"encoding/base64"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/aunefyren/treningheten/files"
)

// --- Audiobookshelf ---

// TestABSRequest covers the authenticated GET against a user's own Audiobookshelf server:
// the token is sent as a bearer credential, the path is appended to the configured server
// URL, and the status is handed back to the caller rather than swallowed.
func TestABSRequest(t *testing.T) {
	var gotPath, gotAuth, gotAccept string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		gotPath = request.URL.RequestURI()
		gotAuth = request.Header.Get("Authorization")
		gotAccept = request.Header.Get("Accept")
		writer.Write([]byte(`{"id":"abs-user"}`))
	}))
	defer server.Close()

	body, status, err := absRequest(server.URL, "/api/me", "the-token")
	if err != nil {
		t.Fatalf("absRequest returned error: %v", err)
	}
	if status != http.StatusOK {
		t.Errorf("status = %d, want 200", status)
	}
	if string(body) != `{"id":"abs-user"}` {
		t.Errorf("body = %s, want it passed through unchanged", body)
	}
	if gotPath != "/api/me" {
		t.Errorf("requested %q, want /api/me", gotPath)
	}
	if gotAuth != "Bearer the-token" {
		t.Errorf("Authorization = %q, want a bearer token", gotAuth)
	}
	if gotAccept != "application/json" {
		t.Errorf("Accept = %q, want application/json", gotAccept)
	}

	// A server URL with a trailing slash must not double the path separator.
	if _, _, err := absRequest(server.URL+"/", "/api/me", "the-token"); err != nil {
		t.Fatalf("absRequest with a trailing slash returned error: %v", err)
	}
	if gotPath != "/api/me" {
		t.Errorf("requested %q after a trailing slash, want /api/me", gotPath)
	}
}

// TestABSRequestFailures covers the two failure shapes: a non-200 comes back as a status
// for the caller to interpret (not an error), while an unreachable server is an error.
func TestABSRequestFailures(t *testing.T) {
	unauthorized := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(http.StatusUnauthorized)
	}))
	defer unauthorized.Close()

	_, status, err := absRequest(unauthorized.URL, "/api/me", "bad-token")
	if err != nil {
		t.Errorf("a 401 came back as an error (%v); the caller needs the status", err)
	}
	if status != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", status)
	}

	dead := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	deadURL := dead.URL
	dead.Close()

	if _, _, err := absRequest(deadURL, "/api/me", "the-token"); err == nil {
		t.Errorf("an unreachable server returned no error")
	}
	if _, _, err := absRequest("://not a url", "/api/me", "the-token"); err == nil {
		t.Errorf("an unparseable server URL returned no error")
	}
}

// TestABSFetchListeningSessions covers the history pull: it asks the user-scoped endpoint
// (no privacy filtering needed) with the configured page size, and unwraps the sessions.
func TestABSFetchListeningSessions(t *testing.T) {
	var gotQuery url.Values
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if !strings.HasPrefix(request.URL.Path, "/api/me/listening-sessions") {
			t.Errorf("requested %q, want the user-scoped listening-sessions endpoint", request.URL.Path)
		}
		gotQuery = request.URL.Query()
		writer.Write([]byte(`{"sessions":[
			{"id":"s1","libraryItemId":"li1","displayTitle":"A Book","displayAuthor":"An Author","mediaType":"book","timeListening":1800,"startedAt":1754200000000,"updatedAt":1754201800000},
			{"id":"s2","libraryItemId":"li2","displayTitle":"An Episode","displayAuthor":"A Show","mediaType":"podcast","timeListening":600,"startedAt":1754210000000}
		]}`))
	}))
	defer server.Close()

	sessions, err := absFetchListeningSessions(server.URL, "the-token")
	if err != nil {
		t.Fatalf("absFetchListeningSessions returned error: %v", err)
	}
	if len(sessions) != 2 {
		t.Fatalf("got %d sessions, want 2", len(sessions))
	}
	if sessions[0].DisplayTitle != "A Book" || sessions[0].MediaType != "book" {
		t.Errorf("first session = %+v, want the book", sessions[0])
	}
	// The native media type is what lights up the typed rail nodes, so it must survive.
	if sessions[1].MediaType != "podcast" {
		t.Errorf("second session media type = %q, want podcast", sessions[1].MediaType)
	}
	if sessions[0].TimeListening != 1800 || sessions[0].StartedAt != 1754200000000 {
		t.Errorf("session timings did not survive: %+v", sessions[0])
	}
	if got := gotQuery.Get("itemsPerPage"); got != "100" {
		t.Errorf("itemsPerPage = %q, want the configured page size", got)
	}
	if got := gotQuery.Get("page"); got != "0" {
		t.Errorf("page = %q, want the first page", got)
	}
}

// TestABSFetchListeningSessionsFailures covers that a rejected token or a malformed body is
// reported rather than silently becoming an empty history — an empty pull is treated as
// "nothing to store" by the non-destructive guard, which would mask a broken connection.
func TestABSFetchListeningSessionsFailures(t *testing.T) {
	for _, status := range []int{http.StatusUnauthorized, http.StatusInternalServerError} {
		server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			writer.WriteHeader(status)
		}))

		sessions, err := absFetchListeningSessions(server.URL, "the-token")
		server.Close()

		if err == nil {
			t.Errorf("status %d returned no error", status)
		}
		if sessions != nil {
			t.Errorf("status %d returned %d sessions, want nil", status, len(sessions))
		}
	}

	malformed := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Write([]byte(`{"sessions": [`))
	}))
	defer malformed.Close()

	if _, err := absFetchListeningSessions(malformed.URL, "the-token"); err == nil {
		t.Errorf("a malformed body returned no error")
	}
}

// --- Spotify ---

// stubSpotify points both Spotify endpoints at a test server and installs app credentials,
// restoring everything afterwards.
func stubSpotify(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()

	server := httptest.NewServer(handler)

	previousToken, previousHistory := spotifyTokenURL, spotifyRecentlyPlayedURL
	previousConfig := files.ConfigFile.Media.Spotify
	spotifyTokenURL = server.URL + "/api/token"
	spotifyRecentlyPlayedURL = server.URL + "/v1/me/player/recently-played"
	files.ConfigFile.Media.Spotify.ClientID = "the-client-id"
	files.ConfigFile.Media.Spotify.ClientSecret = "the-client-secret"

	t.Cleanup(func() {
		spotifyTokenURL, spotifyRecentlyPlayedURL = previousToken, previousHistory
		files.ConfigFile.Media.Spotify = previousConfig
		server.Close()
	})

	return server
}

// TestSpotifyTokenRequest covers the token exchange used by both the code grant and the
// refresh grant: the app credentials go in a Basic header (never in the form), the form is
// posted url-encoded, and the response is unmarshalled.
func TestSpotifyTokenRequest(t *testing.T) {
	var gotAuth, gotContentType, gotBody string
	stubSpotify(t, func(writer http.ResponseWriter, request *http.Request) {
		gotAuth = request.Header.Get("Authorization")
		gotContentType = request.Header.Get("Content-Type")
		body, _ := io.ReadAll(request.Body)
		gotBody = string(body)
		writer.Write([]byte(`{"access_token":"at","refresh_token":"rt","expires_in":3600,"token_type":"Bearer"}`))
	})

	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", "the-code")

	tokens, err := spotifyTokenRequest(form)
	if err != nil {
		t.Fatalf("spotifyTokenRequest returned error: %v", err)
	}
	if tokens.AccessToken != "at" || tokens.RefreshToken != "rt" || tokens.ExpiresIn != 3600 {
		t.Errorf("tokens = %+v, want the parsed response", tokens)
	}

	wantAuth := "Basic " + base64.StdEncoding.EncodeToString([]byte("the-client-id:the-client-secret"))
	if gotAuth != wantAuth {
		t.Errorf("Authorization = %q, want the app credentials in a Basic header", gotAuth)
	}
	if gotContentType != "application/x-www-form-urlencoded" {
		t.Errorf("Content-Type = %q, want form encoding", gotContentType)
	}
	if !strings.Contains(gotBody, "grant_type=authorization_code") || !strings.Contains(gotBody, "code=the-code") {
		t.Errorf("body = %q, want the form fields", gotBody)
	}
	if strings.Contains(gotBody, "the-client-secret") {
		t.Errorf("the client secret was posted in the form body: %q", gotBody)
	}
}

// TestSpotifyTokenRequestFailures covers a rejected exchange (an expired code or bad
// credentials) and a malformed response surfacing as errors.
func TestSpotifyTokenRequestFailures(t *testing.T) {
	stubSpotify(t, func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(http.StatusBadRequest)
		writer.Write([]byte(`{"error":"invalid_grant"}`))
	})
	if _, err := spotifyTokenRequest(url.Values{}); err == nil {
		t.Errorf("a rejected token request returned no error")
	}

	stubSpotify(t, func(writer http.ResponseWriter, request *http.Request) {
		writer.Write([]byte(`not json`))
	})
	if _, err := spotifyTokenRequest(url.Values{}); err == nil {
		t.Errorf("a malformed token response returned no error")
	}
}

// TestSpotifyFetchRecentlyPlayed covers the history pull: a bearer token, the 50-item cap,
// and the play items unwrapped with their track detail intact.
func TestSpotifyFetchRecentlyPlayed(t *testing.T) {
	var gotAuth string
	var gotQuery url.Values
	stubSpotify(t, func(writer http.ResponseWriter, request *http.Request) {
		gotAuth = request.Header.Get("Authorization")
		gotQuery = request.URL.Query()
		writer.Write([]byte(`{"items":[
			{"played_at":"2026-08-03T10:05:00.000Z","track":{"id":"t1","name":"A Song","duration_ms":210000,"artists":[{"name":"An Artist"}],"album":{"name":"An Album"}}}
		]}`))
	})

	history, err := spotifyFetchRecentlyPlayed("the-access-token")
	if err != nil {
		t.Fatalf("spotifyFetchRecentlyPlayed returned error: %v", err)
	}
	if len(history) != 1 {
		t.Fatalf("got %d items, want 1", len(history))
	}
	if history[0].Track.Name != "A Song" || history[0].PlayedAt != "2026-08-03T10:05:00.000Z" {
		t.Errorf("item = %+v, want the parsed play", history[0])
	}
	if history[0].Track.DurationMs != 210000 || len(history[0].Track.Artists) != 1 {
		t.Errorf("track detail did not survive: %+v", history[0].Track)
	}
	if gotAuth != "Bearer the-access-token" {
		t.Errorf("Authorization = %q, want a bearer token", gotAuth)
	}
	if got := gotQuery.Get("limit"); got != "50" {
		t.Errorf("limit = %q, want 50", got)
	}
}

// TestSpotifyFetchRecentlyPlayedForbidden covers the 403 an app in development mode returns
// for a user who isn't allowlisted. It has its own sentinel so the caller can tell the user
// something actionable instead of "sync failed".
func TestSpotifyFetchRecentlyPlayedForbidden(t *testing.T) {
	stubSpotify(t, func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(http.StatusForbidden)
	})

	_, err := spotifyFetchRecentlyPlayed("the-access-token")
	if !errors.Is(err, ErrSpotifyForbidden) {
		t.Errorf("error = %v, want the ErrSpotifyForbidden sentinel", err)
	}
}

// TestSpotifyFetchRecentlyPlayedFailures covers the remaining failure paths: an expired
// token (401) and a malformed body are errors, not an empty history.
func TestSpotifyFetchRecentlyPlayedFailures(t *testing.T) {
	stubSpotify(t, func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(http.StatusUnauthorized)
	})
	history, err := spotifyFetchRecentlyPlayed("expired-token")
	if err == nil {
		t.Errorf("a 401 returned no error")
	}
	if errors.Is(err, ErrSpotifyForbidden) {
		t.Errorf("a 401 was reported as the 403 sentinel")
	}
	if history != nil {
		t.Errorf("a 401 returned %d items, want nil", len(history))
	}

	stubSpotify(t, func(writer http.ResponseWriter, request *http.Request) {
		writer.Write([]byte(`{"items": [`))
	})
	if _, err := spotifyFetchRecentlyPlayed("the-access-token"); err == nil {
		t.Errorf("a malformed body returned no error")
	}
}
