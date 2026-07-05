package controllers

import (
	"testing"
	"time"

	"github.com/aunefyren/treningheten/models"
)

func strPtrLocal(s string) *string { return &s }
func i64Ptr(v int64) *int64        { return &v }

// mediaPlay is a small builder for a playback row in these tests.
func mediaPlay(mediaType, title, artist string, length int64) models.MediaPlaybackObject {
	return models.MediaPlaybackObject{
		MediaType:   mediaType,
		Title:       title,
		Artist:      strPtrLocal(artist),
		TrackLength: i64Ptr(length),
		StartedAt:   time.Now(),
	}
}

// TestComputeActionMediaStatisticsNilOnEmpty: no playback rows → nil block, so the
// frontend leaves the soundtrack section out.
func TestComputeActionMediaStatisticsNilOnEmpty(t *testing.T) {
	if got := computeActionMediaStatistics(nil); got != nil {
		t.Fatalf("expected nil for empty playback, got %+v", got)
	}
	if got := computeActionMediaStatistics([]models.MediaPlaybackObject{}); got != nil {
		t.Fatalf("expected nil for empty slice, got %+v", got)
	}
}

// TestComputeActionMediaStatisticsSongs: songs drive the counts, listening time, and the
// most-played track/artist tallies.
func TestComputeActionMediaStatisticsSongs(t *testing.T) {
	playback := []models.MediaPlaybackObject{
		mediaPlay(models.MediaTypeSong, "One", "Metallica", 100),
		mediaPlay(models.MediaTypeSong, "One", "Metallica", 100),
		mediaPlay(models.MediaTypeSong, "Enter Sandman", "Metallica", 200),
		mediaPlay(models.MediaTypeSong, "Paranoid", "Black Sabbath", 60),
	}

	stats := computeActionMediaStatistics(playback)
	if stats == nil {
		t.Fatal("expected non-nil stats")
	}
	if stats.Songs != 4 {
		t.Errorf("Songs = %d, want 4", stats.Songs)
	}
	if stats.UniqueArtists != 2 {
		t.Errorf("UniqueArtists = %d, want 2", stats.UniqueArtists)
	}
	if int64(stats.ListeningTime) != 460 {
		t.Errorf("ListeningTime = %d, want 460 seconds", int64(stats.ListeningTime))
	}
	if stats.TopTrack == nil || stats.TopTrack.Title != "One" || stats.TopTrack.Count != 2 {
		t.Errorf("TopTrack = %+v, want One x2", stats.TopTrack)
	}
	if stats.TopArtist == nil || stats.TopArtist.Title != "Metallica" || stats.TopArtist.Count != 3 {
		t.Errorf("TopArtist = %+v, want Metallica x3", stats.TopArtist)
	}
	if stats.SpokenTime != 0 {
		t.Errorf("SpokenTime = %d, want 0", int64(stats.SpokenTime))
	}
}

// TestComputeActionMediaStatisticsSpokenFolded: podcasts/audiobooks fold into SpokenTime
// and never appear in the song tallies.
func TestComputeActionMediaStatisticsSpokenFolded(t *testing.T) {
	playback := []models.MediaPlaybackObject{
		mediaPlay(models.MediaTypeSong, "Run", "Artist", 180),
		mediaPlay(models.MediaTypePodcast, "Episode 12", "Some Show", 1200),
		mediaPlay(models.MediaTypeAudiobook, "Chapter 3", "Author", 1800),
	}

	stats := computeActionMediaStatistics(playback)
	if stats == nil {
		t.Fatal("expected non-nil stats")
	}
	if stats.Songs != 1 {
		t.Errorf("Songs = %d, want 1", stats.Songs)
	}
	if int64(stats.SpokenTime) != 3000 {
		t.Errorf("SpokenTime = %d, want 3000 seconds", int64(stats.SpokenTime))
	}
	if stats.TopTrack == nil || stats.TopTrack.Title != "Run" {
		t.Errorf("TopTrack = %+v, want Run", stats.TopTrack)
	}
}

// TestComputeActionMediaStatisticsSpanFallback: when TrackLength is absent, the play's
// span is derived from StartedAt→EndedAt.
func TestComputeActionMediaStatisticsSpanFallback(t *testing.T) {
	start := time.Now()
	end := start.Add(3 * time.Minute)
	playback := []models.MediaPlaybackObject{
		{MediaType: models.MediaTypeSong, Title: "No Length", Artist: strPtrLocal("A"), StartedAt: start, EndedAt: &end},
	}

	stats := computeActionMediaStatistics(playback)
	if stats == nil {
		t.Fatal("expected non-nil stats")
	}
	if int64(stats.ListeningTime) != 180 {
		t.Errorf("ListeningTime = %d, want 180 seconds", int64(stats.ListeningTime))
	}
}
