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
