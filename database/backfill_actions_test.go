package database

import (
	"testing"
	"time"

	"github.com/aunefyren/treningheten/models"

	"github.com/google/uuid"
)

func loadAction(t *testing.T, id uuid.UUID) models.Action {
	t.Helper()
	var action models.Action
	if err := Instance.Where("id = ?", id).First(&action).Error; err != nil {
		t.Fatalf("load action: %v", err)
	}
	return action
}

func TestBackfillActionCategories(t *testing.T) {
	newTestDB(t)

	newAction := func(name string, actionType string, bodyPart string, hevyTemplateID *string, category string) uuid.UUID {
		action := models.Action{Enabled: true, Name: name, Type: actionType, BodyPart: bodyPart, HevyTemplateID: hevyTemplateID, Category: category}
		action.ID = uuid.New()
		insertRow(t, &action)
		return action.ID
	}
	hevyBench := newAction("Bench Press (Dumbbell)", "lifting", "chest", strPtr("H1"), "")
	hevySpin := newAction("Spinning", "timing", "cardio", strPtr("H2"), "")
	hevyStretch := newAction("Stretching", "timing", "other", strPtr("H3"), "")
	userLift := newAction("Garage Press", "lifting", "Arms", nil, "")
	userRide := newAction("Gravel Loop", "moving", "", nil, "")
	userTimed := newAction("Trampoline", "timing", "", nil, "")
	chosen := newAction("Kept", "lifting", "", nil, models.ActionCategoryMobility)

	backfillActionCategories()

	tests := []struct {
		name          string
		id            uuid.UUID
		wantCategory  string
		wantEquipment string
	}{
		{"hevy lift with equipment", hevyBench, models.ActionCategoryStrength, models.ActionEquipmentDumbbell},
		{"hevy cardio", hevySpin, models.ActionCategoryCardio, ""},
		{"hevy other is mobility", hevyStretch, models.ActionCategoryMobility, ""},
		{"user lifting", userLift, models.ActionCategoryStrength, ""},
		{"user moving", userRide, models.ActionCategoryCardio, ""},
		{"user timing", userTimed, models.ActionCategoryGeneral, ""},
		{"existing category kept", chosen, models.ActionCategoryMobility, ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			action := loadAction(t, tc.id)
			if action.Category != tc.wantCategory || action.Equipment != tc.wantEquipment {
				t.Fatalf("category/equipment = %q/%q, want %q/%q", action.Category, action.Equipment, tc.wantCategory, tc.wantEquipment)
			}
		})
	}

	// Self-limiting: nothing is left without a category, so a second run is a no-op.
	var pending int64
	if err := Instance.Model(&models.Action{}).Where("category IS NULL OR category = ?", "").Count(&pending).Error; err != nil {
		t.Fatal(err)
	}
	if pending != 0 {
		t.Fatalf("%d actions still without a category", pending)
	}
	backfillActionCategories()
}

func TestBackfillHevyOperationNotes(t *testing.T) {
	newTestDB(t)
	user := makeTestUser(t, "hevy-notes@test.dev", nil)

	day := models.ExerciseDay{UserID: &user.ID, Enabled: true, Date: time.Now()}
	day.ID = uuid.New()
	insertRow(t, &day)
	newExercise := func(hevyWorkoutID *string) uuid.UUID {
		exercise := models.Exercise{ExerciseDayID: day.ID, Enabled: true, IsOn: true, HevyWorkoutID: hevyWorkoutID}
		exercise.ID = uuid.New()
		insertRow(t, &exercise)
		return exercise.ID
	}
	hevySession := newExercise(strPtr("workout-1"))
	stravaSession := newExercise(nil)

	action := models.Action{Enabled: true, Name: "Squat (Barbell)", Type: "lifting", Category: models.ActionCategoryStrength}
	action.ID = uuid.New()
	insertRow(t, &action)

	newOperation := func(exerciseID uuid.UUID, actionID *uuid.UUID, note *string, description *string) uuid.UUID {
		operation := models.Operation{ExerciseID: exerciseID, ActionID: actionID, Enabled: true, Type: "lifting", WeightUnit: "kg", DistanceUnit: "km", Note: note, Description: description}
		operation.ID = uuid.New()
		insertRow(t, &operation)
		return operation.ID
	}
	catalogNote := newOperation(hevySession, &action.ID, strPtr("deep"), nil)
	customTitle := newOperation(hevySession, nil, strPtr("My Secret Move"), strPtr("slow"))
	bothSet := newOperation(hevySession, &action.ID, strPtr("keep"), strPtr("already described"))
	stravaTitle := newOperation(stravaSession, &action.ID, strPtr("Morning Lift"), nil)

	backfillHevyOperationNotes()

	tests := []struct {
		name            string
		id              uuid.UUID
		wantNote        *string
		wantDescription *string
	}{
		{"catalog note moves to description", catalogNote, nil, strPtr("deep")},
		{"custom exercise keeps its title", customTitle, strPtr("My Secret Move"), strPtr("slow")},
		{"existing description is not overwritten", bothSet, strPtr("keep"), strPtr("already described")},
		{"non-Hevy session untouched", stravaTitle, strPtr("Morning Lift"), nil},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			operation := loadOperation(t, tc.id)
			if !equalStringPtr(operation.Note, tc.wantNote) || !equalStringPtr(operation.Description, tc.wantDescription) {
				t.Fatalf("note/description = %v/%v, want %v/%v", deref(operation.Note), deref(operation.Description), deref(tc.wantNote), deref(tc.wantDescription))
			}
		})
	}

	// Self-limiting: a second run changes nothing.
	backfillHevyOperationNotes()
	if operation := loadOperation(t, catalogNote); operation.Note != nil || deref(operation.Description) != "deep" {
		t.Fatalf("second run changed the moved row: %v/%v", deref(operation.Note), deref(operation.Description))
	}
}

func equalStringPtr(a, b *string) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

func deref(value *string) string {
	if value == nil {
		return "<nil>"
	}
	return *value
}

func TestActionBackfillsScanFailure(t *testing.T) {
	newTestDB(t)
	sqlDB, err := Instance.DB()
	if err != nil {
		t.Fatalf("failed to get sql db: %v", err)
	}
	sqlDB.Close()

	// A failed scan or update is logged and skipped, never a panic during startup.
	backfillActionCategories()
	backfillHevyOperationNotes()
}
