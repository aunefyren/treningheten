package middlewares

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestRateLimitMiddlewareAnswers429WithRetryAfter(t *testing.T) {
	limiter := newSlidingWindow(2, time.Minute)
	router := gin.New()
	router.GET("/limited", rateLimit(limiter), func(context *gin.Context) { context.Status(http.StatusOK) })

	hit := func(ip string) *httptest.ResponseRecorder {
		request := httptest.NewRequest("GET", "/limited", nil)
		request.RemoteAddr = ip + ":1234"
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, request)
		return recorder
	}

	for i := 0; i < 2; i++ {
		if code := hit("10.0.0.1").Code; code != http.StatusOK {
			t.Fatalf("request %d: status = %d, want 200", i+1, code)
		}
	}
	blocked := hit("10.0.0.1")
	if blocked.Code != http.StatusTooManyRequests {
		t.Fatalf("third request: status = %d, want 429", blocked.Code)
	}
	if seconds, err := strconv.Atoi(blocked.Header().Get("Retry-After")); err != nil || seconds < 1 || seconds > 61 {
		t.Errorf("Retry-After = %q, want the seconds until a slot frees", blocked.Header().Get("Retry-After"))
	}
	// Another client is unaffected.
	if code := hit("10.0.0.2").Code; code != http.StatusOK {
		t.Errorf("other client: status = %d, want 200", code)
	}

	// The exported constructors are wired to their limiters.
	for _, middleware := range []gin.HandlerFunc{RateLimitTokenEndpoint(), RateLimitOpenEndpoints(), RateLimitClientRegistration()} {
		if middleware == nil {
			t.Error("nil rate-limit middleware")
		}
	}
}

func TestLockoutAndBusyResponses(t *testing.T) {
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	AbortLoginLocked(context, 90*time.Second)
	if recorder.Code != http.StatusTooManyRequests || recorder.Header().Get("Retry-After") != "91" || !context.IsAborted() {
		t.Errorf("lockout: status %d, Retry-After %q", recorder.Code, recorder.Header().Get("Retry-After"))
	}

	recorder = httptest.NewRecorder()
	context, _ = gin.CreateTestContext(recorder)
	AbortPasswordCostBusy(context)
	if recorder.Code != http.StatusServiceUnavailable || recorder.Header().Get("Retry-After") == "" || !context.IsAborted() {
		t.Errorf("busy: status %d, Retry-After %q", recorder.Code, recorder.Header().Get("Retry-After"))
	}
}
