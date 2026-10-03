package controllers

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/aunefyren/treningheten/database"
	"github.com/aunefyren/treningheten/files"
	"github.com/aunefyren/treningheten/models"

	"github.com/google/uuid"
)

// withInlineIntegrationRecovery runs recovery backfills on the calling goroutine, so a
// test can assert on their result and none outlives the test's database.
func withInlineIntegrationRecovery(t *testing.T) {
	t.Helper()
	previous := startIntegrationRecovery
	startIntegrationRecovery = func(_ string, fn func()) { fn() }
	t.Cleanup(func() { startIntegrationRecovery = previous })
}

// storePlexConnection writes a usable Plex connection with the given token, the way a
// completed PIN flow does.
func storePlexConnection(t *testing.T, userID uuid.UUID, token string, serverURL string) {
	t.Helper()
	accountID := "1"
	if _, err := upsertMediaConnection(userID, models.MediaProviderPlex, token, &serverURL, &accountID); err != nil {
		t.Fatal(err)
	}
}

// seedIntegrationStatus puts a connection in a known health state.
func seedIntegrationStatus(t *testing.T, userID uuid.UUID, status string, failingSince time.Time, notified bool) {
	t.Helper()
	row := models.IntegrationStatus{UserID: userID, Provider: models.IntegrationProviderPlex, Status: status, FailingSince: &failingSince}
	if notified {
		row.NotifiedAt = &failingSince
	}
	if _, err := database.SaveIntegrationStatus(row); err != nil {
		t.Fatal(err)
	}
}

func plexStatusRow(t *testing.T, userID uuid.UUID) *models.IntegrationStatus {
	t.Helper()
	row, err := database.GetIntegrationStatus(userID, models.IntegrationProviderPlex)
	if err != nil {
		t.Fatal(err)
	}
	return row
}

func TestIntegrationErrorClassification(t *testing.T) {
	cases := []struct {
		name    string
		failure error
		want    string
	}{
		{"auth", integrationAuthError("Plex rejected the token."), models.IntegrationStatusAuthFailed},
		{"unavailable", integrationUnavailableError("Plex is down."), models.IntegrationStatusUnavailable},
		{"wrapped auth", fmt.Errorf("sync failed: %w", integrationAuthError("x")), models.IntegrationStatusAuthFailed},
		{"plain error", errors.New("Failed to write rows."), ""},
		{"nil", nil, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := integrationErrorStatus(c.failure); got != c.want {
				t.Errorf("integrationErrorStatus = %q, want %q", got, c.want)
			}
		})
	}

	// The provider's own message survives the tagging.
	if message := integrationAuthError("Plex history rejected the Plex token.").Error(); message != "Plex history rejected the Plex token." {
		t.Errorf("message = %q", message)
	}
}

func TestPlexServerStatusError(t *testing.T) {
	cases := map[int]string{
		http.StatusOK:                  "",
		http.StatusUnauthorized:        models.IntegrationStatusAuthFailed,
		http.StatusForbidden:           models.IntegrationStatusAuthFailed,
		http.StatusNotFound:            models.IntegrationStatusUnavailable,
		http.StatusInternalServerError: models.IntegrationStatusUnavailable,
		http.StatusBadGateway:          models.IntegrationStatusUnavailable,
	}
	for statusCode, want := range cases {
		t.Run(http.StatusText(statusCode), func(t *testing.T) {
			err := plexServerStatusError("Plex history", statusCode)
			if (err == nil) != (want == "") {
				t.Fatalf("plexServerStatusError(%d) = %v", statusCode, err)
			}
			if got := integrationErrorStatus(err); got != want {
				t.Errorf("status = %q, want %q", got, want)
			}
		})
	}
}

func TestIntegrationAlertText(t *testing.T) {
	cases := []struct {
		name     string
		provider string
		status   string
		reason   string
		want     []string
	}{
		{"plex rejected", models.IntegrationProviderPlex, models.IntegrationStatusAuthFailed, "", []string{"Plex connection", "Reconnect"}},
		{"plex unavailable", models.IntegrationProviderPlex, models.IntegrationStatusUnavailable, "", []string{"Plex hasn't responded"}},
		{"plex setup", models.IntegrationProviderPlex, models.IntegrationStatusAuthFailed, models.IntegrationReasonSetupIncomplete, []string{"Plex connection isn't finished"}},
		{"spotify allowlist", models.IntegrationProviderSpotify, models.IntegrationStatusAuthFailed, models.IntegrationReasonNotAllowlisted, []string{"Spotify account", "admin"}},
		{"strava rejected", models.IntegrationProviderStrava, models.IntegrationStatusAuthFailed, "", []string{"Strava connection"}},
		{"hevy rejected", models.IntegrationProviderHevy, models.IntegrationStatusAuthFailed, "", []string{"Hevy connection"}},
		{"audiobookshelf down", models.IntegrationProviderAudiobookshelf, models.IntegrationStatusUnavailable, "", []string{"Audiobookshelf hasn't responded"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			body := integrationAlertBody(c.provider, c.status, c.reason)
			for _, want := range c.want {
				if !strings.Contains(body, want) {
					t.Errorf("body %q lacks %q", body, want)
				}
			}
		})
	}
	if name := integrationDisplayName("somethingelse"); name != "somethingelse" {
		t.Errorf("unknown provider display name = %q", name)
	}
}

func TestRecordIntegrationFailureTransitions(t *testing.T) {
	newControllerTestDB(t)
	user := createTestUser(t, "transitions@plex.test", "Transitions")

	// An error that says nothing about the connection is not tracked.
	recordIntegrationFailure(user.ID, models.IntegrationProviderPlex, errors.New("Failed to write rows."))
	if row := plexStatusRow(t, user.ID); row != nil {
		t.Fatalf("a non-health error created a status row: %+v", row)
	}

	// A first outage is only noted: inside the grace period it still reads as ok.
	recordIntegrationFailure(user.ID, models.IntegrationProviderPlex, integrationUnavailableError("down"))
	row := plexStatusRow(t, user.ID)
	if row == nil || row.Status != models.IntegrationStatusOK || row.FailingSince == nil || row.NotifiedAt != nil {
		t.Fatalf("after a first outage = %+v, want ok with failing_since and no notification", row)
	}
	if health := integrationHealthForUser(user.ID, models.IntegrationProviderPlex); health.Status != models.IntegrationStatusOK || health.FailingSince != nil {
		t.Errorf("shown health inside the grace period = %+v; want ok", health)
	}

	// An earlier gap start (a failed backfill) moves failing_since back.
	earlier := time.Now().Add(-25 * time.Hour)
	recordIntegrationFailureSince(user.ID, models.IntegrationProviderPlex, integrationUnavailableError("down"), earlier)
	row = plexStatusRow(t, user.ID)
	if row.FailingSince == nil || row.FailingSince.Sub(earlier).Abs() > time.Second {
		t.Fatalf("failing_since = %v, want %v", row.FailingSince, earlier)
	}
	// ...and since that is past the grace period, the connection is now unavailable.
	if row.Status != models.IntegrationStatusUnavailable || row.NotifiedAt == nil {
		t.Fatalf("after the grace period = %+v, want unavailable and notified", row)
	}
	firstNotice := *row.NotifiedAt

	// A rejected credential is worse: the status changes and the user is told again.
	recordIntegrationFailure(user.ID, models.IntegrationProviderPlex, integrationAuthError("rejected"))
	row = plexStatusRow(t, user.ID)
	if row.Status != models.IntegrationStatusAuthFailed || row.NotifiedAt == nil || row.NotifiedAt.Equal(firstNotice) {
		t.Fatalf("after an auth failure = %+v, want auth_failed with a fresh notification", row)
	}
	if row.FailingSince.Sub(earlier).Abs() > time.Second {
		t.Errorf("an auth failure moved failing_since to %v", row.FailingSince)
	}

	// The server also going quiet doesn't hide that the user has to reconnect.
	recordIntegrationFailure(user.ID, models.IntegrationProviderPlex, integrationUnavailableError("down"))
	if row = plexStatusRow(t, user.ID); row.Status != models.IntegrationStatusAuthFailed {
		t.Errorf("status after a later outage = %q, want auth_failed", row.Status)
	}
	if health := integrationHealthForUser(user.ID, models.IntegrationProviderPlex); health.Status != models.IntegrationStatusAuthFailed || health.FailingSince == nil {
		t.Errorf("shown health = %+v; want auth_failed with a start time", health)
	}

	// Clearing reports where the gap began, once.
	since, ok := clearIntegrationStatus(user.ID, models.IntegrationProviderPlex)
	if !ok || since.Sub(earlier).Abs() > time.Second {
		t.Errorf("clearIntegrationStatus = %v, %v; want the gap start", since, ok)
	}
	if _, ok := clearIntegrationStatus(user.ID, models.IntegrationProviderPlex); ok {
		t.Errorf("a second clear reported a gap")
	}

	// A row without a gap start (never written by the tracker, but possible by hand) is
	// cleared without a re-pull.
	if _, err := database.SaveIntegrationStatus(models.IntegrationStatus{UserID: user.ID, Provider: models.IntegrationProviderPlex, Status: models.IntegrationStatusAuthFailed}); err != nil {
		t.Fatal(err)
	}
	if _, ok := clearIntegrationStatus(user.ID, models.IntegrationProviderPlex); ok {
		t.Errorf("a row without failing_since reported a gap")
	}
	if row := plexStatusRow(t, user.ID); row != nil {
		t.Errorf("row without failing_since was not cleared: %+v", row)
	}
}

// TestPlexBrokenConnectionIsReportedAndRecovered is the production incident: a revoked
// Plex token silently dropped the soundtrack from new sessions. The user must be told
// once, see it on the account page, and get the missed history back after reconnecting.
func TestPlexBrokenConnectionIsReportedAndRecovered(t *testing.T) {
	h := newAPIHarness(t)
	withMedia(t)
	withVAPIDKeys(t)
	withInlineIntegrationRecovery(t)
	pmsURL := fakePlex(t, todayAt(10))
	user, token := h.user("broken@plex.test", false)

	// One device takes account alerts by default (an older client that doesn't send the
	// field); another has opted out.
	phone, phoneDeliveries := pushEndpoint(t, http.StatusCreated)
	laptop, laptopDeliveries := pushEndpoint(t, http.StatusCreated)
	h.expect(http.StatusCreated, "POST", "/api/auth/notifications/subscribe", token, models.SubscriptionCreationRequest{Subscription: browserSubscription(t, phone.URL)})
	optOut := models.SubscriptionCreationRequest{Subscription: browserSubscription(t, laptop.URL)}
	optOut.Settings.AccountAlert = boolPtr(false)
	h.expect(http.StatusCreated, "POST", "/api/auth/notifications/subscribe", token, optOut)
	stored := h.expect(http.StatusCreated, "POST", "/api/auth/notifications/subscription", token, models.SubscriptionGetRequest{Endpoint: laptop.URL})
	if field(t, stored, "subscription", "account_alert") != false {
		t.Fatalf("opt-out not stored on a new subscription: %v", stored)
	}

	storePlexConnection(t, user.ID, "revoked-token", pmsURL)
	sessionID := timedSession(t, h, token, todayAt(10))

	// The pull fails on the rejected token: no soundtrack, the user gets one push.
	synced := h.ok("POST", "/api/auth/exercises/"+sessionID+"/media-sync", token, nil)
	if warning, _ := synced["warning"].(string); !strings.Contains(warning, "rejected the Plex token") {
		t.Errorf("re-pull warning = %v", synced["warning"])
	}
	if rows := playbackRows(t, sessionID, "plex"); len(rows) != 0 {
		t.Fatalf("plex rows with a revoked token = %d", len(rows))
	}
	if row := plexStatusRow(t, user.ID); row == nil || row.Status != models.IntegrationStatusAuthFailed {
		t.Fatalf("status after the rejected pull = %+v", row)
	}
	if phoneDeliveries.Load() != 1 || laptopDeliveries.Load() != 0 {
		t.Fatalf("deliveries phone/laptop = %d/%d, want 1/0", phoneDeliveries.Load(), laptopDeliveries.Load())
	}

	// Further failures — the daily check, another pull — don't repeat the notice.
	IntegrationHealthCheckForAllUsers()
	h.ok("POST", "/api/auth/exercises/"+sessionID+"/media-sync", token, nil)
	if phoneDeliveries.Load() != 1 {
		t.Errorf("deliveries after repeated failures = %d, want still 1", phoneDeliveries.Load())
	}

	// The account page shows the broken connection.
	connections := h.ok("GET", "/api/auth/media/connections", token, nil)
	list, _ := connections["connections"].([]any)
	if len(list) != 1 || field(t, list[0], "status") != models.IntegrationStatusAuthFailed || field(t, list[0], "failing_since") == nil {
		t.Fatalf("connections = %v", connections)
	}

	// Another provider stamping the session as pulled and settled is what used to hide
	// the gap for good.
	parsedSessionID := uuid.MustParse(sessionID)
	if err := database.SetExerciseMediaRetrievedAt(parsedSessionID, time.Now()); err != nil {
		t.Fatal(err)
	}
	if err := database.SetExerciseMediaSettled(parsedSessionID, true); err != nil {
		t.Fatal(err)
	}

	// The user reconnects; the next check sees a working token and re-pulls the gap.
	storePlexConnection(t, user.ID, "plex-token", pmsURL)
	IntegrationHealthCheckForAllUsers()
	if row := plexStatusRow(t, user.ID); row != nil {
		t.Errorf("status survived the recovery: %+v", row)
	}
	if rows := playbackRows(t, sessionID, "plex"); len(rows) != 1 {
		t.Errorf("plex rows after recovery = %d, want the missed track", len(rows))
	}
	connections = h.ok("GET", "/api/auth/media/connections", token, nil)
	list, _ = connections["connections"].([]any)
	if field(t, list[0], "status") != models.IntegrationStatusOK {
		t.Errorf("status after recovery = %v", field(t, list[0], "status"))
	}

	// A later breakage is a new one: the user is told again.
	storePlexConnection(t, user.ID, "revoked-token", pmsURL)
	IntegrationHealthCheckForAllUsers()
	if phoneDeliveries.Load() != 2 {
		t.Errorf("deliveries after a second breakage = %d, want 2", phoneDeliveries.Load())
	}
}

func TestPlexReconnectRePullsTheGap(t *testing.T) {
	h := newAPIHarness(t)
	withMedia(t)
	withInlineIntegrationRecovery(t)
	pmsURL := fakePlex(t, todayAt(10))
	user, token := h.user("reconnect@plex.test", false)

	storePlexConnection(t, user.ID, "revoked-token", pmsURL)
	sessionID := timedSession(t, h, token, todayAt(10))
	seedIntegrationStatus(t, user.ID, models.IntegrationStatusAuthFailed, time.Now().Add(-time.Hour), true)

	// Running the PIN flow again stores the new token, clears the status and re-pulls.
	h.ok("POST", "/api/auth/media/plex/pin", token, nil)
	done := h.ok("POST", "/api/auth/media/plex/pin/77/check", token, nil)
	if field(t, done, "result", "connection", "status") != models.IntegrationStatusOK {
		t.Errorf("connection status after reconnect = %v", field(t, done, "result", "connection", "status"))
	}
	if row := plexStatusRow(t, user.ID); row != nil {
		t.Errorf("status survived the reconnect: %+v", row)
	}
	if rows := playbackRows(t, sessionID, "plex"); len(rows) != 1 {
		t.Errorf("plex rows after reconnect = %d, want the missed track", len(rows))
	}

	// Fixing the server URL by hand resumes an unavailable connection the same way.
	database.Instance.Where("exercise_id = ?", sessionID).Delete(&models.MediaPlayback{})
	seedIntegrationStatus(t, user.ID, models.IntegrationStatusUnavailable, time.Now().Add(-48*time.Hour), true)
	h.ok("PUT", "/api/auth/media/plex/server", token, models.PlexServerURLRequest{ServerURL: pmsURL})
	if row := plexStatusRow(t, user.ID); row != nil {
		t.Errorf("status survived the server fix: %+v", row)
	}
	if rows := playbackRows(t, sessionID, "plex"); len(rows) != 1 {
		t.Errorf("plex rows after the server fix = %d", len(rows))
	}

	// Disconnecting drops the status with the connection.
	seedIntegrationStatus(t, user.ID, models.IntegrationStatusAuthFailed, time.Now(), true)
	h.ok("DELETE", "/api/auth/media/plex", token, nil)
	if row := plexStatusRow(t, user.ID); row != nil {
		t.Errorf("status survived the disconnect: %+v", row)
	}
}

func TestPlexBackfillKeepsTheGapWhenItFailsAgain(t *testing.T) {
	h := newAPIHarness(t)
	withMedia(t)
	withInlineIntegrationRecovery(t)
	pmsURL := fakePlex(t, todayAt(10))
	user, token := h.user("refail@plex.test", false)

	storePlexConnection(t, user.ID, "revoked-token", pmsURL)
	timedSession(t, h, token, todayAt(10))

	gapStart := time.Now().Add(-3 * time.Hour)
	recoverIntegration(user.ID, models.IntegrationProviderPlex, gapStart)

	row := plexStatusRow(t, user.ID)
	if row == nil || row.Status != models.IntegrationStatusAuthFailed || row.FailingSince == nil || row.FailingSince.Sub(gapStart).Abs() > time.Second {
		t.Fatalf("status after a failed backfill = %+v, want auth_failed from the original gap start", row)
	}
}

func TestPlexHealthCheckEdges(t *testing.T) {
	h := newAPIHarness(t)
	withMedia(t)
	pmsURL := fakePlex(t, todayAt(10))
	user, _ := h.user("edges@plex.test", false)

	// No connection: nothing to check.
	if err := checkMediaConnectionForUser(user.ID, models.MediaProviderPlex); err != nil {
		t.Errorf("no connection: %v", err)
	}

	// A token that can't be decrypted is our problem, not the provider's: not tracked.
	garbage := "not-ciphertext"
	broken := models.MediaConnection{Enabled: true, UserID: user.ID, Provider: models.MediaProviderPlex, AccessToken: &garbage, ServerURL: &pmsURL}
	broken.ID = uuid.New()
	if _, err := database.CreateMediaConnectionInDB(broken); err != nil {
		t.Fatal(err)
	}
	if err := checkMediaConnectionForUser(user.ID, models.MediaProviderPlex); err == nil {
		t.Errorf("an undecryptable token passed the check")
	}
	if row := plexStatusRow(t, user.ID); row != nil {
		t.Errorf("a decrypt failure was recorded as a connection problem: %+v", row)
	}

	// A server that refuses connections, with no other advertised address that answers,
	// is an outage, not a rejected token.
	previousResources := plexResourcesURL
	plexResourcesURL = "http://127.0.0.1:1/api/v2/resources"
	deadURL := "http://127.0.0.1:1"
	storePlexConnection(t, user.ID, "plex-token", deadURL)
	err := checkMediaConnectionForUser(user.ID, models.MediaProviderPlex)
	plexResourcesURL = previousResources
	if integrationErrorStatus(err) != models.IntegrationStatusUnavailable {
		t.Errorf("unreachable server: %v", err)
	}
	if row := plexStatusRow(t, user.ID); row == nil || row.Status != models.IntegrationStatusOK || row.FailingSince == nil {
		t.Errorf("a fresh outage = %+v, want noted but still ok", row)
	}

	// Disabled providers are skipped by the cron entry point (withMedia restores the
	// flags); an unknown provider has nothing to check, sync or recover.
	files.ConfigFile.Media.Plex.Enabled = false
	files.ConfigFile.Media.Spotify.Enabled = false
	files.ConfigFile.Media.Audiobookshelf.Enabled = false
	IntegrationHealthCheckForAllUsers()
	if mediaProviderEnabled("unknown") || mediaProviderSync("unknown") != nil || mediaProviderCheck("unknown", &models.MediaConnection{}) == nil {
		t.Errorf("an unknown media provider is treated as real")
	}
	recoverIntegration(user.ID, "unknown", time.Now().Add(-365*24*time.Hour))
	mediaBackfillSince(user, "unknown", time.Now())
}
