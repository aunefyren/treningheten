package controllers

import (
	"testing"
	"time"

	"github.com/aunefyren/treningheten/models"
)

func TestIsPlexAudioListen(t *testing.T) {
	cases := map[string]bool{
		"track":   true,
		"TRACK":   true,
		"episode": false, // TV episode = watching, not listening
		"movie":   false,
		"clip":    false,
		"":        false,
	}
	for in, want := range cases {
		if got := isPlexAudioListen(in); got != want {
			t.Errorf("isPlexAudioListen(%q) = %v, want %v", in, got, want)
		}
	}
}

func TestBuildPlexPlaybackForWindow(t *testing.T) {
	start := time.Date(2026, 6, 27, 10, 0, 0, 0, time.UTC)
	end := start.Add(time.Hour) // 11:00

	at := func(min int) int64 { return start.Add(time.Duration(min) * time.Minute).Unix() }

	items := []models.PlexHistoryMetadata{
		// Inside the window: a full song with artist/album/duration.
		{RatingKey: "1", Title: "In Window", GrandparentTitle: "Artist", ParentTitle: "Album", Type: "track", ViewedAt: at(20), Duration: 180000},
		// Before the window (and before grace) — excluded.
		{RatingKey: "2", Title: "Too Early", Type: "track", ViewedAt: at(-30)},
		// Inside grace just after the end — included.
		{RatingKey: "3", Title: "Grace", Type: "track", ViewedAt: end.Add(3 * time.Minute).Unix()},
		// A TV episode inside the window — excluded (video, not listening).
		{RatingKey: "4", Title: "Vox Machina", Type: "episode", ViewedAt: at(40)},
		// Another track inside the window.
		{RatingKey: "5", Title: "Late Track", GrandparentTitle: "Band", Type: "track", ViewedAt: at(45)},
	}

	got := buildPlexPlaybackForWindow(items, start, end)

	if len(got) != 3 {
		t.Fatalf("expected 3 matched rows, got %d (%+v)", len(got), got)
	}
	for _, row := range got {
		if row.Title == "Vox Machina" {
			t.Errorf("video episode should be filtered out of the audio timeline")
		}
	}

	// Row 1: full mapping with clamped/known length.
	first := got[0]
	if first.Title != "In Window" {
		t.Errorf("title: got %q", first.Title)
	}
	if first.Artist == nil || *first.Artist != "Artist" {
		t.Errorf("artist not mapped: %v", first.Artist)
	}
	if first.Album == nil || *first.Album != "Album" {
		t.Errorf("album not mapped: %v", first.Album)
	}
	if first.MediaType != models.MediaTypeSong {
		t.Errorf("media type: got %q", first.MediaType)
	}
	if first.TrackLength == nil || *first.TrackLength != 180 {
		t.Errorf("track length (seconds): got %v, want 180", first.TrackLength)
	}
	if first.EndedAt == nil {
		t.Errorf("expected EndedAt to be set when duration is known")
	} else if want := first.StartedAt.Add(180 * time.Second); !first.EndedAt.Equal(want) {
		t.Errorf("EndedAt: got %v, want %v", first.EndedAt, want)
	}

	// A duration-less track maps with no length/EndedAt.
	var late *models.MediaPlayback
	for i := range got {
		if got[i].Title == "Late Track" {
			late = &got[i]
		}
	}
	if late == nil {
		t.Fatalf("expected the duration-less track to be present")
	}
	if late.MediaType != models.MediaTypeSong {
		t.Errorf("media type: got %q", late.MediaType)
	}
	if late.TrackLength != nil || late.EndedAt != nil {
		t.Errorf("expected no length/EndedAt for duration-less item")
	}
}

func TestBuildPlexPlaybackClampsToActivityEnd(t *testing.T) {
	start := time.Date(2026, 6, 27, 10, 0, 0, 0, time.UTC)
	end := start.Add(30 * time.Minute) // 10:30

	// A 10-minute track scrobbled at 10:25 would naturally end at 10:35, past the
	// activity end — it must clamp to 10:30.
	items := []models.PlexHistoryMetadata{
		{RatingKey: "1", Title: "Long", Type: "track", ViewedAt: start.Add(25 * time.Minute).Unix(), Duration: 600000},
	}

	got := buildPlexPlaybackForWindow(items, start, end)
	if len(got) != 1 {
		t.Fatalf("expected 1 row, got %d", len(got))
	}
	if got[0].EndedAt == nil {
		t.Fatalf("expected EndedAt set")
	}
	if !got[0].EndedAt.Equal(end) {
		t.Errorf("EndedAt should clamp to activity end %v, got %v", end, got[0].EndedAt)
	}
}

func TestResolveSessionWindow(t *testing.T) {
	startTime := time.Date(2026, 6, 27, 8, 0, 0, 0, time.UTC)
	duration := time.Duration(3600) // raw seconds count (repo convention)

	// Explicit session Duration wins.
	ex := models.Exercise{Time: &startTime, Duration: &duration}
	gotStart, gotEnd, ok := resolveSessionWindow(ex, 0)
	if !ok {
		t.Fatalf("expected a trustworthy window for a real clock time")
	}
	if !gotStart.Equal(startTime) {
		t.Errorf("start: got %v, want %v", gotStart, startTime)
	}
	if want := startTime.Add(time.Hour); !gotEnd.Equal(want) {
		t.Errorf("end: got %v, want %v", gotEnd, want)
	}

	// No session Duration but an operations/sets fallback → use the fallback.
	exFallback := models.Exercise{Time: &startTime}
	_, gotEnd, ok = resolveSessionWindow(exFallback, 1800)
	if !ok {
		t.Fatalf("expected a trustworthy window with a fallback duration")
	}
	if want := startTime.Add(30 * time.Minute); !gotEnd.Equal(want) {
		t.Errorf("fallback window end: got %v, want %v", gotEnd, want)
	}

	// No duration anywhere → default window length.
	_, gotEnd, ok = resolveSessionWindow(exFallback, 0)
	if !ok {
		t.Fatalf("expected a trustworthy window on the default fallback")
	}
	if want := startTime.Add(defaultActivityWindow); !gotEnd.Equal(want) {
		t.Errorf("default window end: got %v, want %v", gotEnd, want)
	}

	// No Time at all → not trustworthy (skip matching).
	if _, _, ok := resolveSessionWindow(models.Exercise{Duration: &duration}, 0); ok {
		t.Errorf("expected ok=false when the session has no time")
	}

	// Date-only midnight stamp (manual past-day log) → not trustworthy.
	midnight := time.Date(2026, 6, 27, 0, 0, 0, 0, time.UTC)
	if _, _, ok := resolveSessionWindow(models.Exercise{Time: &midnight, Duration: &duration}, 0); ok {
		t.Errorf("expected ok=false for a date-only midnight session")
	}
}
