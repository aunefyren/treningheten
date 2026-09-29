package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestFetchTemplatesPagesFiltersAndSorts(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Header.Get("api-key") != "pro-key" {
			writer.WriteHeader(http.StatusUnauthorized)
			_, _ = writer.Write([]byte(`{"error":"bad key"}`))
			return
		}
		pages := map[string]templatesResponse{
			"1": {Page: 1, PageCount: 2, ExerciseTemplates: []template{{ID: "b", Title: "squat"}, {ID: "c", Title: "Mine", IsCustom: true}}},
			"2": {Page: 2, PageCount: 2, ExerciseTemplates: []template{{ID: "a", Title: "Bench Press"}}},
		}
		_ = json.NewEncoder(writer).Encode(pages[request.URL.Query().Get("page")])
	}))
	defer server.Close()

	var out, log bytes.Buffer
	if err := fetchTemplates(server.Client(), server.URL, "pro-key", &out, &log); err != nil {
		t.Fatalf("fetchTemplates: %v", err)
	}

	var templates []template
	if err := json.Unmarshal(out.Bytes(), &templates); err != nil {
		t.Fatal(err)
	}
	if len(templates) != 2 || templates[0].Title != "Bench Press" || templates[1].Title != "squat" {
		t.Errorf("templates = %+v, want the two official ones sorted case-insensitively", templates)
	}
	if !strings.Contains(log.String(), "done: 2 non-custom templates") {
		t.Errorf("progress log = %q", log.String())
	}

	if err := fetchTemplates(server.Client(), server.URL, "wrong", &out, &log); err == nil || !strings.Contains(err.Error(), "401") {
		t.Errorf("rejected key: err = %v, want the 401 surfaced", err)
	}
}

func TestFetchTemplatesFailures(t *testing.T) {
	garbage := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		_, _ = writer.Write([]byte("not json"))
	}))
	defer garbage.Close()

	var out, log bytes.Buffer
	if err := fetchTemplates(garbage.Client(), garbage.URL, "k", &out, &log); err == nil {
		t.Error("expected a parse error")
	}
	if err := fetchTemplates(http.DefaultClient, "http://127.0.0.1:1", "k", &out, &log); err == nil {
		t.Error("expected a connection error")
	}
	if err := fetchTemplates(http.DefaultClient, "://bad-url", "k", &out, &log); err == nil {
		t.Error("expected a request-building error")
	}
}
