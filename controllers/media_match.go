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
	trackLengthSec int64 // 0 = unknown
}

// playbackForWindow keeps the events whose start time falls within the activity
// window (plus grace) and turns them into MediaPlayback rows. StartedAt is the play
// time; EndedAt is StartedAt + track length, clamped to the activity end when the
// track actually started inside the activity. Identity fields (id/operation/
// provider) are filled later by ReplaceMediaPlaybackForOperationProvider.
func playbackForWindow(events []mediaPlayEvent, start, end time.Time) []models.MediaPlayback {
	playback := []models.MediaPlayback{}

	matchStart := start.Add(-mediaMatchGrace)
	matchEnd := end.Add(mediaMatchGrace)

	for _, event := range events {
		if event.startedAt.IsZero() || event.startedAt.Before(matchStart) || event.startedAt.After(matchEnd) {
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

		row := models.MediaPlayback{
			MediaType: mediaType,
			Title:     title,
			StartedAt: event.startedAt,
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
			ended := event.startedAt.Add(time.Duration(length) * time.Second)
			if !event.startedAt.After(end) && ended.After(end) {
				ended = end
			}
			row.EndedAt = &ended
		}

		playback = append(playback, row)
	}

	return playback
}
