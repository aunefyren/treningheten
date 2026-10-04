package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

var windowStart = time.Date(2026, 10, 1, 17, 0, 0, 0, time.FixedZone("CEST", 2*60*60))

func ms(offset time.Duration) int64 {
	return windowStart.Add(offset).UnixMilli()
}

// absSession builds a raw session as ABS serialises it.
func absSession(id, episode string, startOffset, lastActiveOffset time.Duration, listened, startPos, currentPos float64) map[string]any {
	return map[string]any{
		"id":            id,
		"userId":        "user-secret",
		"libraryItemId": "show",
		"episodeId":     episode,
		"displayTitle":  "Episode " + episode,
		"displayAuthor": "The Show",
		"mediaType":     "podcast",
		"timeListening": listened,
		"startTime":     startPos,
		"currentTime":   currentPos,
		"startedAt":     ms(startOffset),
		"updatedAt":     ms(lastActiveOffset),
		"deviceInfo": map[string]any{
			"deviceId":   "device-123",
			"clientName": "Abs Android",
			"deviceName": "Someone's phone",
			"ipAddress":  "10.0.0.5",
		},
	}
}

// newABSServer serves the given sessions newest-first in pages, like ABS, and counts calls.
func newABSServer(t *testing.T, sessions []map[string]any, calls *int) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		*calls++
		if request.Header.Get("Authorization") != "Bearer good" {
			writer.WriteHeader(http.StatusUnauthorized)
			_, _ = writer.Write([]byte("Unauthorized"))
			return
		}
		var page int
		fmt.Sscanf(request.URL.Query().Get("page"), "%d", &page)
		from := page * pageSize
		to := from + pageSize
		if from > len(sessions) {
			from = len(sessions)
		}
		if to > len(sessions) {
			to = len(sessions)
		}
		_ = json.NewEncoder(writer).Encode(map[string]any{
			"numPages": (len(sessions) + pageSize - 1) / pageSize,
			"page":     page,
			"sessions": sessions[from:to],
		})
	}))
}

func env(values map[string]string) func(string) string {
	return func(key string) string { return values[key] }
}

func TestRunDumpsRedactedSessionsTableAndMatch(t *testing.T) {
	sessions := []map[string]any{
		// Newest first, like ABS. The episode is split over two sessions with a small
		// wall-clock gap and a near-zero position gap; the third is long before the window.
		absSession("second-session", "ep25", 21*time.Minute, 43*time.Minute, 1320, 1500, 2820),
		absSession("first-session", "ep25", -10*time.Minute, 20*time.Minute+30*time.Second, 1800, 0, 1495),
		absSession("old-session", "ep24", -5*time.Hour, -4*time.Hour, 3600, 0, 3600),
	}
	calls := 0
	server := newABSServer(t, sessions, &calls)
	defer server.Close()

	var out, log bytes.Buffer
	args := []string{"-from", windowStart.Format(time.RFC3339), "-to", windowStart.Add(time.Hour).Format(time.RFC3339), "-match"}
	if err := run(args, env(map[string]string{"ABS_URL": server.URL + "/", "ABS_TOKEN": "good"}), server.Client(), &out, &log); err != nil {
		t.Fatalf("run: %v\nlog:\n%s", err, log.String())
	}

	var dump probeOutput
	if err := json.Unmarshal(out.Bytes(), &dump); err != nil {
		t.Fatalf("stdout is not JSON: %v", err)
	}
	if len(dump.Sessions) != 2 || dump.Sessions[0]["id"] != "first-session" || dump.Sessions[1]["id"] != "second-session" {
		t.Fatalf("sessions = %v, want the two in-window ones ordered by start", dump.Sessions)
	}
	if strings.Contains(out.String(), "user-secret") || strings.Contains(out.String(), "10.0.0.5") ||
		strings.Contains(out.String(), "Someone's phone") || strings.Contains(out.String(), "device-123") {
		t.Errorf("dump leaks identifying fields:\n%s", out.String())
	}
	if dump.Sessions[0]["startTime"] == nil || dump.Sessions[0]["currentTime"] == nil {
		t.Errorf("dump dropped the position fields: %v", dump.Sessions[0])
	}

	for _, want := range []string{
		"2 of 3 fetched sessions",
		"Episode ep25",
		"+00:30", // wall gap: second started 30s after the first was last active
		"+00:05", // position gap: resumed 5s past where the first stopped
		"Abs Android",
		"matcher output",
		"↑00:00",
	} {
		if !strings.Contains(log.String(), want) {
			t.Errorf("log missing %q:\n%s", want, log.String())
		}
	}
	if calls != 1 {
		t.Errorf("server calls = %d, want 1 (a short page ends the history)", calls)
	}
}

func TestRunRejectsBadInput(t *testing.T) {
	from := windowStart.Format(time.RFC3339)
	to := windowStart.Add(time.Hour).Format(time.RFC3339)
	credentials := env(map[string]string{"ABS_URL": "http://abs", "ABS_TOKEN": "t"})

	tests := []struct {
		name   string
		args   []string
		getenv func(string) string
		want   string
	}{
		{"missing credentials", []string{"-from", from, "-to", to}, env(nil), "ABS_URL"},
		{"bad from", []string{"-from", "yesterday", "-to", to}, credentials, "-from"},
		{"bad to", []string{"-from", from, "-to", "later"}, credentials, "-to"},
		{"inverted window", []string{"-from", to, "-to", from}, credentials, "after -from"},
		{"unknown flag", []string{"-nope"}, credentials, "not defined"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var out, log bytes.Buffer
			err := run(test.args, test.getenv, http.DefaultClient, &out, &log)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Errorf("err = %v, want it to mention %q", err, test.want)
			}
		})
	}
}

func TestFetchSessionsErrors(t *testing.T) {
	t.Run("http status", func(t *testing.T) {
		calls := 0
		server := newABSServer(t, nil, &calls)
		defer server.Close()
		_, err := fetchSessions(server.Client(), server.URL, "bad", windowStart, 5, &bytes.Buffer{})
		if err == nil || !strings.Contains(err.Error(), "401") {
			t.Errorf("err = %v, want a 401", err)
		}
	})

	t.Run("invalid json", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
			_, _ = writer.Write([]byte("<html>"))
		}))
		defer server.Close()
		_, err := fetchSessions(server.Client(), server.URL, "good", windowStart, 5, &bytes.Buffer{})
		if err == nil || !strings.Contains(err.Error(), "parsing page 0") {
			t.Errorf("err = %v, want a parse error", err)
		}
	})

	t.Run("unreachable", func(t *testing.T) {
		server := httptest.NewServer(http.NotFoundHandler())
		server.Close()
		_, err := fetchSessions(http.DefaultClient, server.URL, "good", windowStart, 5, &bytes.Buffer{})
		if err == nil || !strings.Contains(err.Error(), "calling Audiobookshelf") {
			t.Errorf("err = %v, want a transport error", err)
		}
	})

	t.Run("bad url", func(t *testing.T) {
		_, err := fetchSessions(http.DefaultClient, "http://bad host", "good", windowStart, 5, &bytes.Buffer{})
		if err == nil || !strings.Contains(err.Error(), "building request") {
			t.Errorf("err = %v, want a request error", err)
		}
	})
}

func TestFetchSessionsPaging(t *testing.T) {
	// Two full pages of recent sessions then one of old ones.
	full := func(offset time.Duration) []map[string]any {
		page := make([]map[string]any, pageSize)
		for i := range page {
			page[i] = absSession(fmt.Sprint(offset, i), "ep", offset, offset+time.Minute, 60, 0, 60)
		}
		return page
	}
	sessions := append(append(full(0), full(-time.Hour)...), full(-48*time.Hour)...)

	tests := []struct {
		name      string
		since     time.Time
		maxPages  int
		wantCalls int
	}{
		{"stops once a page reaches before since", windowStart.Add(-30 * time.Minute), 10, 2},
		{"stops at the page limit", windowStart.Add(-72 * time.Hour), 2, 2},
		{"stops at the last page", windowStart.Add(-72 * time.Hour), 10, 3},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			calls := 0
			server := newABSServer(t, sessions, &calls)
			defer server.Close()
			got, err := fetchSessions(server.Client(), server.URL, "good", test.since, test.maxPages, &bytes.Buffer{})
			if err != nil {
				t.Fatal(err)
			}
			if calls != test.wantCalls || len(got) != test.wantCalls*pageSize {
				t.Errorf("calls = %d, sessions = %d; want %d calls", calls, len(got), test.wantCalls)
			}
		})
	}
}

func TestFilterSessionsSkipsUnstartedAndOutOfWindow(t *testing.T) {
	sessions := []session{
		{"id": "never-started"},
		{"id": "after", "startedAt": float64(ms(2 * time.Hour))},
		{"id": "before", "startedAt": float64(ms(-2 * time.Hour)), "updatedAt": float64(ms(-90 * time.Minute))},
		{"id": "no-update", "startedAt": float64(ms(10 * time.Minute))},
	}
	got := filterSessions(sessions, windowStart, windowStart.Add(time.Hour))
	if len(got) != 1 || got[0]["id"] != "no-update" {
		t.Errorf("got %v, want only the in-window session", got)
	}
}

func TestRedactSessionWithoutDeviceInfo(t *testing.T) {
	s := session{"userId": "u", "deviceInfo": "unexpected"}
	redactSession(s)
	if _, ok := s["userId"]; ok {
		t.Error("userId kept")
	}
	if _, ok := s["deviceInfo"]; ok {
		t.Error("malformed deviceInfo kept")
	}
	if deviceLabel(s) != "" {
		t.Error("label for a session without device info")
	}
}

func TestWriteMatchShowsUnknownLengthAndEnd(t *testing.T) {
	var log bytes.Buffer
	sessions := []session{{"id": "s", "displayTitle": "Bare", "mediaType": "book", "startedAt": float64(ms(10 * time.Minute))}}
	if err := writeMatch(&log, sessions, windowStart, windowStart.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	if fields := strings.Fields(strings.Split(log.String(), "\n")[2]); strings.Join(fields, " ") != "10:00 Bare - - s" {
		t.Errorf("log = %q, want dashes for the unknown listened time and end", log.String())
	}
}

func TestWriteMatchRejectsUnencodableAndMistypedSessions(t *testing.T) {
	if err := writeMatch(&bytes.Buffer{}, []session{{"bad": make(chan int)}}, windowStart, windowStart); err == nil {
		t.Error("want an encoding error")
	}
	if err := writeMatch(&bytes.Buffer{}, []session{{"startedAt": "noon"}}, windowStart, windowStart); err == nil {
		t.Error("want a decoding error")
	}
}

func TestFormatting(t *testing.T) {
	tests := []struct {
		got, want string
	}{
		{clock(90 * time.Minute), "1:30:00"},
		{clock(-75 * time.Second), "-01:15"},
		{signedDuration(5 * time.Second), "+00:05"},
		{signedDuration(-5 * time.Second), "-00:05"},
		{shorten("abcdef", 4), "abc…"},
		{shorten("abc", 4), "abc"},
		{firstNonEmpty(" ", "b"), "b"},
		{firstNonEmpty(), ""},
	}
	for _, test := range tests {
		if test.got != test.want {
			t.Errorf("got %q, want %q", test.got, test.want)
		}
	}
}

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) { return 0, fmt.Errorf("disk full") }

func TestRunPropagatesFetchAndWriteErrors(t *testing.T) {
	calls := 0
	server := newABSServer(t, []map[string]any{absSession("s", "ep", 0, time.Minute, 60, 0, 60)}, &calls)
	defer server.Close()
	args := []string{"-from", windowStart.Format(time.RFC3339), "-to", windowStart.Add(time.Hour).Format(time.RFC3339), "-url", server.URL}

	if err := run(append(args, "-token", "bad"), env(nil), server.Client(), &bytes.Buffer{}, &bytes.Buffer{}); err == nil {
		t.Error("want the fetch error")
	}
	err := run(append(args, "-token", "good"), env(nil), server.Client(), failingWriter{}, &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "writing JSON") {
		t.Errorf("err = %v, want a write error", err)
	}
}
