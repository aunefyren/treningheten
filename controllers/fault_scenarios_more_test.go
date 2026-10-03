package controllers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/aunefyren/treningheten/database"
	"github.com/aunefyren/treningheten/files"
	"github.com/aunefyren/treningheten/models"

	"github.com/google/uuid"
)

// callScenario sweeps a direct function call (a sync or scheduled job) instead of a request.
func callScenario(name string, setup func(t *testing.T, h *apiHarness, w faultWorld) func() error) faultScenario {
	return faultScenario{name: name, wantStatus: http.StatusOK, setup: func(t *testing.T, h *apiHarness) faultRequest {
		return faultRequest{call: setup(t, h, buildFaultWorld(t, h))}
	}}
}

func statusError(recorder *httptest.ResponseRecorder, want int) error {
	if recorder.Code != want {
		return fmt.Errorf("status %d: %s", recorder.Code, recorder.Body.String())
	}
	return nil
}

// registerWithPassword creates a user through the real registration path (so the password
// is a real bcrypt hash) and returns the user id.
func registerWithPassword(t *testing.T, h *apiHarness, w faultWorld, email string) uuid.UUID {
	t.Helper()
	invite := h.expect(http.StatusCreated, "POST", "/api/admin/invites", w.adminToken, nil)
	h.expect(http.StatusCreated, "POST", "/api/open/users", "", models.UserCreationRequest{
		FirstName: "Pass", LastName: "Word", Email: email, Password: "Password123", PasswordRepeat: "Password123",
		InviteCode: field(t, invite, "invitation").(string),
	})
	user, err := database.GetAllUserInformationByEmail(email)
	if err != nil {
		t.Fatal(err)
	}
	return user.ID
}

func connectStravaInWorld(t *testing.T, h *apiHarness, w faultWorld) *fakeStrava {
	t.Helper()
	fake := withStrava(t)
	gearID := "g77"
	fake.activities = []models.StravaGetActivitiesRequestReply{
		stravaActivity(3001, "Run", todayAt(1), &gearID, false),
		stravaActivity(3002, "Ride", todayAt(2), nil, true),
	}
	return fake
}

func moreFaultScenarios() []faultScenario {
	return []faultScenario{
		// Accounts and OAuth (these hash/compare real passwords, so they're slower).
		scenario("register user", http.StatusCreated, func(t *testing.T, h *apiHarness, w faultWorld) faultRequest {
			withControllerSMTP(t)
			invite := h.expect(http.StatusCreated, "POST", "/api/admin/invites", w.adminToken, nil)
			return faultRequest{"POST", "/api/open/users", "", models.UserCreationRequest{
				FirstName: "New", LastName: "User", Email: "new@fault.test", Password: "Password123", PasswordRepeat: "Password123",
				InviteCode: field(t, invite, "invitation").(string),
			}, nil}
		}),
		scenario("resend verification", http.StatusOK, func(t *testing.T, h *apiHarness, w faultWorld) faultRequest {
			withControllerSMTP(t)
			return faultRequest{"POST", "/api/open/users/verification", w.memberToken, nil, nil}
		}),
		scenario("verify user", http.StatusOK, func(t *testing.T, h *apiHarness, w faultWorld) faultRequest {
			server := withControllerSMTP(t)
			h.ok("POST", "/api/open/users/verification", w.memberToken, nil)
			code := lastMailMatching(t, server, `this code: <b>([^<]+)</b>`)
			return faultRequest{"POST", "/api/open/users/verify/" + code, w.memberToken, nil, nil}
		}),
		scenario("request password reset", http.StatusOK, func(t *testing.T, h *apiHarness, w faultWorld) faultRequest {
			withControllerSMTP(t)
			files.ConfigFile.TreninghetenExternalURL = "https://trening.test"
			return faultRequest{"POST", "/api/open/users/reset", "", map[string]string{"email": "member@fault.test"}, nil}
		}),
		scenario("check reset code", http.StatusOK, func(t *testing.T, h *apiHarness, w faultWorld) faultRequest {
			code, err := database.GenerateRandomResetCodeForUser(w.member.ID, true)
			if err != nil {
				t.Fatal(err)
			}
			return faultRequest{"GET", "/api/open/users/reset/" + code, "", nil, nil}
		}),
		scenario("change password", http.StatusOK, func(t *testing.T, h *apiHarness, w faultWorld) faultRequest {
			code, err := database.GenerateRandomResetCodeForUser(w.member.ID, true)
			if err != nil {
				t.Fatal(err)
			}
			return faultRequest{"POST", "/api/open/users/password", "", models.UserUpdatePasswordRequest{ResetCode: code, Password: "Password456", PasswordRepeat: "Password456"}, nil}
		}),
		callScenario("password login", func(t *testing.T, h *apiHarness, w faultWorld) func() error {
			registerWithPassword(t, h, w, "login@fault.test")
			return func() error {
				return statusError(h.doForm("/api/oauth/token", url.Values{"grant_type": {"password"}, "client_id": {models.FirstPartyClientID}, "username": {"login@fault.test"}, "password": {"Password123"}}), http.StatusOK)
			}
		}),
		callScenario("update account", func(t *testing.T, h *apiHarness, w faultWorld) func() error {
			userID := registerWithPassword(t, h, w, "update@fault.test")
			_, login := passwordLogin(h, "update@fault.test", "Password123")
			token := login["access_token"].(string)
			birth := time.Now().AddDate(-30, 0, 0)
			maxHR, restHR := 190, 50
			return func() error {
				return statusError(h.do("POST", "/api/auth/users/"+userID.String(), token, models.UserUpdateRequest{
					Email: "update2@fault.test", OldPassword: "Password123", Password: "Password456", PasswordRepeat: "Password456",
					BirthDate: &birth, MaxHeartrate: &maxHR, RestingHeartrate: &restHR, ShareActivities: boolPtr(false), ShareStatistics: boolPtr(true),
				}), http.StatusOK)
			}
		}),
		callScenario("refresh token", func(t *testing.T, h *apiHarness, w faultWorld) func() error {
			registerWithPassword(t, h, w, "refresh@fault.test")
			_, login := passwordLogin(h, "refresh@fault.test", "Password123")
			refresh := login["refresh_token"].(string)
			return func() error {
				return statusError(h.doForm("/api/oauth/token", url.Values{"grant_type": {"refresh_token"}, "client_id": {models.FirstPartyClientID}, "refresh_token": {refresh}}), http.StatusOK)
			}
		}),
		callScenario("authorization code flow", func(t *testing.T, h *apiHarness, w faultWorld) func() error {
			registered := h.expect(http.StatusCreated, "POST", "/api/oauth/register", "", OAuthClientRegistrationRequest{RedirectURIs: []string{"http://127.0.0.1/cb"}})
			clientID := registered["client_id"].(string)
			verifier, challenge := pkcePair()
			return func() error {
				form := url.Values{"client_id": {clientID}, "redirect_uri": {"http://127.0.0.1/cb"}, "code_challenge": {challenge}, "code_challenge_method": {"S256"}, "approve": {"true"}}
				request := newFormRequest("/api/oauth/authorize/decision", form)
				request.Header.Set("Authorization", "Bearer "+w.memberToken)
				decision := serve(h, request)
				if err := statusError(decision, http.StatusOK); err != nil {
					return err
				}
				var body map[string]string
				_ = json.Unmarshal(decision.Body.Bytes(), &body)
				redirect, _ := url.Parse(body["redirect"])
				return statusError(h.doForm("/api/oauth/token", url.Values{"grant_type": {"authorization_code"}, "client_id": {clientID}, "code": {redirect.Query().Get("code")}, "redirect_uri": {"http://127.0.0.1/cb"}, "code_verifier": {verifier}}), http.StatusOK)
			}
		}),
		scenario("register oauth client", http.StatusCreated, func(t *testing.T, h *apiHarness, w faultWorld) faultRequest {
			return faultRequest{"POST", "/api/oauth/register", "", OAuthClientRegistrationRequest{RedirectURIs: []string{"https://a/cb"}, TokenEndpointAuthMethod: "client_secret_post"}, nil}
		}),

		// Strava
		scenario("strava connect and sync", http.StatusOK, func(t *testing.T, h *apiHarness, w faultWorld) faultRequest {
			connectStravaInWorld(t, h, w)
			return faultRequest{"POST", "/api/auth/users/" + w.member.ID.String() + "/strava", w.memberToken, models.UserStravaCodeUpdateRequest{StravaCode: "code"}, nil}
		}),
		callScenario("strava resync", func(t *testing.T, h *apiHarness, w faultWorld) func() error {
			connectStravaInWorld(t, h, w)
			h.ok("POST", "/api/auth/users/"+w.member.ID.String()+"/strava", w.memberToken, models.UserStravaCodeUpdateRequest{StravaCode: "code"})
			user, _ := database.GetAllUserInformation(w.member.ID)
			return func() error { return StravaSyncWeekForUser(user, time.Now()) }
		}),
		callScenario("strava sync by activity id", func(t *testing.T, h *apiHarness, w faultWorld) func() error {
			connectStravaInWorld(t, h, w)
			h.ok("POST", "/api/auth/users/"+w.member.ID.String()+"/strava", w.memberToken, models.UserStravaCodeUpdateRequest{StravaCode: "code"})
			user, _ := database.GetAllUserInformation(w.member.ID)
			return func() error { SyncStravaActivitiesForUsers([]models.User{user}, []string{"3001", "3002"}); return nil }
		}),
		scenario("strava combine", http.StatusCreated, func(t *testing.T, h *apiHarness, w faultWorld) faultRequest {
			connectStravaInWorld(t, h, w)
			h.ok("POST", "/api/auth/users/"+w.member.ID.String()+"/strava", w.memberToken, models.UserStravaCodeUpdateRequest{StravaCode: "code"})
			return faultRequest{"POST", "/api/auth/exercises/strava-combine", w.memberToken, []string{"3001", "3002"}, nil}
		}),
		scenario("strava divide", http.StatusCreated, func(t *testing.T, h *apiHarness, w faultWorld) faultRequest {
			connectStravaInWorld(t, h, w)
			h.ok("POST", "/api/auth/users/"+w.member.ID.String()+"/strava", w.memberToken, models.UserStravaCodeUpdateRequest{StravaCode: "code"})
			h.expect(http.StatusCreated, "POST", "/api/auth/exercises/strava-combine", w.memberToken, []string{"3001", "3002"})
			exercise, _ := database.GetExerciseForUserWithStravaID(w.member.ID, "3001")
			return faultRequest{"POST", "/api/auth/exercises/" + exercise.ID.String() + "/strava-divide", w.memberToken, nil, nil}
		}),
		scenario("strava set resync", http.StatusOK, func(t *testing.T, h *apiHarness, w faultWorld) faultRequest {
			connectStravaInWorld(t, h, w)
			h.ok("POST", "/api/auth/users/"+w.member.ID.String()+"/strava", w.memberToken, models.UserStravaCodeUpdateRequest{StravaCode: "code"})
			exercise, _ := database.GetExerciseForUserWithStravaID(w.member.ID, "3001")
			operations, _ := database.GetOperationsByExerciseID(exercise.ID)
			sets, _ := database.GetOperationSetsByOperationID(operations[0].ID)
			return faultRequest{"POST", "/api/auth/operation-sets/" + sets[0].ID.String() + "/strava-sync", w.memberToken, nil, nil}
		}),
		scenario("strava disconnect", http.StatusOK, func(t *testing.T, h *apiHarness, w faultWorld) faultRequest {
			return faultRequest{"DELETE", "/api/auth/users/" + w.member.ID.String() + "/strava", w.memberToken, nil, nil}
		}),

		// Hevy
		callScenario("hevy backfill", func(t *testing.T, h *apiHarness, w faultWorld) func() error {
			withHevy(t)
			fakeHevyAPI(t, []models.HevyWorkout{hevyWorkout("w1", todayAt(3), "Legs")}, nil)
			encrypted, _ := encryptHevyAPIKey("key")
			database.Instance.Model(&models.User{}).Where("id = ?", w.member.ID).Update("hevy_api_key", encrypted)
			user, _ := database.GetAllUserInformation(w.member.ID)
			return func() error { return HevyBackfillForUser(user) }
		}),
		callScenario("hevy events", func(t *testing.T, h *apiHarness, w faultWorld) func() error {
			withHevy(t)
			first := hevyWorkout("w1", todayAt(3), "Legs")
			fakeHevyAPI(t, []models.HevyWorkout{first, hevyWorkout("w2", todayAt(5), "Arms")}, nil)
			encrypted, _ := encryptHevyAPIKey("key")
			database.Instance.Model(&models.User{}).Where("id = ?", w.member.ID).Update("hevy_api_key", encrypted)
			user, _ := database.GetAllUserInformation(w.member.ID)
			if err := HevyBackfillForUser(user); err != nil {
				t.Fatal(err)
			}
			edited := first
			edited.Exercises = edited.Exercises[:1]
			fakeHevyAPI(t, nil, []models.HevyWorkoutEvent{{Type: "updated", Workout: &edited}, {Type: "deleted", ID: "w2"}})
			user, _ = database.GetAllUserInformation(w.member.ID)
			return func() error { return HevyEventsSyncForUser(user) }
		}),
		scenario("hevy connect", http.StatusOK, func(t *testing.T, h *apiHarness, w faultWorld) faultRequest {
			withHevy(t)
			fakeHevyAPI(t, nil, nil)
			return faultRequest{"POST", "/api/auth/users/" + w.member.ID.String() + "/hevy", w.memberToken, models.UserHevyAPIKeyUpdateRequest{HevyAPIKey: "key"}, nil}
		}),

		// Media
		scenario("audiobookshelf connect", http.StatusOK, func(t *testing.T, h *apiHarness, w faultWorld) faultRequest {
			withMedia(t)
			abs := fakeAudiobookshelf(t, todayAt(10))
			return faultRequest{"POST", "/api/auth/media/audiobookshelf/connect", w.memberToken, models.AudiobookshelfConnectRequest{ServerURL: abs.URL, Token: "abs-token"}, nil}
		}),
		scenario("media sync", http.StatusOK, func(t *testing.T, h *apiHarness, w faultWorld) faultRequest {
			withMedia(t)
			abs := fakeAudiobookshelf(t, todayAt(10))
			h.ok("POST", "/api/auth/media/audiobookshelf/connect", w.memberToken, models.AudiobookshelfConnectRequest{ServerURL: abs.URL, Token: "abs-token"})
			h.ok("PUT", "/api/auth/exercises/"+w.sessionID, w.memberToken, models.ExerciseUpdateRequest{Note: "m", IsOn: true, Duration: int64Ptr(2400), Time: todayAt(10).Format(time.RFC3339)})
			return faultRequest{"POST", "/api/auth/exercises/" + w.sessionID + "/media-sync", w.memberToken, nil, nil}
		}),
		callScenario("media reconcile", func(t *testing.T, h *apiHarness, w faultWorld) func() error {
			withMedia(t)
			abs := fakeAudiobookshelf(t, todayAt(10))
			h.ok("POST", "/api/auth/media/audiobookshelf/connect", w.memberToken, models.AudiobookshelfConnectRequest{ServerURL: abs.URL, Token: "abs-token"})
			h.ok("PUT", "/api/auth/exercises/"+w.sessionID, w.memberToken, models.ExerciseUpdateRequest{Note: "m", IsOn: true, Duration: int64Ptr(2400), Time: todayAt(10).Format(time.RFC3339)})
			return func() error { MediaReconcileForAllUsers(); return nil }
		}),
		scenario("media connections", http.StatusOK, func(t *testing.T, h *apiHarness, w faultWorld) faultRequest {
			withMedia(t)
			abs := fakeAudiobookshelf(t, todayAt(10))
			h.ok("POST", "/api/auth/media/audiobookshelf/connect", w.memberToken, models.AudiobookshelfConnectRequest{ServerURL: abs.URL, Token: "abs-token"})
			return faultRequest{"GET", "/api/auth/media/connections", w.memberToken, nil, nil}
		}),
		scenario("media disconnect", http.StatusOK, func(t *testing.T, h *apiHarness, w faultWorld) faultRequest {
			withMedia(t)
			abs := fakeAudiobookshelf(t, todayAt(10))
			h.ok("POST", "/api/auth/media/audiobookshelf/connect", w.memberToken, models.AudiobookshelfConnectRequest{ServerURL: abs.URL, Token: "abs-token"})
			return faultRequest{"DELETE", "/api/auth/media/audiobookshelf", w.memberToken, nil, nil}
		}),

		// Notifications
		scenario("subscribe push", http.StatusCreated, func(t *testing.T, h *apiHarness, w faultWorld) faultRequest {
			withVAPIDKeys(t)
			endpoint, _ := pushEndpoint(t, http.StatusCreated)
			request := models.SubscriptionCreationRequest{Subscription: browserSubscription(t, endpoint.URL)}
			request.Settings.NewsAlert = true
			// Opting out takes a second write (the column defaults to true); sweep it too.
			request.Settings.AccountAlert = boolPtr(false)
			return faultRequest{"POST", "/api/auth/notifications/subscribe", w.memberToken, request, nil}
		}),
		scenario("push all devices", http.StatusCreated, func(t *testing.T, h *apiHarness, w faultWorld) faultRequest {
			withVAPIDKeys(t)
			endpoint, _ := pushEndpoint(t, http.StatusCreated)
			request := models.SubscriptionCreationRequest{Subscription: browserSubscription(t, endpoint.URL)}
			h.expect(http.StatusCreated, "POST", "/api/auth/notifications/subscribe", w.memberToken, request)
			return faultRequest{"POST", "/api/admin/notifications/push/all-devices", w.adminToken, models.NotificationCreationRequest{Title: "t", Body: "b", UserID: w.member.ID}, nil}
		}),
		scenario("update push settings", http.StatusCreated, func(t *testing.T, h *apiHarness, w faultWorld) faultRequest {
			withVAPIDKeys(t)
			endpoint, _ := pushEndpoint(t, http.StatusCreated)
			h.expect(http.StatusCreated, "POST", "/api/auth/notifications/subscribe", w.memberToken, models.SubscriptionCreationRequest{Subscription: browserSubscription(t, endpoint.URL)})
			return faultRequest{"POST", "/api/auth/notifications/subscription/update", w.memberToken, models.SubscriptionUpdateRequest{Endpoint: endpoint.URL, NewsAlert: true, AccountAlert: boolPtr(false)}, nil}
		}),

		// Scheduled jobs and the rest
		callScenario("process last week", func(t *testing.T, h *apiHarness, w faultWorld) func() error {
			withControllerSMTP(t)
			return func() error { ProcessLastWeek(); return nil }
		}),
		callScenario("sunday reminders", func(t *testing.T, h *apiHarness, w faultWorld) func() error {
			withControllerSMTP(t)
			database.Instance.Model(&models.User{}).Where("id = ?", w.member.ID).Update("sunday_alert", true)
			return func() error { SendSundayReminders(); return nil }
		}),
		scenario("ollama greeting", http.StatusOK, func(t *testing.T, h *apiHarness, w faultWorld) faultRequest {
			withOllama(t)
			return faultRequest{"GET", "/api/auth/ai/frontpage", w.memberToken, nil, nil}
		}),
		scenario("exercise day by weekday", http.StatusOK, func(t *testing.T, h *apiHarness, w faultWorld) faultRequest {
			return faultRequest{"GET", "/api/auth/exercise-days/week?weekDay=1", w.adminToken, nil, nil}
		}),
		scenario("reduce registered week", http.StatusCreated, func(t *testing.T, h *apiHarness, w faultWorld) faultRequest {
			h.expect(http.StatusCreated, "POST", "/api/auth/exercises/week", w.adminToken, models.WeekCreationRequest{Days: currentWeekDays(3), TimeZone: "UTC"})
			return faultRequest{"POST", "/api/auth/exercises/week", w.adminToken, models.WeekCreationRequest{Days: currentWeekDays(1), TimeZone: "UTC"}, nil}
		}),
		scenario("withdraw goal", http.StatusCreated, func(t *testing.T, h *apiHarness, w faultWorld) faultRequest {
			future := seedSeason(t, "Future fault season", time.Now().AddDate(0, 0, 10), time.Now().AddDate(0, 0, 40), false)
			goal := seedGoal(t, w.member.ID, future, 2)
			return faultRequest{"DELETE", "/api/auth/goals/" + goal.ID.String(), w.memberToken, nil, nil}
		}),
		scenario("delete invite", http.StatusOK, func(t *testing.T, h *apiHarness, w faultWorld) faultRequest {
			h.expect(http.StatusCreated, "POST", "/api/admin/invites", w.adminToken, nil)
			invites := field(t, h.ok("GET", "/api/admin/invites", w.adminToken, nil), "invites").([]any)
			return faultRequest{"DELETE", "/api/admin/invites/" + idOf(t, invites[0]), w.adminToken, nil, nil}
		}),
		scenario("revoke pat", http.StatusOK, func(t *testing.T, h *apiHarness, w faultWorld) faultRequest {
			pat := h.expect(http.StatusCreated, "POST", "/api/auth/pats", w.memberToken, models.PATCreationRequest{Name: "ci", Scope: "api:read", ExpiresInDays: 10})
			return faultRequest{"DELETE", "/api/auth/pats/" + idOf(t, pat, "data", "pat"), w.memberToken, nil, nil}
		}),
		scenario("prize received", http.StatusOK, func(t *testing.T, h *apiHarness, w faultWorld) faultRequest {
			database.Instance.Model(&models.Debt{}).Where("id = ?", w.debtID).Update("winner_id", w.admin.ID)
			return faultRequest{"POST", "/api/auth/debts/" + w.debtID.String() + "/received", w.adminToken, nil, nil}
		}),
	}
}

var _ = errors.New
