package controllers

import (
	"testing"
	"time"

	"github.com/aunefyren/treningheten/models"
)

func TestSpotifySmallestImage(t *testing.T) {
	if got := spotifySmallestImage(nil); got != "" {
		t.Errorf("expected empty for no images, got %q", got)
	}
	images := []models.SpotifyImage{
		{URL: "big", Width: 640},
		{URL: "mid", Width: 300},
		{URL: "small", Width: 64},
	}
	if got := spotifySmallestImage(images); got != "small" {
		t.Errorf("expected the last (smallest) image, got %q", got)
	}
}

func TestBuildSpotifyPlaybackForWindow(t *testing.T) {
	start := time.Date(2026, 6, 28, 10, 0, 0, 0, time.UTC)
	end := start.Add(time.Hour)

	rfc := func(min int) string { return start.Add(time.Duration(min) * time.Minute).Format(time.RFC3339) }

	items := []models.SpotifyPlayHistory{
		{
			PlayedAt: rfc(20),
			Track: models.SpotifyTrack{
				ID:         "t1",
				Name:       "Dracula",
				DurationMs: 180000,
				Artists:    []models.SpotifyArtist{{Name: "Tame Impala"}},
				Album:      models.SpotifyAlbum{Name: "Currents", Images: []models.SpotifyImage{{URL: "big"}, {URL: "small"}}},
			},
		},
		{
			// Two artists join with a comma.
			PlayedAt: rfc(35),
			Track: models.SpotifyTrack{
				ID:      "t2",
				Name:    "Collab",
				Artists: []models.SpotifyArtist{{Name: "A"}, {Name: "B"}},
				Album:   models.SpotifyAlbum{Name: "Split"},
			},
		},
		{
			// Outside the window (well before) — excluded.
			PlayedAt: rfc(-40),
			Track:    models.SpotifyTrack{ID: "t3", Name: "Too Early"},
		},
		{
			// Unparseable timestamp — skipped, no panic.
			PlayedAt: "not-a-time",
			Track:    models.SpotifyTrack{ID: "t4", Name: "Bad"},
		},
	}

	got := buildSpotifyPlaybackForWindow(items, start, end)
	if len(got) != 2 {
		t.Fatalf("expected 2 matched rows, got %d (%+v)", len(got), got)
	}

	first := got[0]
	if first.Title != "Dracula" {
		t.Errorf("title: got %q", first.Title)
	}
	if first.Artist == nil || *first.Artist != "Tame Impala" {
		t.Errorf("artist: got %v", first.Artist)
	}
	if first.ArtworkURL == nil || *first.ArtworkURL != "small" {
		t.Errorf("artwork should be the smallest image, got %v", first.ArtworkURL)
	}
	if first.TrackLength == nil || *first.TrackLength != 180 {
		t.Errorf("track length seconds: got %v", first.TrackLength)
	}
	if first.ProviderItemID == nil || *first.ProviderItemID != "t1" {
		t.Errorf("provider item id: got %v", first.ProviderItemID)
	}
	// played_at is when the track stopped: the 3-minute track ran 17:00–20:00.
	if want := start.Add(17 * time.Minute); !first.StartedAt.Equal(want) {
		t.Errorf("StartedAt should be played_at minus the track length: got %v, want %v", first.StartedAt, want)
	}
	if want := start.Add(20 * time.Minute); first.EndedAt == nil || !first.EndedAt.Equal(want) {
		t.Errorf("EndedAt should be played_at: got %v, want %v", first.EndedAt, want)
	}

	var collab *models.MediaPlayback
	for i := range got {
		if got[i].Title == "Collab" {
			collab = &got[i]
		}
	}
	if collab == nil {
		t.Fatalf("expected the multi-artist row")
	}
	if collab.Artist == nil || *collab.Artist != "A, B" {
		t.Errorf("multi-artist should join with comma, got %v", collab.Artist)
	}
}

// TestBuildSpotifyPlaybackTreatsPlayedAtAsFinish pins the scrobble semantics at the run's
// edges: a track that finished just into the run started before it, and one that
// finished before the run (outside the grace) is not part of it.
func TestBuildSpotifyPlaybackTreatsPlayedAtAsFinish(t *testing.T) {
	start := time.Date(2026, 10, 2, 14, 22, 52, 0, time.UTC)
	end := start.Add(time.Hour)

	cases := []struct {
		name          string
		playedAt      time.Time
		durationMs    int64
		wantMatched   bool
		wantStart     time.Time
		wantStartedBe bool
	}{
		// The reported case: a 339 s track logged 9:21 into the run started 3:42 in.
		{"inside the run", start.Add(9*time.Minute + 21*time.Second), 339000, true, start.Add(3*time.Minute + 42*time.Second), false},
		{"started before, finished inside", start.Add(time.Minute), 339000, true, start, true},
		{"finished before the grace", start.Add(-5*time.Minute - time.Second), 339000, false, time.Time{}, false},
		{"finished within the grace after the end", end.Add(4 * time.Minute), 120000, true, end.Add(2 * time.Minute), false},
		{"unknown length keeps played_at as start", start.Add(30 * time.Minute), 0, true, start.Add(30 * time.Minute), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			items := []models.SpotifyPlayHistory{{
				PlayedAt: tc.playedAt.Format(time.RFC3339Nano),
				Track:    models.SpotifyTrack{ID: "t1", Name: "Let's Groove", DurationMs: tc.durationMs},
			}}
			got := buildSpotifyPlaybackForWindow(items, start, end)
			if !tc.wantMatched {
				if len(got) != 0 {
					t.Fatalf("expected no match, got %+v", got)
				}
				return
			}
			if len(got) != 1 {
				t.Fatalf("expected 1 row, got %d", len(got))
			}
			if !got[0].StartedAt.Equal(tc.wantStart) {
				t.Errorf("StartedAt: got %v, want %v", got[0].StartedAt, tc.wantStart)
			}
			if got[0].StartedBefore != tc.wantStartedBe {
				t.Errorf("StartedBefore: got %v, want %v", got[0].StartedBefore, tc.wantStartedBe)
			}
		})
	}
}
