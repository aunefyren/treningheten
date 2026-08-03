package auth

import (
	"testing"

	"github.com/aunefyren/treningheten/models"
)

// TestScopeChecks covers the read/write/admin decisions the auth middleware makes for every
// scope string it can see. The important property is whole-token matching: a scope is a
// space-delimited set, so a token that merely has "api" as a prefix or substring must not
// grant anything.
func TestScopeChecks(t *testing.T) {
	tests := []struct {
		name  string
		scope string
		read  bool
		write bool
		admin bool
	}{
		{"empty", "", false, false, false},
		{"legacy api implies read and write", models.ScopeAPI, true, true, false},
		{"read only", models.ScopeAPIRead, true, false, false},
		{"write implies read", models.ScopeAPIWrite, true, true, false},
		{"admin implies read and write", models.ScopeAdmin, true, true, true},
		{"read plus admin", models.ScopeAPIRead + " " + models.ScopeAdmin, true, true, true},
		{"user login scope", ScopeForUser(false), true, true, false},
		{"admin login scope", ScopeForUser(true), true, true, true},
		{"extra whitespace is ignored", "  api:read   admin  ", true, true, true},
		{"unknown scope grants nothing", "openid profile", false, false, false},

		// Whole-token matching: none of these may be mistaken for a real scope.
		{"prefix of api is not api", "ap", false, false, false},
		{"api with suffix is not api", "apix", false, false, false},
		{"api:readonly is not api:read", "api:readonly", false, false, false},
		{"admin as substring is not admin", "administrator", false, false, false},
		{"colon-joined scopes are not split", "api:read:write", false, false, false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := ScopeCanRead(test.scope); got != test.read {
				t.Errorf("ScopeCanRead(%q) = %v, want %v", test.scope, got, test.read)
			}
			if got := ScopeCanWrite(test.scope); got != test.write {
				t.Errorf("ScopeCanWrite(%q) = %v, want %v", test.scope, got, test.write)
			}
			if got := ScopeHasAdmin(test.scope); got != test.admin {
				t.Errorf("ScopeHasAdmin(%q) = %v, want %v", test.scope, got, test.admin)
			}
		})
	}
}

// TestScopeForUser covers that only an admin login is granted the admin scope.
func TestScopeForUser(t *testing.T) {
	if scope := ScopeForUser(false); ScopeHasAdmin(scope) {
		t.Errorf("ScopeForUser(false) = %q, which grants admin", scope)
	}
	if scope := ScopeForUser(true); !ScopeHasAdmin(scope) {
		t.Errorf("ScopeForUser(true) = %q, which does not grant admin", scope)
	}
	// Both must still be usable for ordinary API calls.
	for _, admin := range []bool{false, true} {
		if scope := ScopeForUser(admin); !ScopeCanRead(scope) || !ScopeCanWrite(scope) {
			t.Errorf("ScopeForUser(%v) = %q, want read+write", admin, scope)
		}
	}
}
