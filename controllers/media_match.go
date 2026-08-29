package controllers

import (
	"sort"
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
	mediaType string // defaults to song when empty
	title     string
	artist    string
	album     string
	// Provider identity, all optional. providerItemID must be the most specific
	// playable thing the provider names (a podcast episode, not its show) — it is what
	// coalesceOverlappingEvents groups on. See MediaPlayback for what each one is.
	providerItemID    string
	providerParentID  string
	providerGUID      string
	providerSessionID string
	artworkURL        string
	startedAt         time.Time
	// coverageEnd is the real wall-clock end of the listen, when the provider knows it.
	// It lets a long item (a podcast/audiobook started well before the activity but
	// still playing through it) match on interval overlap. Zero = unknown, and the
	// match falls back to start-only — fine for scrobble providers (Plex/Spotify) whose
	// timestamp is logged at finish and so already lands inside the window.
	coverageEnd    time.Time
	trackLengthSec int64 // 0 = unknown
	// trackLengthIsListened distinguishes what trackLengthSec means, which decides how
	// two merged plays combine it: time actually *listened* (Audiobookshelf) adds up,
	// whereas the item's own length (a Plex/Spotify track) does not.
	trackLengthIsListened bool
}

// mediaMergeIdentity is the key overlapping plays are grouped by: the most specific
// provider id available, else the durable GUID, else the item's name. Falling back to
// the name keeps the merge working for providers that give no usable id, at the cost of
// grouping two genuinely different items that share a title — acceptable, since they
// still only merge when their play spans overlap.
func mediaMergeIdentity(event mediaPlayEvent) string {
	if id := strings.TrimSpace(event.providerItemID); id != "" {
		return "item:" + id
	}
	if id := strings.TrimSpace(event.providerGUID); id != "" {
		return "guid:" + id
	}
	return "name:" + strings.ToLower(strings.TrimSpace(event.title)) + "|" +
		strings.ToLower(strings.TrimSpace(event.artist))
}

// coalesceOverlappingEvents merges plays of the same item whose spans overlap into one
// event covering the whole stretch.
//
// Providers routinely report one continuous listen as several records: Audiobookshelf
// opens a fresh listening session on a device switch or an unclean close and leaves the
// old one behind, so a single podcast episode can arrive as three overlapping sessions.
// Rendered raw those become three timeline rows for one thing — and, because the display
// start is clamped to the session start, several of them stack on 00:00.
//
// Overlap is required, not merely a shared identity: replaying an item later in a
// workout is a real second listen and must stay its own row. Touching spans join
// (one ends exactly where the next begins); a real gap does not, so a pause and resume
// still reads as two listens.
func coalesceOverlappingEvents(events []mediaPlayEvent) []mediaPlayEvent {
	if len(events) < 2 {
		return events
	}

	// Sort by start so a single forward pass can absorb every overlapping follower.
	ordered := make([]mediaPlayEvent, len(events))
	copy(ordered, events)
	sort.SliceStable(ordered, func(i, j int) bool {
		return ordered[i].startedAt.Before(ordered[j].startedAt)
	})

	merged := []mediaPlayEvent{}
	// Index of the open (still-extendable) merge for each identity.
	open := map[string]int{}

	for _, event := range ordered {
		identity := mediaMergeIdentity(event)
		end := eventCoverageEnd(event)

		if at, ok := open[identity]; ok && !event.startedAt.After(eventCoverageEnd(merged[at])) {
			merged[at] = mergeEvents(merged[at], event, end)
			continue
		}

		merged = append(merged, event)
		open[identity] = len(merged) - 1
	}

	return merged
}

// eventCoverageEnd is an event's best-known end: the provider's own end when it has one,
// else start + length, else the start itself (a zero-width span).
func eventCoverageEnd(event mediaPlayEvent) time.Time {
	if !event.coverageEnd.IsZero() && event.coverageEnd.After(event.startedAt) {
		return event.coverageEnd
	}
	if event.trackLengthSec > 0 {
		return event.startedAt.Add(time.Duration(event.trackLengthSec) * time.Second)
	}
	return event.startedAt
}

// mergeEvents folds `next` into `into`, keeping the earliest start and latest end and
// filling any metadata the first record was missing.
func mergeEvents(into, next mediaPlayEvent, nextEnd time.Time) mediaPlayEvent {
	if next.startedAt.Before(into.startedAt) {
		into.startedAt = next.startedAt
	}
	if nextEnd.After(eventCoverageEnd(into)) {
		into.coverageEnd = nextEnd
	}

	if into.trackLengthIsListened || next.trackLengthIsListened {
		// Listened time is additive: two sessions on one episode listened to the sum.
		into.trackLengthSec += next.trackLengthSec
		into.trackLengthIsListened = true
	} else if next.trackLengthSec > into.trackLengthSec {
		into.trackLengthSec = next.trackLengthSec
	}

	into.mediaType = firstNonEmpty(into.mediaType, next.mediaType)
	into.title = firstNonEmpty(into.title, next.title)
	into.artist = firstNonEmpty(into.artist, next.artist)
	into.album = firstNonEmpty(into.album, next.album)
	into.artworkURL = firstNonEmpty(into.artworkURL, next.artworkURL)
	into.providerItemID = firstNonEmpty(into.providerItemID, next.providerItemID)
	into.providerParentID = firstNonEmpty(into.providerParentID, next.providerParentID)
	into.providerGUID = firstNonEmpty(into.providerGUID, next.providerGUID)
	// The merged row keeps the first session's id; the others described the same listen.
	into.providerSessionID = firstNonEmpty(into.providerSessionID, next.providerSessionID)

	return into
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

	// Merge duplicate records of one listen before matching, so an item split across
	// several provider records is tested (and displayed) as the single span it was.
	for _, event := range coalesceOverlappingEvents(events) {
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
			// Record that the clamp moved the start, so the timeline can say "already
			// playing" rather than claim the listen began at 00:00.
			StartedBefore: event.startedAt.Before(start),
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
		if id := strings.TrimSpace(event.providerParentID); id != "" {
			row.ProviderParentID = &id
		}
		if id := strings.TrimSpace(event.providerGUID); id != "" {
			row.ProviderGUID = &id
		}
		if id := strings.TrimSpace(event.providerSessionID); id != "" {
			row.ProviderSessionID = &id
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
