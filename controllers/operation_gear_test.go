package controllers

import (
	"testing"

	"github.com/aunefyren/treningheten/database"
	"github.com/aunefyren/treningheten/models"

	"github.com/google/uuid"
)

// seedGear inserts an enabled gear row for a user and returns it.
func seedGear(t *testing.T, userID uuid.UUID, name string) models.Gear {
	t.Helper()

	gear := models.Gear{
		Enabled: true,
		UserID:  userID,
		Name:    name,
		Type:    "shoe",
	}
	gear.ID = uuid.New()

	created, err := database.CreateGearInDB(gear)
	if err != nil {
		t.Fatalf("failed to seed gear: %v", err)
	}
	return created
}

// TestResolveGearIDForUser covers the per-operation gear write path used by
// APIUpdateOperation: clearing, assigning owned gear, and the two rejections
// (malformed id, gear not owned by the user).
func TestResolveGearIDForUser(t *testing.T) {
	newControllerTestDB(t)

	user := createTestUser(t, "gear-owner@example.com", "Owner")
	other := createTestUser(t, "gear-other@example.com", "Other")

	gear := seedGear(t, user.ID, "Trail shoes")
	othersGear := seedGear(t, other.ID, "Someone else's bike")

	t.Run("empty clears gear", func(t *testing.T) {
		gearID, clientErr, serverErr := resolveGearIDForUser("", user.ID)
		if serverErr != nil {
			t.Fatalf("unexpected server error: %v", serverErr)
		}
		if clientErr != "" {
			t.Fatalf("unexpected client error: %q", clientErr)
		}
		if gearID != nil {
			t.Fatalf("expected nil gear id, got %v", *gearID)
		}
	})

	t.Run("assigns owned gear", func(t *testing.T) {
		gearID, clientErr, serverErr := resolveGearIDForUser(gear.ID.String(), user.ID)
		if serverErr != nil {
			t.Fatalf("unexpected server error: %v", serverErr)
		}
		if clientErr != "" {
			t.Fatalf("unexpected client error: %q", clientErr)
		}
		if gearID == nil || *gearID != gear.ID {
			t.Fatalf("expected gear id %v, got %v", gear.ID, gearID)
		}
	})

	t.Run("rejects malformed id", func(t *testing.T) {
		gearID, clientErr, serverErr := resolveGearIDForUser("not-a-uuid", user.ID)
		if serverErr != nil {
			t.Fatalf("unexpected server error: %v", serverErr)
		}
		if clientErr == "" {
			t.Fatal("expected a client error for a malformed gear id")
		}
		if gearID != nil {
			t.Fatalf("expected nil gear id, got %v", *gearID)
		}
	})

	t.Run("rejects gear owned by another user", func(t *testing.T) {
		gearID, clientErr, serverErr := resolveGearIDForUser(othersGear.ID.String(), user.ID)
		if serverErr != nil {
			t.Fatalf("unexpected server error: %v", serverErr)
		}
		if clientErr == "" {
			t.Fatal("expected a client error for gear the user does not own")
		}
		if gearID != nil {
			t.Fatalf("expected nil gear id, got %v", *gearID)
		}
	})
}
