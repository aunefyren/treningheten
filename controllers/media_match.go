package controllers

import (
	"math"
	"sort"
	"strings"
	"time"

	"github.com/aunefyren/treningheten/models"
)

// mediaMatchGrace widens the match window slightly on each side: manual start times
// are approximate, and a provider's timestamp can land a little off the activity's.
const mediaMatchGrace = 5 * time.Minute

// Resumed-listen merging (listened-time items only). A pause shorter than
// mediaResumeMaxPause doesn't split a listen into two rows; a resume must also pick up
// within mediaResumePositionSlack of where the previous record stopped, so restarting
// an episode from the top stays a separate listen.
const (
	mediaResumeMaxPause      = 15 * time.Minute
	mediaResumePositionSlack = 60.0 // seconds of item position
)

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
	// It lets an item that started before the activity but played into it (a long
	// podcast, or a scrobbled track that finished early in the run) match on interval
	// overlap. Zero = unknown, and the match falls back to start-only.
	coverageEnd    time.Time
	trackLengthSec int64 // 0 = unknown
	// trackLengthIsListened distinguishes what trackLengthSec means, which decides how
	// two merged plays combine it: time actually *listened* (Audiobookshelf) adds up,
	// whereas the item's own length (a Plex/Spotify track) does not.
	trackLengthIsListened bool
	// Playback positions within the item (seconds) at the start and end of the record,
	// when the provider reports them (Audiobookshelf). A record that starts where the
	// previous one stopped is a resumed listen.
	hasPosition   bool
	positionStart float64
	positionEnd   float64
	// listenedInWindowSec is the listened time inside the activity window, computed per
	// record before merging (playbackForWindow) so each record's own span is used for
	// clipping, and summed when records merge.
	listenedInWindowSec int64
	// titleWeightSec is the length of the record that supplied the title. A merge takes
	// the title of its longest record: some clients log seconds-long blips named after
	// the show rather than the episode.
	titleWeightSec int64
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
// (one ends exactly where the next begins). The exception is a resumed listen of a
// listened-time item (see continuesListen): a short pause doesn't split it.
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

		if at, ok := open[identity]; ok && continuesListen(merged[at], event) {
			merged[at] = mergeEvents(merged[at], event, end)
			continue
		}

		merged = append(merged, event)
		open[identity] = len(merged) - 1
	}

	return merged
}

// continuesListen reports whether `next` (starting no earlier than `previous`) belongs
// to the same listen: their spans overlap or touch, or — for listened-time items with
// positions — `next` resumes where `previous` stopped after a pause of at most
// mediaResumeMaxPause.
func continuesListen(previous, next mediaPlayEvent) bool {
	previousEnd := eventCoverageEnd(previous)
	if !next.startedAt.After(previousEnd) {
		return true
	}

	if !previous.trackLengthIsListened || !next.trackLengthIsListened ||
		!previous.hasPosition || !next.hasPosition {
		return false
	}
	return next.startedAt.Sub(previousEnd) <= mediaResumeMaxPause &&
		math.Abs(next.positionStart-previous.positionEnd) <= mediaResumePositionSlack
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
	// The longest record names the merge; weigh before the lengths are combined.
	intoWeight := into.titleWeightSec
	if intoWeight == 0 {
		intoWeight = into.trackLengthSec
	}
	into.titleWeightSec = intoWeight
	if strings.TrimSpace(next.title) != "" && next.trackLengthSec > intoWeight {
		into.title = next.title
		into.titleWeightSec = next.trackLengthSec
	}

	if next.startedAt.Before(into.startedAt) {
		into.startedAt = next.startedAt
	}
	if nextEnd.After(eventCoverageEnd(into)) {
		into.coverageEnd = nextEnd
		// The listen now stops where the later record stopped.
		into.positionEnd = next.positionEnd
	}
	into.hasPosition = into.hasPosition && next.hasPosition
	into.listenedInWindowSec += next.listenedInWindowSec

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

// scrobbleSpan turns a scrobble — a play logged when it *finished* (Plex viewedAt,
// Spotify played_at) — into its play span: it started one track length earlier. With
// the length unknown the finish time is all there is, so it stands in as the start and
// coverageEnd stays zero (the start-only match).
func scrobbleSpan(finishedAt time.Time, lengthSec int64) (startedAt, coverageEnd time.Time) {
	if lengthSec <= 0 {
		return finishedAt, time.Time{}
	}
	return finishedAt.Add(-time.Duration(lengthSec) * time.Second), finishedAt
}

// listenedWithinWindow scales a listened-time item (an Audiobookshelf session) down to
// the part that fell inside the activity window. The provider's listened total covers
// the whole listen — including any before the workout began or after it ended — so
// shown raw, a podcast started before a run and finished after it claims more minutes
// than the run lasted. Pauses are assumed spread evenly over the wall-clock span: the
// result is listened × overlap ÷ span, capped at the overlap itself — merged duplicate
// sessions can report more listened time than wall-clock span, and nobody listens
// longer than the window they were in. With no known end the span is the listened time
// itself, so this reduces to the overlap.
func listenedWithinWindow(event mediaPlayEvent, start, end time.Time) int64 {
	if event.trackLengthSec <= 0 {
		return 0
	}

	// With a positive listened time the span is always positive: eventCoverageEnd falls
	// back to start + listened when the provider gave no later end.
	spanEnd := eventCoverageEnd(event)
	span := spanEnd.Sub(event.startedAt)

	overlap := windowOverlap(event.startedAt, spanEnd, start, end)
	if overlap <= 0 {
		return 0
	}

	listened := int64(math.Round(float64(event.trackLengthSec) * overlap.Seconds() / span.Seconds()))
	if capSec := int64(overlap.Seconds()); listened > capSec {
		listened = capSec
	}
	return listened
}

// windowOverlap is how much of [from, to] falls inside [start, end]; zero when disjoint.
func windowOverlap(from, to, start, end time.Time) time.Duration {
	if from.Before(start) {
		from = start
	}
	if to.After(end) {
		to = end
	}
	if overlap := to.Sub(from); overlap > 0 {
		return overlap
	}
	return 0
}

// listenedInWindow clips each listened-time record to the window and drops the records
// with nothing heard inside it, before merging. Without this a listen that stopped just
// before the workout matched through the grace and showed as "already playing" with no
// minutes, and a pre-workout record within a pause of the workout's first one would pull
// the merged start back before the session.
func listenedInWindow(events []mediaPlayEvent, start, end time.Time) []mediaPlayEvent {
	kept := make([]mediaPlayEvent, 0, len(events))
	for _, event := range events {
		if event.trackLengthIsListened && event.trackLengthSec > 0 {
			event.listenedInWindowSec = listenedWithinWindow(event, start, end)
			if event.listenedInWindowSec == 0 {
				continue
			}
		}
		kept = append(kept, event)
	}
	return kept
}

// playbackForWindow keeps the events whose play span overlaps the activity window
// (plus grace) and turns them into MediaPlayback rows. StartedAt is the play start,
// clamped up to the activity start; EndedAt is the known end (else StartedAt + track
// length), clamped to the activity end when the item started inside the activity.
// TrackLength is the item length, or for listened-time items the time listened within
// the window (listenedWithinWindow). Identity fields (id/exercise/provider) are filled
// later by
// ReplaceMediaPlaybackForExerciseProvider.
func playbackForWindow(events []mediaPlayEvent, start, end time.Time) []models.MediaPlayback {
	playback := []models.MediaPlayback{}

	matchStart := start.Add(-mediaMatchGrace)
	matchEnd := end.Add(mediaMatchGrace)

	// Merge duplicate records of one listen before matching, so an item split across
	// several provider records is tested (and displayed) as the single span it was.
	for _, event := range coalesceOverlappingEvents(listenedInWindow(events, start, end)) {
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

		length := event.trackLengthSec
		if event.trackLengthIsListened {
			// The records' in-window shares, capped at the merged span's overlap: merged
			// duplicate records can claim more listening than the time they cover.
			length = event.listenedInWindowSec
			overlap := int64(windowOverlap(event.startedAt, eventCoverageEnd(event), start, end).Seconds())
			if length > overlap {
				length = overlap
			}
		}
		if length > 0 {
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
