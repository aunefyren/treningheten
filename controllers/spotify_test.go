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
