package controllers

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/aunefyren/treningheten/auth"
	"github.com/aunefyren/treningheten/database"
	"github.com/aunefyren/treningheten/middlewares"
	"github.com/aunefyren/treningheten/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// apiHarness drives handlers through a real gin router with the real Auth middleware, on
// top of the in-memory test DB — so a test exercises routing, token authentication, the
// admin gate and the handler together, the way a client does.
//
// The route table mirrors the /api groups in main.go's initRouter (which lives in package
// main and so can't be called from here). Rate-limit middlewares are left out on purpose:
// their state is process-global and would make tests order-dependent. When adding a route
// to initRouter, add it here too if a test needs it.
type apiHarness struct {
	t      *testing.T
	router *gin.Engine
}

func newAPIHarness(t *testing.T) *apiHarness {
	t.Helper()

	gin.SetMode(gin.TestMode)
	newControllerTestDB(t)
	withTestSigningKey(t)
	// Startup seeds the curated actions and the first-party OAuth client; several read
	// paths (e.g. the "general" action fallback) depend on them.
	database.SeedActions()
	database.SeedOAuthClients()

	router := gin.New()
	api := router.Group("/api")

	oauth := api.Group("/oauth")
	oauth.POST("/token", OAuthToken)
	oauth.GET("/authorize", OAuthAuthorizeInfo)
	oauth.POST("/authorize/decision", middlewares.Auth(false), OAuthAuthorizeDecision)
	oauth.POST("/register", OAuthRegister)
	oauth.POST("/revoke", OAuthRevoke)

	open := api.Group("/open")
	open.POST("/users", RegisterUser)
	open.POST("/users/reset", APIResetPassword)
	open.GET("/users/reset/:resetCode", APIVerifyResetCode)
	open.POST("/users/password", APIChangePassword)
	open.POST("/users/verify/:code", VerifyUser)
	open.POST("/users/verification", SendUserVerificationCode)

	a := api.Group("/auth").Use(middlewares.Auth(false))
	a.POST("/tokens/validate", ValidateToken)
	a.GET("/seasons", APIGetSeasons)
	a.GET("/seasons/:season_id", APIGetSeason)
	a.GET("/seasons/:season_id/weeks", APIGetSeasonWeeks)
	a.GET("/seasons/:season_id/weeks-personal", APIGetSeasonWeeksPersonal)
	a.GET("/seasons/get-on-going", APIGetOngoingSeasons)
	a.GET("/seasons/:season_id/leaderboard", APIGetCurrentSeasonLeaderboard)
	a.GET("/seasons/:season_id/activities", APIGetCurrentSeasonActivities)
	a.POST("/goals", APIRegisterGoalToSeason)
	a.DELETE("/goals/:goal_id", APIDeleteGoalToSeason)
	a.GET("/goals", APIGetGoals)
	a.GET("/exercise-days", APIGetExerciseDays)
	a.GET("/exercise-days/:exercise_day_id", APIGetExerciseDay)
	a.POST("/exercise-days/:exercise_day_id", APIUpdateExerciseDay)
	a.GET("/exercise-days/week", APIGetExerciseDayInWeek)
	a.POST("/exercises/week", APIRegisterWeek)
	a.GET("/exercises/week", APIGetWeek)
	a.POST("/exercises", APICreateExercise)
	a.PUT("/exercises/:exercise_id", APIUpdateExercise)
	a.POST("/exercises/:exercise_id/strava-divide", APIStravaDivide)
	a.POST("/exercises/strava-combine", APIStravaCombine)
	a.GET("/activities", APIGetActivityFeed)
	a.GET("/activities/shared", APIGetSharedActivities)
	a.GET("/operations", APIGetOperationsForUser)
	a.POST("/operations", APICreateOperationForUser)
	a.GET("/operations/:operation_id", APIGetOperation)
	a.PUT("/operations/:operation_id", APIUpdateOperation)
	a.DELETE("/operations/:operation_id", APIDeleteOperation)
	a.GET("/actions", APIGetActions)
	a.POST("/actions", APICreateAction)
	a.GET("/actions/:action_id/statistics", APIGetActionStatistics)
	a.GET("/activity-goal-settings", APIGetActivityGoalSettings)
	a.PUT("/activity-goal-settings", APISetActivityGoalSetting)
	a.GET("/operation-sets", APIGetOperationSets)
	a.POST("/operation-sets", APICreateOperationSetForUser)
	a.PUT("/operation-sets/:operation_set_id", APIUpdateOperationSet)
	a.DELETE("/operation-sets/:operation_set_id", APIDeleteOperationSet)
	a.POST("/operation-sets/:operation_set_id/strava-sync", APISyncStravaOperationSet)
	a.GET("/gear", APIGetGearForUser)
	a.POST("/gear", APICreateGear)
	a.PUT("/gear/:gear_id", APIUpdateGear)
	a.DELETE("/gear/:gear_id", APIDeleteGear)
	a.PUT("/exercises/:exercise_id/gear", APISetGearForExercise)
	a.GET("/weights", APIGetWeightsForUser)
	a.GET("/weights/:weight_id", APIGetWeightForUser)
	a.POST("/weights", APICreateWeightForUser)
	a.DELETE("/weights/:weight_id", APIDeleteWeightForUser)
	a.POST("/sickleave/:season_id", APIRegisterSickleave)
	a.GET("/news", GetNews)
	a.GET("/news/:news_id", GetNewsPost)
	a.GET("/users/:user_id", GetUser)
	a.POST("/users/:user_id/strava", APISetStravaCode)
	a.DELETE("/users/:user_id/strava", APIDeleteStravaConnection)
	a.POST("/users/:user_id/strava-sync", APISyncStravaForUser)
	a.POST("/users/:user_id/hevy", APISetHevyAPIKey)
	a.DELETE("/users/:user_id/hevy", APIDeleteHevyAPIKey)
	a.GET("/media/connections", APIGetMediaConnections)
	a.DELETE("/media/:provider", APIDeleteMediaConnection)
	a.POST("/media/plex/pin", APICreatePlexPin)
	a.POST("/media/plex/pin/:pin_id/check", APICheckPlexPin)
	a.PUT("/media/plex/server", APISetPlexServerURL)
	a.POST("/media/spotify/callback", APISpotifyCallback)
	a.POST("/media/audiobookshelf/connect", APIAudiobookshelfConnect)
	a.POST("/exercises/:exercise_id/media-sync", APISyncMediaForExercise)
	a.POST("/users/:user_id/hevy-sync", APISyncHevyForUser)
	a.GET("/users", GetUsers)
	a.POST("/users/:user_id", UpdateUser)
	a.PATCH("/users/:user_id", APIPartialUpdateUser)
	a.GET("/users/:user_id/activities", APIGetUserActivities)
	a.GET("/users/:user_id/statistics", APIGetUserStatistics)
	a.GET("/debts/unchosen", APIGetUnchosenDebt)
	a.GET("/debts/:debt_id", APIGetDebt)
	a.POST("/debts/:debt_id/choose", APIChooseWinnerForDebt)
	a.GET("/debts", APIGetDebtOverview)
	a.POST("/debts/:debt_id/received", APISetPrizeReceived)
	a.GET("/achievements", APIGetAchievements)
	a.POST("/notifications/subscribe", APISubscribeToNotification)
	a.POST("/notifications/subscription", APIGetSubscriptionForEndpoint)
	a.POST("/notifications/subscription/update", APIUpdateSubscriptionForEndpoint)
	a.GET("/ai/frontpage", APIGetOllamaFrontPageMessageForUser)
	a.POST("/pats", APICreatePersonalAccessToken)
	a.GET("/pats", APIGetPersonalAccessTokens)
	a.DELETE("/pats/:pat_id", APIDeletePersonalAccessToken)

	images := api.Group("/auth").Use(middlewares.AuthImageReadOnly())
	images.GET("/users/:user_id/image", APIGetUserProfileImage)
	images.GET("/achievements/:achievement_id/image", APIGetAchievementsImage)

	admin := api.Group("/admin").Use(middlewares.Auth(true))
	admin.POST("/invites", RegisterInvite)
	admin.GET("/invites", APIGetAllInvites)
	admin.DELETE("/invites/:invite_id", APIDeleteInvite)
	admin.POST("/seasons", APIRegisterSeason)
	admin.POST("/news", RegisterNewsPost)
	admin.DELETE("/news/:news_id", DeleteNewsPost)
	admin.GET("/server-info", APIGetServerInfo)
	admin.GET("/stats", APIGetAdminStats)
	admin.GET("/exercise-days", APIAdminGetExerciseDays)
	admin.POST("/debts", APIGenerateDebtForWeek)
	admin.POST("/users/:user_id/achievement-delegations", APIGiveUserAnAchievement)
	admin.GET("/prizes", APIGetPrizes)
	admin.POST("/prizes", APIRegisterPrize)
	admin.POST("/notifications/push/all-devices", APIPushNotificationToAllDevicesForUser)
	admin.POST("/strava/sync-activities-for-users", APISyncStravaActivitiesForUsers)
	admin.POST("/media/sync-for-users", APISyncMediaForUsers)

	router.GET("/.well-known/oauth-authorization-server", OAuthAuthorizationServerMetadata)
	router.GET("/.well-known/oauth-protected-resource", OAuthProtectedResourceMetadata)

	return &apiHarness{t: t, router: router}
}

// user creates an enabled, verified user and mints an access token for it. An admin gets
// the admin flag in the DB and the admin scope in the token, as a real admin login would.
func (h *apiHarness) user(email string, admin bool) (models.User, string) {
	h.t.Helper()

	user := models.User{FirstName: "Test", LastName: "User", Email: email, Password: "hashed", Enabled: true, Verified: true, Admin: &admin}
	user.ID = uuid.New()
	created, err := database.RegisterUserInDB(user)
	if err != nil {
		h.t.Fatalf("failed to create user %s: %v", email, err)
	}

	token, _, err := auth.GenerateAccessToken(created.ID, admin, auth.ScopeForUser(admin), models.FirstPartyClientID)
	if err != nil {
		h.t.Fatalf("failed to mint access token: %v", err)
	}
	return created, token
}

// do sends a request through the router. body may be nil, a string (sent as-is) or any
// value (JSON-encoded).
func (h *apiHarness) do(method, path, token string, body any) *httptest.ResponseRecorder {
	h.t.Helper()

	var reader io.Reader
	switch b := body.(type) {
	case nil:
	case string:
		reader = bytes.NewBufferString(b)
	default:
		encoded, err := json.Marshal(b)
		if err != nil {
			h.t.Fatalf("failed to marshal body: %v", err)
		}
		reader = bytes.NewBuffer(encoded)
	}

	request := httptest.NewRequest(method, path, reader)
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}

	recorder := httptest.NewRecorder()
	h.router.ServeHTTP(recorder, request)
	return recorder
}

// doForm sends a form-encoded POST, the way OAuth clients call the token endpoint.
func (h *apiHarness) doForm(path string, form url.Values) *httptest.ResponseRecorder {
	h.t.Helper()

	request := httptest.NewRequest("POST", path, strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	recorder := httptest.NewRecorder()
	h.router.ServeHTTP(recorder, request)
	return recorder
}

// decodeBody decodes a JSON object body, or returns an empty map for anything else.
func decodeBody(t *testing.T, body []byte) map[string]any {
	t.Helper()
	decoded := map[string]any{}
	if bytes.HasPrefix(bytes.TrimSpace(body), []byte("{")) {
		if err := json.Unmarshal(body, &decoded); err != nil {
			t.Fatalf("invalid JSON response: %v", err)
		}
	}
	return decoded
}

// expect sends a request and fails the test unless it answers with the wanted status,
// then decodes the JSON body into a generic map.
func (h *apiHarness) expect(want int, method, path, token string, body any) map[string]any {
	h.t.Helper()

	recorder := h.do(method, path, token, body)
	if recorder.Code != want {
		h.t.Fatalf("%s %s: status = %d, want %d; body: %s", method, path, recorder.Code, want, recorder.Body.String())
	}

	decoded := map[string]any{}
	if recorder.Body.Len() > 0 && recorder.Header().Get("Content-Type") != "" &&
		bytes.HasPrefix(bytes.TrimSpace(recorder.Body.Bytes()), []byte("{")) {
		if err := json.Unmarshal(recorder.Body.Bytes(), &decoded); err != nil {
			h.t.Fatalf("%s %s: invalid JSON response: %v", method, path, err)
		}
	}
	return decoded
}

// ok is expect(200, …).
func (h *apiHarness) ok(method, path, token string, body any) map[string]any {
	h.t.Helper()
	return h.expect(http.StatusOK, method, path, token, body)
}

// field walks a decoded JSON object along the given keys, failing if any step is missing.
func field(t *testing.T, value any, keys ...string) any {
	t.Helper()
	for _, key := range keys {
		object, isObject := value.(map[string]any)
		if !isObject {
			t.Fatalf("expected an object at %q, got %T", key, value)
		}
		next, found := object[key]
		if !found {
			t.Fatalf("missing key %q in %v", key, object)
		}
		value = next
	}
	return value
}

// idOf returns the "id" of a decoded JSON object nested under the given keys.
func idOf(t *testing.T, value any, keys ...string) string {
	t.Helper()
	id, isString := field(t, value, append(keys, "id")...).(string)
	if !isString || id == "" {
		t.Fatalf("no id at %v", keys)
	}
	return id
}
