package controllers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/aunefyren/treningheten/files"
	"github.com/aunefyren/treningheten/models"
)

// fakeOllama is an OpenAI-compatible chat endpoint that records the user payloads it is
// sent and answers with a fixed greeting (or a configured failure).
type fakeOllama struct {
	mutex    sync.Mutex
	payloads []string
	auth     []string
	status   int
	body     string
}

func (f *fakeOllama) requests() int {
	f.mutex.Lock()
	defer f.mutex.Unlock()
	return len(f.payloads)
}

func withOllama(t *testing.T) *fakeOllama {
	t.Helper()

	fake := &fakeOllama{status: http.StatusOK, body: `{"choices":[{"message":{"role":"assistant","content":"  Nice Tuesday work!  "}}]}`}
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if injectUpstreamFault(writer) {
			return
		}
		if request.URL.Path != "/v1/chat/completions" {
			t.Errorf("unexpected Ollama path %s", request.URL.Path)
		}
		var chat ollamaChatRequest
		if err := json.NewDecoder(request.Body).Decode(&chat); err != nil {
			t.Errorf("bad chat request: %v", err)
		}
		fake.mutex.Lock()
		if len(chat.Messages) == 2 {
			fake.payloads = append(fake.payloads, chat.Messages[1].Content)
		}
		fake.auth = append(fake.auth, request.Header.Get("Authorization"))
		status, body := fake.status, fake.body
		fake.mutex.Unlock()
		writer.WriteHeader(status)
		_, _ = writer.Write([]byte(body))
	}))
	t.Cleanup(server.Close)

	previous := files.ConfigFile.Ollama
	files.ConfigFile.Ollama = models.OllamaSettings{Enabled: true, URL: server.URL, Model: "tiny", APIKey: "secret"}
	t.Cleanup(func() { files.ConfigFile.Ollama = previous })

	ollamaCacheMu.Lock()
	ollamaCache = map[string]ollamaCacheEntry{}
	ollamaCacheMu.Unlock()
	return fake
}

func TestOllamaFrontPageMessage(t *testing.T) {
	h := newAPIHarness(t)
	user, token := h.user("coachme@ollama.test", false)

	// Disabled: the endpoint reports a failure rather than calling anything.
	h.expect(http.StatusBadRequest, "GET", "/api/auth/ai/frontpage", token, nil)

	fake := withOllama(t)

	season := seedOngoingSeason(t, "Coached season", 2, 6)
	seedGoal(t, user.ID, season, 3)
	sessionID := createTodaySession(t, h, token)
	h.expect(http.StatusCreated, "POST", "/api/auth/operations", token, models.OperationCreationRequest{
		ExerciseID: mustUUID(t, sessionID), Type: "moving", WeightUnit: "kg", DistanceUnit: "km",
	})

	message := h.ok("GET", "/api/auth/ai/frontpage", token, nil)
	if message["data"] != "Nice Tuesday work!" {
		t.Errorf("message = %v, want the trimmed model reply", message["data"])
	}
	if fake.requests() != 1 {
		t.Fatalf("model calls = %d, want 1", fake.requests())
	}
	if fake.auth[0] != "Bearer secret" {
		t.Errorf("Authorization = %q, want the configured API key", fake.auth[0])
	}

	var payload map[string]any
	if err := json.Unmarshal([]byte(fake.payloads[0]), &payload); err != nil {
		t.Fatalf("payload is not JSON: %v\n%s", err, fake.payloads[0])
	}
	if payload["workouts_logged_this_week"] != float64(1) {
		t.Errorf("workouts_logged_this_week = %v, want 1", payload["workouts_logged_this_week"])
	}
	if payload["in_any_season"] != true || !strings.Contains(fake.payloads[0], "Coached season") {
		t.Errorf("season facts missing from payload: %s", fake.payloads[0])
	}

	// Same facts → served from cache, no second model call.
	h.ok("GET", "/api/auth/ai/frontpage", token, nil)
	if fake.requests() != 1 {
		t.Errorf("model calls after a cached read = %d, want still 1", fake.requests())
	}

	// New facts invalidate the cache; the async refresh and the startup pre-cache run too.
	createTodaySession(t, h, token)
	OllamaAsyncRefreshCacheForUser(user.ID)
	if fake.requests() != 2 {
		t.Errorf("model calls after new data = %d, want 2", fake.requests())
	}
	OllamaPreCacheForAllUsers()

	// Failures surface as errors, not as a cached empty message.
	for name, response := range map[string]struct {
		status int
		body   string
	}{
		"server error": {http.StatusInternalServerError, `{}`},
		"no choices":   {http.StatusOK, `{"choices":[]}`},
		"garbage":      {http.StatusOK, `not json`},
	} {
		fake.mutex.Lock()
		fake.status, fake.body = response.status, response.body
		fake.mutex.Unlock()
		ollamaCacheMu.Lock()
		ollamaCache = map[string]ollamaCacheEntry{} // force a fresh model call
		ollamaCacheMu.Unlock()
		if code := h.do("GET", "/api/auth/ai/frontpage", token, nil).Code; code != http.StatusBadRequest {
			t.Errorf("%s: status = %d, want 400", name, code)
		}
	}
}

func TestOllamaConfigurationGaps(t *testing.T) {
	newAPIHarness(t)
	withOllama(t)
	user := makeControllerTestUser(t, "config@ollama.test")

	for name, settings := range map[string]models.OllamaSettings{
		"no url":   {Enabled: true, Model: "tiny"},
		"no model": {Enabled: true, URL: "http://127.0.0.1:1"},
		"offline":  {Enabled: true, URL: "http://127.0.0.1:1", Model: "tiny"},
	} {
		files.ConfigFile.Ollama = settings
		if _, err := OllamaGenerateFrontPageMessage(t.Context(), user.ID, time.Now()); err == nil {
			t.Errorf("%s: expected an error", name)
		}
	}
}
