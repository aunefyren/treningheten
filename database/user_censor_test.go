package database

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/aunefyren/treningheten/models"
)

// The censored getters return models.PublicUser, an allowlist type: a field that is not on
// it cannot be served cross-user at all, which is why these tests assert on the *shape* of
// what comes back rather than on individual fields being blanked. (They used to check for a
// literal "REDACTED"; that assertion is now unrepresentable, which is the point.)
//
// They also pin the explicit uncensored siblings, which the background jobs need.

func seedSensitiveUser(t *testing.T, email string, sundayAlert bool) models.User {
	t.Helper()

	return makeTestUser(t, email, func(u *models.User) {
		reset := "RESETCODE123"
		verification := "VERIFYCODE123"
		strava := "r:strava-refresh-token"
		u.ResetCode = &reset
		u.VerificationCode = &verification
		u.StravaCode = &strava
		u.SundayAlert = sundayAlert
	})
}

// publicUserJSONFields is every key a models.PublicUser may put on the wire. It is spelled
// out here, rather than derived from the struct, precisely so that adding a field to
// PublicUser fails this test until somebody confirms that peers may see it. That is the
// whole safety property: the previous blocklist censor silently published BirthDate,
// MaxHeartrate, RestingHeartrate and ObservedMaxHeartrate for exactly this reason.
var publicUserJSONFields = map[string]bool{
	"id":                 true,
	"created_at":         true,
	"first_name":         true,
	"last_name":          true,
	"admin":              true,
	"strava_id":          true,
	"strava_public":      true,
	"hevy_profile_url":   true,
	"hevy_public":        true,
	"share_statistics":   true,
	"wheel_color":        true,
	"wheel_border_color": true,
	"wheel_emoji":        true,
}

func assertCensored(t *testing.T, users []models.PublicUser, context string) {
	t.Helper()

	if len(users) == 0 {
		t.Fatalf("%s returned no users", context)
	}
	for _, user := range users {
		assertOnlyPublicFields(t, user, context)
	}
}

// assertOnlyPublicFields marshals the value and checks that every key is on the allowlist.
// Marshalling rather than reflecting over the struct is deliberate: what reaches another
// user is the JSON, so that is what is checked.
func assertOnlyPublicFields(t *testing.T, user models.PublicUser, context string) {
	t.Helper()

	encoded, err := json.Marshal(user)
	if err != nil {
		t.Fatalf("%s: failed to marshal public user: %v", context, err)
	}

	var fields map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &fields); err != nil {
		t.Fatalf("%s: failed to decode marshalled public user: %v", context, err)
	}

	for field := range fields {
		if !publicUserJSONFields[field] {
			t.Errorf("%s: serialized unexpected field %q — add it to publicUserJSONFields only if peers may see it", context, field)
		}
	}
	for field := range publicUserJSONFields {
		if _, ok := fields[field]; !ok {
			t.Errorf("%s: expected field %q is missing from the public user", context, field)
		}
	}
}

// A user row carrying every sensitive field there is must still censor down to the
// allowlist — including the health fields (birth date, heart rates) that the previous
// blocklist censor let through to every authenticated caller.
func TestCensorUserObjectKeepsOnlyPublicFields(t *testing.T) {
	newTestDB(t)

	birthDate := time.Date(1990, 5, 17, 0, 0, 0, 0, time.UTC)
	maxHR := 195
	restingHR := 48
	observedHR := 191
	hevyKey := "hevy-api-key"
	user := seedSensitiveUser(t, "everything@censor.test", true)
	user.BirthDate = &birthDate
	user.MaxHeartrate = &maxHR
	user.RestingHeartrate = &restingHR
	user.ObservedMaxHeartrate = &observedHR
	user.HevyAPIKey = &hevyKey

	publicUser := CensorUserObject(user)

	assertOnlyPublicFields(t, publicUser, "CensorUserObject")

	encoded, err := json.Marshal(publicUser)
	if err != nil {
		t.Fatalf("failed to marshal public user: %v", err)
	}
	// Belt and braces for the string-valued secrets: they must not appear anywhere in the
	// payload, whatever key they might have been given. The numeric health fields are not
	// scanned for this way — short numbers collide with uuids and timestamps — but their
	// absence is already covered by the field allowlist above.
	for _, secret := range []string{"hevy-api-key", "RESETCODE123", "VERIFYCODE123", "strava-refresh-token", "everything@censor.test"} {
		if strings.Contains(string(encoded), secret) {
			t.Errorf("public user payload leaked %q: %s", secret, encoded)
		}
	}
}

// A Strava id is only a profile link while its owner keeps the link public.
func TestCensorUserObjectHidesPrivateStravaID(t *testing.T) {
	newTestDB(t)

	stravaID := "123456"
	private := false
	public := true

	user := seedSensitiveUser(t, "strava@censor.test", false)
	user.StravaID = &stravaID
	user.StravaPublic = &private

	if got := CensorUserObject(user); got.StravaID != nil || got.StravaPublic != nil {
		t.Errorf("private Strava link exposed: id=%v public=%v", got.StravaID, got.StravaPublic)
	}

	user.StravaPublic = &public
	got := CensorUserObject(user)
	if got.StravaID == nil || *got.StravaID != stravaID {
		t.Errorf("public Strava id = %v, want %q", got.StravaID, stravaID)
	}
}

func TestGetUsersInformationCensorsEveryRow(t *testing.T) {
	newTestDB(t)

	seedSensitiveUser(t, "one@censor.test", false)
	seedSensitiveUser(t, "two@censor.test", false)

	users, err := GetUsersInformation()
	if err != nil {
		t.Fatalf("GetUsersInformation() error: %v", err)
	}
	if len(users) != 2 {
		t.Fatalf("GetUsersInformation() returned %d users, want 2", len(users))
	}
	assertCensored(t, users, "GetUsersInformation")
}

func TestGetAllUsersWithSundayAlertsEnabledCensorsEveryRow(t *testing.T) {
	newTestDB(t)

	seedSensitiveUser(t, "alert@censor.test", true)
	seedSensitiveUser(t, "quiet@censor.test", false)

	users, err := GetAllUsersWithSundayAlertsEnabled()
	if err != nil {
		t.Fatalf("GetAllUsersWithSundayAlertsEnabled() error: %v", err)
	}
	if len(users) != 1 {
		t.Fatalf("returned %d users, want only the one with alerts enabled", len(users))
	}
	assertCensored(t, users, "GetAllUsersWithSundayAlertsEnabled")
}

func TestUncensoredGettersKeepCredentials(t *testing.T) {
	newTestDB(t)

	seedSensitiveUser(t, "job@censor.test", true)

	// The Strava sync selects on StravaCode, so its getter must not censor.
	all, err := GetAllUsersUncensored()
	if err != nil {
		t.Fatalf("GetAllUsersUncensored() error: %v", err)
	}
	if len(all) != 1 {
		t.Fatalf("GetAllUsersUncensored() returned %d users, want 1", len(all))
	}
	if all[0].StravaCode == nil || *all[0].StravaCode != "r:strava-refresh-token" {
		t.Errorf("GetAllUsersUncensored() dropped StravaCode: %v", all[0].StravaCode)
	}

	// The Sunday reminder is an e-mail, so its getter must not censor the address.
	alerts, err := GetAllUsersWithSundayAlertsEnabledUncensored()
	if err != nil {
		t.Fatalf("GetAllUsersWithSundayAlertsEnabledUncensored() error: %v", err)
	}
	if len(alerts) != 1 {
		t.Fatalf("returned %d users, want 1", len(alerts))
	}
	if alerts[0].Email != "job@censor.test" {
		t.Errorf("Email = %q, want it preserved", alerts[0].Email)
	}
}
