package controllers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aunefyren/treningheten/files"
	"github.com/aunefyren/treningheten/models"
)

// plexConnection is a shorthand for building a candidate connection.
func plexConnection(uri, protocol string, local, relay bool) models.PlexConnection {
	return models.PlexConnection{URI: uri, Protocol: protocol, Local: local, Relay: relay}
}

// TestProbePlexServer covers the reachability probe: a 200 from /identity means reachable,
// anything else (or an unreachable host) does not, and the Plex auth headers are sent.
func TestProbePlexServer(t *testing.T) {
	var probedPath, token, clientID string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		probedPath = request.URL.Path
		token = request.Header.Get("X-Plex-Token")
		clientID = request.Header.Get("X-Plex-Client-Identifier")
		writer.Write([]byte(`{"MediaContainer":{"machineIdentifier":"abc"}}`))
	}))
	defer server.Close()

	if !probePlexServer(server.URL, "the-token") {
		t.Errorf("a reachable server probed as unreachable")
	}
	if probedPath != "/identity" {
		t.Errorf("probed %q, want /identity", probedPath)
	}
	if token != "the-token" {
		t.Errorf("X-Plex-Token = %q, want the token", token)
	}
	if clientID != files.ConfigFile.Media.Plex.ClientIdentifier {
		t.Errorf("X-Plex-Client-Identifier = %q, want the configured client id", clientID)
	}

	// A trailing slash must not produce a doubled path.
	if !probePlexServer(server.URL+"/", "the-token") {
		t.Errorf("a trailing slash broke the probe")
	}
	if probedPath != "/identity" {
		t.Errorf("probed %q after a trailing slash, want /identity", probedPath)
	}

	t.Run("unauthorized is not reachable", func(t *testing.T) {
		unauthorized := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			writer.WriteHeader(http.StatusUnauthorized)
		}))
		defer unauthorized.Close()

		if probePlexServer(unauthorized.URL, "the-token") {
			t.Errorf("a server rejecting the token probed as reachable")
		}
	})

	t.Run("dead host is not reachable", func(t *testing.T) {
		dead := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
		deadURL := dead.URL
		dead.Close() // nothing is listening now

		if probePlexServer(deadURL, "the-token") {
			t.Errorf("a closed server probed as reachable")
		}
		if probePlexServer("://not a url", "the-token") {
			t.Errorf("an unparseable URI probed as reachable")
		}
	})
}

// TestSelectReachablePlexServer covers discovery end to end: the ranked candidates are
// probed in order and the first reachable one wins, even when a higher-ranked candidate is
// unreachable — the case that motivated probe-based discovery over "trust the first URI".
func TestSelectReachablePlexServer(t *testing.T) {
	reachable := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Write([]byte(`{}`))
	}))
	defer reachable.Close()

	unreachable := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	unreachableURL := unreachable.URL
	unreachable.Close()

	// The unreachable LAN address outranks the reachable WAN one, so it is tried first.
	resources := []models.PlexResource{{
		Provides: "server",
		Connections: []models.PlexConnection{
			plexConnection(reachable.URL, "http", false, false),
			plexConnection(unreachableURL, "http", true, false),
		},
	}}

	if got := selectReachablePlexServer(resources, "the-token"); got != reachable.URL {
		t.Errorf("selected %q, want the reachable server %q", got, reachable.URL)
	}

	t.Run("nothing reachable", func(t *testing.T) {
		resources := []models.PlexResource{{
			Provides:    "server",
			Connections: []models.PlexConnection{plexConnection(unreachableURL, "http", true, false)},
		}}
		if got := selectReachablePlexServer(resources, "the-token"); got != "" {
			t.Errorf("selected %q, want empty when nothing is reachable", got)
		}
	})

	if got := selectReachablePlexServer(nil, "the-token"); got != "" {
		t.Errorf("selected %q from no resources, want empty", got)
	}
}

// TestResolvePlexServerAccountID covers the privacy-critical mapping from a plex.tv
// identity to the server-local account id history is scoped by. It must return "" — which
// makes the caller fail closed and store nothing — rather than guessing, since a wrong id
// would attribute another household member's listening to this user.
func TestResolvePlexServerAccountID(t *testing.T) {
	const accountsBody = `{"MediaContainer":{"Account":[
		{"id":0,"name":""},
		{"id":1,"name":"OwnerName"},
		{"id":42,"name":"Housemate"}
	]}}`

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/accounts" {
			t.Errorf("requested %q, want /accounts", request.URL.Path)
		}
		writer.Write([]byte(accountsBody))
	}))
	defer server.Close()

	if got := resolvePlexServerAccountID(server.URL, "the-token", "OwnerName"); got != "1" {
		t.Errorf("account id = %q, want 1", got)
	}
	// Matching is case-insensitive and tolerates surrounding whitespace.
	if got := resolvePlexServerAccountID(server.URL, "the-token", "  ownername  "); got != "1" {
		t.Errorf("account id = %q for a differently-cased name, want 1", got)
	}
	// The first matching candidate wins; a non-matching one is skipped.
	if got := resolvePlexServerAccountID(server.URL, "the-token", "unknown@example.com", "Housemate"); got != "42" {
		t.Errorf("account id = %q, want 42", got)
	}

	t.Run("no match fails closed", func(t *testing.T) {
		if got := resolvePlexServerAccountID(server.URL, "the-token", "SomeoneElse"); got != "" {
			t.Errorf("account id = %q, want empty so the caller fails closed", got)
		}
		if got := resolvePlexServerAccountID(server.URL, "the-token"); got != "" {
			t.Errorf("account id = %q with no candidates, want empty", got)
		}
		if got := resolvePlexServerAccountID(server.URL, "the-token", "", "   "); got != "" {
			t.Errorf("account id = %q for blank candidates, want empty", got)
		}
	})

	t.Run("an id of zero is never used", func(t *testing.T) {
		if got := resolvePlexServerAccountID(server.URL, "the-token", ""); got != "" {
			t.Errorf("account id = %q, want the id-0 row skipped", got)
		}
	})

	t.Run("server errors fail closed", func(t *testing.T) {
		failing := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			writer.WriteHeader(http.StatusForbidden)
		}))
		defer failing.Close()
		if got := resolvePlexServerAccountID(failing.URL, "the-token", "OwnerName"); got != "" {
			t.Errorf("account id = %q on a 403, want empty", got)
		}

		malformed := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			writer.Write([]byte(`not json`))
		}))
		defer malformed.Close()
		if got := resolvePlexServerAccountID(malformed.URL, "the-token", "OwnerName"); got != "" {
			t.Errorf("account id = %q on malformed JSON, want empty", got)
		}

		if got := resolvePlexServerAccountID("://not a url", "the-token", "OwnerName"); got != "" {
			t.Errorf("account id = %q for an unparseable URL, want empty", got)
		}
	})
}
