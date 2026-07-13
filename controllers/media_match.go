package controllers

import (
	"strings"
	"time"

	"github.com/aunefyren/treningheten/models"
)

// mediaMatchGrace widens the match window slightly on each side: a track is logged
// when it finishes (Plex viewedAt / Spotify played_at), which can fall just after
// the activity ends, and manual start times are approximate.
const mediaMatchGrace = 5 * time.Minute

// mediaPlayEvent is a provider-neutral played item. Each provider maps its own
// history payload into these, then calls playbackForWindow — so the timestamp-
// overlap matching and the EndedAt clamp live in one place rather than per provider.
type mediaPlayEvent struct {
	mediaType      string // defaults to song when empty
	title          string
	artist         string
	album          string
	providerItemID string
	artworkURL     string
	startedAt      time.Time
	// coverageEnd is the real wall-clock end of the listen, when the provider knows it.
	// It lets a long item (a podcast/audiobook started well before the activity but
	// still playing through it) match on interval overlap. Zero = unknown, and the
	// match falls back to start-only — fine for scrobble providers (Plex/Spotify) whose
	// timestamp is logged at finish and so already lands inside the window.
	coverageEnd    time.Time
	trackLengthSec int64 // 0 = unknown
}

// playbackForWindow keeps the events whose start time falls within the activity
// window (plus grace) and turns them into MediaPlayback rows. StartedAt is the play
// time; EndedAt is StartedAt + track length, clamped to the activity end when the
// track actually started inside the activity. Identity fields (id/exercise/
// provider) are filled later by ReplaceMediaPlaybackForExerciseProvider.
func playbackForWindow(events []mediaPlayEvent, start, end time.Time) []models.MediaPlayback {
	playback := []models.MediaPlayback{}

	matchStart := start.Add(-mediaMatchGrace)
	matchEnd := end.Add(mediaMatchGrace)

	for _, event := range events {
		if event.startedAt.IsZero() {
			continue
		}

		// Match on interval overlap: keep the event when its play span
		// [startedAt, coverageEnd] intersects the (grace-widened) activity window.
		// With coverageEnd unknown this collapses to the original start-only test.
		coverageEnd := event.coverageEnd
		if coverageEnd.Before(event.startedAt) {
			coverageEnd = event.startedAt
		}
		if event.startedAt.After(matchEnd) || coverageEnd.Before(matchStart) {
			continue
		}

		mediaType := event.mediaType
		if mediaType == "" {
			mediaType = models.MediaTypeSong
		}
		title := strings.TrimSpace(event.title)
		if title == "" {
			title = "Unknown"
		}

		// Clamp the displayed start up to the activity start, so an item that began
		// before the workout renders its overlapping portion rather than spilling left.
		displayStart := event.startedAt
		if displayStart.Before(start) {
			displayStart = start
		}

		row := models.MediaPlayback{
			MediaType: mediaType,
			Title:     title,
			StartedAt: displayStart,
		}
		if artist := strings.TrimSpace(event.artist); artist != "" {
			row.Artist = &artist
		}
		if album := strings.TrimSpace(event.album); album != "" {
			row.Album = &album
		}
		if id := strings.TrimSpace(event.providerItemID); id != "" {
			row.ProviderItemID = &id
		}
		if art := strings.TrimSpace(event.artworkURL); art != "" {
			row.ArtworkURL = &art
		}

		if event.trackLengthSec > 0 {
			length := event.trackLengthSec
			row.TrackLength = &length
		}

		// EndedAt is the display span end: the real end when the provider knows it
		// (coverageEnd), otherwise startedAt + track length. Clamp to the activity end
		// when the item started inside it, and never let it precede the clamped start.
		ended := time.Time{}
		if !event.coverageEnd.IsZero() {
			ended = event.coverageEnd
		} else if event.trackLengthSec > 0 {
			ended = event.startedAt.Add(time.Duration(event.trackLengthSec) * time.Second)
		}
		if !ended.IsZero() {
			if !event.startedAt.After(end) && ended.After(end) {
				ended = end
			}
			if ended.Before(displayStart) {
				ended = displayStart
			}
			row.EndedAt = &ended
		}

		playback = append(playback, row)
	}

	return playback
}
