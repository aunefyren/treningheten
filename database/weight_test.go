package database

import (
	"testing"
	"time"

	"github.com/aunefyren/treningheten/models"

	"github.com/google/uuid"
)

// makeTestWeight inserts a weight row for the given user. Because WeightValue.Enabled
// carries a `default: true` tag, GORM omits a zero-value (false) Enabled from the
// INSERT and the DB default wins — so a disabled row is produced the way the app does
// it: insert enabled, then Save it disabled.
func makeTestWeight(t *testing.T, userID uuid.UUID, weight float64, enabled bool) models.WeightValue {
	t.Helper()

	w := models.WeightValue{
		Enabled: true,
		Date:    time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC),
		Weight:  weight,
		UserID:  userID,
	}
	w.ID = uuid.New()

	created, err := CreateWeightInDB(w)
	if err != nil {
		t.Fatalf("CreateWeightInDB() error: %v", err)
	}

	if !enabled {
		created.Enabled = false
		created, err = UpdateWeightInDB(created)
		if err != nil {
			t.Fatalf("UpdateWeightInDB() error disabling weight: %v", err)
		}
	}
	return created
}

func TestCreateAndGetWeightsForUser(t *testing.T) {
	newTestDB(t)
	user := makeTestUser(t, "weight@me.test", nil)

	makeTestWeight(t, user.ID, 80.5, true)
	makeTestWeight(t, user.ID, 81.0, true)

	weights, err := GetEnabledWeightsForUser(user.ID)
	if err != nil {
		t.Fatalf("GetEnabledWeightsForUser() error: %v", err)
	}
	if len(weights) != 2 {
		t.Fatalf("got %d weights, want 2", len(weights))
	}
}

// GetEnabledWeightsForUser must exclude disabled rows and other users' rows.
func TestGetEnabledWeightsForUserFilters(t *testing.T) {
	newTestDB(t)
	user := makeTestUser(t, "mine@weight.test", nil)
	other := makeTestUser(t, "other@weight.test", nil)

	wanted := makeTestWeight(t, user.ID, 75.0, true)
	makeTestWeight(t, user.ID, 76.0, false) // disabled → excluded
	makeTestWeight(t, other.ID, 90.0, true) // other user → excluded

	weights, err := GetEnabledWeightsForUser(user.ID)
	if err != nil {
		t.Fatalf("GetEnabledWeightsForUser() error: %v", err)
	}
	if len(weights) != 1 {
		t.Fatalf("got %d weights, want 1", len(weights))
	}
	if weights[0].ID != wanted.ID {
		t.Errorf("returned weight ID = %v, want %v", weights[0].ID, wanted.ID)
	}
}

func TestGetEnabledWeightByID(t *testing.T) {
	newTestDB(t)
	user := makeTestUser(t, "byid@weight.test", nil)
	other := makeTestUser(t, "byidother@weight.test", nil)
	created := makeTestWeight(t, user.ID, 70.0, true)

	t.Run("found", func(t *testing.T) {
		got, err := GetEnabledWeightsByWeightIDAndUserID(user.ID, created.ID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.Weight != 70.0 {
			t.Errorf("Weight = %v, want 70.0", got.Weight)
		}
	})

	t.Run("wrong user is denied", func(t *testing.T) {
		if _, err := GetEnabledWeightsByWeightIDAndUserID(other.ID, created.ID); err == nil {
			t.Error("expected error fetching another user's weight, got nil")
		}
	})

	t.Run("missing id errors", func(t *testing.T) {
		if _, err := GetEnabledWeightsByWeightIDAndUserID(user.ID, uuid.New()); err == nil {
			t.Error("expected error for missing weight, got nil")
		}
	})
}

// Soft-deleting a weight (Enabled=false via UpdateWeightInDB) removes it from the
// enabled read path.
func TestUpdateWeightSoftDisable(t *testing.T) {
	newTestDB(t)
	user := makeTestUser(t, "disable@weight.test", nil)
	created := makeTestWeight(t, user.ID, 82.0, true)

	created.Enabled = false
	if _, err := UpdateWeightInDB(created); err != nil {
		t.Fatalf("UpdateWeightInDB() error: %v", err)
	}

	weights, err := GetEnabledWeightsForUser(user.ID)
	if err != nil {
		t.Fatalf("GetEnabledWeightsForUser() error: %v", err)
	}
	if len(weights) != 0 {
		t.Errorf("got %d enabled weights after disable, want 0", len(weights))
	}
}

func TestUpdateWeightPersistsValue(t *testing.T) {
	newTestDB(t)
	user := makeTestUser(t, "persist@weight.test", nil)
	created := makeTestWeight(t, user.ID, 82.0, true)

	created.Weight = 79.3
	if _, err := UpdateWeightInDB(created); err != nil {
		t.Fatalf("UpdateWeightInDB() error: %v", err)
	}

	got, err := GetEnabledWeightsByWeightIDAndUserID(user.ID, created.ID)
	if err != nil {
		t.Fatalf("GetEnabledWeightsByWeightIDAndUserID() error: %v", err)
	}
	if got.Weight != 79.3 {
		t.Errorf("Weight = %v, want 79.3", got.Weight)
	}
}
