package controllers

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/aunefyren/treningheten/auth"
	"github.com/aunefyren/treningheten/database"
	"github.com/aunefyren/treningheten/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func makeControllerTestSession(t *testing.T, userID uuid.UUID) models.Exercise {
	t.Helper()

	day := models.ExerciseDay{Date: time.Now(), Enabled: true, UserID: &userID}
	day.ID = uuid.New()
	if err := database.Instance.Omit("User", "Goal").Create(&day).Error; err != nil {
		t.Fatalf("failed to create exercise day: %v", err)
	}

	session := models.Exercise{Enabled: true, IsOn: true, ExerciseDayID: day.ID}
	session.ID = uuid.New()
	if err := database.Instance.Omit("ExerciseDay").Create(&session).Error; err != nil {
		t.Fatalf("failed to create exercise: %v", err)
	}
	return session
}

// postOperation drives APICreateOperationForUser as the given user against the given session.
func postOperation(t *testing.T, callerID uuid.UUID, exerciseID uuid.UUID) *httptest.ResponseRecorder {
	t.Helper()

	token, _, err := auth.GenerateAccessToken(callerID, false, models.ScopeAPI, "test-client")
	if err != nil {
		t.Fatalf("failed to mint access token: %v", err)
	}

	body, err := json.Marshal(models.OperationCreationRequest{
		ExerciseID: exerciseID, Type: "lifting", WeightUnit: "kg", DistanceUnit: "km",
	})
	if err != nil {
		t.Fatalf("failed to marshal request: %v", err)
	}

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest("POST", "/", bytes.NewBuffer(body))
	context.Request.Header.Set("Content-Type", "application/json")
	context.Request.Header.Set("Authorization", "Bearer "+token)

	APICreateOperationForUser(context)
	return recorder
}

func operationCountFor(t *testing.T, exerciseID uuid.UUID) int64 {
	t.Helper()

	var count int64
	if err := database.Instance.Model(&models.Operation{}).Where("exercise_id = ?", exerciseID).Count(&count).Error; err != nil {
		t.Fatalf("failed to count operations: %v", err)
	}
	return count
}

// The ownership check discarded the looked-up exercise and tested only the error, but
// GetExerciseByIDAndUserID reported "not yours / no such session" as (nil error, zero-value
// struct) — so the guard never fired and any authenticated user could attach an activity to
// another user's session.
func TestCreateOperationRejectsAnotherUsersSession(t *testing.T) {
	newControllerTestDB(t)
	withTestSigningKey(t)

	owner := makeControllerTestUser(t, "owner@operation.test")
	stranger := makeControllerTestUser(t, "stranger@operation.test")
	session := makeControllerTestSession(t, owner.ID)

	recorder := postOperation(t, stranger.ID, session.ID)

	if recorder.Code != 404 {
		t.Errorf("status = %d, want 404 for another user's session; body: %s", recorder.Code, recorder.Body.String())
	}
	if got := operationCountFor(t, session.ID); got != 0 {
		t.Errorf("operations attached to the owner's session = %d, want 0", got)
	}
}

func TestCreateOperationRejectsUnknownSession(t *testing.T) {
	newControllerTestDB(t)
	withTestSigningKey(t)

	user := makeControllerTestUser(t, "unknown@operation.test")

	recorder := postOperation(t, user.ID, uuid.New())

	if recorder.Code != 404 {
		t.Errorf("status = %d, want 404 for a session that does not exist; body: %s", recorder.Code, recorder.Body.String())
	}
}

func TestCreateOperationAllowsOwner(t *testing.T) {
	newControllerTestDB(t)
	withTestSigningKey(t)

	owner := makeControllerTestUser(t, "owner2@operation.test")
	session := makeControllerTestSession(t, owner.ID)

	recorder := postOperation(t, owner.ID, session.ID)

	if recorder.Code < 200 || recorder.Code > 299 {
		t.Fatalf("status = %d, want 2xx for the owner; body: %s", recorder.Code, recorder.Body.String())
	}
	if got := operationCountFor(t, session.ID); got != 1 {
		t.Errorf("operations on the owner's session = %d, want 1", got)
	}
}
