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

	for _, route := range h.router.Routes() {
		skip := false
		for _, marker := range externalRoutes {
			if strings.Contains(route.Path, marker) {
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
				id := uuid.NewString()
				params = append(params, gin.Param{Key: segment[1:], Value: id})
				path = strings.Replace(path, segment, id, 1)
			}
		}

		recorder := httptest.NewRecorder()
		context, _ := gin.CreateTestContext(recorder)
		context.Request = httptest.NewRequest(route.Method, path+"?"+robustnessQuery(), strings.NewReader("{}"))
		context.Request.Header.Set("Content-Type", "application/json")
		context.Request.Header.Set("Authorization", "Bearer "+token)
		context.Params = params

		func() {
			defer func() {
				if recovered := recover(); recovered != nil {
					t.Errorf("%s %s panicked: %v", route.Method, route.Path, recovered)
				}
			}()
			route.HandlerFunc(context)
		}()
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
