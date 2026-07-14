package database

import (
	"testing"
	"time"

	"github.com/aunefyren/treningheten/models"

	"github.com/google/uuid"
)

// seedDebtRow inserts a debt (associations omitted) for the given loser and returns its ID.
func seedDebtRow(t *testing.T, loserID uuid.UUID, enabled bool) uuid.UUID {
	t.Helper()
	debt := models.Debt{Date: time.Now(), LoserID: loserID, Enabled: enabled}
	debt.ID = uuid.New()
	if err := Instance.Omit("Season", "Loser", "Winner").Create(&debt).Error; err != nil {
		t.Fatalf("failed to seed debt: %v", err)
	}
	if err := Instance.Model(&models.Debt{}).Where("id = ?", debt.ID).Update("enabled", enabled).Error; err != nil {
		t.Fatalf("failed to set debt enabled: %v", err)
	}
	return debt.ID
}

// seedWheelview inserts a wheelview (associations omitted) and returns its ID.
func seedWheelview(t *testing.T, userID, debtID uuid.UUID) uuid.UUID {
	t.Helper()
	wheelview := models.Wheelview{UserID: userID, DebtID: debtID, Enabled: true, Viewed: false}
	wheelview.ID = uuid.New()
	if err := Instance.Omit("User", "Debt").Create(&wheelview).Error; err != nil {
		t.Fatalf("failed to seed wheelview: %v", err)
	}
	return wheelview.ID
}

func TestCreateWheelviewAndLookups(t *testing.T) {
	newTestDB(t)

	user := makeTestUser(t, "wheel@example.com", nil)
	debtID := seedDebtRow(t, user.ID, true)

	wheelview := models.Wheelview{UserID: user.ID, DebtID: debtID, Enabled: true}
	wheelview.ID = uuid.New()
	if err := CreateWheelview(wheelview); err != nil {
		t.Fatalf("CreateWheelview returned error: %v", err)
	}

	unviewed, ok, err := GetUnviewedWheelviewByDebtIDAndUserID(user.ID, debtID)
	if err != nil {
		t.Fatalf("GetUnviewedWheelviewByDebtIDAndUserID returned error: %v", err)
	}
	if !ok || unviewed.ID != wheelview.ID {
		t.Errorf("expected to find unviewed wheelview, got ok=%v", ok)
	}

	any, ok, err := GetWheelviewByDebtIDAndUserID(user.ID, debtID)
	if err != nil {
		t.Fatalf("GetWheelviewByDebtIDAndUserID returned error: %v", err)
	}
	if !ok || any.ID != wheelview.ID {
		t.Errorf("expected to find wheelview, got ok=%v", ok)
	}

	// Unknown debt → not found, no error.
	_, ok, err = GetUnviewedWheelviewByDebtIDAndUserID(user.ID, uuid.New())
	if err != nil {
		t.Fatalf("GetUnviewedWheelviewByDebtIDAndUserID(missing) returned error: %v", err)
	}
	if ok {
		t.Errorf("expected not found for unknown debt")
	}
}

func TestSetWheelviewToViewedByID(t *testing.T) {
	newTestDB(t)

	user := makeTestUser(t, "wheelviewed@example.com", nil)
	debtID := seedDebtRow(t, user.ID, true)
	wheelviewID := seedWheelview(t, user.ID, debtID)

	if err := SetWheelviewToViewedByID(wheelviewID); err != nil {
		t.Fatalf("SetWheelviewToViewedByID returned error: %v", err)
	}

	// No longer unviewed.
	_, ok, err := GetUnviewedWheelviewByDebtIDAndUserID(user.ID, debtID)
	if err != nil {
		t.Fatalf("GetUnviewedWheelviewByDebtIDAndUserID returned error: %v", err)
	}
	if ok {
		t.Errorf("viewed wheelview should not be returned as unviewed")
	}

	// But still retrievable regardless of viewed status.
	_, ok, err = GetWheelviewByDebtIDAndUserID(user.ID, debtID)
	if err != nil {
		t.Fatalf("GetWheelviewByDebtIDAndUserID returned error: %v", err)
	}
	if !ok {
		t.Errorf("viewed wheelview should still be retrievable")
	}

	// Setting an unknown wheelview to viewed affects no rows and errors.
	if err := SetWheelviewToViewedByID(uuid.New()); err == nil {
		t.Errorf("expected error setting unknown wheelview to viewed")
	}
}

func TestGetUnviewedWheelviewByUserID(t *testing.T) {
	newTestDB(t)

	user := makeTestUser(t, "wheeluser@example.com", nil)

	// Unviewed wheelview on an enabled debt → counted.
	enabledDebt := seedDebtRow(t, user.ID, true)
	seedWheelview(t, user.ID, enabledDebt)

	// Unviewed wheelview on a disabled debt → excluded by the join.
	disabledDebt := seedDebtRow(t, user.ID, false)
	seedWheelview(t, user.ID, disabledDebt)

	views, ok, err := GetUnviewedWheelviewByUserID(user.ID)
	if err != nil {
		t.Fatalf("GetUnviewedWheelviewByUserID returned error: %v", err)
	}
	if !ok || len(views) != 1 {
		t.Errorf("got ok=%v len=%d, want ok=true len=1", ok, len(views))
	}

	// A user with no wheelviews → not found.
	_, ok, err = GetUnviewedWheelviewByUserID(uuid.New())
	if err != nil {
		t.Fatalf("GetUnviewedWheelviewByUserID(none) returned error: %v", err)
	}
	if ok {
		t.Errorf("expected not found for user without wheelviews")
	}
}
