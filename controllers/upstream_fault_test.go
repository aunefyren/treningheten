package controllers

import (
	"fmt"
	"net/http"
	"sync"
	"testing"

	"github.com/aunefyren/treningheten/database"
	"github.com/aunefyren/treningheten/internal/testdb"
)

// upstreamFaults fails exactly the Nth request any fake third-party server (Strava,
// Hevy, Plex, Spotify, Audiobookshelf, Ollama, push) receives after it is armed, in one
// of three ways — the failures a real provider produces.
type upstreamFaultMode int

const (
	upstreamServerError upstreamFaultMode = iota // HTTP 500
	upstreamGarbage                              // 200 with a body that isn't JSON
	upstreamHangUp                               // connection closed without a response
)

var upstreamFaults = &struct {
	sync.Mutex
	armed  bool
	failAt int
	seen   int
	mode   upstreamFaultMode
}{}

// injectUpstreamFault is called first by every fake handler. It reports whether it
// already answered (with the injected failure).
func injectUpstreamFault(writer http.ResponseWriter) bool {
	upstreamFaults.Lock()
	armed, fail, mode := upstreamFaults.armed, false, upstreamFaults.mode
	if armed {
		upstreamFaults.seen++
		fail = upstreamFaults.seen == upstreamFaults.failAt
	}
	upstreamFaults.Unlock()
	if !fail {
		return false
	}

	switch mode {
	case upstreamServerError:
		writer.WriteHeader(http.StatusInternalServerError)
	case upstreamGarbage:
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte("<html>not json</html>"))
	case upstreamHangUp:
		if hijacker, ok := writer.(http.Hijacker); ok {
			if connection, _, err := hijacker.Hijack(); err == nil {
				_ = connection.Close()
				return true
			}
		}
		writer.WriteHeader(http.StatusBadGateway)
	}
	return true
}

func armUpstreamFaults(failAt int, mode upstreamFaultMode) {
	upstreamFaults.Lock()
	defer upstreamFaults.Unlock()
	upstreamFaults.armed, upstreamFaults.failAt, upstreamFaults.seen, upstreamFaults.mode = true, failAt, 0, mode
}

func disarmUpstreamFaults() int {
	upstreamFaults.Lock()
	defer upstreamFaults.Unlock()
	upstreamFaults.armed = false
	return upstreamFaults.seen
}

// sweepUpstream counts a scenario's upstream requests, then replays it once per request
// and failure mode with that request failing. Integrations are allowed to degrade (many
// deliberately keep going when a provider hiccups); they must never panic.
func sweepUpstream(t *testing.T, scenario faultScenario) {
	t.Helper()

	run := func(failAt int, mode upstreamFaultMode) (calls int) {
		t.Run(fmt.Sprintf("upstream-%d-mode-%d", failAt, mode), func(t *testing.T) {
			h := newAPIHarness(t)
			_ = database.Instance
			request := scenario.setup(t, h)

			armUpstreamFaults(failAt, mode)
			defer func() {
				calls = disarmUpstreamFaults()
				if recovered := recover(); recovered != nil {
					t.Errorf("%s: panicked with upstream request %d failing (mode %d): %v", scenario.name, failAt, mode, recovered)
				}
			}()
			if request.call != nil {
				_ = request.call()
				return
			}
			h.do(request.method, request.path, request.token, request.body)
		})
		return calls
	}

	total := run(0, upstreamServerError)
	if total == 0 {
		t.Fatalf("%s: made no upstream requests; the scenario doesn't reach the integration", scenario.name)
	}
	for failAt := 1; failAt <= total; failAt++ {
		for _, mode := range []upstreamFaultMode{upstreamServerError, upstreamGarbage, upstreamHangUp} {
			run(failAt, mode)
		}
	}
}

func TestUpstreamFaultSweep(t *testing.T) {
	if testing.Short() {
		t.Skip("upstream sweep replays every integration scenario per request and failure mode")
	}
	// A fresh database per injected failure; it exercises error handling, not SQL.
	if testdb.Backend() != "sqlite" {
		t.Skip("upstream sweep runs on SQLite only")
	}
	integrations := map[string]bool{
		"strava connect and sync": true, "strava resync": true, "strava sync by activity id": true, "strava set resync": true,
		"hevy backfill": true, "hevy events": true, "hevy connect": true,
		"audiobookshelf connect": true, "media sync": true, "media reconcile": true,
		"plex pin": true, "plex pin check": true, "plex server override": true, "plex sync": true, "plex artwork": true,
		"integration health check": true, "plex broken connection": true, "plex recovery": true,
		"spotify connect": true, "spotify sync": true, "ollama greeting": true, "push all devices": true,
	}
	all := append(append(faultScenarios(), moreFaultScenarios()...), richFaultScenarios()...)
	swept := 0
	for _, scenario := range all {
		if !integrations[scenario.name] {
			continue
		}
		scenario := scenario
		t.Run(scenario.name, func(t *testing.T) { sweepUpstream(t, scenario) })
		swept++
	}
	if swept != len(integrations) {
		t.Errorf("swept %d integration scenarios, want %d — a scenario was renamed", swept, len(integrations))
	}
}
