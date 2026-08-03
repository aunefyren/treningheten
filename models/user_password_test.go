package models

import (
	"strings"
	"testing"
)

// TestHashPasswordAndCheck covers the password hashing used at registration and the check
// used at login: the plaintext must not survive on the struct, and only the correct
// password may verify. Cases are kept few on purpose — bcrypt runs at cost 14, so every
// hash and every comparison costs about a second.
func TestHashPasswordAndCheck(t *testing.T) {
	const password = "correct horse battery staple"

	user := User{}
	if err := user.HashPassword(password); err != nil {
		t.Fatalf("HashPassword returned error: %v", err)
	}

	if user.Password == password {
		t.Fatalf("the plaintext password was stored on the user")
	}
	if !strings.HasPrefix(user.Password, "$2") {
		t.Errorf("stored password %q is not a bcrypt hash", user.Password)
	}

	if err := user.CheckPassword(password); err != nil {
		t.Errorf("CheckPassword rejected the correct password: %v", err)
	}

	wrong := []string{
		"",
		"Correct horse battery staple", // case matters
		user.Password,                  // presenting the hash itself must not work
	}
	for _, attempt := range wrong {
		if err := user.CheckPassword(attempt); err == nil {
			t.Errorf("CheckPassword accepted %q", attempt)
		}
	}
}

// TestHashPasswordIsSalted covers that two users with the same password get different
// hashes, so a leaked table doesn't reveal which accounts share a password.
func TestHashPasswordIsSalted(t *testing.T) {
	if testing.Short() {
		t.Skip("two bcrypt hashes at cost 14 take several seconds")
	}

	const password = "same-password-for-both"

	first := User{}
	second := User{}
	if err := first.HashPassword(password); err != nil {
		t.Fatalf("HashPassword returned error: %v", err)
	}
	if err := second.HashPassword(password); err != nil {
		t.Fatalf("HashPassword returned error: %v", err)
	}

	if first.Password == second.Password {
		t.Errorf("identical passwords produced identical hashes — the hash is not salted")
	}
	// Both must still verify against their own hash.
	if err := first.CheckPassword(password); err != nil {
		t.Errorf("first hash failed to verify: %v", err)
	}
	if err := second.CheckPassword(password); err != nil {
		t.Errorf("second hash failed to verify: %v", err)
	}
}

// TestCheckPasswordAgainstEmptyHash covers a user row with no password set (an unfinished
// registration): nothing may authenticate against it, least of all an empty password.
func TestCheckPasswordAgainstEmptyHash(t *testing.T) {
	user := User{}
	for _, attempt := range []string{"", "anything"} {
		if err := user.CheckPassword(attempt); err == nil {
			t.Errorf("CheckPassword(%q) succeeded against an empty hash", attempt)
		}
	}
}
