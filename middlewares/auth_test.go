package middlewares

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/aunefyren/treningheten/files"
	"github.com/aunefyren/treningheten/logger"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

func TestMain(m *testing.M) {
	if logger.Log == nil {
		l := logrus.New()
		l.SetOutput(io.Discard)
		logger.Log = l
	}
	gin.SetMode(gin.TestMode)
	m.Run()
}

// TestIsReadMethod covers the read/write split that decides whether an `api:read` token may
// proceed. Anything that is not a plain GET/HEAD must count as a write, or a read-only token
// could mutate data.
func TestIsReadMethod(t *testing.T) {
	for _, method := range []string{http.MethodGet, http.MethodHead} {
		if !isReadMethod(method) {
			t.Errorf("isReadMethod(%q) = false, want true", method)
		}
	}

	writes := []string{
		http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete,
		http.MethodOptions, http.MethodConnect, http.MethodTrace,
		"get", "Get", "", "GETX", " GET",
	}
	for _, method := range writes {
		if isReadMethod(method) {
			t.Errorf("isReadMethod(%q) = true, want false", method)
		}
	}
}

// newTestContext returns a gin context writing into a recorder, for testing the challenge
// helpers without standing up a router.
func newTestContext() (*gin.Context, *httptest.ResponseRecorder) {
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodGet, "/api/auth/users", nil)
	return context, recorder
}

// TestBearerChallenge covers the RFC 6750 challenge emitted on a rejected request: the
// status and body, the error code and description, and that the request is aborted so no
// handler runs afterwards.
func TestBearerChallenge(t *testing.T) {
	prevURL := files.ConfigFile.TreninghetenExternalURL
	files.ConfigFile.TreninghetenExternalURL = ""
	t.Cleanup(func() { files.ConfigFile.TreninghetenExternalURL = prevURL })

	context, recorder := newTestContext()
	BearerChallenge(context, http.StatusUnauthorized, "invalid_token", "token has expired")

	if recorder.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", recorder.Code, http.StatusUnauthorized)
	}
	if !context.IsAborted() {
		t.Errorf("the request was not aborted")
	}

	header := recorder.Header().Get("WWW-Authenticate")
	if !strings.HasPrefix(header, `Bearer realm="Treningheten"`) {
		t.Errorf("challenge %q does not start with the Bearer realm", header)
	}
	if !strings.Contains(header, `error="invalid_token"`) {
		t.Errorf("challenge %q is missing the error code", header)
	}
	if !strings.Contains(header, `error_description="token has expired"`) {
		t.Errorf("challenge %q is missing the description", header)
	}
	if strings.Contains(header, "resource_metadata") {
		t.Errorf("challenge %q advertises metadata with no external URL configured", header)
	}
	if !strings.Contains(recorder.Body.String(), "token has expired") {
		t.Errorf("body %q does not carry the description", recorder.Body.String())
	}
}

// TestBearerChallengeWithoutErrorCode covers the bare challenge (no error code), which is
// what an unauthenticated request receives — there is no token to call invalid yet.
func TestBearerChallengeWithoutErrorCode(t *testing.T) {
	prevURL := files.ConfigFile.TreninghetenExternalURL
	files.ConfigFile.TreninghetenExternalURL = ""
	t.Cleanup(func() { files.ConfigFile.TreninghetenExternalURL = prevURL })

	context, recorder := newTestContext()
	BearerChallenge(context, http.StatusUnauthorized, "", "no token")

	header := recorder.Header().Get("WWW-Authenticate")
	if strings.Contains(header, "error=") {
		t.Errorf("challenge %q carries an error code that was not asked for", header)
	}
	if header != `Bearer realm="Treningheten"` {
		t.Errorf("challenge = %q, want the bare realm", header)
	}
}

// TestBearerChallengeAdvertisesResourceMetadata covers the discovery pointer MCP clients
// follow to find the OAuth flow: it appears only when an external URL is configured, and the
// URL's trailing slash is trimmed so the path isn't doubled.
func TestBearerChallengeAdvertisesResourceMetadata(t *testing.T) {
	prevURL := files.ConfigFile.TreninghetenExternalURL
	t.Cleanup(func() { files.ConfigFile.TreninghetenExternalURL = prevURL })

	for _, external := range []string{"https://treningheten.example", "https://treningheten.example/"} {
		files.ConfigFile.TreninghetenExternalURL = external

		context, recorder := newTestContext()
		BearerChallenge(context, http.StatusUnauthorized, "invalid_token", "nope")

		header := recorder.Header().Get("WWW-Authenticate")
		want := `resource_metadata="https://treningheten.example/.well-known/oauth-protected-resource"`
		if !strings.Contains(header, want) {
			t.Errorf("external URL %q produced challenge %q, want it to contain %s", external, header, want)
		}
	}
}

// TestAuthRejectsMissingHeader covers the first gate in the middleware: a request with no
// Authorization header is refused with a challenge, and the wrapped handler never runs. It
// needs no database, because the check happens before any lookup.
func TestAuthRejectsMissingHeader(t *testing.T) {
	prevURL := files.ConfigFile.TreninghetenExternalURL
	files.ConfigFile.TreninghetenExternalURL = ""
	t.Cleanup(func() { files.ConfigFile.TreninghetenExternalURL = prevURL })

	handlerRan := false
	router := gin.New()
	router.GET("/protected", Auth(false), func(context *gin.Context) {
		handlerRan = true
		context.Status(http.StatusOK)
	})

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/protected", nil))

	if recorder.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", recorder.Code, http.StatusUnauthorized)
	}
	if handlerRan {
		t.Errorf("the protected handler ran without an Authorization header")
	}
	if recorder.Header().Get("WWW-Authenticate") == "" {
		t.Errorf("no WWW-Authenticate challenge was emitted")
	}
}
