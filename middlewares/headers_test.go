package middlewares

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func headersFor(t *testing.T, externalURL string) http.Header {
	t.Helper()
	router := gin.New()
	router.Use(SecurityHeaders(externalURL))
	router.GET("/x", func(context *gin.Context) { context.Status(http.StatusOK) })
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest("GET", "/x", nil))
	return recorder.Header()
}

func TestSecurityHeadersAreSet(t *testing.T) {
	header := headersFor(t, "http://example.com")
	want := map[string]string{
		"X-Content-Type-Options":  "nosniff",
		"X-Frame-Options":         "DENY",
		"Content-Security-Policy": "frame-ancestors 'none'",
		"Referrer-Policy":         "strict-origin-when-cross-origin",
	}
	for name, value := range want {
		if got := header.Get(name); got != value {
			t.Errorf("%s = %q, want %q", name, got, value)
		}
	}
}

func TestSecurityHeadersHSTSOnlyForHTTPS(t *testing.T) {
	for _, external := range []string{"https://example.com", "HTTPS://example.com/app"} {
		if got := headersFor(t, external).Get("Strict-Transport-Security"); got != "max-age=31536000" {
			t.Errorf("%q: HSTS = %q, want max-age=31536000", external, got)
		}
	}
	for _, external := range []string{"", "http://example.com", "example.com", "://bad"} {
		if got := headersFor(t, external).Get("Strict-Transport-Security"); got != "" {
			t.Errorf("%q: HSTS = %q, want none", external, got)
		}
	}
}
