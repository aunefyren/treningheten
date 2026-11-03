package controllers

import (
	"encoding/json"
	"math"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/aunefyren/treningheten/models"
)

// absFixtureCase is one workout window and the real listening sessions around it, as
// dumped by scripts/mediaprobe and trimmed to the modelled fields.
type absFixtureCase struct {
	From     time.Time                            `json:"from"`
	To       time.Time                            `json:"to"`
	Sessions []models.AudiobookshelfListenSession `json:"sessions"`
}

// wantTimelineRow is a soundtrack row as the timeline shows it: the offset into the
// session, the "already playing" mark, the title and the minutes listened.
type wantTimelineRow struct {
	at            time.Duration
	startedBefore bool
	titlePrefix   string
	minutes       int64
}

// TestBuildAudiobookshelfPlaybackFromRealSessions replays real ABS history through the
// matcher. The AudioBooth windows are the three reported soundtracks, reconstructed from
// their screenshots; the Abs iOS ones cover the TestFlight app's null listened time and a
// device switch mid-episode.
func TestBuildAudiobookshelfPlaybackFromRealSessions(t *testing.T) {
	raw, err := os.ReadFile("testdata/abs_listening_sessions.json")
	if err != nil {
		t.Fatal(err)
	}
	fixtures := map[string]absFixtureCase{}
	if err := json.Unmarshal(raw, &fixtures); err != nil {
		t.Fatal(err)
	}

	minSec := func(minutes, seconds int) time.Duration {
		return time.Duration(minutes)*time.Minute + time.Duration(seconds)*time.Second
	}

	tests := []struct {
		name string
		want []wantTimelineRow
	}{
		{
			// Was: ↑00:00 1 min (a 2-minute listen paused 34 minutes before the run, its
			// session only closed on resume), then Ep. 10 split in two by a 2-second gap.
			name: "audiobooth-ep10",
			want: []wantTimelineRow{
				{at: minSec(1, 37), titlePrefix: "Ep. 10", minutes: 52},
				{at: minSec(54, 16), titlePrefix: "Ep. 11", minutes: 65},
			},
		},
		{
			// Was: Ep. 13 as three rows split by ~11- and ~10-minute pauses — under the
			// 15-minute resume threshold, so one listen.
			name: "audiobooth-ep13",
			want: []wantTimelineRow{
				{at: minSec(0, 10), titlePrefix: "Ep. 12", minutes: 46},
				{at: minSec(45, 51), titlePrefix: "Ep. 13", minutes: 51},
			},
		},
		{
			// Was: ↑00:00 21 min, ↑00:00 with no minutes (paused the night before, closed
			// when resumed two minutes before the run), and 20:57 22 min.
			name: "audiobooth-ep25",
			want: []wantTimelineRow{
				{at: 0, startedBefore: true, titlePrefix: "Ep. 25", minutes: 43},
				{at: minSec(43, 22), titlePrefix: "Ep. 26", minutes: 13},
			},
		},
		{
			// An 18-second session titled with the show ("The Yard") resumes into the
			// episode; the merged row takes the longer session's episode title.
			name: "audiobooth-show-titled-blip",
			want: []wantTimelineRow{
				{at: minSec(4, 2), titlePrefix: "Ep. 269", minutes: 21},
			},
		},
		{
			// Abs iOS sent timeListening as null: the position delta stands in.
			name: "absios-null-listened",
			want: []wantTimelineRow{
				{at: minSec(5, 34), titlePrefix: "Ep. 270", minutes: 21},
			},
		},
		{
			// Abs Web then Abs iOS, 40 seconds apart, the position carried over.
			name: "absios-cross-device",
			want: []wantTimelineRow{
				{at: minSec(0, 42), titlePrefix: "Ep. 50", minutes: 57},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fixture, ok := fixtures[test.name]
			if !ok {
				t.Fatalf("fixture %q missing", test.name)
			}

			got := buildAudiobookshelfPlaybackForWindow(fixture.Sessions, fixture.From, fixture.To)
			if len(got) != len(test.want) {
				for _, row := range got {
					t.Logf("row: %s at %s", row.Title, row.StartedAt.Sub(fixture.From))
				}
				t.Fatalf("got %d rows, want %d", len(got), len(test.want))
			}

			for i, want := range test.want {
				row := got[i]
				at := row.StartedAt.Sub(fixture.From).Truncate(time.Second) // the timeline drops part-seconds
				minutes := int64(-1)
				if row.TrackLength != nil {
					minutes = int64(math.Round(float64(*row.TrackLength) / 60))
				}
				if !strings.HasPrefix(row.Title, want.titlePrefix) || at != want.at ||
					row.StartedBefore != want.startedBefore || minutes != want.minutes {
					t.Errorf("row %d = %q at %s (before %t) %d min; want %q… at %s (before %t) %d min",
						i, row.Title, at, row.StartedBefore, minutes,
						want.titlePrefix, want.at, want.startedBefore, want.minutes)
				}
			}
		})
	}
}

// TestAudiobookshelfPausedInsideSession covers #41: the ABS iOS app kept a pause inside
// one session (15:44:41-16:39:36, 26:30 listened) around a 15:59-16:37:54 run. The pause
// can't be located from the session, so the run's other listening decides: the synthetic
// Plex row stands in for the tracks the maintainer saw from 01:12 into the run (they could
// not be pulled with mediaprobe, which only reads ABS) and ends at 16:20:30.
func TestAudiobookshelfPausedInsideSession(t *testing.T) {
	raw, err := os.ReadFile("testdata/abs_listening_sessions.json")
	if err != nil {
		t.Fatal(err)
	}
	fixtures := map[string]absFixtureCase{}
	if err := json.Unmarshal(raw, &fixtures); err != nil {
		t.Fatal(err)
	}
	fixture := fixtures["absios-paused-inside-session"]
	at := func(clock string) time.Time {
		parsed, err := time.Parse("2006-01-02 15:04:05 -07:00", "2026-10-27 "+clock+" +01:00")
		if err != nil {
			t.Fatal(err)
		}
		return parsed
	}

	tests := []struct {
		name        string
		otherEnds   []time.Time
		wantAt      time.Duration
		wantBefore  bool
		wantMinutes int64
	}{
		{
			// Resumes when the last Plex track ended, 21:30 into the run.
			name:        "resumes where the last other listen ended",
			otherEnds:   []time.Time{at("16:05:00"), at("16:20:30")},
			wantAt:      21*time.Minute + 30*time.Second,
			wantMinutes: 17,
		},
		{
			// Nothing else to narrow the pause: today's even spread, "already playing".
			name:        "no other listening keeps the even spread",
			wantAt:      0,
			wantBefore:  true,
			wantMinutes: 19,
		},
		{
			// An other listen that ended before the session even started says nothing.
			name:        "other listening before the session is ignored",
			otherEnds:   []time.Time{at("15:30:00")},
			wantAt:      0,
			wantBefore:  true,
			wantMinutes: 19,
		},
		{
			// Resuming at 15:50 would make 49 minutes of playing out of 26:30 listened.
			name:        "a stretch longer than the listened time falls back",
			otherEnds:   []time.Time{at("15:50:00")},
			wantAt:      0,
			wantBefore:  true,
			wantMinutes: 19,
		},
		{
			// Resuming at 16:39:00 leaves under a minute, but the session must have
			// played at least 10 minutes inside the run.
			name:        "a stretch too short for the listening falls back",
			otherEnds:   []time.Time{at("16:39:00")},
			wantAt:      0,
			wantBefore:  true,
			wantMinutes: 19,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := buildAudiobookshelfPlaybackWithOthers(fixture.Sessions, fixture.From, fixture.To, test.otherEnds)
			if len(got) != 1 {
				t.Fatalf("got %d rows, want 1", len(got))
			}
			row := got[0]
			start := row.StartedAt.Sub(fixture.From).Truncate(time.Second)
			minutes := int64(-1)
			if row.TrackLength != nil {
				minutes = int64(math.Round(float64(*row.TrackLength) / 60))
			}
			if start != test.wantAt || row.StartedBefore != test.wantBefore || minutes != test.wantMinutes {
				t.Errorf("row at %s (before %t) %d min; want at %s (before %t) %d min",
					start, row.StartedBefore, minutes, test.wantAt, test.wantBefore, test.wantMinutes)
			}
		})
	}
}
