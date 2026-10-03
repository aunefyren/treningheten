package controllers

import (
	"testing"

	"github.com/aunefyren/treningheten/files"
	"github.com/aunefyren/treningheten/models"
)

// TestServerInfoMedia pins that a media provider only reports as on when the tenant-wide
// media flag is on too, which is how the API gates it.
func TestServerInfoMedia(t *testing.T) {
	cases := []struct {
		name    string
		media   bool
		plex    bool
		spotify bool
		want    map[string]bool
	}{
		{"all off", false, false, false, map[string]bool{"enabled": false, "plex": false, "spotify": false, "audiobookshelf": false}},
		{"providers on but feature off", false, true, true, map[string]bool{"enabled": false, "plex": false, "spotify": false, "audiobookshelf": false}},
		{"feature on, plex only", true, true, false, map[string]bool{"enabled": true, "plex": true, "spotify": false, "audiobookshelf": false}},
		{"feature on, spotify only", true, false, true, map[string]bool{"enabled": true, "plex": false, "spotify": true, "audiobookshelf": false}},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			h := newAPIHarness(t)
			_, adminToken := h.user("admin@server.test", true)

			previous := files.ConfigFile.Media
			t.Cleanup(func() { files.ConfigFile.Media = previous })
			files.ConfigFile.Media = models.MediaSettings{Enabled: testCase.media}
			files.ConfigFile.Media.Plex.Enabled = testCase.plex
			files.ConfigFile.Media.Spotify.Enabled = testCase.spotify

			server := h.ok("GET", "/api/admin/server-info", adminToken, nil)
			for key, want := range testCase.want {
				if got := field(t, server, "server", "media", key); got != want {
					t.Errorf("media.%s: got %v, want %v", key, got, want)
				}
			}
		})
	}
}
