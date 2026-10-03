package controllers

import (
	"net/http"
	"testing"

	"github.com/aunefyren/treningheten/database"
	"github.com/aunefyren/treningheten/models"

	"gorm.io/gorm"
)

// failWritesToTable makes every create and update against one table fail, leaving all other
// statements alone — a narrower tool than the fault sweep's "fail the Nth operation", for
// pinning what a handler does when one specific write is lost.
func failWritesToTable(t *testing.T, db *gorm.DB, table string) {
	t.Helper()

	hook := func(tx *gorm.DB) {
		if tx.Statement.Table == table {
			_ = tx.AddError(errInjectedFault)
		}
	}
	callbacks := db.Callback()
	for _, err := range []error{
		callbacks.Create().Before("gorm:create").Register("fail-table:create", hook),
		callbacks.Update().Before("gorm:update").Register("fail-table:update", hook),
	} {
		if err != nil {
			t.Fatalf("failed to register table fault: %v", err)
		}
	}
}

// TestChooseWinnerFailsWhenTheWinnerIsNotSaved covers that the wheel never announces a
// result it could not store: the spin answers 500 and nothing downstream (wheel views,
// e-mails) presents it as final, so the loser can spin again once the database recovers.
func TestChooseWinnerFailsWhenTheWinnerIsNotSaved(t *testing.T) {
	h := newAPIHarness(t)
	w := buildFaultWorld(t, h)
	failWritesToTable(t, database.Instance, "debts")

	h.expect(http.StatusInternalServerError, "POST", "/api/auth/debts/"+w.debtID.String()+"/choose", w.memberToken, nil)

	debt, found, err := database.GetDebtByDebtID(w.debtID)
	if err != nil || !found {
		t.Fatalf("GetDebtByDebtID returned found=%v err=%v", found, err)
	}
	if debt.WinnerID != nil {
		t.Errorf("winner = %v, want none stored", debt.WinnerID)
	}

	var wheelviews int64
	if err := database.Instance.Model(&models.Wheelview{}).Where("debt_id = ?", w.debtID).Count(&wheelviews).Error; err != nil {
		t.Fatal(err)
	}
	if wheelviews != 0 {
		t.Errorf("%d wheel views created for a spin that was never saved", wheelviews)
	}
}

// TestGenerateDebtReportsAFailedDebtWrite covers that the admin "generate debt" action no
// longer claims success when a loser's debt could not be written, while a clean run still
// writes it (the positive control, so the failure case is known to reach the write).
func TestGenerateDebtReportsAFailedDebtWrite(t *testing.T) {
	tests := []struct {
		name       string
		failDebts  bool
		wantStatus int
		wantDebts  int64
	}{
		{name: "clean run writes the loser's debt", failDebts: false, wantStatus: http.StatusOK, wantDebts: 1},
		{name: "a failed debt write is an error", failDebts: true, wantStatus: http.StatusInternalServerError, wantDebts: 0},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			h := newAPIHarness(t)
			w := buildFaultWorld(t, h)

			// The world ships with last week's debt already recorded; clear it so the
			// generator has a loser to write.
			if err := database.Instance.Model(&models.Debt{}).Where("id = ?", w.debtID).Update("enabled", false).Error; err != nil {
				t.Fatal(err)
			}
			if test.failDebts {
				failWritesToTable(t, database.Instance, "debts")
			}

			h.expect(test.wantStatus, "POST", "/api/admin/debts", w.adminToken, map[string]any{"date": wednesdayOfWeeksAgo(1)})

			var debts int64
			if err := database.Instance.Model(&models.Debt{}).Where("enabled = ?", true).Where("loser_id = ?", w.member.ID).Count(&debts).Error; err != nil {
				t.Fatal(err)
			}
			if debts != test.wantDebts {
				t.Errorf("debts for the loser = %d, want %d", debts, test.wantDebts)
			}
		})
	}
}

// TestRegisterGoalReportsAFailedSickLeaveAllowance covers that joining a season no longer
// answers 201 when the goal's sick-leave allowance could not be created.
func TestRegisterGoalReportsAFailedSickLeaveAllowance(t *testing.T) {
	h := newAPIHarness(t)
	w := buildFaultWorld(t, h)
	if w.season.Sickleave == 0 {
		t.Fatal("the fault world's season grants no sick leave; the test would not reach the write")
	}
	failWritesToTable(t, database.Instance, "sickleaves")

	_, token := h.user("joiner@partial.test", false)
	h.expect(http.StatusInternalServerError, "POST", "/api/auth/goals", token, models.GoalCreationRequest{ExerciseInterval: 2, Competing: true, SeasonID: w.season.ID})
}
