package main

import (
	"flag"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/aunefyren/treningheten/logger"
	"github.com/aunefyren/treningheten/models"

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
	os.Exit(m.Run())
}

func testConfig() models.ConfigStruct {
	return models.ConfigStruct{
		TreninghetenName:        "Treningheten Test",
		TreninghetenDescription: "desc",
		TreninghetenVersion:     "v9.9.9-test",
		VAPIDPublicKey:          "vapid-public-key-for-test",
		Timezone:                "Europe/Oslo",
	}
}

func get(router *gin.Engine, path string) *httptest.ResponseRecorder {
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest("GET", path, nil))
	return recorder
}

// Every file under web/html, web/js, web/json and web/txt must be reachable at its clean
// URL and render through its template without error — a broken {{ }} in any page or
// script would otherwise only show up in a browser.
func TestInitRouterServesEveryWebFile(t *testing.T) {
	router := initRouter(testConfig())

	special := map[string]string{
		"frontpage.html":    "/",
		"user.html":         "/users/some-user",
		"exercise.html":     "/exercises/some-exercise",
		"manifest.json":     "/manifest.json",
		"service-worker.js": "/service-worker.js",
		"robots.txt":        "/robots.txt",
	}

	checked := 0
	for _, directory := range []struct{ dir, prefix string }{
		{"web/html", ""}, {"web/js", "/js"}, {"web/json", "/json"}, {"web/txt", "/txt"},
	} {
		entries, err := os.ReadDir(directory.dir)
		if err != nil {
			t.Fatal(err)
		}
		for _, entry := range entries {
			name := entry.Name()
			path := directory.prefix + "/" + name
			if directory.dir == "web/html" {
				path = "/" + strings.TrimSuffix(name, ".html")
			}
			if override, found := special[name]; found {
				path = override
			}

			response := get(router, path)
			if response.Code != http.StatusOK {
				t.Errorf("GET %s (%s): status = %d, body: %.200s", path, name, response.Code, response.Body.String())
				continue
			}
			if strings.Contains(response.Body.String(), "{{ .") {
				t.Errorf("GET %s: unrendered template placeholder left in output", path)
			}
			checked++
		}
	}
	if checked < 30 {
		t.Errorf("only %d web files checked; did the directory layout change?", checked)
	}
}

func TestInitRouterInjectsConfigIntoTemplates(t *testing.T) {
	router := initRouter(testConfig())

	frontpage := get(router, "/js/frontpage.js")
	if !strings.Contains(frontpage.Body.String(), "vapid-public-key-for-test") {
		t.Error("frontpage.js does not carry the VAPID public key from config")
	}
	if contentType := frontpage.Header().Get("Content-Type"); !strings.HasPrefix(contentType, "application/javascript") {
		t.Errorf("JS content type = %q", contentType)
	}
	if contentType := get(router, "/manifest.json").Header().Get("Content-Type"); !strings.HasPrefix(contentType, "application/json") {
		t.Errorf("manifest content type = %q", contentType)
	}
	if contentType := get(router, "/robots.txt").Header().Get("Content-Type"); !strings.HasPrefix(contentType, "text/plain") {
		t.Errorf("robots content type = %q", contentType)
	}
	// Static assets are served as-is.
	if code := get(router, "/css/does-not-exist.css").Code; code != http.StatusNotFound {
		t.Errorf("missing stylesheet: status = %d, want 404", code)
	}
}

func TestInitRouterProtectsTheAPI(t *testing.T) {
	router := initRouter(testConfig())

	for _, path := range []string{"/api/auth/seasons", "/api/auth/users", "/api/admin/stats"} {
		response := get(router, path)
		if response.Code != http.StatusUnauthorized {
			t.Errorf("GET %s without a token: status = %d, want 401", path, response.Code)
		}
		if !strings.HasPrefix(response.Header().Get("WWW-Authenticate"), "Bearer") {
			t.Errorf("GET %s: no Bearer challenge", path)
		}
	}
	for _, path := range []string{"/.well-known/oauth-authorization-server", "/.well-known/oauth-protected-resource"} {
		if code := get(router, path).Code; code != http.StatusOK {
			t.Errorf("GET %s: status = %d, want 200", path, code)
		}
	}
}

func TestRenderTemplateHandlersReportTemplateErrors(t *testing.T) {
	templates := MustLoadTemplates("./web/txt/*.txt")
	for name, handler := range map[string]gin.HandlerFunc{
		"js":   RenderJSTemplate(templates, "missing.js", nil),
		"json": RenderJSONTemplate(templates, "missing.json", nil),
		"txt":  RenderTextTemplate(templates, "missing.txt", nil),
	} {
		recorder := httptest.NewRecorder()
		context, _ := gin.CreateTestContext(recorder)
		handler(context)
		if recorder.Code != http.StatusInternalServerError {
			t.Errorf("%s: status = %d, want 500 for an unknown template", name, recorder.Code)
		}
	}
	// A glob that matches nothing logs and returns nil rather than panicking.
	if MustLoadTemplates("./web/none/*.nothing") != nil {
		t.Error("expected nil templates for an empty glob")
	}
}

// parseFlags reads the process command line; give it a clean flag set and argv.
func withArgs(t *testing.T, args ...string) {
	t.Helper()
	previousArgs, previousFlags := os.Args, flag.CommandLine
	os.Args = append([]string{"treningheten"}, args...)
	flag.CommandLine = flag.NewFlagSet("treningheten", flag.ContinueOnError)
	t.Cleanup(func() { os.Args, flag.CommandLine = previousArgs, previousFlags })
}

func TestParseFlagsOverridesConfig(t *testing.T) {
	withArgs(t,
		"--port", "9090", "--externalurl", "https://trening.example", "--timezone", "UTC",
		"--dbip", "db", "--dbport", "3307", "--dbname", "tr", "--dbusername", "u", "--dbpassword", "p",
		"--smtphost", "mail", "--smtpport", "25", "--smtpusername", "su", "--smtppassword", "sp", "--smtpfrom", "a@b",
		"--generateinvite", "TRUE",
	)

	config, generateInvite, err := parseFlags(testConfig())
	if err != nil {
		t.Fatal(err)
	}
	if !generateInvite {
		t.Error("generateinvite TRUE was not honoured (case-insensitive)")
	}
	if config.TreninghetenPort != 9090 || config.TreninghetenExternalURL != "https://trening.example" || config.Timezone != "UTC" {
		t.Errorf("server flags not applied: %+v", config)
	}
	if config.DBIP != "db" || config.DBPort != 3307 || config.DBName != "tr" || config.DBUsername != "u" || config.DBPassword != "p" {
		t.Errorf("database flags not applied: %+v", config)
	}
	if !config.SMTPEnabled || config.SMTPHost != "mail" || config.SMTPPort != 25 || config.SMTPFrom != "a@b" {
		t.Errorf("SMTP flags not applied: %+v", config)
	}
}

func TestParseFlagsDefaults(t *testing.T) {
	withArgs(t, "--disablesmtp", "true")

	config := testConfig()
	config.TreninghetenPort = 0
	config, generateInvite, err := parseFlags(config)
	if err != nil {
		t.Fatal(err)
	}
	if generateInvite {
		t.Error("generateinvite defaulted to true")
	}
	if config.SMTPEnabled {
		t.Error("--disablesmtp true left SMTP enabled")
	}
	if config.TreninghetenPort != 8080 {
		t.Errorf("port = %d, want the 8080 fallback for 0", config.TreninghetenPort)
	}
}

// Several handlers (the admin ones, the user list, news) never read the caller and rely
// entirely on the group middleware, so every /api/auth and /api/admin route registered in
// initRouter must refuse an anonymous request before any handler runs.
func TestEveryProtectedRouteRejectsAnonymousCallers(t *testing.T) {
	router := initRouter(testConfig())

	checked := 0
	for _, route := range router.Routes() {
		if !strings.HasPrefix(route.Path, "/api/auth") && !strings.HasPrefix(route.Path, "/api/admin") {
			continue
		}
		path := route.Path
		for _, segment := range strings.Split(route.Path, "/") {
			if strings.HasPrefix(segment, ":") {
				path = strings.Replace(path, segment, "00000000-0000-0000-0000-000000000001", 1)
			}
		}
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, httptest.NewRequest(route.Method, path, strings.NewReader("{}")))
		if recorder.Code != http.StatusUnauthorized {
			t.Errorf("%s %s without a token: status = %d, want 401", route.Method, route.Path, recorder.Code)
		}
		checked++
	}
	if checked < 100 {
		t.Errorf("only %d protected routes checked; did the route table change?", checked)
	}
}
