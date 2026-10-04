// Command mediaprobe dumps the raw Audiobookshelf listening sessions around a workout
// window, so soundtrack-matching bugs can be diagnosed (and turned into test fixtures)
// from real history instead of guesses. It is a read-only developer tool — it is NOT
// part of the running application, and it touches no database.
//
// Usage:
//
//	ABS_URL=https://abs.example.com ABS_TOKEN=your_token \
//	  go run ./scripts/mediaprobe -from 2026-10-01T17:00:00+02:00 -to 2026-10-01T18:30:00+02:00 -match
//
// -from/-to are the workout window (the session start and start + duration). Stdout gets
// the redacted raw sessions as JSON (every field ABS sends, minus the user id and device
// details); stderr gets a readable table and, with -match, the rows the app's matcher
// produces for the window. The token comes from the ABS account settings.
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/aunefyren/treningheten/controllers"
	"github.com/aunefyren/treningheten/models"
)

const pageSize = 100

// session is one raw ABS listening session, kept as a generic map so the dump carries
// every field ABS sends — including ones the app does not model yet.
type session map[string]any

type options struct {
	baseURL  string
	token    string
	from     time.Time
	to       time.Time
	margin   time.Duration
	maxPages int
	match    bool
}

// probeOutput is the stdout document: the window plus the sessions around it, shaped so
// it can be saved straight into a test fixture.
type probeOutput struct {
	From     time.Time `json:"from"`
	To       time.Time `json:"to"`
	Sessions []session `json:"sessions"`
}

func main() {
	client := &http.Client{Timeout: 30 * time.Second}
	if err := run(os.Args[1:], os.Getenv, client, os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

// run parses the arguments, fetches the history and writes the dump, table and match.
func run(args []string, getenv func(string) string, client *http.Client, out io.Writer, log io.Writer) error {
	opts, err := parseOptions(args, getenv, log)
	if err != nil {
		return err
	}

	raw, err := fetchSessions(client, opts.baseURL, opts.token, opts.from.Add(-opts.margin), opts.maxPages, log)
	if err != nil {
		return err
	}

	sessions := filterSessions(raw, opts.from.Add(-opts.margin), opts.to.Add(opts.margin))
	for _, s := range sessions {
		redactSession(s)
	}
	fmt.Fprintf(log, "%d of %d fetched sessions fall within %s of the window\n\n", len(sessions), len(raw), opts.margin)

	encoder := json.NewEncoder(out)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(probeOutput{From: opts.from, To: opts.to, Sessions: sessions}); err != nil {
		return fmt.Errorf("writing JSON: %w", err)
	}

	writeTable(log, sessions, opts.from.Location())

	if opts.match {
		fmt.Fprintln(log)
		if err := writeMatch(log, sessions, opts.from, opts.to); err != nil {
			return err
		}
	}
	return nil
}

func parseOptions(args []string, getenv func(string) string, log io.Writer) (options, error) {
	opts := options{}

	flags := flag.NewFlagSet("mediaprobe", flag.ContinueOnError)
	flags.SetOutput(log)
	urlFlag := flags.String("url", "", "Audiobookshelf server URL (defaults to ABS_URL env)")
	tokenFlag := flags.String("token", "", "Audiobookshelf API token (defaults to ABS_TOKEN env)")
	fromFlag := flags.String("from", "", "workout start, RFC 3339 (e.g. 2026-10-01T17:00:00+02:00)")
	toFlag := flags.String("to", "", "workout end, RFC 3339")
	flags.DurationVar(&opts.margin, "margin", 30*time.Minute, "context kept either side of the window")
	flags.IntVar(&opts.maxPages, "pages", 20, "maximum history pages to read")
	flags.BoolVar(&opts.match, "match", false, "also run the app's matcher over the window")
	if err := flags.Parse(args); err != nil {
		return opts, err
	}

	opts.baseURL = strings.TrimRight(firstNonEmpty(*urlFlag, getenv("ABS_URL")), "/")
	opts.token = firstNonEmpty(*tokenFlag, getenv("ABS_TOKEN"))
	if opts.baseURL == "" || opts.token == "" {
		return opts, fmt.Errorf("provide the server via -url or ABS_URL and the token via -token or ABS_TOKEN")
	}

	var err error
	if opts.from, err = time.Parse(time.RFC3339, *fromFlag); err != nil {
		return opts, fmt.Errorf("-from: %w", err)
	}
	if opts.to, err = time.Parse(time.RFC3339, *toFlag); err != nil {
		return opts, fmt.Errorf("-to: %w", err)
	}
	if !opts.to.After(opts.from) {
		return opts, fmt.Errorf("-to must be after -from")
	}
	return opts, nil
}

// fetchSessions pages back through /api/me/listening-sessions (most recent first) until
// a page reaches a session last active before `since`, the history runs out, or
// maxPages pages are read.
func fetchSessions(client *http.Client, baseURL, token string, since time.Time, maxPages int, log io.Writer) ([]session, error) {
	all := []session{}

	for page := 0; page < maxPages; page++ {
		url := fmt.Sprintf("%s/api/me/listening-sessions?itemsPerPage=%d&page=%d", baseURL, pageSize, page)
		req, err := http.NewRequest(http.MethodGet, url, nil)
		if err != nil {
			return nil, fmt.Errorf("building request: %w", err)
		}
		req.Header.Set("Accept", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("calling Audiobookshelf: %w", err)
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("Audiobookshelf returned %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
		}

		var reply struct {
			NumPages int       `json:"numPages"`
			Sessions []session `json:"sessions"`
		}
		if err := json.Unmarshal(body, &reply); err != nil {
			return nil, fmt.Errorf("parsing page %d: %w", page, err)
		}
		all = append(all, reply.Sessions...)
		fmt.Fprintf(log, "fetched page %d (%d sessions)\n", page, len(reply.Sessions))

		if len(reply.Sessions) < pageSize || (reply.NumPages > 0 && page+1 >= reply.NumPages) {
			break
		}
		if lastActive(reply.Sessions[len(reply.Sessions)-1]).Before(since) {
			break
		}
	}

	return all, nil
}

// filterSessions keeps the sessions whose [startedAt, last active] span overlaps
// [from, to], ordered by start.
func filterSessions(sessions []session, from, to time.Time) []session {
	kept := []session{}
	for _, s := range sessions {
		started := msTime(s, "startedAt")
		if started.IsZero() || started.After(to) || lastActive(s).Before(from) {
			continue
		}
		kept = append(kept, s)
	}

	sort.SliceStable(kept, func(i, j int) bool {
		return msTime(kept[i], "startedAt").Before(msTime(kept[j], "startedAt"))
	})
	return kept
}

// redactSession strips what identifies the listener: the ABS user id and the device's
// name/IP. The device id is replaced by a short hash so device switches stay visible.
func redactSession(s session) {
	delete(s, "userId")

	device, ok := s["deviceInfo"].(map[string]any)
	if !ok {
		delete(s, "deviceInfo")
		return
	}
	kept := map[string]any{}
	for _, key := range []string{"clientName", "clientVersion", "osName", "osVersion"} {
		if value, ok := device[key]; ok {
			kept[key] = value
		}
	}
	if id, ok := device["deviceId"].(string); ok && id != "" {
		sum := sha256.Sum256([]byte(id))
		kept["deviceHash"] = hex.EncodeToString(sum[:4])
	}
	s["deviceInfo"] = kept
}

// writeTable prints one line per session with the gaps to the previous session of the
// same item, both on the wall clock (start − previous last active) and in the item's
// own position (startTime − previous currentTime). A continuation of one listen shows
// a position gap near zero whatever the wall-clock gap.
func writeTable(w io.Writer, sessions []session, loc *time.Location) {
	table := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(table, "SESSION\tITEM\tTITLE\tSTARTED\tLAST ACTIVE\tSPAN\tLISTENED\tPOSITION\tWALL GAP\tPOS GAP\tDEVICE")

	previous := map[string]session{}
	for _, s := range sessions {
		item := itemID(s)
		started := msTime(s, "startedAt")
		active := lastActive(s)

		wallGap, posGap := "", ""
		if prev, ok := previous[item]; ok {
			wallGap = signedDuration(started.Sub(lastActive(prev)))
			posGap = signedDuration(secondsDuration(number(s, "startTime") - number(prev, "currentTime")))
		}
		previous[item] = s

		fmt.Fprintf(table, "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s→%s\t%s\t%s\t%s\n",
			shorten(text(s, "id"), 8),
			shorten(item, 8),
			shorten(text(s, "displayTitle"), 40),
			started.In(loc).Format("15:04:05"),
			active.In(loc).Format("15:04:05"),
			clock(active.Sub(started)),
			clock(secondsDuration(number(s, "timeListening"))),
			clock(secondsDuration(number(s, "startTime"))),
			clock(secondsDuration(number(s, "currentTime"))),
			wallGap,
			posGap,
			deviceLabel(s),
		)
	}
	table.Flush()
}

// writeMatch replays the sessions through the app's own matcher and prints the rows a
// sync would store, stamped like the timeline (offset into the session, ↑ when the
// item was already playing).
func writeMatch(w io.Writer, sessions []session, from, to time.Time) error {
	encoded, err := json.Marshal(sessions)
	if err != nil {
		return fmt.Errorf("re-encoding sessions: %w", err)
	}
	typed := []models.AudiobookshelfListenSession{}
	if err := json.Unmarshal(encoded, &typed); err != nil {
		return fmt.Errorf("decoding sessions for the matcher: %w", err)
	}

	rows := controllers.MatchAudiobookshelfSessions(typed, from, to)
	fmt.Fprintf(w, "matcher output for %s → %s: %d rows\n", from.Format(time.RFC3339), to.Format(time.RFC3339), len(rows))

	table := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(table, "AT\tTITLE\tLISTENED\tENDS AT\tSESSION")
	for _, row := range rows {
		at := clock(row.StartedAt.Sub(from))
		if row.StartedBefore {
			at = "↑" + at
		}
		listened, ends, sessionID := "-", "-", ""
		if row.TrackLength != nil {
			listened = clock(secondsDuration(float64(*row.TrackLength)))
		}
		if row.EndedAt != nil {
			ends = clock(row.EndedAt.Sub(from))
		}
		if row.ProviderSessionID != nil {
			sessionID = shorten(*row.ProviderSessionID, 8)
		}
		fmt.Fprintf(table, "%s\t%s\t%s\t%s\t%s\n", at, shorten(row.Title, 40), listened, ends, sessionID)
	}
	return table.Flush()
}

// itemID is the most specific thing a session played: the episode, else the library item.
func itemID(s session) string {
	return firstNonEmpty(text(s, "episodeId"), text(s, "libraryItemId"))
}

// lastActive is updatedAt, or startedAt when the session was never updated.
func lastActive(s session) time.Time {
	started, updated := msTime(s, "startedAt"), msTime(s, "updatedAt")
	if updated.After(started) {
		return updated
	}
	return started
}

func deviceLabel(s session) string {
	device, ok := s["deviceInfo"].(map[string]any)
	if !ok {
		return ""
	}
	parts := []string{}
	for _, key := range []string{"clientName", "deviceHash"} {
		if value, ok := device[key].(string); ok && value != "" {
			parts = append(parts, value)
		}
	}
	return strings.Join(parts, " ")
}

func text(s session, key string) string {
	value, _ := s[key].(string)
	return value
}

func number(s session, key string) float64 {
	value, _ := s[key].(float64)
	return value
}

// msTime reads an epoch-milliseconds field; zero when absent.
func msTime(s session, key string) time.Time {
	ms := number(s, key)
	if ms <= 0 {
		return time.Time{}
	}
	return time.UnixMilli(int64(ms)).UTC()
}

func secondsDuration(seconds float64) time.Duration {
	return time.Duration(seconds * float64(time.Second))
}

// clock formats a duration as [h:]mm:ss, with a leading minus when negative.
func clock(d time.Duration) string {
	sign := ""
	if d < 0 {
		sign, d = "-", -d
	}
	total := int64(d.Round(time.Second) / time.Second)
	hours, minutes, seconds := total/3600, total%3600/60, total%60
	if hours > 0 {
		return fmt.Sprintf("%s%d:%02d:%02d", sign, hours, minutes, seconds)
	}
	return fmt.Sprintf("%s%02d:%02d", sign, minutes, seconds)
}

func signedDuration(d time.Duration) string {
	if d >= 0 {
		return "+" + clock(d)
	}
	return clock(d)
}

func shorten(value string, max int) string {
	runes := []rune(value)
	if len(runes) <= max {
		return value
	}
	return string(runes[:max-1]) + "…"
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			return value
		}
	}
	return ""
}
