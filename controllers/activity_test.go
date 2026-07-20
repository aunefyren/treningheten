package controllers

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

// feedContextForQuery builds a gin context whose request URL carries the given raw query
// string, so parseActivityFeedFilter (which reads context.Query) can be exercised without
// the auth/DB layers.
func feedContextForQuery(rawQuery string) *gin.Context {
	context, _ := gin.CreateTestContext(httptest.NewRecorder())
	context.Request = httptest.NewRequest("GET", "/?"+rawQuery, nil)
	return context
}

func TestParseActivityFeedFilterDefaults(t *testing.T) {
	filter, err := parseActivityFeedFilter(feedContextForQuery(""))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if filter.Sort != "date" || filter.Order != "desc" {
		t.Errorf("defaults wrong: sort=%q order=%q", filter.Sort, filter.Order)
	}
	if filter.Limit != 30 || filter.Offset != 0 {
		t.Errorf("pagination defaults wrong: limit=%d offset=%d", filter.Limit, filter.Offset)
	}
	if filter.ActionID != nil || filter.Start != nil || filter.End != nil {
		t.Errorf("expected no filters set by default")
	}
	if filter.Query != "" || filter.HasDistance {
		t.Errorf("expected empty query and has_distance=false")
	}
}

func TestParseActivityFeedFilterValidValues(t *testing.T) {
	actionID := "550e8400-e29b-41d4-a716-446655440000"
	q := "action_id=" + actionID +
		"&start=2025-01-01&end=2025-12-31T23:59:59Z" +
		"&q=%20cosmo%20&has_distance=TRUE&sort=DISTANCE&order=ASC&limit=50&offset=10"

	filter, err := parseActivityFeedFilter(feedContextForQuery(q))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if filter.ActionID == nil || filter.ActionID.String() != actionID {
		t.Errorf("action id not parsed: %v", filter.ActionID)
	}
	// A bare YYYY-MM-DD is read as midnight; a full RFC3339 keeps its time.
	if filter.Start == nil || !filter.Start.Equal(time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("start not parsed: %v", filter.Start)
	}
	if filter.End == nil || !filter.End.Equal(time.Date(2025, 12, 31, 23, 59, 59, 0, time.UTC)) {
		t.Errorf("end not parsed: %v", filter.End)
	}
	if filter.Query != "cosmo" {
		t.Errorf("query should be trimmed to %q, got %q", "cosmo", filter.Query)
	}
	if !filter.HasDistance {
		t.Errorf("has_distance should be true (case-insensitive)")
	}
	// Sort and order are lower-cased before validation.
	if filter.Sort != "distance" || filter.Order != "asc" {
		t.Errorf("sort/order not normalized: sort=%q order=%q", filter.Sort, filter.Order)
	}
	if filter.Limit != 50 || filter.Offset != 10 {
		t.Errorf("pagination not parsed: limit=%d offset=%d", filter.Limit, filter.Offset)
	}
}

func TestParseActivityFeedFilterRejectsBadInput(t *testing.T) {
	cases := map[string]string{
		"bad action id": "action_id=not-a-uuid",
		"bad start":     "start=13/25/2025",
		"bad end":       "end=nonsense",
		"bad sort":      "sort=heartrate",
		"bad order":     "order=sideways",
	}
	for name, query := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := parseActivityFeedFilter(feedContextForQuery(query)); err == nil {
				t.Errorf("expected error for %q", query)
			}
		})
	}
}

func TestParseActivityFeedFilterClampsPagination(t *testing.T) {
	cases := []struct {
		query      string
		wantLimit  int
		wantOffset int
	}{
		{"limit=0", 1, 0},          // below floor → 1
		{"limit=999", 100, 0},      // above ceiling → 100
		{"limit=abc", 30, 0},       // unparseable → default 30
		{"offset=-5", 30, 0},       // negative offset ignored → 0
		{"offset=7&limit=5", 5, 7}, // both honoured within range
	}
	for _, c := range cases {
		filter, err := parseActivityFeedFilter(feedContextForQuery(c.query))
		if err != nil {
			t.Fatalf("unexpected error for %q: %v", c.query, err)
		}
		if filter.Limit != c.wantLimit || filter.Offset != c.wantOffset {
			t.Errorf("%q: got limit=%d offset=%d, want limit=%d offset=%d",
				c.query, filter.Limit, filter.Offset, c.wantLimit, c.wantOffset)
		}
	}
}

func TestParseActivityFeedTime(t *testing.T) {
	// Full RFC3339 keeps the instant.
	got, err := parseActivityFeedTime("2025-04-12T18:30:00Z")
	if err != nil {
		t.Fatalf("rfc3339 parse failed: %v", err)
	}
	if !got.Equal(time.Date(2025, 4, 12, 18, 30, 0, 0, time.UTC)) {
		t.Errorf("rfc3339 wrong: %v", got)
	}

	// Bare date is treated as midnight.
	got, err = parseActivityFeedTime("2025-04-12")
	if err != nil {
		t.Fatalf("date parse failed: %v", err)
	}
	if got.Hour() != 0 || got.Minute() != 0 || got.Year() != 2025 {
		t.Errorf("date wrong: %v", got)
	}

	if _, err := parseActivityFeedTime("not-a-date"); err == nil {
		t.Errorf("expected error for garbage input")
	}
}
