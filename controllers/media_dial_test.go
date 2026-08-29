package controllers

import (
	"net/http"
	"net/http/httptest"
	"net/netip"
	"testing"
	"time"

	"github.com/aunefyren/treningheten/files"
	"github.com/aunefyren/treningheten/models"
)

// withMediaPrivateTargets sets the destination policy flag for the duration of a test.
func withMediaPrivateTargets(t *testing.T, allow bool) {
	t.Helper()

	previous := files.ConfigFile.Media
	files.ConfigFile.Media = models.MediaSettings{AllowPrivateTargets: &allow}
	t.Cleanup(func() { files.ConfigFile.Media = previous })
}

func TestMediaTargetIPPolicy(t *testing.T) {
	cases := []struct {
		name         string
		address      string
		allowPrivate bool
		wantAllowed  bool
	}{
		// Cloud metadata and the rest of link-local are refused whatever the config says —
		// no media server lives there, and it is the highest-value SSRF target.
		{"cloud metadata, private allowed", "169.254.169.254", true, false},
		{"cloud metadata, private blocked", "169.254.169.254", false, false},
		{"ipv6 link-local", "fe80::1", true, false},
		{"unspecified", "0.0.0.0", true, false},
		{"multicast", "224.0.0.1", true, false},

		// Loopback and private ranges are legitimate self-hosted destinations, so they
		// follow the flag.
		{"loopback allowed by default", "127.0.0.1", true, true},
		{"loopback blocked when locked down", "127.0.0.1", false, false},
		{"lan allowed by default", "192.168.1.20", true, true},
		{"lan blocked when locked down", "192.168.1.20", false, false},
		{"rfc1918 ten-net", "10.1.2.3", true, true},
		{"ipv6 ula", "fd00::1", true, true},
		{"ipv6 ula blocked when locked down", "fd00::1", false, false},
		// Tailscale and friends sit in carrier-grade NAT space, which IsPrivate misses.
		{"cgnat treated as private", "100.64.1.1", false, false},
		{"cgnat allowed by default", "100.64.1.1", true, true},

		// Public addresses are always fine.
		{"public v4", "93.184.216.34", false, true},
		{"public v6", "2606:2800:220:1:248:1893:25c8:1946", false, true},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			withMediaPrivateTargets(t, testCase.allowPrivate)

			addr, err := netip.ParseAddr(testCase.address)
			if err != nil {
				t.Fatalf("bad test address %q: %v", testCase.address, err)
			}

			gotAllowed := mediaTargetIPAllowed(addr) == nil
			if gotAllowed != testCase.wantAllowed {
				t.Errorf("mediaTargetIPAllowed(%s) allowed = %v, want %v", testCase.address, gotAllowed, testCase.wantAllowed)
			}
		})
	}
}

// An IPv4-mapped IPv6 literal must not smuggle a blocked address past the policy.
func TestMediaTargetIPUnmapsV4MappedAddresses(t *testing.T) {
	withMediaPrivateTargets(t, true)

	addr, err := netip.ParseAddr("::ffff:169.254.169.254")
	if err != nil {
		t.Fatalf("bad address: %v", err)
	}
	if mediaTargetIPAllowed(addr) == nil {
		t.Error("an IPv4-mapped link-local address was allowed; it must be unmapped first")
	}
}

func TestMediaAllowsPrivateTargetsDefaultsOn(t *testing.T) {
	previous := files.ConfigFile.Media
	files.ConfigFile.Media = models.MediaSettings{AllowPrivateTargets: nil}
	t.Cleanup(func() { files.ConfigFile.Media = previous })

	if !mediaAllowsPrivateTargets() {
		t.Error("a config missing the field must default to allowing private targets, or LAN servers break on upgrade")
	}
}

func TestValidateMediaServerURL(t *testing.T) {
	withMediaPrivateTargets(t, true)

	if _, err := validateMediaServerURL("not-a-url"); err == nil {
		t.Error("expected a scheme-less string to be rejected")
	}
	if _, err := validateMediaServerURL("ftp://example.com"); err == nil {
		t.Error("expected a non-http scheme to be rejected")
	}
	if _, err := validateMediaServerURL("http://169.254.169.254/"); err == nil {
		t.Error("expected the metadata address to be rejected even with private targets allowed")
	}

	got, err := validateMediaServerURL("  https://plex.example.com:32400/  ")
	if err != nil {
		t.Fatalf("unexpected error for a normal URL: %v", err)
	}
	if got != "https://plex.example.com:32400" {
		t.Errorf("validateMediaServerURL trimmed to %q, want %q", got, "https://plex.example.com:32400")
	}

	// A LAN address is fine by default — this is the common self-hosted setup.
	if _, err := validateMediaServerURL("http://192.168.1.20:32400"); err != nil {
		t.Errorf("a LAN server URL should be accepted by default, got: %v", err)
	}

	withMediaPrivateTargets(t, false)
	if _, err := validateMediaServerURL("http://192.168.1.20:32400"); err == nil {
		t.Error("a LAN server URL should be rejected when private targets are turned off")
	}
}

// The policy is enforced in the dialer's Control hook, not as a pre-flight string check, so
// it also covers redirects and DNS rebinding. This drives a real client at a real (loopback)
// server to prove the hook is actually wired into the transport.
func TestMediaHTTPClientEnforcesPolicyAtDialTime(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	withMediaPrivateTargets(t, true)
	response, err := mediaHTTPClient(5*time.Second, nil).Get(server.URL)
	if err != nil {
		t.Fatalf("loopback should be reachable when private targets are allowed: %v", err)
	}
	response.Body.Close()

	withMediaPrivateTargets(t, false)
	if _, err := mediaHTTPClient(5*time.Second, nil).Get(server.URL); err == nil {
		t.Error("the request reached a loopback server with private targets turned off; the dial guard is not wired in")
	}
}
