package database

import (
	"errors"
	"testing"
	"time"

	"github.com/aunefyren/treningheten/models"

	"github.com/google/uuid"
)

// Data-layer pieces of the admin "Fix a week" reconciliation (docs/admin-week-fix.md).

func TestGetDebtsForSeasonBetweenDates(t *testing.T) {
	newTestDB(t)

	loser := makeTestUser(t, "seasonweek@example.com", nil)
	seasonID := uuid.New()
	// 2024-01-08 (Mon) .. 2024-01-14 (Sun), in the server's zone like every caller: the
	// whole-day bounds are compared as strings, which Postgres/MySQL read in the session zone
	// (pinned to time.Local by newTestDB), so UTC dates would shift across the edges.
	monday := time.Date(2024, 1, 8, 0, 0, 0, 0, time.Local)
	sunday := time.Date(2024, 1, 14, 23, 59, 59, 0, time.Local)

	first := makeDebt(t, seasonID, loser.ID, nil, monday, false)
	second := makeDebt(t, seasonID, loser.ID, nil, time.Date(2024, 1, 14, 18, 0, 0, 0, time.Local), true)
	makeDebt(t, seasonID, loser.ID, nil, time.Date(2024, 1, 15, 0, 0, 0, 0, time.Local), false) // next week
	makeDebt(t, seasonID, loser.ID, nil, time.Date(2024, 1, 7, 23, 0, 0, 0, time.Local), false) // last week
	makeDebt(t, uuid.New(), loser.ID, nil, monday, false)                                     // other season
	disabled := makeDebt(t, seasonID, loser.ID, nil, monday, false)
	disableRow(t, &models.Debt{}, disabled.ID)
	// Oldest first, by creation.
	Instance.Model(&models.Debt{}).Where("id = ?", second.ID).Update("created_at", time.Now().Add(time.Hour))

	debts, err := GetDebtsForSeasonBetweenDates(seasonID, monday, sunday)
	if err != nil {
		t.Fatal(err)
	}
	if len(debts) != 2 || debts[0].ID != first.ID || debts[1].ID != second.ID {
		t.Errorf("got %+v, want the two enabled debts of the week, oldest first", debts)
	}

	none, err := GetDebtsForSeasonBetweenDates(uuid.New(), monday, sunday)
	if err != nil || len(none) != 0 {
		t.Errorf("unknown season: %v, %v", none, err)
	}
}

func TestGetDebtForWeekForUserInSeasonIDFindsDuplicates(t *testing.T) {
	newTestDB(t)

	loser := makeTestUser(t, "dupeweek@example.com", nil)
	seasonID := uuid.New()
	inWeek := time.Date(2024, 1, 10, 12, 0, 0, 0, time.Local)
	oldest := makeDebt(t, seasonID, loser.ID, nil, inWeek, false)
	newer := makeDebt(t, seasonID, loser.ID, nil, inWeek, false)
	Instance.Model(&models.Debt{}).Where("id = ?", newer.ID).Update("created_at", time.Now().Add(time.Hour))

	found, ok, err := GetDebtForWeekForUserInSeasonID(inWeek, loser.ID, seasonID)
	if err != nil || !ok || found.ID != oldest.ID {
		t.Errorf("got %v, %v, %v; want the oldest of the two", found.ID, ok, err)
	}
}

func TestConditionalDebtUpdates(t *testing.T) {
	winner := uuid.New()
	other := uuid.New()

	tests := []struct {
		name       string
		paid       bool
		disabled   bool
		stored     *uuid.UUID
		seen       *uuid.UUID
		newWinner  *uuid.UUID
		wantChange bool
	}{
		{name: "wheel debt to a winner", stored: nil, seen: nil, newWinner: &winner, wantChange: true},
		{name: "winner back to the wheel", stored: &winner, seen: &winner, newWinner: nil, wantChange: true},
		{name: "winner to another winner", stored: &winner, seen: &winner, newWinner: &other, wantChange: true},
		{name: "spun since it was read", stored: &winner, seen: nil, newWinner: &other, wantChange: false},
		{name: "re-assigned since it was read", stored: &other, seen: &winner, newWinner: nil, wantChange: false},
		{name: "paid", paid: true, stored: &winner, seen: &winner, newWinner: nil, wantChange: false},
		{name: "disabled", disabled: true, stored: nil, seen: nil, newWinner: &winner, wantChange: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			for _, operation := range []string{"set winner", "disable"} {
				t.Run(operation, func(t *testing.T) {
					newTestDB(t)
					loser := makeTestUser(t, "conditional@example.com", nil)
					debt := makeDebt(t, uuid.New(), loser.ID, test.stored, time.Now(), test.paid)
					if test.disabled {
						disableRow(t, &models.Debt{}, debt.ID)
					}

					var err error
					if operation == "set winner" {
						err = SetDebtWinnerIfUnchanged(debt.ID, test.seen, test.newWinner)
					} else {
						err = DisableDebtIfUnchanged(debt.ID, test.seen)
					}

					var stored models.Debt
					if loadErr := Instance.Where("debts.id = ?", debt.ID).First(&stored).Error; loadErr != nil {
						t.Fatal(loadErr)
					}

					if !test.wantChange {
						if !errors.Is(err, ErrDebtChanged) {
							t.Errorf("err = %v, want ErrDebtChanged", err)
						}
						if !sameWinner(stored.WinnerID, test.stored) || stored.Enabled == test.disabled {
							t.Errorf("debt changed anyway: %+v", stored)
						}
						return
					}

					if err != nil {
						t.Fatalf("err = %v", err)
					}
					if operation == "set winner" && (!sameWinner(stored.WinnerID, test.newWinner) || !stored.Enabled) {
						t.Errorf("winner = %v, enabled = %v; want %v, true", stored.WinnerID, stored.Enabled, test.newWinner)
					}
					if operation == "disable" && stored.Enabled {
						t.Error("debt still enabled")
					}
				})
			}
		})
	}
}

func sameWinner(a *uuid.UUID, b *uuid.UUID) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}

func TestDebtHasWheelviews(t *testing.T) {
	newTestDB(t)

	loser := makeTestUser(t, "wheel@example.com", nil)
	spun := makeDebt(t, uuid.New(), loser.ID, nil, time.Now(), false)
	unspun := makeDebt(t, uuid.New(), loser.ID, nil, time.Now(), false)

	view := models.Wheelview{UserID: loser.ID, DebtID: spun.ID}
	view.ID = uuid.New()
	if err := CreateWheelview(view); err != nil {
		t.Fatal(err)
	}

	if has, err := DebtHasWheelviews(spun.ID); err != nil || !has {
		t.Errorf("spun debt: %v, %v", has, err)
	}
	if has, err := DebtHasWheelviews(unspun.ID); err != nil || has {
		t.Errorf("unspun debt: %v, %v", has, err)
	}
}

func TestDisableAchievementDelegationByID(t *testing.T) {
	newTestDB(t)

	user := makeTestUser(t, "revoke@example.com", nil)
	achievement := makeAchievement(t, "Revocable", "Default", 1, true)
	kept := makeDelegation(t, user.ID, achievement.ID, false)
	revoked := makeDelegation(t, user.ID, achievement.ID, false)

	if ok, err := DisableAchievementDelegationByID(revoked.ID); err != nil || !ok {
		t.Fatalf("revoke: %v, %v", ok, err)
	}
	if ok, err := DisableAchievementDelegationByID(revoked.ID); err != nil || ok {
		t.Errorf("revoking twice: %v, %v; want nothing to revoke", ok, err)
	}
	if ok, err := DisableAchievementDelegationByID(uuid.New()); err != nil || ok {
		t.Errorf("unknown delegation: %v, %v", ok, err)
	}

	left, err := GetAchievementDelegationByAchievementIDAndUserID(user.ID, achievement.ID)
	if err != nil || len(left) != 1 || left[0].ID != kept.ID {
		t.Errorf("left = %+v, %v; want only the kept delegation", left, err)
	}
}
