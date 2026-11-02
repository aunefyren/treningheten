package middlewares

import (
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
)

// ContentSecurityPolicy is the strict policy for the whole app. It is sent as
// Content-Security-Policy-Report-Only for now and is meant to be enforced as-is later, so
// keep it a single constant. No report-uri: violations show up in the browser console.
//
// External origins, each one really used by web/:
//   - fonts.googleapis.com (style-src): the Google Fonts stylesheets linked in every page.
//   - fonts.gstatic.com (font-src): the font files those stylesheets load.
//   - cdnjs.cloudflare.com and unpkg.com (script-src, style-src): Chart.js and Leaflet
//     (scripts and leaflet.css) loaded from CDNs on seasons, user, statistics and exercise.
//   - *.tile.openstreetmap.org (img-src): Leaflet map tiles on statistics and exercise.
//   - i.scdn.co (img-src): Spotify cover art, which the API passes through unchanged and
//     the soundtrack card on statistics uses as a CSS background image.
const ContentSecurityPolicy = "default-src 'self'; " +
	"script-src 'self' https://cdnjs.cloudflare.com https://unpkg.com; " +
	"style-src 'self' https://fonts.googleapis.com https://cdnjs.cloudflare.com https://unpkg.com; " +
	"font-src 'self' https://fonts.gstatic.com; " +
	"img-src 'self' data: blob: https://*.tile.openstreetmap.org https://i.scdn.co; " +
	"connect-src 'self'; " +
	"object-src 'none'; " +
	"base-uri 'self'; " +
	"form-action 'self'; " +
	"frame-ancestors 'none'"

// SecurityHeaders sets baseline hardening headers on every response: no MIME sniffing,
// no framing, a conservative referrer policy and, when the instance is served over
// HTTPS, HSTS. The enforced Content-Security-Policy carries only frame-ancestors; the
// strict policy (ContentSecurityPolicy) is sent alongside it as Report-Only until every
// page is compliant (see docs/security.md).
//
// HSTS is sent only when externalURL has an https scheme, so a plain-HTTP instance
// never pins browsers to HTTPS. It deliberately omits includeSubDomains and preload:
// self-hosted instances often share a parent domain with other services.
func SecurityHeaders(externalURL string) gin.HandlerFunc {
	https := false
	if parsed, err := url.Parse(strings.TrimSpace(externalURL)); err == nil {
		https = strings.EqualFold(parsed.Scheme, "https")
	}

	return func(context *gin.Context) {
		header := context.Writer.Header()
		header.Set("X-Content-Type-Options", "nosniff")
		header.Set("X-Frame-Options", "DENY")
		header.Set("Content-Security-Policy", "frame-ancestors 'none'")
		header.Set("Content-Security-Policy-Report-Only", ContentSecurityPolicy)
		header.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		if https {
			header.Set("Strict-Transport-Security", "max-age=31536000")
		}
		context.Next()
	}
}
