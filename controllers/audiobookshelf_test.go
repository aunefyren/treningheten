package controllers

import (
	"testing"
	"time"

	"github.com/aunefyren/treningheten/models"
)

func TestABSClassifyMediaType(t *testing.T) {
	cases := map[string]string{
		"podcast": models.MediaTypePodcast,
		"Podcast": models.MediaTypePodcast,
		"book":    models.MediaTypeAudiobook,
		"":        models.MediaTypeAudiobook,
		"other":   models.MediaTypeAudiobook,
	}
	for in, want := range cases {
		if got := absClassifyMediaType(in); got != want {
			t.Errorf("absClassifyMediaType(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestBuildAudiobookshelfPlaybackForWindow(t *testing.T) {
	start := time.Date(2026, 6, 28, 10, 0, 0, 0, time.UTC)
	end := start.Add(time.Hour)

	ms := func(min int) int64 { return start.Add(time.Duration(min) * time.Minute).UnixMilli() }

	sessions := []models.AudiobookshelfListenSession{
		{
			ID:            "s1",
			LibraryItemID: "li1",
			DisplayTitle:  "Dracula",
			DisplayAuthor: "Bram Stoker",
			MediaType:     "book",
			Duration:      36000,
			TimeListening: 1800,
			StartedAt:     ms(20),
		},
		{
			ID:            "s2",
			LibraryItemID: "li2",
			DisplayTitle:  "Some Episode",
			DisplayAuthor: "A Podcast",
			MediaType:     "podcast",
			TimeListening: 600,
			StartedAt:     ms(35),
		},
		{
			// Outside the window (well before) — excluded.
			ID:           "s3",
			DisplayTitle: "Too Early",
			MediaType:    "book",
			StartedAt:    ms(-40),
		},
		{
			// No start time — skipped, no panic.
			ID:           "s4",
			DisplayTitle: "No Start",
			MediaType:    "book",
			StartedAt:    0,
		},
	}

	got := buildAudiobookshelfPlaybackForWindow(sessions, start, end)
	if len(got) != 2 {
		t.Fatalf("expected 2 matched rows, got %d (%+v)", len(got), got)
	}

	book := got[0]
	if book.Title != "Dracula" {
		t.Errorf("title: got %q", book.Title)
	}
	if book.MediaType != models.MediaTypeAudiobook {
		t.Errorf("book should classify as audiobook, got %q", book.MediaType)
	}
	if book.Artist == nil || *book.Artist != "Bram Stoker" {
		t.Errorf("artist: got %v", book.Artist)
	}
	if book.TrackLength == nil || *book.TrackLength != 1800 {
		t.Errorf("track length should be TimeListening seconds, got %v", book.TrackLength)
	}
	if book.ProviderItemID == nil || *book.ProviderItemID != "li1" {
		t.Errorf("provider item id: got %v", book.ProviderItemID)
	}

	var podcast *models.MediaPlayback
	for i := range got {
		if got[i].Title == "Some Episode" {
			podcast = &got[i]
		}
	}
	if podcast == nil {
		t.Fatalf("expected the podcast row")
	}
	if podcast.MediaType != models.MediaTypePodcast {
		t.Errorf("podcast media type: got %q", podcast.MediaType)
	}
}

// A podcast/audiobook started before the workout but still playing through it must be
// retrieved (matched on interval overlap, not just its start time). This is the ABS
// "started earlier" case reported for the soundtrack feature.
func TestBuildAudiobookshelfPlaybackForWindowStartedBeforeWindow(t *testing.T) {
	start := time.Date(2026, 6, 28, 10, 0, 0, 0, time.UTC)
	end := start.Add(time.Hour)

	ms := func(min int) int64 { return start.Add(time.Duration(min) * time.Minute).UnixMilli() }

	sessions := []models.AudiobookshelfListenSession{
		{
			// Started 15 min before the workout, last activity 30 min into it: overlaps.
			ID:            "early",
			LibraryItemID: "li-early",
			DisplayTitle:  "Long Episode",
			MediaType:     "podcast",
			StartedAt:     ms(-15),
			UpdatedAt:     ms(30),
		},
		{
			// Ended well before the workout (no overlap) — must stay excluded.
			ID:           "done",
			DisplayTitle: "Finished Earlier",
			MediaType:    "podcast",
			StartedAt:    ms(-90),
			UpdatedAt:    ms(-60),
		},
	}

	got := buildAudiobookshelfPlaybackForWindow(sessions, start, end)
	if len(got) != 1 {
		t.Fatalf("expected only the overlapping session, got %d (%+v)", len(got), got)
	}
	row := got[0]
	if row.Title != "Long Episode" {
		t.Fatalf("wrong session matched: %q", row.Title)
	}
	// Display start is clamped up to the workout start.
	if !row.StartedAt.Equal(start) {
		t.Errorf("StartedAt should clamp to workout start %v, got %v", start, row.StartedAt)
	}
	// End is the real session end (UpdatedAt), inside the window.
	if row.EndedAt == nil || !row.EndedAt.Equal(start.Add(30*time.Minute)) {
		t.Errorf("EndedAt should be the session UpdatedAt, got %v", row.EndedAt)
	}
}

// TestBuildAudiobookshelfPlaybackClipsListenedTimeToWindow covers the reported case: a
// podcast started before a 60-minute run and another finished after it read as 40 + 49
// minutes, because TimeListening covers the whole listen. Only the in-window share is
// kept: listened × overlap ÷ span, capped at the overlap.
func TestBuildAudiobookshelfPlaybackClipsListenedTimeToWindow(t *testing.T) {
	start := time.Date(2026, 6, 28, 10, 0, 0, 0, time.UTC)
	end := start.Add(time.Hour)
	ms := func(min int) int64 { return start.Add(time.Duration(min) * time.Minute).UnixMilli() }

	cases := []struct {
		name          string
		startedAt     int64
		updatedAt     int64
		timeListening float64
		want          *int64
	}{
		// 40 min listened over a 45-minute span, 30 of them in the run.
		{"started before the run", ms(-15), ms(30), 2400, int64Ptr(1600)},
		// 49 min listened over a 60-minute span, 39 of them in the run.
		{"ran past the end", ms(21), ms(81), 2940, int64Ptr(1911)},
		{"fully inside is unchanged", ms(10), ms(40), 1500, int64Ptr(1500)},
		// More listened than wall-clock time can't be true of the window: cap at overlap.
		{"capped at the overlap", ms(10), ms(20), 900, int64Ptr(600)},
		// No UpdatedAt: the span is the listened time itself, so the result is the overlap.
		{"unknown end reduces to the overlap", ms(50), 0, 1200, int64Ptr(600)},
		// Within the grace after the end but nothing listened in the run: no row at all,
		// rather than an "already playing" row with no minutes.
		{"only in the grace is dropped", end.Add(2 * time.Minute).UnixMilli(), end.Add(20 * time.Minute).UnixMilli(), 1080, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			sessions := []models.AudiobookshelfListenSession{{
				ID: "s", LibraryItemID: "li", DisplayTitle: "Episode", MediaType: "podcast",
				TimeListening: tc.timeListening, StartedAt: tc.startedAt, UpdatedAt: tc.updatedAt,
			}}
			got := buildAudiobookshelfPlaybackForWindow(sessions, start, end)
			if tc.want == nil {
				if len(got) != 0 {
					t.Fatalf("expected no row, got %+v", got)
				}
				return
			}
			if len(got) != 1 {
				t.Fatalf("expected 1 row, got %d", len(got))
			}
			if got[0].TrackLength == nil || *got[0].TrackLength != *tc.want {
				t.Errorf("TrackLength: got %v, want %d", got[0].TrackLength, *tc.want)
			}
		})
	}
}

func TestListenedWithinWindowWithoutListenedTime(t *testing.T) {
	start := time.Date(2026, 6, 28, 10, 0, 0, 0, time.UTC)
	event := mediaPlayEvent{startedAt: start, trackLengthIsListened: true}
	if got := listenedWithinWindow(event, start, start.Add(time.Hour)); got != 0 {
		t.Errorf("no listened time should give 0, got %d", got)
	}
}

func TestMatchAudiobookshelfSessionsMatchesTheSyncMatcher(t *testing.T) {
	start := time.Date(2026, 6, 28, 10, 0, 0, 0, time.UTC)
	end := start.Add(time.Hour)

	sessions := []models.AudiobookshelfListenSession{
		{ID: "in", EpisodeID: "ep1", LibraryItemID: "show", DisplayTitle: "Episode", MediaType: "podcast",
			TimeListening: 600, StartedAt: start.Add(10 * time.Minute).UnixMilli()},
		{ID: "out", EpisodeID: "ep2", LibraryItemID: "show", DisplayTitle: "Later", MediaType: "podcast",
			TimeListening: 600, StartedAt: end.Add(time.Hour).UnixMilli()},
	}

	got := MatchAudiobookshelfSessions(sessions, start, end)
	want := buildAudiobookshelfPlaybackForWindow(sessions, start, end)

	if len(got) != 1 || len(want) != 1 {
		t.Fatalf("got %d rows, want 1 (sync matcher gave %d)", len(got), len(want))
	}
	if got[0].Title != want[0].Title || !got[0].StartedAt.Equal(want[0].StartedAt) {
		t.Errorf("got %+v, want %+v", got[0], want[0])
	}
}

func TestABSListenedSeconds(t *testing.T) {
	tests := []struct {
		name    string
		session models.AudiobookshelfListenSession
		want    int64
	}{
		{"listened time as sent", models.AudiobookshelfListenSession{TimeListening: 600.4, StartTime: 0, CurrentTime: 900}, 600},
		{"null listened falls back to the position delta", models.AudiobookshelfListenSession{StartTime: 1332, CurrentTime: 2604}, 1272},
		{"nothing to go on", models.AudiobookshelfListenSession{StartTime: 50, CurrentTime: 50}, 0},
		{"capped at the wall-clock span", models.AudiobookshelfListenSession{TimeListening: 900, StartedAt: 1_000, UpdatedAt: 601_000}, 600},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := absListenedSeconds(test.session); got != test.want {
				t.Errorf("got %d, want %d", got, test.want)
			}
		})
	}
}

// absCoverageEnd ends a session at UpdatedAt unless the session held a pause and was
// closed when the next session resumed it — then listening was at its start.
func TestABSCoverageEnd(t *testing.T) {
	start := time.Date(2026, 6, 28, 10, 0, 0, 0, time.UTC)
	ms := func(offset time.Duration) int64 { return start.Add(offset).UnixMilli() }
	paused := models.AudiobookshelfListenSession{StartedAt: ms(0), UpdatedAt: ms(40 * time.Minute)}
	nextAt := func(offset time.Duration) *models.AudiobookshelfListenSession {
		return &models.AudiobookshelfListenSession{StartedAt: ms(offset)}
	}

	tests := []struct {
		name     string
		session  models.AudiobookshelfListenSession
		listened int64
		next     *models.AudiobookshelfListenSession
		want     time.Time
	}{
		{"continuous listen ends at UpdatedAt", models.AudiobookshelfListenSession{StartedAt: ms(0), UpdatedAt: ms(11 * time.Minute)}, 600, nextAt(11*time.Minute + 5*time.Second), start.Add(11 * time.Minute)},
		{"closed on resume ends after the listened time", paused, 600, nextAt(40*time.Minute + 10*time.Second), start.Add(10 * time.Minute)},
		{"pause with no resume handoff is spread to UpdatedAt", paused, 600, nextAt(2 * time.Hour), start.Add(40 * time.Minute)},
		{"pause with no later session is spread to UpdatedAt", paused, 600, nil, start.Add(40 * time.Minute)},
		{"unknown listened time ends at UpdatedAt", paused, 0, nextAt(40 * time.Minute), start.Add(40 * time.Minute)},
		{"no UpdatedAt ends after the listened time", models.AudiobookshelfListenSession{StartedAt: ms(0)}, 600, nil, start.Add(10 * time.Minute)},
		{"nothing known is zero", models.AudiobookshelfListenSession{StartedAt: ms(0)}, 0, nil, time.Time{}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := absCoverageEnd(test.session, test.listened, test.next); !got.Equal(test.want) {
				t.Errorf("got %s, want %s", got, test.want)
			}
		})
	}
}
