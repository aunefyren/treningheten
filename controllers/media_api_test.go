package controllers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/aunefyren/treningheten/database"
	"github.com/aunefyren/treningheten/files"
	"github.com/aunefyren/treningheten/models"

	"github.com/google/uuid"
)

// withMedia turns the media feature and every provider on, allowing loopback targets so
// httptest servers are reachable.
func withMedia(t *testing.T) {
	t.Helper()

	previous := files.ConfigFile.Media
	key, _ := files.GenerateSecureKey(32)
	allow := true
	files.ConfigFile.Media.Enabled = true
	files.ConfigFile.Media.TokenKey = key
	files.ConfigFile.Media.AllowPrivateTargets = &allow
	files.ConfigFile.Media.Plex.Enabled = true
	files.ConfigFile.Media.Plex.ClientIdentifier = "test-client-identifier"
	files.ConfigFile.Media.Spotify.Enabled = true
	files.ConfigFile.Media.Spotify.RedirectURI = "http://localhost/oauth"
	files.ConfigFile.Media.Audiobookshelf.Enabled = true
	t.Cleanup(func() { files.ConfigFile.Media = previous })
}

// timedSession creates today's session with a start time and a 40-minute duration, the
// window the soundtrack is matched against.
func timedSession(t *testing.T, h *apiHarness, token string, start time.Time) string {
	t.Helper()

	sessionID := createTodaySession(t, h, token)
	h.ok("PUT", "/api/auth/exercises/"+sessionID, token, models.ExerciseUpdateRequest{
		Note: "with music", IsOn: true, Duration: int64Ptr(2400), Time: start.Format(time.RFC3339),
	})
	return sessionID
}

func playbackRows(t *testing.T, sessionID string, provider string) []models.MediaPlayback {
	t.Helper()
	var rows []models.MediaPlayback
	query := database.Instance.Where("exercise_id = ?", sessionID)
	if provider != "" {
		query = query.Where("provider = ?", provider)
	}
	if err := query.Find(&rows).Error; err != nil {
		t.Fatal(err)
	}
	return rows
}

func fakeAudiobookshelf(t *testing.T, windowStart time.Time) *httptest.Server {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Header.Get("Authorization") != "Bearer abs-token" {
			writer.WriteHeader(http.StatusUnauthorized)
			return
		}
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/api/me":
			_ = json.NewEncoder(writer).Encode(models.AudiobookshelfUser{ID: "abs-user", Username: "listener"})
		case "/api/me/listening-sessions":
			_ = json.NewEncoder(writer).Encode(models.AudiobookshelfListeningSessionsResponse{Total: 2, Sessions: []models.AudiobookshelfListenSession{
				{ID: "s1", LibraryItemID: "book-1", DisplayTitle: "A Long Book", DisplayAuthor: "Author", MediaType: "book",
					Duration: 36000, TimeListening: 900, StartedAt: windowStart.Add(2 * time.Minute).UnixMilli(), UpdatedAt: windowStart.Add(17 * time.Minute).UnixMilli()},
				{ID: "s2", LibraryItemID: "show-1", EpisodeID: "ep-9", DisplayTitle: "Episode 9", MediaType: "podcast",
					Duration: 1800, TimeListening: 600, StartedAt: windowStart.Add(20 * time.Minute).UnixMilli(), UpdatedAt: windowStart.Add(30 * time.Minute).UnixMilli()},
			}})
		default:
			writer.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(server.Close)
	return server
}

func TestAudiobookshelfConnectSyncAndDisconnect(t *testing.T) {
	h := newAPIHarness(t)
	_, adminToken := h.user("admin@media.test", true)
	user, token := h.user("listener@media.test", false)

	// Media off: the endpoints are hidden.
	h.expect(http.StatusNotFound, "GET", "/api/auth/media/connections", token, nil)

	withMedia(t)
	start := todayAt(10)
	abs := fakeAudiobookshelf(t, start)

	h.expect(http.StatusBadRequest, "POST", "/api/auth/media/audiobookshelf/connect", token, models.AudiobookshelfConnectRequest{ServerURL: "not a url", Token: "abs-token"})
	if code := h.do("POST", "/api/auth/media/audiobookshelf/connect", token, models.AudiobookshelfConnectRequest{ServerURL: abs.URL, Token: "wrong"}).Code; code < 400 {
		t.Errorf("connect with a rejected token: status = %d, want an error", code)
	}
	h.ok("POST", "/api/auth/media/audiobookshelf/connect", token, models.AudiobookshelfConnectRequest{ServerURL: abs.URL, Token: "abs-token"})

	connections := h.ok("GET", "/api/auth/media/connections", token, nil)
	list := field(t, connections, "connections").([]any)
	if len(list) != 1 || field(t, list[0], "provider") != "audiobookshelf" || field(t, list[0], "connected") != true {
		t.Fatalf("connections = %v", list)
	}

	sessionID := timedSession(t, h, token, start)
	h.ok("POST", "/api/auth/exercises/"+sessionID+"/media-sync", token, nil)
	rows := playbackRows(t, sessionID, "audiobookshelf")
	if len(rows) != 2 {
		t.Fatalf("playback rows = %d, want the book and the podcast episode", len(rows))
	}
	kinds := rows[0].MediaType + "," + rows[1].MediaType
	if !strings.Contains(kinds, "audiobook") || !strings.Contains(kinds, "podcast") {
		t.Errorf("media types = %s, want audiobook and podcast", kinds)
	}

	// The soundtrack is on the session read path.
	day := h.ok("GET", "/api/auth/exercise-days/week?today=true", token, nil)
	if !strings.Contains(mustJSON(t, day), "A Long Book") {
		t.Error("the soundtrack does not show on the session")
	}

	// Another user can't pull media into this session.
	_, strangerToken := h.user("stranger@media.test", false)
	if code := h.do("POST", "/api/auth/exercises/"+sessionID+"/media-sync", strangerToken, nil).Code; code < 400 {
		t.Errorf("stranger media sync: status = %d, want an error", code)
	}
	h.expect(http.StatusBadRequest, "POST", "/api/auth/exercises/nope/media-sync", token, nil)

	// Background paths: admin bulk sync and the hourly reconcile are idempotent.
	h.expect(http.StatusAccepted, "POST", "/api/admin/media/sync-for-users", adminToken, models.MediaSyncForUsersRequest{UserIDs: []string{user.ID.String()}})
	MediaSyncForUsers([]models.User{user}, []string{sessionID})
	MediaReconcileForAllUsers()
	if got := len(playbackRows(t, sessionID, "audiobookshelf")); got != 2 {
		t.Errorf("playback rows after re-syncs = %d, want still 2", got)
	}

	h.expect(http.StatusBadRequest, "DELETE", "/api/auth/media/not-a-provider", token, nil)
	h.ok("DELETE", "/api/auth/media/audiobookshelf", token, nil)
	after := h.ok("GET", "/api/auth/media/connections", token, nil)
	if list, _ := after["connections"].([]any); len(list) != 0 {
		t.Errorf("connections after disconnect = %v", list)
	}
}

func TestSpotifyCallbackAndSync(t *testing.T) {
	h := newAPIHarness(t)
	withMedia(t)
	user, token := h.user("spotify@media.test", false)

	start := todayAt(11)
	var tokenCalls atomic.Int32
	stubSpotify(t, func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/api/token":
			tokenCalls.Add(1)
			if request.FormValue("code") == "bad-code" {
				writer.WriteHeader(http.StatusBadRequest)
				return
			}
			_ = json.NewEncoder(writer).Encode(models.SpotifyTokenResponse{AccessToken: "spotify-access", RefreshToken: "spotify-refresh", ExpiresIn: 3600})
		case "/v1/me/player/recently-played":
			played := func(id, name string, minutes int) map[string]any {
				return map[string]any{
					"played_at": start.Add(time.Duration(minutes) * time.Minute).UTC().Format(time.RFC3339),
					"track": map[string]any{"id": id, "uri": "spotify:track:" + id, "name": name, "duration_ms": 180000,
						"artists": []map[string]any{{"name": "Band"}},
						"album":   map[string]any{"id": "al", "name": "Album", "images": []map[string]any{{"url": "https://img/1", "width": 64, "height": 64}}}},
				}
			}
			_ = json.NewEncoder(writer).Encode(map[string]any{"items": []any{played("t1", "Song One", 5), played("t2", "Song Two", 9)}})
		default:
			writer.WriteHeader(http.StatusNotFound)
		}
	})

	if code := h.do("POST", "/api/auth/media/spotify/callback", token, models.SpotifyCallbackRequest{Code: "bad-code"}).Code; code < 400 {
		t.Errorf("rejected code: status = %d, want an error", code)
	}
	h.expect(http.StatusBadRequest, "POST", "/api/auth/media/spotify/callback", token, models.SpotifyCallbackRequest{Code: ""})
	h.ok("POST", "/api/auth/media/spotify/callback", token, models.SpotifyCallbackRequest{Code: "good-code"})

	sessionID := timedSession(t, h, token, start)
	h.ok("POST", "/api/auth/exercises/"+sessionID+"/media-sync", token, nil)
	rows := playbackRows(t, sessionID, "spotify")
	if len(rows) != 2 {
		t.Fatalf("spotify playback rows = %d, want 2", len(rows))
	}

	// An expired access token is refreshed before the pull.
	connection, err := database.GetMediaConnectionForUserProvider(user.ID, models.MediaProviderSpotify)
	if err != nil || connection == nil {
		t.Fatalf("no spotify connection: %v", err)
	}
	past := time.Now().Add(-time.Hour)
	connection.TokenExpiresAt = &past
	if err := database.Instance.Save(connection).Error; err != nil {
		t.Fatal(err)
	}
	before := tokenCalls.Load()
	h.ok("POST", "/api/auth/exercises/"+sessionID+"/media-sync", token, nil)
	if tokenCalls.Load() == before {
		t.Error("an expired Spotify token was not refreshed")
	}
}

func TestPlexPinFlowServerAndSync(t *testing.T) {
	h := newAPIHarness(t)
	withMedia(t)
	user, token := h.user("plex@media.test", false)
	start := todayAt(12)

	pms := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Header.Get("X-Plex-Token") != "plex-token" {
			writer.WriteHeader(http.StatusUnauthorized)
			return
		}
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/library/metadata/101/thumb":
			writer.Header().Set("Content-Type", "image/jpeg")
			_, _ = writer.Write([]byte("fake-jpeg-bytes"))
			return
		case "/identity":
			_, _ = writer.Write([]byte(`{"MediaContainer":{"machineIdentifier":"pms"}}`))
		case "/accounts":
			_, _ = writer.Write([]byte(`{"MediaContainer":{"Account":[{"id":1,"name":"plexuser"},{"id":2,"name":"someone-else"}]}}`))
		case "/library/sections":
			_, _ = writer.Write([]byte(`{"MediaContainer":{"Directory":[{"key":"1","type":"artist","agent":"tv.plex.agents.music","title":"Music"},{"key":"2","type":"artist","agent":"com.plexapp.agents.audnexus","title":"Audiobooks"}]}}`))
		case "/status/sessions/history/all":
			if request.URL.Query().Get("accountID") != "1" {
				t.Errorf("history not scoped to the server-local account: %s", request.URL.RawQuery)
			}
			viewed := func(minutes int) int64 { return start.Add(time.Duration(minutes) * time.Minute).Unix() }
			_ = json.NewEncoder(writer).Encode(map[string]any{"MediaContainer": map[string]any{"size": 3, "Metadata": []map[string]any{
				{"ratingKey": "101", "title": "Track A", "grandparentTitle": "Artist", "type": "track", "librarySectionID": "1", "viewedAt": viewed(4), "accountID": 1, "duration": 200000, "thumb": "/library/metadata/101/thumb"},
				{"ratingKey": "102", "title": "Chapter 1", "type": "track", "librarySectionID": 2, "viewedAt": viewed(20), "accountID": 1, "duration": 900000},
				{"ratingKey": "103", "title": "A Movie", "type": "movie", "librarySectionID": "3", "viewedAt": viewed(25), "accountID": 1},
			}}})
		default:
			writer.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(pms.Close)

	var authorized atomic.Bool
	plexTV := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Header.Get("X-Plex-Client-Identifier") != "test-client-identifier" {
			t.Errorf("plex.tv request without the install's client identifier")
		}
		writer.Header().Set("Content-Type", "application/json")
		switch {
		case request.Method == "POST" && request.URL.Path == "/api/v2/pins":
			writer.WriteHeader(http.StatusCreated)
			_, _ = writer.Write([]byte(`{"id":77,"code":"ABCD"}`))
		case request.URL.Path == "/api/v2/pins/77":
			if authorized.Load() {
				_, _ = writer.Write([]byte(`{"id":77,"code":"ABCD","authToken":"plex-token"}`))
			} else {
				_, _ = writer.Write([]byte(`{"id":77,"code":"ABCD","authToken":null}`))
			}
		case request.URL.Path == "/api/v2/user":
			_, _ = writer.Write([]byte(`{"id":999,"uuid":"u","username":"plexuser","email":"plex@media.test"}`))
		case request.URL.Path == "/api/v2/resources":
			_ = json.NewEncoder(writer).Encode([]models.PlexResource{{
				Name: "Home", Provides: "server", Owned: true,
				Connections: []models.PlexConnection{{URI: pms.URL, Local: true, Protocol: "http"}},
			}})
		default:
			writer.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(plexTV.Close)

	previousPins, previousUser, previousResources := plexPinsURL, plexUserURL, plexResourcesURL
	plexPinsURL, plexUserURL, plexResourcesURL = plexTV.URL+"/api/v2/pins", plexTV.URL+"/api/v2/user", plexTV.URL+"/api/v2/resources"
	t.Cleanup(func() { plexPinsURL, plexUserURL, plexResourcesURL = previousPins, previousUser, previousResources })

	pin := h.ok("POST", "/api/auth/media/plex/pin", token, nil)
	if url, _ := field(t, pin, "pin", "auth_url").(string); !strings.Contains(url, "ABCD") {
		t.Errorf("auth url %q does not carry the PIN code", url)
	}

	// Someone else can't claim this user's PIN.
	_, strangerToken := h.user("stranger@plex.test", false)
	h.expect(http.StatusNotFound, "POST", "/api/auth/media/plex/pin/77/check", strangerToken, nil)
	h.expect(http.StatusBadRequest, "POST", "/api/auth/media/plex/pin/abc/check", token, nil)

	pending := h.ok("POST", "/api/auth/media/plex/pin/77/check", token, nil)
	if field(t, pending, "result", "authorized") != false {
		t.Errorf("unapproved PIN reported authorized: %v", pending)
	}
	authorized.Store(true)
	done := h.ok("POST", "/api/auth/media/plex/pin/77/check", token, nil)
	if field(t, done, "result", "connection", "server_url") != pms.URL {
		t.Errorf("discovered server = %v, want the reachable PMS", field(t, done, "result", "connection", "server_url"))
	}

	sessionID := timedSession(t, h, token, start)
	h.ok("POST", "/api/auth/exercises/"+sessionID+"/media-sync", token, nil)
	rows := playbackRows(t, sessionID, "plex")
	if len(rows) != 2 {
		t.Fatalf("plex playback rows = %d, want the track and the audiobook chapter (the movie is dropped)", len(rows))
	}

	// Manual server override (e.g. behind a reverse proxy) is probed and stored.
	h.expect(http.StatusBadRequest, "PUT", "/api/auth/media/plex/server", token, models.PlexServerURLRequest{ServerURL: "ftp://nope"})
	h.ok("PUT", "/api/auth/media/plex/server", token, models.PlexServerURLRequest{ServerURL: pms.URL})
	h.expect(http.StatusBadRequest, "PUT", "/api/auth/media/plex/server", strangerToken, models.PlexServerURLRequest{ServerURL: pms.URL})

	TriggerMediaSyncForExercise(user, uuid.MustParse(sessionID))

	// Cover art is proxied with the user's own token (an <img> can only send the cookie).
	artworkRequest := httptest.NewRequest("GET", "/api/auth/media/plex/artwork?path=/library/metadata/101/thumb", nil)
	artworkRequest.AddCookie(&http.Cookie{Name: "treningheten", Value: token})
	artwork := serve(h, artworkRequest)
	if artwork.Code != http.StatusOK || artwork.Body.String() != "fake-jpeg-bytes" || artwork.Header().Get("Cache-Control") != "private, max-age=86400" {
		t.Errorf("artwork: status %d, body %q, cache %q", artwork.Code, artwork.Body.String(), artwork.Header().Get("Cache-Control"))
	}
	h.expect(http.StatusBadRequest, "GET", "/api/auth/media/plex/artwork?path=/../../etc/passwd", token, nil)
	h.expect(http.StatusBadRequest, "GET", "/api/auth/media/plex/artwork?path=https://evil.test/x", token, nil)
	h.expect(http.StatusNotFound, "GET", "/api/auth/media/plex/artwork?path=/library/metadata/999/thumb", token, nil)
	h.expect(http.StatusNotFound, "GET", "/api/auth/media/plex/artwork?path=/library/metadata/101/thumb", strangerToken, nil)

	// Admin bulk sync by session id; malformed ids are rejected up front.
	_, adminToken := h.user("admin@plex.test", true)
	h.expect(http.StatusAccepted, "POST", "/api/admin/media/sync-for-users", adminToken, models.MediaSyncForUsersRequest{ExerciseIDs: []string{sessionID}})
	h.expect(http.StatusBadRequest, "POST", "/api/admin/media/sync-for-users", adminToken, models.MediaSyncForUsersRequest{UserIDs: []string{"nope"}})
	h.expect(http.StatusBadRequest, "POST", "/api/admin/media/sync-for-users", adminToken, "{broken")
}

func mustJSON(t *testing.T, value any) string {
	t.Helper()
	encoded, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return string(encoded)
}
