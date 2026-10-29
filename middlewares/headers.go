package middlewares

import (
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
)

// SecurityHeaders sets baseline hardening headers on every response: no MIME sniffing,
// no framing, a conservative referrer policy and, when the instance is served over
// HTTPS, HSTS. The Content-Security-Policy carries only frame-ancestors for now; a
// script-restricting policy is tracked separately.
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
		header.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		if https {
			header.Set("Strict-Transport-Security", "max-age=31536000")
		}
		context.Next()
	}
}
