package database

import (
	"testing"

	"github.com/aunefyren/treningheten/models"

	"github.com/google/uuid"
)

func TestGetGearForUserOrderingAndScope(t *testing.T) {
	newTestDB(t)

	user := makeTestUser(t, "gearlist@example.com", nil)

	// Ordering is retired ASC, then name ASC → active "Alpha" before retired "Zeta".
	active, err := CreateGearInDB(models.Gear{GormModel: models.GormModel{ID: uuid.New()}, Enabled: true, UserID: user.ID, Name: "Alpha", Type: "shoe"})
	if err != nil {
		t.Fatalf("failed to create active gear: %v", err)
	}
	retired, err := CreateGearInDB(models.Gear{GormModel: models.GormModel{ID: uuid.New()}, Enabled: true, UserID: user.ID, Name: "Zeta", Type: "shoe", Retired: true})
	if err != nil {
		t.Fatalf("failed to create retired gear: %v", err)
	}

	// Disabled gear must be excluded.
	disabled, err := CreateGearInDB(models.Gear{GormModel: models.GormModel{ID: uuid.New()}, Enabled: true, UserID: user.ID, Name: "Gone", Type: "shoe"})
	if err != nil {
		t.Fatalf("failed to create disabled gear: %v", err)
	}
	disableRow(t, &models.Gear{}, disabled.ID)

	gear, err := GetGearForUser(user.ID)
	if err != nil {
		t.Fatalf("GetGearForUser returned error: %v", err)
	}
	if len(gear) != 2 {
		t.Fatalf("got %d gear, want 2 (enabled only)", len(gear))
	}
	if gear[0].ID != active.ID || gear[1].ID != retired.ID {
		t.Errorf("gear not ordered retired asc, name asc: got %q then %q", gear[0].Name, gear[1].Name)
	}
}

func TestGetGearByIDAndUpdate(t *testing.T) {
	newTestDB(t)

	user := makeTestUser(t, "gearupdate@example.com", nil)
	gear, err := CreateGearInDB(models.Gear{GormModel: models.GormModel{ID: uuid.New()}, Enabled: true, UserID: user.ID, Name: "Pegasus", Type: "shoe"})
	if err != nil {
		t.Fatalf("failed to create gear: %v", err)
	}

	// GetGearByID is not user-scoped.
	found, err := GetGearByID(gear.ID)
	if err != nil {
		t.Fatalf("GetGearByID returned error: %v", err)
	}
	if found == nil || found.ID != gear.ID {
		t.Errorf("expected to find gear by id, got %v", found)
	}

	missing, err := GetGearByID(uuid.New())
	if err != nil {
		t.Fatalf("GetGearByID(missing) returned error: %v", err)
	}
	if missing != nil {
		t.Errorf("expected nil for unknown gear id, got %v", missing)
	}

	gear.Name = "Vaporfly"
	updated, err := UpdateGearInDB(gear)
	if err != nil {
		t.Fatalf("UpdateGearInDB returned error: %v", err)
	}
	if updated.Name != "Vaporfly" {
		t.Errorf("gear name: got %q, want %q", updated.Name, "Vaporfly")
	}
	reloaded, _ := GetGearByID(gear.ID)
	if reloaded.Name != "Vaporfly" {
		t.Errorf("persisted gear name: got %q, want %q", reloaded.Name, "Vaporfly")
	}
}
