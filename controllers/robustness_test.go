package controllers

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/aunefyren/treningheten/auth"
	"github.com/aunefyren/treningheten/database"
	"github.com/aunefyren/treningheten/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// externalRoutes can reach third-party services when their integration is enabled; they
// are gated off in the harness config, but skip them so this test can never make a real
// network call.
var externalRoutes = []string{"strava", "hevy", "plex", "spotify", "audiobookshelf", "media", "/ai/"}

// robustnessQuery fills every query parameter a handler reads with a plausible value, so
// requests get past parameter validation and into the data layer.
func robustnessQuery() string {
	now := time.Now().UTC()
	values := url.Values{
		"today": {"true"}, "year": {now.Format("2006")}, "goal": {uuid.NewString()}, "operation_id": {uuid.NewString()},
		"start": {now.AddDate(0, -1, 0).Format(time.RFC3339)}, "end": {now.Format(time.RFC3339)},
		"user": {uuid.NewString()}, "countdown": {"true"}, "potential": {"true"}, "thumbnail": {"true"},
		"client_id": {"treningheten-web"}, "redirect_uri": {"http://x"}, "response_type": {"code"},
	}
	return values.Encode()
}

// callEveryHandler invokes each routed controller directly — bypassing the auth
// middleware, which would itself need the database — as an admin with a valid token and
// random path ids. It fails the test on any panic.
func callEveryHandler(t *testing.T, h *apiHarness, token string) (calls int) {
	t.Helper()
	return callEveryHandlerWith(t, h, token, "", "{}", nil)
}

// callEveryHandlerWith is callEveryHandler with a fixed path-parameter value (empty means
// a fresh UUID), a raw body, and an optional check on each response.
func callEveryHandlerWith(t *testing.T, h *apiHarness, token string, paramValue string, body string,
	check func(t *testing.T, method, path string, recorder *httptest.ResponseRecorder)) (calls int) {
	t.Helper()

	// Without a valid token or with a malformed id, every handler returns before any
	// network call (and the integrations are gated off in the harness anyway), so only
	// the fully valid sweep needs to skip the external routes.
	skipExternal := token != "" && paramValue == "" && body == "{}"
	for _, route := range h.router.Routes() {
		skip := false
		for _, marker := range externalRoutes {
			if skipExternal && strings.Contains(route.Path, marker) {
				skip = true
			}
		}
		if skip {
			continue
		}

		path := route.Path
		var params gin.Params
		for _, segment := range strings.Split(route.Path, "/") {
			if strings.HasPrefix(segment, ":") {
				id := paramValue
				if id == "" {
					id = uuid.NewString()
				}
				params = append(params, gin.Param{Key: segment[1:], Value: id})
				path = strings.Replace(path, segment, id, 1)
			}
		}

		recorder := httptest.NewRecorder()
		context, _ := gin.CreateTestContext(recorder)
		context.Request = httptest.NewRequest(route.Method, path+"?"+robustnessQuery(), strings.NewReader(body))
		context.Request.Header.Set("Content-Type", "application/json")
		if token != "" {
			context.Request.Header.Set("Authorization", "Bearer "+token)
		}
		context.Params = params

		func() {
			defer func() {
				if recovered := recover(); recovered != nil {
					t.Errorf("%s %s panicked: %v", route.Method, route.Path, recovered)
				}
			}()
			route.HandlerFunc(context)
		}()
		if check != nil {
			check(t, route.Method, route.Path, recorder)
		}
		calls++
	}
	return calls
}

// Every handler must answer unknown ids with an error response, never a panic — the
// nil-error dereferences fixed earlier ("err.Error()" on a not-found branch) were exactly
// this shape.
func TestHandlersSurviveUnknownIDs(t *testing.T) {
	h := newAPIHarness(t)
	admin, token := h.user("robust@robust.test", true)
	_ = admin

	if calls := callEveryHandler(t, h, token); calls < 80 {
		t.Errorf("only %d handlers exercised; did the route table shrink?", calls)
	}
}

// With the database gone, handlers must fail with an error response, not panic or claim
// success.
func TestHandlersSurviveADeadDatabase(t *testing.T) {
	h := newAPIHarness(t)
	admin, _ := h.user("deaddb@robust.test", true)
	token, _, err := auth.GenerateAccessToken(admin.ID, true, auth.ScopeForUser(true), models.FirstPartyClientID)
	if err != nil {
		t.Fatal(err)
	}

	sqlDB, err := database.Instance.DB()
	if err != nil {
		t.Fatal(err)
	}
	_ = sqlDB.Close()

	callEveryHandler(t, h, token)

	// Spot-check that data endpoints report the failure rather than an empty success.
	for _, path := range []string{"/api/auth/seasons", "/api/auth/goals", "/api/auth/weights", "/api/auth/gear", "/api/auth/news"} {
		for _, route := range h.router.Routes() {
			if route.Method != "GET" || route.Path != path {
				continue
			}
			recorder := httptest.NewRecorder()
			context, _ := gin.CreateTestContext(recorder)
			context.Request = httptest.NewRequest("GET", path, nil)
			context.Request.Header.Set("Authorization", "Bearer "+token)
			route.HandlerFunc(context)
			if recorder.Code < http.StatusBadRequest {
				t.Errorf("GET %s with the database down: status = %d, want an error", path, recorder.Code)
			}
		}
	}
	_ = fmt.Sprint()
}

// Called without a valid token — as if mounted without the auth middleware — handlers
// must not panic. (Several admin handlers never read the caller and rely on the router's
// middleware alone; main_test.go's TestEveryProtectedRouteRejectsAnonymousCallers checks
// that every such route actually sits behind it.)
func TestHandlersSurviveCallsWithoutAValidToken(t *testing.T) {
	h := newAPIHarness(t)
	// Integrations on, so their handlers get past the "enabled" gate to the token check.
	// Without a token each stops there, before any network call.
	withMedia(t)
	withHevy(t)
	withStrava(t)
	callEveryHandlerWith(t, h, "", "", "{}", nil)
	callEveryHandlerWith(t, h, "forged.token.value", "", "{}", nil)
}

// Malformed path ids and request bodies are client errors, never panics.
func TestHandlersRejectMalformedInput(t *testing.T) {
	h := newAPIHarness(t)
	_, token := h.user("malformed@robust.test", true)

	callEveryHandlerWith(t, h, token, "not-a-uuid", "{}", nil)
	callEveryHandlerWith(t, h, token, "", "{this is not json", nil)
	callEveryHandlerWith(t, h, token, "", `{"date":"yesterday","exercise_interval":"many","weight":"heavy"}`, nil)
}

// Query parameters that don't parse are client errors.
func TestHandlersRejectMalformedQueryParameters(t *testing.T) {
	h := newAPIHarness(t)
	_, token := h.user("query@robust.test", false)

	for _, path := range []string{
		"/api/auth/exercise-days?year=twenty",
		"/api/auth/exercise-days?goal=nope",
		"/api/auth/exercise-days/week?weekDay=eight",
		"/api/auth/achievements?user=nope",
		"/api/auth/actions/" + uuid.NewString() + "/statistics?start=2026-01-01T00:00:00Z&end=later",
	} {
		if code := h.do("GET", path, token, nil).Code; code != http.StatusBadRequest {
			t.Errorf("GET %s: status = %d, want 400", path, code)
		}
	}
	if code := h.do("POST", "/api/auth/users/"+uuid.NewString()+"/strava-sync?pointInTime=soon", token, nil).Code; code != http.StatusBadRequest {
		t.Errorf("strava-sync with a bad pointInTime: status = %d, want 400", code)
	}
}
