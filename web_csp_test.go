package main

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// cspVendoredFiles are third-party scripts in web/js that are not ours to rewrite. They
// only touch styles through the CSSOM, which the policy allows.
var cspVendoredFiles = map[string]bool{
	"Winwheel.js":     true,
	"TweenMax.min.js": true,
}

// cspViolations are the markup and code patterns the strict Content-Security-Policy
// (middlewares.ContentSecurityPolicy) blocks or that the refactor rules ban. Attribute
// patterns require the value's opening quote (or an escaped quote or template
// placeholder) straight after the "=", which is how markup is written in web/ and keeps
// JS such as `var one = ...` from matching.
var cspViolations = []struct {
	pattern *regexp.Regexp
	rule    string
}{
	{regexp.MustCompile(`(?i)[\s"'/]on[a-z]+=["'\\$]`), "inline event handler attribute (on...=); use data-action and on_action"},
	{regexp.MustCompile(`(?i)[\s"'/]style=["'\\$]`), "inline style attribute (style=); use a class, the hidden attribute or a data-* style"},
	{regexp.MustCompile(`(?i)javascript:`), "javascript: URL; use a button with data-action"},
	{regexp.MustCompile(`(?i)<style[\s>]`), "style element; put the CSS in web/css"},
	{regexp.MustCompile(`setAttribute\(\s*["']style["']`), "setAttribute('style', ...); set element.style.x through the CSSOM"},
	{regexp.MustCompile(`cssText`), "style.cssText; set element.style.x through the CSSOM"},
}

// scriptTag matches an opening <script> tag; one without a src attribute is inline.
var scriptTag = regexp.MustCompile(`(?i)<script\b[^>]*>`)

// isCommentLine reports whether a line is a whole-line JS or HTML comment, which may name
// the banned constructs (e.g. "formerly an inline <script>") without using them.
func isCommentLine(line string) bool {
	trimmed := strings.TrimSpace(line)
	for _, prefix := range []string{"//", "/*", "*", "<!--"} {
		if strings.HasPrefix(trimmed, prefix) {
			return true
		}
	}
	return false
}

// TestWebAssetsAreCSPClean keeps web/ compatible with the enforced strict CSP: no inline
// handlers, style attributes, style elements, inline scripts or javascript: URLs, in the
// page templates or in JS-built markup. The browser would silently block them.
func TestWebAssetsAreCSPClean(t *testing.T) {
	var files []string
	for _, pattern := range []string{"web/html/*.html", "web/js/*.js"} {
		matches, err := filepath.Glob(pattern)
		if err != nil {
			t.Fatal(err)
		}
		files = append(files, matches...)
	}
	if len(files) == 0 {
		t.Fatal("no web assets found; run the test from the repository root")
	}

	for _, file := range files {
		if cspVendoredFiles[filepath.Base(file)] {
			continue
		}
		content, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		for number, line := range strings.Split(string(content), "\n") {
			if isCommentLine(line) {
				continue
			}
			for _, violation := range cspViolations {
				if violation.pattern.MatchString(line) {
					t.Errorf("%s:%d: %s (strict CSP, see docs/security.md#strict-content-security-policy)\n\t%s",
						file, number+1, violation.rule, strings.TrimSpace(line))
				}
			}
			for _, tag := range scriptTag.FindAllString(line, -1) {
				if !strings.Contains(strings.ToLower(tag), "src=") {
					t.Errorf("%s:%d: inline <script> block; move the code into the page's JS file (strict CSP, see docs/security.md#strict-content-security-policy)\n\t%s",
						file, number+1, strings.TrimSpace(line))
				}
			}
		}
	}
}

// The guard must actually catch what it claims to; a pattern that silently stopped
// matching would let violations back in.
func TestCSPViolationPatterns(t *testing.T) {
	bad := []string{
		`<button onclick="save()">`,
		`html += '<img onerror=\'x\'>'`,
		"`<input oninput=\"${handler}\">`",
		`<div style="display:none">`,
		"`<span style=\"width: ${pct}%\"></span>`",
		`<a href="javascript:void(0)">`,
		`<style>.x { color: red; }</style>`,
		`element.setAttribute('style', 'border: 0')`,
		`element.style.cssText = "border: 0"`,
	}
	for _, line := range bad {
		matched := false
		for _, violation := range cspViolations {
			matched = matched || violation.pattern.MatchString(line)
		}
		if !matched {
			t.Errorf("no CSP pattern matches %q", line)
		}
	}

	good := []string{
		`<button data-action="save" data-id="${id}">`,
		`var one = firstGrapheme(value);`,
		`element.style.display = "none";`,
		`element.onclick = function() {};`,
		`<div class="x" hidden>`,
		`aria-controls="stats-panel-${tab.key}"`,
	}
	for _, line := range good {
		for _, violation := range cspViolations {
			if violation.pattern.MatchString(line) {
				t.Errorf("CSP pattern %q wrongly matches %q", violation.pattern, line)
			}
		}
	}

	if tags := scriptTag.FindAllString(`<script>alert(1)</script><script src="/js/a.js"></script>`, -1); len(tags) != 2 {
		t.Errorf("scriptTag found %d tags, want 2", len(tags))
	}
}
