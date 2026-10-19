package controllers

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/aunefyren/treningheten/database"
	"github.com/aunefyren/treningheten/models"
	"github.com/aunefyren/treningheten/utilities"

	"github.com/google/uuid"
)

func statusRow(t *testing.T, userID uuid.UUID, provider string) *models.IntegrationStatus {
	t.Helper()
	row, err := database.GetIntegrationStatus(userID, provider)
	if err != nil {
		t.Fatal(err)
	}
	return row
}

// withDeadPlexResources points plex.tv's server list at a closed port, so rediscovery
// finds nothing.
func withDeadPlexResources(t *testing.T) {
	t.Helper()
	quiesceBackgroundWork(t)
	previous := plexResourcesURL
	plexResourcesURL = "http://127.0.0.1:1/api/v2/resources"
	restoreAfterBackgroundWork(t, func() { plexResourcesURL = previous })
}

func TestPlexRediscoversAMovedServer(t *testing.T) {
	h := newAPIHarness(t)
	withMedia(t)
	withInlineIntegrationRecovery(t)
	pmsURL := fakePlex(t, todayAt(10))
	user, token := h.user("moved@plex.test", false)

	// The stored address went dead (e.g. a new IP) and the server has been unreachable
	// past the grace period.
	storePlexConnection(t, user.ID, "plex-token", "http://127.0.0.1:1")
	sessionID := timedSession(t, h, token, todayAt(10))
	seedIntegrationStatus(t, user.ID, models.IntegrationStatusUnavailable, time.Now().Add(-48*time.Hour), true)

	// The daily check finds the server's current address through plex.tv, switches to it,
	// and the recovery re-pulls the gap.
	if err := checkMediaConnectionForUser(user.ID, models.MediaProviderPlex); err != nil {
		t.Fatalf("check after the server moved: %v", err)
	}
	connection, _ := database.GetMediaConnectionForUserProvider(user.ID, models.MediaProviderPlex)
	if connection.ServerURL == nil || *connection.ServerURL != pmsURL {
		t.Errorf("server URL = %v, want the rediscovered %s", connection.ServerURL, pmsURL)
	}
	if row := plexStatusRow(t, user.ID); row != nil {
		t.Errorf("status survived the rediscovery: %+v", row)
	}
	if rows := playbackRows(t, sessionID, "plex"); len(rows) != 1 {
		t.Errorf("plex rows after rediscovery = %d, want the missed track", len(rows))
	}

	// Nothing better to find: a working server is left alone.
	if changed, _ := plexRediscoverServer(connection, "plex-token"); changed {
		t.Errorf("rediscovery reported a change for a server that is already current")
	}
}

func TestPlexSetupIncompleteIsReported(t *testing.T) {
	h := newAPIHarness(t)
	withMedia(t)
	withInlineIntegrationRecovery(t)
	pmsURL := fakePlex(t, todayAt(10))
	user, token := h.user("setup@plex.test", false)
	sessionID := timedSession(t, h, token, todayAt(10))

	// Connected, but discovery never found a server (and still can't).
	if _, err := upsertMediaConnection(user.ID, models.MediaProviderPlex, "plex-token", nil, nil); err != nil {
		t.Fatal(err)
	}
	previousResources := plexResourcesURL
	plexResourcesURL = "http://127.0.0.1:1/api/v2/resources"
	err := checkMediaConnectionForUser(user.ID, models.MediaProviderPlex)
	if integrationErrorReason(err) != models.IntegrationReasonSetupIncomplete {
		t.Fatalf("check without a server = %v, want setup_incomplete", err)
	}
	row := plexStatusRow(t, user.ID)
	if row == nil || row.Status != models.IntegrationStatusAuthFailed || row.Reason != models.IntegrationReasonSetupIncomplete {
		t.Fatalf("status = %+v, want auth_failed / setup_incomplete", row)
	}
	connections := h.ok("GET", "/api/auth/media/connections", token, nil)
	list, _ := connections["connections"].([]any)
	if len(list) != 1 || field(t, list[0], "status_reason") != models.IntegrationReasonSetupIncomplete {
		t.Errorf("connections = %v", connections)
	}

	// The pull skips the session without failing it, and keeps the status.
	if err := PlexSyncExerciseForUser(user, mustExercise(t, user.ID, sessionID)); err != nil {
		t.Errorf("pull without a server: %v", err)
	}

	// A server but no account id is incomplete too: history can't be scoped to the user.
	if _, err := upsertMediaConnection(user.ID, models.MediaProviderPlex, "plex-token", &pmsURL, nil); err != nil {
		t.Fatal(err)
	}
	if err := PlexSyncExerciseForUser(user, mustExercise(t, user.ID, sessionID)); err != nil {
		t.Errorf("pull without an account id: %v", err)
	}
	if row := plexStatusRow(t, user.ID); row == nil || row.Reason != models.IntegrationReasonSetupIncomplete {
		t.Errorf("status without an account id = %+v", row)
	}

	// Once plex.tv lists a server that answers, the check completes the setup itself.
	plexResourcesURL = previousResources
	if err := checkMediaConnectionForUser(user.ID, models.MediaProviderPlex); err != nil {
		t.Fatalf("check once a server is discoverable: %v", err)
	}
	connection, _ := database.GetMediaConnectionForUserProvider(user.ID, models.MediaProviderPlex)
	if connection.AccountID == nil || *connection.AccountID != "1" {
		t.Errorf("account id = %v, want the resolved server-local id", connection.AccountID)
	}
	if row := plexStatusRow(t, user.ID); row != nil {
		t.Errorf("status survived the completed setup: %+v", row)
	}
	if rows := playbackRows(t, sessionID, "plex"); len(rows) != 1 {
		t.Errorf("plex rows after setup = %d", len(rows))
	}
}

func mustExercise(t *testing.T, userID uuid.UUID, sessionID string) models.Exercise {
	t.Helper()
	exercise, err := database.GetExerciseByIDAndUserID(uuid.MustParse(sessionID), userID)
	if err != nil || exercise == nil {
		t.Fatalf("load session %s: %v", sessionID, err)
	}
	return *exercise
}

func TestSpotifyErrorClassification(t *testing.T) {
	t.Run("token rejection", func(t *testing.T) {
		cases := []struct {
			status int
			body   string
			want   bool
		}{
			{http.StatusUnauthorized, ``, true},
			{http.StatusBadRequest, `{"error":"invalid_grant"}`, true},
			{http.StatusBadRequest, `{"error":"invalid_client"}`, false},
			{http.StatusBadRequest, `not json`, false},
			{http.StatusInternalServerError, `{"error":"invalid_grant"}`, false},
		}
		for _, c := range cases {
			if got := spotifyTokenRejected(c.status, []byte(c.body)); got != c.want {
				t.Errorf("spotifyTokenRejected(%d, %s) = %v, want %v", c.status, c.body, got, c.want)
			}
		}
	})

	for status, want := range map[int]string{
		http.StatusUnauthorized:        models.IntegrationStatusAuthFailed,
		http.StatusForbidden:           models.IntegrationStatusAuthFailed,
		http.StatusInternalServerError: models.IntegrationStatusUnavailable,
		http.StatusOK:                  models.IntegrationStatusUnavailable, // a garbage 200
	} {
		t.Run("history "+http.StatusText(status), func(t *testing.T) {
			stubSpotify(t, func(writer http.ResponseWriter, request *http.Request) {
				writer.WriteHeader(status)
				_, _ = writer.Write([]byte("not json"))
			})
			_, err := spotifyFetchRecentlyPlayed("token")
			if integrationErrorStatus(err) != want {
				t.Errorf("status %d → %v (%q), want %q", status, err, integrationErrorStatus(err), want)
			}
			if status == http.StatusForbidden && (!errors.Is(err, ErrSpotifyForbidden) || integrationErrorReason(err) != models.IntegrationReasonNotAllowlisted) {
				t.Errorf("403 lost its sentinel or reason: %v", err)
			}
		})
	}

	t.Run("token endpoint statuses", func(t *testing.T) {
		stubSpotify(t, func(writer http.ResponseWriter, request *http.Request) {
			writer.WriteHeader(http.StatusBadRequest)
			_, _ = writer.Write([]byte(`{"error":"invalid_grant"}`))
		})
		if _, err := spotifyTokenRequest(nil); integrationErrorStatus(err) != models.IntegrationStatusAuthFailed {
			t.Errorf("invalid_grant → %v", err)
		}
	})

	t.Run("unreachable", func(t *testing.T) {
		server := stubSpotify(t, func(writer http.ResponseWriter, request *http.Request) {})
		server.Close()
		if _, err := spotifyFetchRecentlyPlayed("token"); integrationErrorStatus(err) != models.IntegrationStatusUnavailable {
			t.Errorf("history from a dead host → %v", err)
		}
		if _, err := spotifyTokenRequest(nil); integrationErrorStatus(err) != models.IntegrationStatusUnavailable {
			t.Errorf("token from a dead host → %v", err)
		}
	})
}

// TestSpotifyNotAllowlistedAndRecovery: an account missing from the Spotify app's
// allowlist is reported (reconnecting can't fix it, so the message says to ask the
// admin), and once it works the last day's history is re-pulled.
func TestSpotifyNotAllowlistedAndRecovery(t *testing.T) {
	h := newAPIHarness(t)
	withMedia(t)
	withInlineIntegrationRecovery(t)
	var forbidden atomic.Bool
	forbidden.Store(true)
	ok := spotifyFakeHandler(todayAt(10))
	stubSpotify(t, func(writer http.ResponseWriter, request *http.Request) {
		if forbidden.Load() && request.URL.Path != "/api/token" {
			writer.WriteHeader(http.StatusForbidden)
			return
		}
		ok(writer, request)
	})
	user, token := h.user("allowlist@spotify.test", false)
	if _, err := storeSpotifyTokens(user.ID, models.SpotifyTokenResponse{AccessToken: "a", RefreshToken: "r", ExpiresIn: 3600}); err != nil {
		t.Fatal(err)
	}
	sessionID := timedSession(t, h, token, todayAt(10))

	IntegrationHealthCheckForAllUsers()
	row := statusRow(t, user.ID, models.IntegrationProviderSpotify)
	if row == nil || row.Status != models.IntegrationStatusAuthFailed || row.Reason != models.IntegrationReasonNotAllowlisted {
		t.Fatalf("status = %+v, want auth_failed / not_allowlisted", row)
	}

	// The pull fails the same way and keeps the reason.
	if err := SpotifySyncExerciseForUser(user, mustExercise(t, user.ID, sessionID)); !errors.Is(err, ErrSpotifyForbidden) {
		t.Errorf("pull while not allowlisted = %v", err)
	}

	// The admin adds the account; reconnecting clears the status and re-pulls.
	forbidden.Store(false)
	h.ok("POST", "/api/auth/media/spotify/callback", token, models.SpotifyCallbackRequest{Code: "code"})
	if row := statusRow(t, user.ID, models.IntegrationProviderSpotify); row != nil {
		t.Errorf("status survived the reconnect: %+v", row)
	}
	if rows := playbackRows(t, sessionID, "spotify"); len(rows) != 2 {
		t.Errorf("spotify rows after recovery = %d, want 2", len(rows))
	}

	// Spotify only remembers a day: an older session is not re-pulled however long the
	// gap was.
	old := timedSession(t, h, token, todayAt(10))
	database.Instance.Model(&models.Exercise{}).Where("id = ?", old).Update("created_at", time.Now().Add(-48*time.Hour))
	mediaBackfillSince(user, models.MediaProviderSpotify, time.Now().Add(-72*time.Hour))
	if rows := playbackRows(t, old, "spotify"); len(rows) != 0 {
		t.Errorf("a session older than Spotify's history window was re-pulled: %d rows", len(rows))
	}
}

func TestAudiobookshelfHealth(t *testing.T) {
	for status, want := range map[int]string{
		http.StatusOK:                  "",
		http.StatusUnauthorized:        models.IntegrationStatusAuthFailed,
		http.StatusForbidden:           models.IntegrationStatusAuthFailed,
		http.StatusBadGateway:          models.IntegrationStatusUnavailable,
		http.StatusInternalServerError: models.IntegrationStatusUnavailable,
	} {
		if got := integrationErrorStatus(absStatusError("Audiobookshelf history", status)); got != want {
			t.Errorf("absStatusError(%d) → %q, want %q", status, got, want)
		}
	}

	h := newAPIHarness(t)
	withMedia(t)
	withInlineIntegrationRecovery(t)
	abs := fakeAudiobookshelf(t, todayAt(10))
	user, token := h.user("health@abs.test", false)
	sessionID := timedSession(t, h, token, todayAt(10))

	// A token the server no longer accepts.
	accountID := "abs-user"
	if _, err := upsertMediaConnection(user.ID, models.MediaProviderAudiobookshelf, "old-token", &abs.URL, &accountID); err != nil {
		t.Fatal(err)
	}
	IntegrationHealthCheckForAllUsers()
	if row := statusRow(t, user.ID, models.IntegrationProviderAudiobookshelf); row == nil || row.Status != models.IntegrationStatusAuthFailed {
		t.Fatalf("status = %+v, want auth_failed", row)
	}
	if err := AudiobookshelfSyncExerciseForUser(user, mustExercise(t, user.ID, sessionID)); integrationErrorStatus(err) != models.IntegrationStatusAuthFailed {
		t.Errorf("pull with a rejected token = %v", err)
	}

	// Reconnecting with a fresh token clears it and re-pulls.
	h.ok("POST", "/api/auth/media/audiobookshelf/connect", token, models.AudiobookshelfConnectRequest{ServerURL: abs.URL, Token: "abs-token"})
	if row := statusRow(t, user.ID, models.IntegrationProviderAudiobookshelf); row != nil {
		t.Errorf("status survived the reconnect: %+v", row)
	}
	if rows := playbackRows(t, sessionID, "audiobookshelf"); len(rows) == 0 {
		t.Errorf("no audiobookshelf rows after recovery")
	}

	// The check's own edges: a server that's gone is an outage; no server or an
	// undecryptable token is not checked against the provider at all.
	if _, _, err := absRequest("http://127.0.0.1:1", "/api/me", "abs-token"); integrationErrorStatus(err) != models.IntegrationStatusUnavailable {
		t.Errorf("request to a dead server → %v", err)
	}
	garbage := "not-ciphertext"
	if err := absCheckConnection(&models.MediaConnection{AccessToken: &garbage}); err != nil {
		t.Errorf("check without a server = %v", err)
	}
	if err := absCheckConnection(&models.MediaConnection{AccessToken: &garbage, ServerURL: &abs.URL}); err == nil || integrationErrorStatus(err) != "" {
		t.Errorf("undecryptable token = %v, want a plain error", err)
	}
}

func TestHevyRejectedKeyIsReported(t *testing.T) {
	for status, want := range map[int]string{
		http.StatusUnauthorized:        models.IntegrationStatusAuthFailed,
		http.StatusForbidden:           models.IntegrationStatusAuthFailed,
		http.StatusInternalServerError: models.IntegrationStatusUnavailable,
	} {
		stubHevyAPI(t, func(writer http.ResponseWriter, request *http.Request) { writer.WriteHeader(status) })
		if _, err := hevyAPIGet("key", "/user/info"); integrationErrorStatus(err) != want {
			t.Errorf("Hevy %d → %v, want %q", status, err, want)
		}
	}
	previous := hevyAPIBaseURL
	hevyAPIBaseURL = "http://127.0.0.1:1"
	if _, err := hevyAPIGet("key", "/user/info"); integrationErrorStatus(err) != models.IntegrationStatusUnavailable {
		t.Errorf("Hevy unreachable → %v", err)
	}
	hevyAPIBaseURL = previous

	h := newAPIHarness(t)
	withHevy(t)
	fakeHevyAPI(t, nil, nil)
	user, token := h.user("rejected@hevy.test", false)
	userPath := "/api/auth/users/" + user.ID.String()

	setKey := func(key string) models.User {
		encrypted, _ := encryptHevyAPIKey(key)
		database.Instance.Model(&models.User{}).Where("id = ?", user.ID).Updates(map[string]any{"hevy_api_key": encrypted, "hevy_last_sync": time.Now().Add(-time.Hour)})
		stored, _ := database.GetAllUserInformation(user.ID)
		return stored
	}

	// The key was regenerated (or PRO lapsed): the hourly sync reports it.
	if err := HevyEventsSyncForUser(setKey("rejected-key")); integrationErrorStatus(err) != models.IntegrationStatusAuthFailed {
		t.Fatalf("events sync with a rejected key = %v", err)
	}
	self := h.ok("GET", userPath, token, nil)
	if field(t, self, "user", "integration_health", "hevy", "status") != models.IntegrationStatusAuthFailed {
		t.Errorf("own user health = %v", field(t, self, "user", "integration_health"))
	}
	if err := HevyBackfillForUser(setKey("rejected-key")); integrationErrorStatus(err) != models.IntegrationStatusAuthFailed {
		t.Errorf("backfill with a rejected key = %v", err)
	}

	// A working key: the next run clears the status (the baseline never moved, so that
	// run already covers the gap).
	if err := HevyEventsSyncForUser(setKey("key")); err != nil {
		t.Fatalf("events sync with a working key: %v", err)
	}
	if row := statusRow(t, user.ID, models.IntegrationProviderHevy); row != nil {
		t.Errorf("status survived a working sync: %+v", row)
	}
	recoverIntegration(user.ID, models.IntegrationProviderHevy, time.Now().Add(-time.Hour))

	// Disconnecting drops a status along with the key.
	recordIntegrationFailure(user.ID, models.IntegrationProviderHevy, integrationAuthError("the Hevy API key was rejected"))
	h.ok("DELETE", userPath+"/hevy", token, nil)
	if row := statusRow(t, user.ID, models.IntegrationProviderHevy); row != nil {
		t.Errorf("status survived the Hevy disconnect: %+v", row)
	}
}

func TestStravaBackfillReSyncsMissedWeeks(t *testing.T) {
	h := newAPIHarness(t)
	fake := withStrava(t)
	withInlineIntegrationRecovery(t)
	user, token := h.user("backfill@strava.test", false)
	h.ok("POST", "/api/auth/users/"+user.ID.String()+"/strava", token, models.UserStravaCodeUpdateRequest{StravaCode: "code"})
	stored, _ := database.GetAllUserInformation(user.ID)

	// A gap inside the current week has no earlier weeks to re-sync.
	before := fake.tokenCalls.Load()
	stravaBackfillSince(stored, time.Now())
	if fake.tokenCalls.Load() != before {
		t.Errorf("a gap in the current week re-synced %d weeks", fake.tokenCalls.Load()-before)
	}

	// A gap of 15 days re-syncs every earlier week it touched, one token exchange each.
	since := time.Now().Add(-15 * 24 * time.Hour)
	currentMonday, _ := utilities.FindEarlierMonday(time.Now())
	wantWeeks := 0
	for week := since; ; week = week.AddDate(0, 0, 7) {
		monday, _ := utilities.FindEarlierMonday(week)
		if !monday.Before(currentMonday) {
			break
		}
		wantWeeks++
	}
	before = fake.tokenCalls.Load()
	stravaBackfillSince(stored, since)
	if got := int(fake.tokenCalls.Load() - before); got != wantWeeks {
		t.Errorf("weeks re-synced = %d, want %d", got, wantWeeks)
	}

	// Revoked part-way: the backfill stops and keeps the gap's original start.
	fake.revoke.Store(true)
	stored, _ = database.GetAllUserInformation(user.ID)
	stravaBackfillSince(stored, since)
	row := statusRow(t, user.ID, models.IntegrationProviderStrava)
	if row == nil || row.Status != models.IntegrationStatusAuthFailed || row.FailingSince == nil || row.FailingSince.Sub(since).Abs() > time.Second {
		t.Errorf("status after a failed backfill = %+v, want auth_failed from %v", row, since)
	}
}

// withPlexTVUserStatus points plex.tv's account lookup at a stub that answers with
// status (0: unreachable), and its server list at a closed port.
func withPlexTVUserStatus(t *testing.T, status int) {
	t.Helper()
	withDeadPlexResources(t)
	quiesceBackgroundWork(t)
	previous := plexUserURL
	restoreAfterBackgroundWork(t, func() { plexUserURL = previous })
	if status == 0 {
		plexUserURL = "http://127.0.0.1:1/api/v2/user"
		return
	}
	plexTV := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(status)
		if status == http.StatusOK {
			_, _ = writer.Write([]byte(`{"id":9,"username":"plexuser","email":"p@x"}`))
		}
	}))
	t.Cleanup(plexTV.Close)
	plexUserURL = plexTV.URL + "/api/v2/user"
}

// A Plex whose server has gone quiet can also have had its token revoked at plex.tv. The
// rediscovery's plex.tv lookup tells the two apart, so the user is told to reconnect
// rather than to wait for a server that, once back, would still refuse the token.
func TestPlexCheckEscalatesARevokedToken(t *testing.T) {
	cases := []struct {
		name       string
		incomplete bool
		plexTV     int
		wantStatus string
		wantReason string
	}{
		{"server down, token revoked", false, http.StatusUnauthorized, models.IntegrationStatusAuthFailed, ""},
		{"server down, token forbidden", false, http.StatusForbidden, models.IntegrationStatusAuthFailed, ""},
		{"server down, plex.tv down too", false, http.StatusInternalServerError, models.IntegrationStatusUnavailable, ""},
		{"server down, plex.tv unreachable", false, 0, models.IntegrationStatusUnavailable, ""},
		{"server down, token fine", false, http.StatusOK, models.IntegrationStatusUnavailable, ""},
		{"setup unfinished, token revoked", true, http.StatusUnauthorized, models.IntegrationStatusAuthFailed, ""},
		{"setup unfinished, plex.tv down", true, http.StatusBadGateway, models.IntegrationStatusAuthFailed, models.IntegrationReasonSetupIncomplete},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			newControllerTestDB(t)
			withMedia(t)
			withPlexTVUserStatus(t, c.plexTV)
			user := createTestUser(t, "revoked@plex.test", "Revoked")

			if c.incomplete {
				if _, err := upsertMediaConnection(user.ID, models.MediaProviderPlex, "plex-token", nil, nil); err != nil {
					t.Fatal(err)
				}
			} else {
				storePlexConnection(t, user.ID, "plex-token", "http://127.0.0.1:1")
			}
			connection, err := database.GetMediaConnectionForUserProvider(user.ID, models.MediaProviderPlex)
			if err != nil || connection == nil {
				t.Fatalf("load connection: %v", err)
			}

			err = plexCheckConnection(connection)
			if got := integrationErrorStatus(err); got != c.wantStatus {
				t.Errorf("status = %q (%v), want %q", got, err, c.wantStatus)
			}
			if got := integrationErrorReason(err); got != c.wantReason {
				t.Errorf("reason = %q, want %q", got, c.wantReason)
			}
		})
	}
}

func TestPlexFetchAccountAndServerClassification(t *testing.T) {
	cases := map[int]string{
		http.StatusUnauthorized:        models.IntegrationStatusAuthFailed,
		http.StatusForbidden:           models.IntegrationStatusAuthFailed,
		http.StatusInternalServerError: models.IntegrationStatusUnavailable,
		http.StatusNotFound:            models.IntegrationStatusUnavailable,
	}
	withMedia(t)
	for status, want := range cases {
		t.Run(http.StatusText(status), func(t *testing.T) {
			withPlexTVUserStatus(t, status)
			_, _, err := plexFetchAccountAndServer("plex-token")
			if got := integrationErrorStatus(err); got != want {
				t.Errorf("status = %q (%v), want %q", got, err, want)
			}
			// The PIN flow's message is unchanged by the tagging.
			if err == nil || err.Error() != "Plex user lookup failed." {
				t.Errorf("message = %v", err)
			}
		})
	}

	// A reply that isn't an account is the provider misbehaving: an outage.
	plexTV := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		_, _ = writer.Write([]byte("not json"))
	}))
	t.Cleanup(plexTV.Close)
	quiesceBackgroundWork(t)
	previous := plexUserURL
	plexUserURL = plexTV.URL
	restoreAfterBackgroundWork(t, func() { plexUserURL = previous })
	if _, _, err := plexFetchAccountAndServer("plex-token"); integrationErrorStatus(err) != models.IntegrationStatusUnavailable {
		t.Errorf("unparseable account = %v, want an outage", err)
	}
}
