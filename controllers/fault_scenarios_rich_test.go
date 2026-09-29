package controllers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/aunefyren/treningheten/database"
	"github.com/aunefyren/treningheten/models"

	"github.com/google/uuid"
)

// fakePlex starts a plex.tv + Plex Media Server pair (PIN 77 authorises immediately) and
// points the plex.tv URLs at it. Returns the PMS URL.
func fakePlex(t *testing.T, start time.Time) string {
	t.Helper()
	pms := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if injectUpstreamFault(writer) {
			return
		}
		if request.Header.Get("X-Plex-Token") != "plex-token" {
			writer.WriteHeader(http.StatusUnauthorized)
			return
		}
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/identity":
			_, _ = writer.Write([]byte(`{}`))
		case "/accounts":
			_, _ = writer.Write([]byte(`{"MediaContainer":{"Account":[{"id":1,"name":"plexuser"}]}}`))
		case "/library/sections":
			_, _ = writer.Write([]byte(`{"MediaContainer":{"Directory":[{"key":"1","type":"artist","agent":"tv.plex.agents.music","title":"Music"}]}}`))
		case "/status/sessions/history/all":
			_ = json.NewEncoder(writer).Encode(map[string]any{"MediaContainer": map[string]any{"Metadata": []map[string]any{
				{"ratingKey": "101", "title": "Track", "type": "track", "librarySectionID": "1", "viewedAt": start.Add(5 * time.Minute).Unix(), "accountID": 1, "duration": 200000},
			}}})
		case "/library/metadata/101/thumb":
			writer.Header().Set("Content-Type", "image/jpeg")
			_, _ = writer.Write([]byte("jpeg"))
		default:
			writer.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(pms.Close)

	plexTV := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if injectUpstreamFault(writer) {
			return
		}
		writer.Header().Set("Content-Type", "application/json")
		switch {
		case strings.HasSuffix(request.URL.Path, "/pins") && request.Method == "POST":
			writer.WriteHeader(http.StatusCreated)
			_, _ = writer.Write([]byte(`{"id":77,"code":"ABCD"}`))
		case strings.HasSuffix(request.URL.Path, "/pins/77"):
			_, _ = writer.Write([]byte(`{"id":77,"code":"ABCD","authToken":"plex-token"}`))
		case strings.HasSuffix(request.URL.Path, "/user"):
			_, _ = writer.Write([]byte(`{"id":9,"username":"plexuser","email":"p@x"}`))
		case strings.HasSuffix(request.URL.Path, "/resources"):
			_ = json.NewEncoder(writer).Encode([]models.PlexResource{{Provides: "server", Owned: true, Connections: []models.PlexConnection{{URI: pms.URL, Local: true}}}})
		default:
			writer.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(plexTV.Close)

	previousPins, previousUser, previousResources := plexPinsURL, plexUserURL, plexResourcesURL
	plexPinsURL, plexUserURL, plexResourcesURL = plexTV.URL+"/api/v2/pins", plexTV.URL+"/api/v2/user", plexTV.URL+"/api/v2/resources"
	t.Cleanup(func() { plexPinsURL, plexUserURL, plexResourcesURL = previousPins, previousUser, previousResources })
	return pms.URL
}

// connectPlex runs the PIN flow for the member and gives the session a timed window.
func connectPlex(t *testing.T, h *apiHarness, w faultWorld) string {
	t.Helper()
	withMedia(t)
	pmsURL := fakePlex(t, todayAt(10))
	h.ok("POST", "/api/auth/media/plex/pin", w.memberToken, nil)
	h.ok("POST", "/api/auth/media/plex/pin/77/check", w.memberToken, nil)
	h.ok("PUT", "/api/auth/exercises/"+w.sessionID, w.memberToken, models.ExerciseUpdateRequest{Note: "m", IsOn: true, Duration: int64Ptr(2400), Time: todayAt(10).Format(time.RFC3339)})
	return pmsURL
}

// seedRichWeek logs last week so that many achievement rules fire: every day, three
// sessions and a long note on one day, and the member's birthday on another.
func seedRichWeek(t *testing.T, w faultWorld) {
	t.Helper()
	monday := wednesdayOfWeeksAgo(1).AddDate(0, 0, -2)
	for i := 0; i < 7; i++ {
		count := 1
		if i == 2 {
			count = 3
		}
		seedExerciseDayWithExercises(t, w.admin.ID, monday.AddDate(0, 0, i), count)
	}
	database.Instance.Model(&models.ExerciseDay{}).Where("user_id = ?", w.admin.ID).Update("note", strings.Repeat("A long training diary entry. ", 4))
	birthday := monday.AddDate(-30, 0, 3)
	database.Instance.Model(&models.User{}).Where("id = ?", w.admin.ID).Update("birth_date", birthday)
}

func richFaultScenarios() []faultScenario {
	return []faultScenario{
		scenario("debt with winner", http.StatusOK, func(t *testing.T, h *apiHarness, w faultWorld) faultRequest {
			database.Instance.Model(&models.Debt{}).Where("id = ?", w.debtID).Update("winner_id", w.admin.ID)
			return faultRequest{"GET", "/api/auth/debts/" + w.debtID.String(), w.adminToken, nil, nil}
		}),
		scenario("debt overview with winner", http.StatusOK, func(t *testing.T, h *apiHarness, w faultWorld) faultRequest {
			database.Instance.Model(&models.Debt{}).Where("id = ?", w.debtID).Update("winner_id", w.admin.ID)
			return faultRequest{"GET", "/api/auth/debts", w.adminToken, nil, nil}
		}),
		callScenario("rich week processing", func(t *testing.T, h *apiHarness, w faultWorld) func() error {
			withControllerSMTP(t)
			seedRichWeek(t, w)
			return func() error { return ProcessWeekOfSeason(w.season, wednesdayOfWeeksAgo(1), true, true, nil) }
		}),
		callScenario("season end processing", func(t *testing.T, h *apiHarness, w faultWorld) func() error {
			withControllerSMTP(t)
			seedRichWeek(t, w)
			ended := w.season
			ended.End = wednesdayOfWeeksAgo(1).AddDate(0, 0, 4)
			return func() error { return ProcessWeekOfSeason(ended, wednesdayOfWeeksAgo(1), true, true, nil) }
		}),
		scenario("plex pin", http.StatusOK, func(t *testing.T, h *apiHarness, w faultWorld) faultRequest {
			withMedia(t)
			fakePlex(t, todayAt(10))
			return faultRequest{"POST", "/api/auth/media/plex/pin", w.memberToken, nil, nil}
		}),
		scenario("plex pin check", http.StatusOK, func(t *testing.T, h *apiHarness, w faultWorld) faultRequest {
			withMedia(t)
			fakePlex(t, todayAt(10))
			h.ok("POST", "/api/auth/media/plex/pin", w.memberToken, nil)
			return faultRequest{"POST", "/api/auth/media/plex/pin/77/check", w.memberToken, nil, nil}
		}),
		scenario("plex server override", http.StatusOK, func(t *testing.T, h *apiHarness, w faultWorld) faultRequest {
			pmsURL := connectPlex(t, h, w)
			return faultRequest{"PUT", "/api/auth/media/plex/server", w.memberToken, models.PlexServerURLRequest{ServerURL: pmsURL}, nil}
		}),
		scenario("plex sync", http.StatusOK, func(t *testing.T, h *apiHarness, w faultWorld) faultRequest {
			connectPlex(t, h, w)
			return faultRequest{"POST", "/api/auth/exercises/" + w.sessionID + "/media-sync", w.memberToken, nil, nil}
		}),
		scenario("plex artwork", http.StatusOK, func(t *testing.T, h *apiHarness, w faultWorld) faultRequest {
			connectPlex(t, h, w)
			return faultRequest{"GET", "/api/auth/media/plex/artwork?path=/library/metadata/101/thumb", w.memberToken, nil, nil}
		}),
		scenario("spotify connect", http.StatusOK, func(t *testing.T, h *apiHarness, w faultWorld) faultRequest {
			withMedia(t)
			stubSpotify(t, spotifyFakeHandler(todayAt(10)))
			return faultRequest{"POST", "/api/auth/media/spotify/callback", w.memberToken, models.SpotifyCallbackRequest{Code: "code"}, nil}
		}),
		scenario("spotify sync", http.StatusOK, func(t *testing.T, h *apiHarness, w faultWorld) faultRequest {
			withMedia(t)
			stubSpotify(t, spotifyFakeHandler(todayAt(10)))
			h.ok("POST", "/api/auth/media/spotify/callback", w.memberToken, models.SpotifyCallbackRequest{Code: "code"})
			h.ok("PUT", "/api/auth/exercises/"+w.sessionID, w.memberToken, models.ExerciseUpdateRequest{Note: "m", IsOn: true, Duration: int64Ptr(2400), Time: todayAt(10).Format(time.RFC3339)})
			return faultRequest{"POST", "/api/auth/exercises/" + w.sessionID + "/media-sync", w.memberToken, nil, nil}
		}),
		scenario("hevy disconnect", http.StatusOK, func(t *testing.T, h *apiHarness, w faultWorld) faultRequest {
			withHevy(t)
			encrypted, _ := encryptHevyAPIKey("key")
			database.Instance.Model(&models.User{}).Where("id = ?", w.member.ID).Update("hevy_api_key", encrypted)
			return faultRequest{"DELETE", "/api/auth/users/" + w.member.ID.String() + "/hevy", w.memberToken, nil, nil}
		}),
		scenario("hevy manual sync", http.StatusOK, func(t *testing.T, h *apiHarness, w faultWorld) faultRequest {
			withHevy(t)
			fakeHevyAPI(t, nil, nil)
			encrypted, _ := encryptHevyAPIKey("key")
			database.Instance.Model(&models.User{}).Where("id = ?", w.member.ID).Update("hevy_api_key", encrypted)
			return faultRequest{"POST", "/api/auth/users/" + w.member.ID.String() + "/hevy-sync", w.memberToken, nil, nil}
		}),
		scenario("strava admin sync", http.StatusAccepted, func(t *testing.T, h *apiHarness, w faultWorld) faultRequest {
			connectStravaInWorld(t, h, w)
			h.ok("POST", "/api/auth/users/"+w.member.ID.String()+"/strava", w.memberToken, models.UserStravaCodeUpdateRequest{StravaCode: "code"})
			return faultRequest{"POST", "/api/admin/strava/sync-activities-for-users", w.adminToken, models.StravaSyncActivitiesForUsersRequest{UserIDs: []string{w.member.ID.String()}}, nil}
		}),
		scenario("media admin sync", http.StatusAccepted, func(t *testing.T, h *apiHarness, w faultWorld) faultRequest {
			connectPlex(t, h, w)
			return faultRequest{"POST", "/api/admin/media/sync-for-users", w.adminToken, models.MediaSyncForUsersRequest{UserIDs: []string{w.member.ID.String()}}, nil}
		}),
		scenario("get subscription", http.StatusCreated, func(t *testing.T, h *apiHarness, w faultWorld) faultRequest {
			withVAPIDKeys(t)
			endpoint, _ := pushEndpoint(t, http.StatusCreated)
			h.expect(http.StatusCreated, "POST", "/api/auth/notifications/subscribe", w.memberToken, models.SubscriptionCreationRequest{Subscription: browserSubscription(t, endpoint.URL)})
			return faultRequest{"POST", "/api/auth/notifications/subscription", w.memberToken, models.SubscriptionGetRequest{Endpoint: endpoint.URL}, nil}
		}),
		scenario("profile image upload", http.StatusOK, func(t *testing.T, h *apiHarness, w faultWorld) faultRequest {
			withImageDirs(t)
			userID := registerWithPassword(t, h, w, "photo@fault.test")
			_, login := passwordLogin(h, "photo@fault.test", "Password123")
			return faultRequest{"POST", "/api/auth/users/" + userID.String(), login["access_token"].(string), models.UserUpdateRequest{
				Email: "photo@fault.test", OldPassword: "Password123", ProfileImage: encodedImage(t, "jpeg", noisyImage(120)),
			}, nil}
		}),
		scenario("weight entry", http.StatusOK, func(t *testing.T, h *apiHarness, w faultWorld) faultRequest {
			return faultRequest{"GET", "/api/auth/weights/" + w.weightID, w.memberToken, nil, nil}
		}),
	}
}

// spotifyFakeHandler serves the token exchange and a two-track history around start.
func spotifyFakeHandler(start time.Time) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		if injectUpstreamFault(writer) {
			return
		}
		writer.Header().Set("Content-Type", "application/json")
		if request.URL.Path == "/api/token" {
			_ = json.NewEncoder(writer).Encode(models.SpotifyTokenResponse{AccessToken: "a", RefreshToken: "r", ExpiresIn: 3600})
			return
		}
		item := func(id string, minutes int) map[string]any {
			return map[string]any{"played_at": start.Add(time.Duration(minutes) * time.Minute).UTC().Format(time.RFC3339),
				"track": map[string]any{"id": id, "uri": "spotify:track:" + id, "name": id, "duration_ms": 180000,
					"artists": []map[string]any{{"name": "A"}}, "album": map[string]any{"name": "B", "images": []map[string]any{{"url": "u", "width": 64}}}}}
		}
		_ = json.NewEncoder(writer).Encode(map[string]any{"items": []any{item("t1", 5), item("t2", 9)}})
	}
}

var _ = uuid.New
