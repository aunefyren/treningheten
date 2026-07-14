package database

import (
	"testing"
	"time"

	"github.com/aunefyren/treningheten/models"

	"github.com/google/uuid"
)

func TestUserVerificationLifecycle(t *testing.T) {
	newTestDB(t)

	user := makeTestUser(t, "verify@example.com", func(u *models.User) { u.Verified = false })

	// No verification code yet.
	has, err := VerifyUserHasVerificationCode(user.ID)
	if err != nil {
		t.Fatalf("VerifyUserHasVerificationCode returned error: %v", err)
	}
	if has {
		t.Errorf("expected no verification code initially")
	}

	code, err := GenerateRandomVerificationCodeForUser(user.ID)
	if err != nil {
		t.Fatalf("GenerateRandomVerificationCodeForUser returned error: %v", err)
	}
	if len(code) != 8 {
		t.Errorf("verification code length: got %d, want 8", len(code))
	}

	has, err = VerifyUserHasVerificationCode(user.ID)
	if err != nil {
		t.Fatalf("VerifyUserHasVerificationCode returned error: %v", err)
	}
	if !has {
		t.Errorf("expected verification code after generation")
	}

	matches, expiry, err := VerifyUserVerificationCodeMatches(user.ID, code)
	if err != nil {
		t.Fatalf("VerifyUserVerificationCodeMatches returned error: %v", err)
	}
	if !matches || expiry == nil {
		t.Errorf("expected code to match with an expiry, got matches=%v expiry=%v", matches, expiry)
	}
	wrong, _, err := VerifyUserVerificationCodeMatches(user.ID, "WRONGONE")
	if err != nil {
		t.Fatalf("VerifyUserVerificationCodeMatches(wrong) returned error: %v", err)
	}
	if wrong {
		t.Errorf("expected wrong code to not match")
	}

	// Verified / enabled flags.
	verified, err := VerifyUserIsVerified(user.ID)
	if err != nil {
		t.Fatalf("VerifyUserIsVerified returned error: %v", err)
	}
	if verified {
		t.Errorf("expected user to start unverified")
	}
	if err := SetUserVerification(user.ID, true); err != nil {
		t.Fatalf("SetUserVerification returned error: %v", err)
	}
	verified, err = VerifyUserIsVerified(user.ID)
	if err != nil {
		t.Fatalf("VerifyUserIsVerified returned error: %v", err)
	}
	if !verified {
		t.Errorf("expected user to be verified after SetUserVerification")
	}

	enabled, err := VerifyUserIsEnabled(user.ID)
	if err != nil {
		t.Fatalf("VerifyUserIsEnabled returned error: %v", err)
	}
	if !enabled {
		t.Errorf("expected user to be enabled")
	}
}

func TestUpdateUserValuesByUserID(t *testing.T) {
	newTestDB(t)

	user := makeTestUser(t, "updatevalues@example.com", nil)
	birth := time.Date(1990, 5, 20, 0, 0, 0, 0, time.UTC)

	if err := UpdateUserValuesByUserID(user.ID, "changed@example.com", "newhash", true, &birth); err != nil {
		t.Fatalf("UpdateUserValuesByUserID returned error: %v", err)
	}

	got, err := GetAllUserInformation(user.ID)
	if err != nil {
		t.Fatalf("GetAllUserInformation returned error: %v", err)
	}
	if got.Email != "changed@example.com" {
		t.Errorf("email: got %q, want %q", got.Email, "changed@example.com")
	}
	if got.Password != "newhash" {
		t.Errorf("password: got %q, want %q", got.Password, "newhash")
	}
	if !got.SundayAlert {
		t.Errorf("expected sunday alert enabled")
	}
	if got.BirthDate == nil || !got.BirthDate.Equal(birth) {
		t.Errorf("birth date: got %v, want %v", got.BirthDate, birth)
	}

	// Updating an unknown user affects no rows and errors.
	if err := UpdateEmailValueByUserID(uuid.New(), "x@example.com"); err == nil {
		t.Errorf("expected error updating email of unknown user")
	}
}

func TestGetUsersInformationAndByEmail(t *testing.T) {
	newTestDB(t)

	u1 := makeTestUser(t, "listed1@example.com", nil)
	makeTestUser(t, "listed2@example.com", nil)
	makeTestUser(t, "listeddisabled@example.com", func(u *models.User) { u.Enabled = false })

	all, err := GetUsersInformation()
	if err != nil {
		t.Fatalf("GetUsersInformation returned error: %v", err)
	}
	if len(all) != 2 {
		t.Errorf("got %d enabled users, want 2", len(all))
	}

	// Censored by-email lookup still finds the user, but redacts the email field.
	censored, err := GetUserInformationByEmail("listed1@example.com")
	if err != nil {
		t.Fatalf("GetUserInformationByEmail returned error: %v", err)
	}
	if censored.ID != u1.ID || censored.Email != "REDACTED" {
		t.Errorf("censored lookup: id=%v email=%q", censored.ID, censored.Email)
	}

	// Uncensored by-email lookup keeps the email.
	full, err := GetAllUserInformationByEmail("listed1@example.com")
	if err != nil {
		t.Fatalf("GetAllUserInformationByEmail returned error: %v", err)
	}
	if full.Email != "listed1@example.com" {
		t.Errorf("uncensored email: got %q, want %q", full.Email, "listed1@example.com")
	}

	if _, err := GetAllUserInformationByEmail("missing@example.com"); err == nil {
		t.Errorf("expected error for unknown email")
	}
}

func TestGetAllUsersWithSundayAlertsEnabled(t *testing.T) {
	newTestDB(t)

	makeTestUser(t, "sundayon@example.com", func(u *models.User) { u.SundayAlert = true })
	makeTestUser(t, "sundayoff@example.com", nil)

	users, err := GetAllUsersWithSundayAlertsEnabled()
	if err != nil {
		t.Fatalf("GetAllUsersWithSundayAlertsEnabled returned error: %v", err)
	}
	if len(users) != 1 {
		t.Errorf("got %d sunday-alert users, want 1", len(users))
	}
}

func TestGenerateResetCodeAndLookup(t *testing.T) {
	newTestDB(t)

	user := makeTestUser(t, "reset@example.com", nil)

	code, err := GenerateRandomResetCodeForUser(user.ID, true)
	if err != nil {
		t.Fatalf("GenerateRandomResetCodeForUser returned error: %v", err)
	}
	if len(code) != 16 {
		t.Errorf("reset code length: got %d, want 16", len(code))
	}

	found, err := GetAllUserInformationByResetCode(code)
	if err != nil {
		t.Fatalf("GetAllUserInformationByResetCode returned error: %v", err)
	}
	if found.ID != user.ID {
		t.Errorf("reset code lookup returned wrong user")
	}

	if _, err := GetAllUserInformationByResetCode("NOSUCHCODE"); err == nil {
		t.Errorf("expected error for unknown reset code")
	}
}

func TestGetUserEmailByUserID(t *testing.T) {
	newTestDB(t)

	user := makeTestUser(t, "emailbyid@example.com", nil)

	email, ok, err := GetUserEmailByUserID(user.ID)
	if err != nil {
		t.Fatalf("GetUserEmailByUserID returned error: %v", err)
	}
	if !ok || email != "emailbyid@example.com" {
		t.Errorf("got ok=%v email=%q", ok, email)
	}

	if _, ok, err := GetUserEmailByUserID(uuid.New()); ok || err == nil {
		t.Errorf("expected not-found error for unknown user")
	}
}

func TestGetStravaUsersWithinSeason(t *testing.T) {
	newTestDB(t)

	season := makeSeason(t, "StravaSeason", time.Now(), time.Now().Add(30*24*time.Hour), true)

	// Connected user with a goal in the season → included.
	inSeason := makeTestUser(t, "stravaseason@example.com", func(u *models.User) { u.StravaCode = strPtr("r:token") })
	makeGoal(t, inSeason.ID, season.ID, true)

	// Connected user without a goal in the season → excluded.
	makeTestUser(t, "stravanoseason@example.com", func(u *models.User) { u.StravaCode = strPtr("r:token") })

	users, err := GetStravaUsersWithinSeason(season.ID)
	if err != nil {
		t.Fatalf("GetStravaUsersWithinSeason returned error: %v", err)
	}
	if len(users) != 1 || users[0].ID != inSeason.ID {
		t.Errorf("got %d strava users in season, want 1", len(users))
	}
}

func TestGetHevyUsersAndEnabledCount(t *testing.T) {
	newTestDB(t)

	makeTestUser(t, "hevy@example.com", func(u *models.User) { u.HevyAPIKey = strPtr("api-key") })
	makeTestUser(t, "nohevy@example.com", nil)
	makeTestUser(t, "hevydisabled@example.com", func(u *models.User) {
		u.HevyAPIKey = strPtr("api-key")
		u.Enabled = false
	})

	hevy, err := GetHevyUsers()
	if err != nil {
		t.Fatalf("GetHevyUsers returned error: %v", err)
	}
	if len(hevy) != 1 {
		t.Errorf("got %d hevy users, want 1 (disabled excluded)", len(hevy))
	}

	count, err := GetAmountOfEnabledUsers()
	if err != nil {
		t.Fatalf("GetAmountOfEnabledUsers returned error: %v", err)
	}
	if count != 2 {
		t.Errorf("enabled user count: got %d, want 2", count)
	}
}
