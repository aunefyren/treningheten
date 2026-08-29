package controllers

import (
	"bytes"
	"encoding/base64"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/aunefyren/treningheten/auth"
	"github.com/aunefyren/treningheten/database"
	"github.com/aunefyren/treningheten/files"
	"github.com/aunefyren/treningheten/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// withTestSigningKey installs a valid signing key so tokens can be minted and parsed
// in-process. A key that fails to decode would send files.GetPrivateKey down its
// ResetSecureKey path, which writes config/config.json — never do that from a test.
func withTestSigningKey(t *testing.T) {
	t.Helper()

	previous := files.ConfigFile
	files.ConfigFile.PrivateKey = base64.StdEncoding.EncodeToString(bytes.Repeat([]byte("k"), 64))
	// No external URL → tokens carry no audience claim, so ParseToken skips the check.
	files.ConfigFile.TreninghetenExternalURL = ""
	files.ConfigFile.SMTPEnabled = false

	t.Cleanup(func() { files.ConfigFile = previous })
}

func makeControllerTestUser(t *testing.T, email string) models.User {
	t.Helper()

	user := models.User{FirstName: "Test", LastName: "User", Email: email, Password: "hashed", Enabled: true, Verified: true}
	user.ID = uuid.New()

	created, err := database.RegisterUserInDB(user)
	if err != nil {
		t.Fatalf("failed to create user %s: %v", email, err)
	}
	return created
}

func makeControllerTestDay(t *testing.T, userID uuid.UUID, note string) models.ExerciseDay {
	t.Helper()

	day := models.ExerciseDay{Date: time.Now(), Note: note, Enabled: true, UserID: &userID}
	day.ID = uuid.New()

	if err := database.Instance.Create(&day).Error; err != nil {
		t.Fatalf("failed to create exercise day: %v", err)
	}
	return day
}

// postDayNote drives APIUpdateExerciseDay as the given user against the given day.
func postDayNote(t *testing.T, callerID uuid.UUID, dayID uuid.UUID, note string) *httptest.ResponseRecorder {
	t.Helper()

	token, _, err := auth.GenerateAccessToken(callerID, false, models.ScopeAPI, "test-client")
	if err != nil {
		t.Fatalf("failed to mint access token: %v", err)
	}

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest("POST", "/", bytes.NewBufferString(`{"note":"`+note+`"}`))
	context.Request.Header.Set("Content-Type", "application/json")
	context.Request.Header.Set("Authorization", "Bearer "+token)
	context.Params = gin.Params{{Key: "exercise_day_id", Value: dayID.String()}}

	APIUpdateExerciseDay(context)
	return recorder
}

func noteFor(t *testing.T, dayID uuid.UUID) string {
	t.Helper()

	var day models.ExerciseDay
	if err := database.Instance.Where("id = ?", dayID).First(&day).Error; err != nil {
		t.Fatalf("failed to re-read exercise day: %v", err)
	}
	return day.Note
}

// The write used to load the day by ID alone, so any authenticated user could overwrite
// anyone's note — which, with the note rendered into innerHTML, was a stored-XSS vector
// against the owner. The lookup must be scoped to the caller.
func TestUpdateExerciseDayRejectsNonOwner(t *testing.T) {
	newControllerTestDB(t)
	withTestSigningKey(t)

	owner := makeControllerTestUser(t, "owner@day.test")
	stranger := makeControllerTestUser(t, "stranger@day.test")
	day := makeControllerTestDay(t, owner.ID, "original note")

	recorder := postDayNote(t, stranger.ID, day.ID, "<img src=x onerror=alert(1)>")

	if recorder.Code != 404 {
		t.Errorf("status = %d, want 404 for another user's day; body: %s", recorder.Code, recorder.Body.String())
	}
	if got := noteFor(t, day.ID); got != "original note" {
		t.Errorf("note = %q, want it untouched by the non-owner", got)
	}
}

func TestUpdateExerciseDayAllowsOwner(t *testing.T) {
	newControllerTestDB(t)
	withTestSigningKey(t)

	owner := makeControllerTestUser(t, "owner2@day.test")
	day := makeControllerTestDay(t, owner.ID, "original note")

	recorder := postDayNote(t, owner.ID, day.ID, "my updated note")

	if recorder.Code < 200 || recorder.Code > 299 {
		t.Fatalf("status = %d, want 2xx for the owner; body: %s", recorder.Code, recorder.Body.String())
	}
	if got := noteFor(t, day.ID); got != "my updated note" {
		t.Errorf("note = %q, want it updated by the owner", got)
	}
}
