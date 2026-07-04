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
