package database

import (
	"testing"
	"time"

	"github.com/aunefyren/treningheten/models"

	"github.com/google/uuid"
)

// seedExerciseWithID inserts an exercise under a day and returns its ID.
func seedExerciseWithID(t *testing.T, dayID uuid.UUID) uuid.UUID {
	t.Helper()
	exercise := models.Exercise{Enabled: true, IsOn: true, ExerciseDayID: dayID}
	exercise.ID = uuid.New()
	if err := Instance.Omit("ExerciseDay").Create(&exercise).Error; err != nil {
		t.Fatalf("failed to seed exercise: %v", err)
	}
	return exercise.ID
}

// seedOperationWithGear inserts an operation (optionally linked to gear) and returns its ID.
func seedOperationWithGear(t *testing.T, exerciseID uuid.UUID, gearID *uuid.UUID) uuid.UUID {
	t.Helper()
	operation := models.Operation{Enabled: true, ExerciseID: exerciseID, GearID: gearID, Type: "running"}
	operation.ID = uuid.New()
	if err := Instance.Omit("Exercise", "Action", "Gear").Create(&operation).Error; err != nil {
		t.Fatalf("failed to seed operation: %v", err)
	}
	return operation.ID
}

// seedOperationSetWithDistance inserts an operation set carrying a distance (km).
func seedOperationSetWithDistance(t *testing.T, operationID uuid.UUID, distance float64) {
	t.Helper()
	d := distance
	set := models.OperationSet{Enabled: true, OperationID: operationID, Distance: &d}
	set.ID = uuid.New()
	if err := Instance.Omit("Operation").Create(&set).Error; err != nil {
		t.Fatalf("failed to seed operation set: %v", err)
	}
}

func TestGetGearDistanceTotalsForUser(t *testing.T) {
	newTestDB(t)
	user := makeTestUser(t, "gear@example.com", nil)

	shoe, err := CreateGearInDB(models.Gear{GormModel: models.GormModel{ID: uuid.New()}, Enabled: true, UserID: user.ID, Name: "Pegasus", Type: "shoe"})
	if err != nil {
		t.Fatalf("failed to create shoe gear: %v", err)
	}
	bike, err := CreateGearInDB(models.Gear{GormModel: models.GormModel{ID: uuid.New()}, Enabled: true, UserID: user.ID, Name: "Roadie", Type: "bike"})
	if err != nil {
		t.Fatalf("failed to create bike gear: %v", err)
	}
	unused, err := CreateGearInDB(models.Gear{GormModel: models.GormModel{ID: uuid.New()}, Enabled: true, UserID: user.ID, Name: "Spare", Type: "shoe"})
	if err != nil {
		t.Fatalf("failed to create unused gear: %v", err)
	}

	dayID := seedDay(t, user.ID, time.Now(), true)
	exerciseID := seedExerciseWithID(t, dayID)

	// Shoe: two operations summing to 8.0 km.
	op1 := seedOperationWithGear(t, exerciseID, &shoe.ID)
	seedOperationSetWithDistance(t, op1, 5.0)
	op2 := seedOperationWithGear(t, exerciseID, &shoe.ID)
	seedOperationSetWithDistance(t, op2, 3.0)

	// Bike: one operation with two sets summing to 12.5 km.
	op3 := seedOperationWithGear(t, exerciseID, &bike.ID)
	seedOperationSetWithDistance(t, op3, 10.0)
	seedOperationSetWithDistance(t, op3, 2.5)

	// An operation with no gear must not be counted toward any gear.
	op4 := seedOperationWithGear(t, exerciseID, nil)
	seedOperationSetWithDistance(t, op4, 4.0)

	totals, err := GetGearDistanceTotalsForUser(user.ID)
	if err != nil {
		t.Fatalf("GetGearDistanceTotalsForUser returned error: %v", err)
	}

	if got := totals[shoe.ID]; got != 8.0 {
		t.Errorf("shoe distance: got %v, want 8.0", got)
	}
	if got := totals[bike.ID]; got != 12.5 {
		t.Errorf("bike distance: got %v, want 12.5", got)
	}
	if _, ok := totals[unused.ID]; ok {
		t.Errorf("unused gear should not appear in totals, got %v", totals[unused.ID])
	}
}

func TestGetGearByStravaGearIDAndUserID(t *testing.T) {
	newTestDB(t)
	user := makeTestUser(t, "stravagear@example.com", nil)

	stravaID := "g12345"
	_, err := CreateGearInDB(models.Gear{GormModel: models.GormModel{ID: uuid.New()}, Enabled: true, UserID: user.ID, Name: "Synced", Type: "shoe", StravaGearID: &stravaID})
	if err != nil {
		t.Fatalf("failed to create strava gear: %v", err)
	}

	found, err := GetGearByStravaGearIDAndUserID(stravaID, user.ID)
	if err != nil {
		t.Fatalf("lookup returned error: %v", err)
	}
	if found == nil {
		t.Fatalf("expected to find gear by strava id, got nil")
	}

	missing, err := GetGearByStravaGearIDAndUserID("b99999", user.ID)
	if err != nil {
		t.Fatalf("lookup returned error: %v", err)
	}
	if missing != nil {
		t.Errorf("expected nil for unknown strava gear id, got %v", missing)
	}
}

func TestUnsetPrimaryGearForUser(t *testing.T) {
	newTestDB(t)
	user := makeTestUser(t, "primarygear@example.com", nil)

	old, err := CreateGearInDB(models.Gear{GormModel: models.GormModel{ID: uuid.New()}, Enabled: true, UserID: user.ID, Name: "Old", Type: "shoe", IsPrimary: true})
	if err != nil {
		t.Fatalf("failed to create old primary: %v", err)
	}
	newPrimary, err := CreateGearInDB(models.Gear{GormModel: models.GormModel{ID: uuid.New()}, Enabled: true, UserID: user.ID, Name: "New", Type: "shoe", IsPrimary: true})
	if err != nil {
		t.Fatalf("failed to create new primary: %v", err)
	}

	if err := UnsetPrimaryGearForUser(user.ID, newPrimary.ID); err != nil {
		t.Fatalf("UnsetPrimaryGearForUser returned error: %v", err)
	}

	demoted, err := GetGearByIDAndUserID(old.ID, user.ID)
	if err != nil || demoted == nil {
		t.Fatalf("failed to reload old gear: %v", err)
	}
	if demoted.IsPrimary {
		t.Errorf("old gear should have been demoted, still primary")
	}

	kept, err := GetGearByIDAndUserID(newPrimary.ID, user.ID)
	if err != nil || kept == nil {
		t.Fatalf("failed to reload new gear: %v", err)
	}
	if !kept.IsPrimary {
		t.Errorf("new primary gear should remain primary")
	}
}
