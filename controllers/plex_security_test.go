package controllers

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

// TLS verification used to be skipped for every PMS connection, which exposed the user's
// decrypted X-Plex-Token to anyone on the path to a public server. It must now be skipped
// only for hosts that cannot present a publicly-verifiable certificate.
func TestPlexTLSVerificationSkipped(t *testing.T) {
	cases := []struct {
		host string
		want bool
	}{
		// Cannot be verified: no public CA issues for these.
		{"192.168.1.20", true},
		{"10.0.0.5", true},
		{"127.0.0.1", true},
		{"::1", true},
		{"localhost", true},
		{"plex.lan.local", true},
		{"192-168-1-20.abc123.plex.direct", true},
		{"PLEX.DIRECT", true},

		// Reached across the internet: verify like anything else.
		{"plex.example.com", false},
		{"media.mydomain.net", false},
		{"example.com", false},
		{"notplex.direct.example.com", false},
		{"", false},
	}

	for _, testCase := range cases {
		if got := plexTLSVerificationSkipped(testCase.host); got != testCase.want {
			t.Errorf("plexTLSVerificationSkipped(%q) = %v, want %v", testCase.host, got, testCase.want)
		}
	}
}

// A plex.tv PIN is redeemable by whoever polls it with this install's (shared) client
// identifier, and PIN ids are sequential — so a PIN must only be claimable by the user who
// created it.
func TestPlexPinOwnership(t *testing.T) {
	owner := uuid.New()
	stranger := uuid.New()
	pinID := int64(4242)

	t.Cleanup(func() { forgetPlexPin(pinID) })

	if plexPinBelongsTo(pinID, owner) {
		t.Fatal("an unknown PIN must not belong to anyone")
	}

	rememberPlexPin(pinID, owner)

	if !plexPinBelongsTo(pinID, owner) {
		t.Error("the creating user should be able to redeem their own PIN")
	}
	if plexPinBelongsTo(pinID, stranger) {
		t.Error("another user was able to claim someone else's in-flight PIN")
	}

	forgetPlexPin(pinID)
	if plexPinBelongsTo(pinID, owner) {
		t.Error("a redeemed PIN should no longer be claimable")
	}
}

func TestPlexPinOwnershipExpires(t *testing.T) {
	owner := uuid.New()
	pinID := int64(4343)

	t.Cleanup(func() { forgetPlexPin(pinID) })

	// Seed a record that is already older than the TTL.
	plexPinOwnersMutex.Lock()
	plexPinOwners[pinID] = plexPinOwner{userID: owner, createdAt: time.Now().Add(-plexPinTTL - time.Minute)}
	plexPinOwnersMutex.Unlock()

	if plexPinBelongsTo(pinID, owner) {
		t.Error("an expired PIN record should not be claimable")
	}
}
