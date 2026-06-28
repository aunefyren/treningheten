package controllers

import (
	"net/url"
	"strings"
	"testing"

	"github.com/aunefyren/treningheten/models"
)

func TestBuildPlexAuthURL(t *testing.T) {
	got := buildPlexAuthURL("client-123", "ABCD", "Treningheten")

	prefix := plexAuthAppURL + "#?"
	if !strings.HasPrefix(got, prefix) {
		t.Fatalf("expected auth url to start with %q, got %q", prefix, got)
	}

	// The params live in the fragment; parse them to assert they round-trip.
	v, err := url.ParseQuery(strings.TrimPrefix(got, prefix))
	if err != nil {
		t.Fatalf("failed to parse auth url fragment: %v", err)
	}
	if v.Get("clientID") != "client-123" {
		t.Errorf("clientID: got %q", v.Get("clientID"))
	}
	if v.Get("code") != "ABCD" {
		t.Errorf("code: got %q", v.Get("code"))
	}
	if v.Get("context[device][product]") != "Treningheten" {
		t.Errorf("product: got %q", v.Get("context[device][product]"))
	}
}

func TestRankPlexServerConnections(t *testing.T) {
	t.Run("no resources yields no candidates", func(t *testing.T) {
		if got := rankPlexServerConnections(nil); len(got) != 0 {
			t.Errorf("expected no candidates, got %v", got)
		}
	})

	t.Run("ignores non-server resources and URI-less connections", func(t *testing.T) {
		resources := []models.PlexResource{
			{Provides: "client", Connections: []models.PlexConnection{{URI: "https://phone", Protocol: "https"}}},
			{Provides: "server", Connections: []models.PlexConnection{{URI: "", Protocol: "https"}}},
		}
		if got := rankPlexServerConnections(resources); len(got) != 0 {
			t.Errorf("expected no candidates, got %v", got)
		}
	})

	t.Run("orders non-relay local https first, relay last", func(t *testing.T) {
		resources := []models.PlexResource{
			{
				Provides: "server",
				Owned:    true,
				Connections: []models.PlexConnection{
					{URI: "https://relay.plex.direct", Protocol: "https", Relay: true},
					{URI: "https://remote.plex.direct:32400", Protocol: "https", Local: false},
					{URI: "https://local.plex.direct:32400", Protocol: "https", Local: true},
					{URI: "http://192.168.1.10:32400", Protocol: "http", Local: true},
				},
			},
		}
		got := rankPlexServerConnections(resources)
		want := []string{
			"https://local.plex.direct:32400",  // non-relay + local + https
			"http://192.168.1.10:32400",        // non-relay + local
			"https://remote.plex.direct:32400", // non-relay + https (remote)
			"https://relay.plex.direct",        // relay last
		}
		if len(got) != len(want) {
			t.Fatalf("got %d candidates, want %d (%v)", len(got), len(want), got)
		}
		for i := range want {
			if got[i] != want[i] {
				t.Errorf("position %d: got %q, want %q (full: %v)", i, got[i], want[i], got)
			}
		}
	})
}
