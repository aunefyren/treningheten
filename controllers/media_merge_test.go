package controllers

import (
	"testing"
	"time"

	"github.com/aunefyren/treningheten/models"
)

// The reported bug: Audiobookshelf reported one continuous listen of a podcast episode
// as three overlapping listening sessions (it opens a fresh one on a device switch or an
// unclean close). Rendered raw that is three timeline rows for one thing — two of which
// stack on 00:00, because a start before the session is clamped up to the session start.
// They must merge into the single span they actually were, without swallowing the next
// episode.
func TestBuildAudiobookshelfPlaybackMergesOverlappingSessionsOfOneEpisode(t *testing.T) {
	start := time.Date(2026, 6, 28, 10, 0, 0, 0, time.UTC)
	end := start.Add(90 * time.Minute)

	ms := func(min int) int64 { return start.Add(time.Duration(min) * time.Minute).UnixMilli() }

	sessions := []models.AudiobookshelfListenSession{
		{
			ID: "sess-a", LibraryItemID: "show-the-yard", EpisodeID: "ep-25",
			DisplayTitle: "Ep. 25", DisplayAuthor: "The Yard", MediaType: "podcast",
			TimeListening: 720, StartedAt: ms(-10), UpdatedAt: ms(12),
		},
		{
			ID: "sess-b", LibraryItemID: "show-the-yard", EpisodeID: "ep-25",
			DisplayTitle: "Ep. 25", DisplayAuthor: "The Yard", MediaType: "podcast",
			TimeListening: 1380, StartedAt: ms(-4), UpdatedAt: ms(23),
		},
		{
			ID: "sess-c", LibraryItemID: "show-the-yard", EpisodeID: "ep-25",
			DisplayTitle: "Ep. 25", DisplayAuthor: "The Yard", MediaType: "podcast",
			TimeListening: 1320, StartedAt: ms(21), UpdatedAt: ms(43),
		},
		{
			ID: "sess-d", LibraryItemID: "show-the-yard", EpisodeID: "ep-26",
			DisplayTitle: "Ep. 26", DisplayAuthor: "The Yard", MediaType: "podcast",
			TimeListening: 2580, StartedAt: ms(43), UpdatedAt: ms(86),
		},
	}

	got := buildAudiobookshelfPlaybackForWindow(sessions, start, end)
	if len(got) != 2 {
		for _, row := range got {
			t.Logf("row: %s started %s", row.Title, row.StartedAt)
		}
		t.Fatalf("expected the three Ep. 25 sessions to merge into one row alongside Ep. 26, got %d rows", len(got))
	}

	merged := got[0]
	if merged.Title != "Ep. 25" {
		t.Fatalf("expected the merged episode first (earliest start), got %q", merged.Title)
	}
	// Display start is clamped to the session start, and the flag records why.
	if !merged.StartedAt.Equal(start) {
		t.Errorf("merged start should clamp to the session start, got %s", merged.StartedAt)
	}
	if !merged.StartedBefore {
		t.Error("merged row began before the session, so StartedBefore must be set")
	}
	// The span runs to the last session's end, not the first's.
	wantEnd := start.Add(43 * time.Minute)
	if merged.EndedAt == nil || !merged.EndedAt.Equal(wantEnd) {
		t.Errorf("merged end: got %v, want %s", merged.EndedAt, wantEnd)
	}
	// ABS reports time listened, so merged sessions add up (720+1380+1320 = 3420 s over a
	// 53-minute span) before being clipped to the window: scaled to the 43 in-window
	// minutes that exceeds the overlap itself, so it caps at 43 minutes.
	if merged.TrackLength == nil || *merged.TrackLength != 43*60 {
		t.Errorf("merged listened time should be the in-window share capped at the overlap, got %v", merged.TrackLength)
	}
	if merged.ProviderItemID == nil || *merged.ProviderItemID != "ep-25" {
		t.Errorf("provider item id should be the episode, got %v", merged.ProviderItemID)
	}
	if merged.ProviderParentID == nil || *merged.ProviderParentID != "show-the-yard" {
		t.Errorf("provider parent id should be the show, got %v", merged.ProviderParentID)
	}
	if merged.ProviderSessionID == nil || *merged.ProviderSessionID != "sess-a" {
		t.Errorf("merged row should keep the first session id, got %v", merged.ProviderSessionID)
	}

	next := got[1]
	if next.Title != "Ep. 26" {
		t.Fatalf("second row should be the next episode, got %q", next.Title)
	}
	if next.StartedBefore {
		t.Error("Ep. 26 began inside the session, so StartedBefore must be false")
	}
}

// Two episodes of one show overlapping in time must stay separate. The library item is
// the *show*, so keying on it alone would collapse them into each other — this is why
// the episode id is the identity.
func TestBuildAudiobookshelfPlaybackKeepsDifferentEpisodesApart(t *testing.T) {
	start := time.Date(2026, 6, 28, 10, 0, 0, 0, time.UTC)
	end := start.Add(time.Hour)

	ms := func(min int) int64 { return start.Add(time.Duration(min) * time.Minute).UnixMilli() }

	sessions := []models.AudiobookshelfListenSession{
		{
			ID: "a", LibraryItemID: "show-1", EpisodeID: "ep-1", DisplayTitle: "Ep. 1",
			MediaType: "podcast", StartedAt: ms(5), UpdatedAt: ms(30),
		},
		{
			ID: "b", LibraryItemID: "show-1", EpisodeID: "ep-2", DisplayTitle: "Ep. 2",
			MediaType: "podcast", StartedAt: ms(20), UpdatedAt: ms(50),
		},
	}

	got := buildAudiobookshelfPlaybackForWindow(sessions, start, end)
	if len(got) != 2 {
		t.Fatalf("different episodes of one show must not merge, got %d rows", len(got))
	}
}

// A real gap between two plays of the same item is a pause and resume, not a duplicate
// record — it stays two rows.
func TestCoalesceKeepsNonOverlappingReplaysSeparate(t *testing.T) {
	base := time.Date(2026, 6, 28, 10, 0, 0, 0, time.UTC)

	events := []mediaPlayEvent{
		{title: "Roads", providerItemID: "t1", startedAt: base, coverageEnd: base.Add(4 * time.Minute)},
		{title: "Roads", providerItemID: "t1", startedAt: base.Add(20 * time.Minute), coverageEnd: base.Add(24 * time.Minute)},
	}

	got := coalesceOverlappingEvents(events)
	if len(got) != 2 {
		t.Fatalf("a replay after a gap is a second listen, got %d events", len(got))
	}
}

// Where the length is the item's own (Plex/Spotify), merging must not add lengths up —
// two overlapping records of one track do not make it twice as long.
func TestCoalesceTakesLongestItemLengthWhenNotListenedTime(t *testing.T) {
	base := time.Date(2026, 6, 28, 10, 0, 0, 0, time.UTC)

	events := []mediaPlayEvent{
		{title: "Roads", providerItemID: "t1", startedAt: base, trackLengthSec: 240},
		{title: "Roads", providerItemID: "t1", startedAt: base.Add(time.Minute), trackLengthSec: 260},
	}

	got := coalesceOverlappingEvents(events)
	if len(got) != 1 {
		t.Fatalf("expected the overlapping records to merge, got %d", len(got))
	}
	if got[0].trackLengthSec != 260 {
		t.Errorf("item length should be the longest, not the sum: got %d", got[0].trackLengthSec)
	}
}

// Events with no usable provider id fall back to matching on name, so the merge still
// works for a provider that names nothing.
func TestCoalesceFallsBackToNameWhenNoIDs(t *testing.T) {
	base := time.Date(2026, 6, 28, 10, 0, 0, 0, time.UTC)

	events := []mediaPlayEvent{
		{title: "Roads", artist: "Portishead", startedAt: base, coverageEnd: base.Add(5 * time.Minute)},
		{title: "roads", artist: "Portishead", startedAt: base.Add(2 * time.Minute), coverageEnd: base.Add(7 * time.Minute)},
		{title: "Glory Box", artist: "Portishead", startedAt: base.Add(3 * time.Minute), coverageEnd: base.Add(8 * time.Minute)},
	}

	got := coalesceOverlappingEvents(events)
	if len(got) != 2 {
		t.Fatalf("expected the two Roads records to merge and Glory Box to stay, got %d", len(got))
	}
}

// Plex hands over an agent GUID and the show/artist rating key; both are stored even
// though nothing reads them yet, because history only reaches back hours to days and
// they cannot be backfilled later.
func TestBuildPlexPlaybackCapturesProviderIdentity(t *testing.T) {
	start := time.Date(2026, 6, 28, 10, 0, 0, 0, time.UTC)
	end := start.Add(time.Hour)

	items := []models.PlexHistoryMetadata{{
		RatingKey:            "12345",
		Guid:                 "plex://track/5d07d1b8",
		ParentRatingKey:      "999",
		GrandparentRatingKey: "777",
		Title:                "Roads",
		GrandparentTitle:     "Portishead",
		ParentTitle:          "Dummy",
		Type:                 "track",
		Duration:             300000,
		ViewedAt:             start.Add(10 * time.Minute).Unix(),
	}}

	got := buildPlexPlaybackForWindow(items, nil, start, end)
	if len(got) != 1 {
		t.Fatalf("expected 1 row, got %d", len(got))
	}
	row := got[0]
	if row.ProviderItemID == nil || *row.ProviderItemID != "12345" {
		t.Errorf("provider item id: got %v", row.ProviderItemID)
	}
	if row.ProviderGUID == nil || *row.ProviderGUID != "plex://track/5d07d1b8" {
		t.Errorf("provider guid: got %v", row.ProviderGUID)
	}
	// The grandparent (artist/show) wins over the parent (album) as the container.
	if row.ProviderParentID == nil || *row.ProviderParentID != "777" {
		t.Errorf("provider parent id: got %v", row.ProviderParentID)
	}
}

// Spotify's stable "spotify:track:…" URI and the album id are stored alongside the
// track id, for the same reason.
func TestBuildSpotifyPlaybackCapturesProviderIdentity(t *testing.T) {
	start := time.Date(2026, 6, 28, 10, 0, 0, 0, time.UTC)
	end := start.Add(time.Hour)

	items := []models.SpotifyPlayHistory{{
		PlayedAt: start.Add(10 * time.Minute).Format(time.RFC3339),
		Track: models.SpotifyTrack{
			ID:         "4kflIGfjdZJW4ot2ioixTB",
			URI:        "spotify:track:4kflIGfjdZJW4ot2ioixTB",
			Name:       "Roads",
			DurationMs: 302000,
			Artists:    []models.SpotifyArtist{{Name: "Portishead"}},
			Album:      models.SpotifyAlbum{ID: "3539EbNgIdEDGBKkUf4wno", Name: "Dummy"},
		},
	}}

	got := buildSpotifyPlaybackForWindow(items, start, end)
	if len(got) != 1 {
		t.Fatalf("expected 1 row, got %d", len(got))
	}
	row := got[0]
	if row.ProviderGUID == nil || *row.ProviderGUID != "spotify:track:4kflIGfjdZJW4ot2ioixTB" {
		t.Errorf("provider guid: got %v", row.ProviderGUID)
	}
	if row.ProviderParentID == nil || *row.ProviderParentID != "3539EbNgIdEDGBKkUf4wno" {
		t.Errorf("provider parent id: got %v", row.ProviderParentID)
	}
}

func TestScrobbleSpan(t *testing.T) {
	finished := time.Date(2026, 10, 2, 14, 32, 13, 0, time.UTC)
	cases := []struct {
		name      string
		lengthSec int64
		wantStart time.Time
		wantEnd   time.Time
	}{
		{"known length starts a track earlier", 339, finished.Add(-339 * time.Second), finished},
		{"unknown length falls back to the finish", 0, finished, time.Time{}},
		{"negative length treated as unknown", -5, finished, time.Time{}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gotStart, gotEnd := scrobbleSpan(finished, tc.lengthSec)
			if !gotStart.Equal(tc.wantStart) || !gotEnd.Equal(tc.wantEnd) {
				t.Errorf("scrobbleSpan = (%v, %v), want (%v, %v)", gotStart, gotEnd, tc.wantStart, tc.wantEnd)
			}
		})
	}
}
