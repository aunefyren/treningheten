package controllers

import (
	"testing"

	"github.com/aunefyren/treningheten/models"
)

func TestResolveMediaArtworkURL(t *testing.T) {
	strPtr := func(s string) *string { return &s }

	t.Run("plex thumb path is rewritten to the proxy", func(t *testing.T) {
		got := resolveMediaArtworkURL(models.MediaProviderPlex, strPtr("/library/metadata/1/thumb/9"))
		want := "/api/auth/media/plex/artwork?path=%2Flibrary%2Fmetadata%2F1%2Fthumb%2F9"
		if got == nil || *got != want {
			t.Errorf("plex artwork: got %v, want %q", got, want)
		}
	})

	t.Run("spotify public url passes through", func(t *testing.T) {
		url := "https://i.scdn.co/image/abc"
		got := resolveMediaArtworkURL(models.MediaProviderSpotify, strPtr(url))
		if got == nil || *got != url {
			t.Errorf("spotify artwork should pass through: got %v", got)
		}
	})

	t.Run("nil and empty stay untouched", func(t *testing.T) {
		if got := resolveMediaArtworkURL(models.MediaProviderPlex, nil); got != nil {
			t.Errorf("nil artwork: got %v, want nil", got)
		}
		if got := resolveMediaArtworkURL(models.MediaProviderPlex, strPtr("")); got == nil || *got != "" {
			t.Errorf("empty artwork: got %v, want empty", got)
		}
	})

	t.Run("plex non-library value is not rewritten", func(t *testing.T) {
		got := resolveMediaArtworkURL(models.MediaProviderPlex, strPtr("http://evil/x"))
		if got == nil || *got != "http://evil/x" {
			t.Errorf("non-library plex value should pass through unchanged: got %v", got)
		}
	})
}
