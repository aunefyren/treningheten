package controllers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// stubHevyAPI points hevyAPIBaseURL at a test server for the duration of a test and returns
// the requests it received, so a test can assert on the paths and headers the client sent.
func stubHevyAPI(t *testing.T, handler http.HandlerFunc) *[]*http.Request {
	t.Helper()

	received := []*http.Request{}
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		received = append(received, request.Clone(request.Context()))
		handler(writer, request)
	}))

	previous := hevyAPIBaseURL
	hevyAPIBaseURL = server.URL
	t.Cleanup(func() {
		hevyAPIBaseURL = previous
		server.Close()
	})

	return &received
}

// TestHevyAPIGetSendsCredentials covers the request the client builds: the API key travels
// in the `api-key` header (not a bearer token), and the caller's path is appended verbatim.
func TestHevyAPIGetSendsCredentials(t *testing.T) {
	received := stubHevyAPI(t, func(writer http.ResponseWriter, request *http.Request) {
		writer.Write([]byte(`{"ok":true}`))
	})

	body, err := hevyAPIGet("the-api-key", "/workouts?page=2&pageSize=10")
	if err != nil {
		t.Fatalf("hevyAPIGet returned error: %v", err)
	}
	if string(body) != `{"ok":true}` {
		t.Errorf("body = %s, want the response passed through unchanged", body)
	}

	if len(*received) != 1 {
		t.Fatalf("got %d requests, want 1", len(*received))
	}
	request := (*received)[0]
	if request.Method != http.MethodGet {
		t.Errorf("method = %s, want GET", request.Method)
	}
	if got := request.Header.Get("api-key"); got != "the-api-key" {
		t.Errorf("api-key header = %q, want the key", got)
	}
	if request.Header.Get("Authorization") != "" {
		t.Errorf("the key was sent as an Authorization header; Hevy expects api-key")
	}
	if request.URL.Path != "/workouts" || request.URL.RawQuery != "page=2&pageSize=10" {
		t.Errorf("requested %s?%s, want the path and query passed through", request.URL.Path, request.URL.RawQuery)
	}
}

// TestHevyAPIGetErrors covers how the client reports failures — a rejected key must be
// distinguishable from any other failure, because it is what tells a user to re-enter it.
func TestHevyAPIGetErrors(t *testing.T) {
	t.Run("empty key never leaves the process", func(t *testing.T) {
		received := stubHevyAPI(t, func(writer http.ResponseWriter, request *http.Request) {
			writer.Write([]byte(`{}`))
		})

		for _, key := range []string{"", "   ", "\t"} {
			if _, err := hevyAPIGet(key, "/user/info"); err == nil {
				t.Errorf("hevyAPIGet(%q) returned no error", key)
			}
		}
		if len(*received) != 0 {
			t.Errorf("an empty key still produced %d requests to Hevy", len(*received))
		}
	})

	statuses := map[int]string{
		http.StatusUnauthorized:        "rejected",
		http.StatusForbidden:           "rejected",
		http.StatusTooManyRequests:     "unexpected response",
		http.StatusInternalServerError: "unexpected response",
		http.StatusNotFound:            "unexpected response",
	}
	for status, wantFragment := range statuses {
		t.Run(http.StatusText(status), func(t *testing.T) {
			stubHevyAPI(t, func(writer http.ResponseWriter, request *http.Request) {
				writer.WriteHeader(status)
			})

			_, err := hevyAPIGet("the-api-key", "/user/info")
			if err == nil {
				t.Fatalf("status %d returned no error", status)
			}
			if !strings.Contains(err.Error(), wantFragment) {
				t.Errorf("status %d gave %q, want it to mention %q", status, err, wantFragment)
			}
		})
	}
}

// TestHevyValidateAPIKey covers the connect-time check: a 200 yields the account it belongs
// to, and a rejected key surfaces as an error rather than an empty account.
func TestHevyValidateAPIKey(t *testing.T) {
	received := stubHevyAPI(t, func(writer http.ResponseWriter, request *http.Request) {
		writer.Write([]byte(`{"data":{"id":"user-123","name":"Test Athlete","url":"https://hevy.com/user/test"}}`))
	})

	userInfo, err := hevyValidateAPIKey("the-api-key")
	if err != nil {
		t.Fatalf("hevyValidateAPIKey returned error: %v", err)
	}
	if userInfo.ID != "user-123" || userInfo.Name != "Test Athlete" {
		t.Errorf("user info = %+v, want the unwrapped data object", userInfo)
	}
	if len(*received) != 1 || (*received)[0].URL.Path != "/user/info" {
		t.Errorf("validation did not call /user/info")
	}

	t.Run("rejected key", func(t *testing.T) {
		stubHevyAPI(t, func(writer http.ResponseWriter, request *http.Request) {
			writer.WriteHeader(http.StatusUnauthorized)
		})
		if _, err := hevyValidateAPIKey("bad-key"); err == nil {
			t.Errorf("a rejected key validated successfully")
		}
	})

	t.Run("malformed response", func(t *testing.T) {
		stubHevyAPI(t, func(writer http.ResponseWriter, request *http.Request) {
			writer.Write([]byte(`not json`))
		})
		if _, err := hevyValidateAPIKey("the-api-key"); err == nil {
			t.Errorf("a malformed response validated successfully")
		}
	})
}

// TestHevyFetchExerciseTemplatesPages covers the paging loop: every page is requested in
// order and merged into one template map keyed by template id.
func TestHevyFetchExerciseTemplatesPages(t *testing.T) {
	pages := map[string]string{
		"1": `{"page":1,"page_count":3,"exercise_templates":[{"id":"A","title":"Bench Press","type":"weight_reps"}]}`,
		"2": `{"page":2,"page_count":3,"exercise_templates":[{"id":"B","title":"Running","type":"distance_duration"}]}`,
		"3": `{"page":3,"page_count":3,"exercise_templates":[{"id":"C","title":"Plank","type":"duration"}]}`,
	}

	received := stubHevyAPI(t, func(writer http.ResponseWriter, request *http.Request) {
		page := request.URL.Query().Get("page")
		body, ok := pages[page]
		if !ok {
			t.Errorf("unexpected page requested: %q", page)
			writer.WriteHeader(http.StatusNotFound)
			return
		}
		writer.Write([]byte(body))
	})

	templates, err := hevyFetchExerciseTemplates("the-api-key")
	if err != nil {
		t.Fatalf("hevyFetchExerciseTemplates returned error: %v", err)
	}

	if len(templates) != 3 {
		t.Errorf("got %d templates, want all 3 pages merged", len(templates))
	}
	if templates["B"].Title != "Running" {
		t.Errorf("template B = %+v, want the second page's entry", templates["B"])
	}
	if len(*received) != 3 {
		t.Errorf("made %d requests, want one per page", len(*received))
	}
	for i, request := range *received {
		if got := request.URL.Query().Get("page"); got != []string{"1", "2", "3"}[i] {
			t.Errorf("request %d asked for page %q", i, got)
		}
		if got := request.URL.Query().Get("pageSize"); got != "100" {
			t.Errorf("pageSize = %q, want Hevy's maximum 100", got)
		}
	}
}

// TestHevyFetchExerciseTemplatesStopsOnEmptyCatalog covers the guard against an endless
// loop when Hevy reports no pages at all.
func TestHevyFetchExerciseTemplatesStopsOnEmptyCatalog(t *testing.T) {
	received := stubHevyAPI(t, func(writer http.ResponseWriter, request *http.Request) {
		writer.Write([]byte(`{"page":1,"page_count":0,"exercise_templates":[]}`))
	})

	templates, err := hevyFetchExerciseTemplates("the-api-key")
	if err != nil {
		t.Fatalf("hevyFetchExerciseTemplates returned error: %v", err)
	}
	if len(templates) != 0 {
		t.Errorf("got %d templates, want none", len(templates))
	}
	if len(*received) != 1 {
		t.Errorf("made %d requests for an empty catalog, want 1", len(*received))
	}
}

// TestHevyFetchExerciseTemplatesPropagatesFailure covers that a failure partway through
// paging aborts rather than returning a half-built catalog — a partial map would silently
// reclassify the missing exercises as custom on import.
func TestHevyFetchExerciseTemplatesPropagatesFailure(t *testing.T) {
	stubHevyAPI(t, func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Query().Get("page") == "1" {
			writer.Write([]byte(`{"page":1,"page_count":2,"exercise_templates":[{"id":"A","title":"Bench Press"}]}`))
			return
		}
		writer.WriteHeader(http.StatusInternalServerError)
	})

	templates, err := hevyFetchExerciseTemplates("the-api-key")
	if err == nil {
		t.Fatalf("a failed page returned no error")
	}
	if templates != nil {
		t.Errorf("got a partial catalog of %d templates, want nil", len(templates))
	}
}

// TestHevyTemplateResponseShape pins the wire shape the client unmarshals, so a rename in
// the model is caught here rather than by an empty catalog in production.
func TestHevyTemplateResponseShape(t *testing.T) {
	const payload = `{"page":1,"page_count":1,"exercise_templates":[
		{"id":"AC1BB830","title":"Running","type":"distance_duration","primary_muscle_group":"cardio","is_custom":false}
	]}`

	var response struct {
		Page      int `json:"page"`
		PageCount int `json:"page_count"`
	}
	if err := json.Unmarshal([]byte(payload), &response); err != nil {
		t.Fatalf("failed to parse the payload: %v", err)
	}

	stubHevyAPI(t, func(writer http.ResponseWriter, request *http.Request) {
		writer.Write([]byte(payload))
	})

	templates, err := hevyFetchExerciseTemplates("the-api-key")
	if err != nil {
		t.Fatalf("hevyFetchExerciseTemplates returned error: %v", err)
	}
	template, ok := templates["AC1BB830"]
	if !ok {
		t.Fatalf("template not keyed by its id: %+v", templates)
	}
	if template.Title != "Running" || template.Type != "distance_duration" || template.PrimaryMuscleGroup != "cardio" || template.IsCustom {
		t.Errorf("template = %+v, want every field populated from the payload", template)
	}
}
