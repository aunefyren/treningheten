package auth

import (
	"strings"

	"github.com/aunefyren/treningheten/models"
)

// scopeSet splits a space-delimited scope string into a lookup set. Matching is
// done on whole tokens so that "api:read" is never mistaken for "api".
func scopeSet(scope string) map[string]bool {
	set := map[string]bool{}
	for _, s := range strings.Fields(scope) {
		set[s] = true
	}
	return set
}

// ScopeHasAdmin reports whether the scope grants admin access.
func ScopeHasAdmin(scope string) bool {
	return scopeSet(scope)[models.ScopeAdmin]
}

// ScopeCanWrite reports whether the scope permits write (non-GET) operations.
// The legacy "api" scope and "admin" both imply write.
func ScopeCanWrite(scope string) bool {
	set := scopeSet(scope)
	return set[models.ScopeAPI] || set[models.ScopeAPIWrite] || set[models.ScopeAdmin]
}

// ScopeCanRead reports whether the scope permits any API access at all.
func ScopeCanRead(scope string) bool {
	set := scopeSet(scope)
	return set[models.ScopeAPI] || set[models.ScopeAPIRead] || set[models.ScopeAPIWrite] || set[models.ScopeAdmin]
}
