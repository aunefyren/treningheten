package controllers

import (
	"crypto/tls"
	"errors"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strings"
	"syscall"
	"time"

	"github.com/aunefyren/treningheten/files"
)

// Outbound safety for the media integrations.
//
// Plex and Audiobookshelf are the only integrations whose destination is chosen by the
// *user* (a server URL they type in) rather than fixed in code, which makes them the app's
// SSRF surface: without a guard, any authenticated user could point the server at an
// internal address and — through the artwork proxy, which streams the response back — read
// it. See docs/media.md, "Outbound request safety".
//
// The check runs in the dialer's Control hook rather than as a pre-flight string test, so
// it sees the IP actually being connected to. That closes DNS rebinding (a name that
// resolves to a public address when validated and a private one when fetched) and covers
// redirects for free, since each redirect hop dials again.

var errMediaTargetBlocked = errors.New("destination address is not allowed")

// mediaAllowsPrivateTargets reports whether private/loopback destinations may be reached.
// It defaults to true: a self-hosted Plex or Audiobookshelf on the LAN — or on 127.0.0.1
// next to Treningheten itself — is the normal deployment, and defaulting to false would
// break it. An instance with untrusted users and nothing worth reaching on its network can
// set media.allow_private_targets to false in config.json.
func mediaAllowsPrivateTargets() bool {
	if files.ConfigFile.Media.AllowPrivateTargets == nil {
		return true
	}
	return *files.ConfigFile.Media.AllowPrivateTargets
}

// mediaTargetIPAllowed applies the destination policy to a resolved address.
//
// Link-local is refused unconditionally, whatever the config says: 169.254.169.254 is the
// cloud metadata endpoint, and no media server legitimately lives on a link-local address.
// Loopback and private ranges are legitimate destinations here, so they follow the config
// flag. Carrier-grade NAT (100.64.0.0/10) counts as private because that is where a
// Tailscale-reachable server sits.
func mediaTargetIPAllowed(addr netip.Addr) error {
	addr = addr.Unmap()
	if !addr.IsValid() {
		return errMediaTargetBlocked
	}

	switch {
	case addr.IsUnspecified(),
		addr.IsLinkLocalUnicast(),
		addr.IsLinkLocalMulticast(),
		addr.IsInterfaceLocalMulticast(),
		addr.IsMulticast():
		return errMediaTargetBlocked
	}

	if addr.IsLoopback() || addr.IsPrivate() || isCarrierGradeNAT(addr) {
		if !mediaAllowsPrivateTargets() {
			return errMediaTargetBlocked
		}
	}

	return nil
}

// carrierGradeNATPrefix is 100.64.0.0/10, which netip's IsPrivate does not cover but which
// is where Tailscale and similar overlays put a reachable server.
var carrierGradeNATPrefix = netip.MustParsePrefix("100.64.0.0/10")

// isCarrierGradeNAT reports whether the address sits in that range.
func isCarrierGradeNAT(addr netip.Addr) bool {
	if !addr.Is4() {
		return false
	}
	return carrierGradeNATPrefix.Contains(addr)
}

// mediaDialControl is the net.Dialer Control hook: it runs after DNS resolution with the
// concrete address about to be connected to.
func mediaDialControl(network string, address string, _ syscall.RawConn) error {
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return errMediaTargetBlocked
	}
	addr, err := netip.ParseAddr(host)
	if err != nil {
		// Control is called with a resolved literal; anything else is unexpected.
		return errMediaTargetBlocked
	}
	return mediaTargetIPAllowed(addr)
}

// mediaHTTPClient builds an HTTP client whose connections are subject to the destination
// policy above. tlsConfig may be nil for default verification.
func mediaHTTPClient(timeout time.Duration, tlsConfig *tls.Config) *http.Client {
	dialer := &net.Dialer{
		Timeout:   10 * time.Second,
		KeepAlive: 30 * time.Second,
		Control:   mediaDialControl,
	}

	return &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			DialContext:           dialer.DialContext,
			TLSClientConfig:       tlsConfig,
			TLSHandshakeTimeout:   10 * time.Second,
			ResponseHeaderTimeout: timeout,
			// Kept so proxied deployments keep working (this is what these requests
			// did before). Note the caveat in docs/media.md: with a proxy configured
			// the dial targets the proxy, so filtering happens there instead.
			Proxy: http.ProxyFromEnvironment,
		},
	}
}

// validateMediaServerURL parses and vets a user-supplied provider server URL, returning the
// trimmed URL ready to store. It is a pre-flight courtesy so the user gets a clear message
// instead of a generic connection failure — the dialer hook is what actually enforces the
// policy, so a name that resolves differently later is still blocked at connect time.
func validateMediaServerURL(rawURL string) (string, error) {
	trimmed := strings.TrimRight(strings.TrimSpace(rawURL), "/")

	parsed, err := url.Parse(trimmed)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return "", errors.New("Enter a full server URL, including http:// or https://.")
	}

	hostname := parsed.Hostname()
	if hostname == "" {
		return "", errors.New("Enter a full server URL, including http:// or https://.")
	}

	// A literal IP can be checked outright. A name is resolved best-effort: if it doesn't
	// resolve we let it through and leave the verdict to the dialer, so a server that is
	// briefly unresolvable isn't rejected as unsafe.
	if addr, addrErr := netip.ParseAddr(hostname); addrErr == nil {
		if mediaTargetIPAllowed(addr) != nil {
			return "", errors.New("That address is not allowed as a server URL.")
		}
		return trimmed, nil
	}

	resolved, resolveErr := net.LookupHost(hostname)
	if resolveErr != nil {
		return trimmed, nil
	}
	for _, candidate := range resolved {
		addr, addrErr := netip.ParseAddr(candidate)
		if addrErr != nil {
			continue
		}
		if mediaTargetIPAllowed(addr) != nil {
			return "", errors.New("That address is not allowed as a server URL.")
		}
	}

	return trimmed, nil
}
