package database

import (
	"testing"
	"time"

	"github.com/aunefyren/treningheten/models"

	"github.com/google/uuid"
)

// makeDebt inserts a debt (associations omitted) and returns it.
func makeDebt(t *testing.T, seasonID, loserID uuid.UUID, winnerID *uuid.UUID, date time.Time, paid bool) models.Debt {
	t.Helper()
	debt := models.Debt{Date: date, SeasonID: seasonID, LoserID: loserID, WinnerID: winnerID, Paid: paid, Enabled: true}
	debt.ID = uuid.New()
	insertRow(t, &debt)
	return debt
}

func TestRegisterAndGetDebtByID(t *testing.T) {
	newTestDB(t)

	loser := makeTestUser(t, "debtloser@example.com", nil)
	debt := models.Debt{Date: time.Now(), SeasonID: uuid.New(), LoserID: loser.ID, Enabled: true}
	debt.ID = uuid.New()

	created, err := RegisterDebtInDB(debt)
	if err != nil {
		t.Fatalf("RegisterDebtInDB returned error: %v", err)
	}

	found, ok, err := GetDebtByDebtID(created.ID)
	if err != nil {
		t.Fatalf("GetDebtByDebtID returned error: %v", err)
	}
	if !ok || found.ID != created.ID {
		t.Errorf("expected to find debt, got ok=%v", ok)
	}

	_, ok, err = GetDebtByDebtID(uuid.New())
	if err != nil {
		t.Fatalf("GetDebtByDebtID(missing) returned error: %v", err)
	}
	if ok {
		t.Errorf("expected not found for unknown debt id")
	}
}

func TestGetDebtForWeek(t *testing.T) {
	newTestDB(t)

	loser := makeTestUser(t, "debtweek@example.com", nil)
	seasonID := uuid.New()
	// 2024-01-10 is a Wednesday; its week runs Mon 2024-01-08 .. Sun 2024-01-14.
	inWeek := time.Date(2024, 1, 10, 12, 0, 0, 0, time.UTC)
	makeDebt(t, seasonID, loser.ID, nil, inWeek, false)

	got, ok, err := GetDebtForWeekForUser(inWeek, loser.ID)
	if err != nil {
		t.Fatalf("GetDebtForWeekForUser returned error: %v", err)
	}
	if !ok || got.LoserID != loser.ID {
		t.Errorf("expected to find debt for the week, got ok=%v", ok)
	}

	// A date in a different week finds nothing.
	otherWeek := time.Date(2024, 2, 10, 12, 0, 0, 0, time.UTC)
	_, ok, err = GetDebtForWeekForUser(otherWeek, loser.ID)
	if err != nil {
		t.Fatalf("GetDebtForWeekForUser(other) returned error: %v", err)
	}
	if ok {
		t.Errorf("expected no debt in an unrelated week")
	}

	// Season-scoped variant: right season hits, wrong season misses.
	gotSeason, ok, err := GetDebtForWeekForUserInSeasonID(inWeek, loser.ID, seasonID)
	if err != nil {
		t.Fatalf("GetDebtForWeekForUserInSeasonID returned error: %v", err)
	}
	if !ok || gotSeason.LoserID != loser.ID {
		t.Errorf("expected to find season debt for the week, got ok=%v", ok)
	}
	_, ok, err = GetDebtForWeekForUserInSeasonID(inWeek, loser.ID, uuid.New())
	if err != nil {
		t.Fatalf("GetDebtForWeekForUserInSeasonID(other season) returned error: %v", err)
	}
	if ok {
		t.Errorf("expected no debt for a different season")
	}
}

func TestUnchosenDebtAndUpdateWinner(t *testing.T) {
	newTestDB(t)

	loser := makeTestUser(t, "unchosenloser@example.com", nil)
	winner := makeTestUser(t, "unchosenwinner@example.com", nil)
	debt := makeDebt(t, uuid.New(), loser.ID, nil, time.Now(), false)

	unchosen, ok, err := GetUnchosenDebtForUserByUserID(loser.ID)
	if err != nil {
		t.Fatalf("GetUnchosenDebtForUserByUserID returned error: %v", err)
	}
	if !ok || len(unchosen) != 1 {
		t.Errorf("expected 1 unchosen debt, got ok=%v len=%d", ok, len(unchosen))
	}

	if err := UpdateDebtWinner(debt.ID, winner.ID); err != nil {
		t.Fatalf("UpdateDebtWinner returned error: %v", err)
	}

	// Now that a winner is set, it is no longer unchosen.
	_, ok, err = GetUnchosenDebtForUserByUserID(loser.ID)
	if err != nil {
		t.Fatalf("GetUnchosenDebtForUserByUserID (after) returned error: %v", err)
	}
	if ok {
		t.Errorf("expected no unchosen debts after a winner was set")
	}

	// Updating an unknown debt affects no rows and errors.
	if err := UpdateDebtWinner(uuid.New(), winner.ID); err == nil {
		t.Errorf("expected error updating winner of unknown debt")
	}
}

func TestUnpaidUnreceivedAndPaidUpdate(t *testing.T) {
	newTestDB(t)

	loser := makeTestUser(t, "unpaidloser@example.com", nil)
	winner := makeTestUser(t, "unpaidwinner@example.com", nil)
	debt := makeDebt(t, uuid.New(), loser.ID, &winner.ID, time.Now(), false)

	unpaid, ok, err := GetUnpaidDebtForUser(loser.ID)
	if err != nil {
		t.Fatalf("GetUnpaidDebtForUser returned error: %v", err)
	}
	if !ok || len(unpaid) != 1 {
		t.Errorf("expected 1 unpaid debt for loser, got ok=%v len=%d", ok, len(unpaid))
	}

	unreceived, ok, err := GetUnreceivedDebtByUserID(winner.ID)
	if err != nil {
		t.Fatalf("GetUnreceivedDebtByUserID returned error: %v", err)
	}
	if !ok || len(unreceived) != 1 {
		t.Errorf("expected 1 unreceived debt for winner, got ok=%v len=%d", ok, len(unreceived))
	}

	// The paid update is scoped to the winner: the loser cannot mark it paid.
	if err := UpdateDebtPaidStatus(debt.ID, loser.ID); err == nil {
		t.Errorf("expected error when a non-winner marks a debt paid")
	}
	if err := UpdateDebtPaidStatus(debt.ID, winner.ID); err != nil {
		t.Fatalf("UpdateDebtPaidStatus returned error: %v", err)
	}

	// Once paid, it drops out of both unpaid/unreceived listings.
	_, ok, _ = GetUnpaidDebtForUser(loser.ID)
	if ok {
		t.Errorf("expected no unpaid debts after payment")
	}
	_, ok, _ = GetUnreceivedDebtByUserID(winner.ID)
	if ok {
		t.Errorf("expected no unreceived debts after payment")
	}
}

func TestGetDebtInSeasonWonLost(t *testing.T) {
	newTestDB(t)

	loser := makeTestUser(t, "seasonloser@example.com", nil)
	winner := makeTestUser(t, "seasonwinner@example.com", nil)
	seasonID := uuid.New()
	otherSeason := uuid.New()

	makeDebt(t, seasonID, loser.ID, &winner.ID, time.Now(), false)
	// A debt in another season must not be counted.
	makeDebt(t, otherSeason, loser.ID, &winner.ID, time.Now(), false)

	won, ok, err := GetDebtInSeasonWonByUserID(seasonID, winner.ID)
	if err != nil {
		t.Fatalf("GetDebtInSeasonWonByUserID returned error: %v", err)
	}
	if !ok || len(won) != 1 {
		t.Errorf("won debts: got ok=%v len=%d, want 1", ok, len(won))
	}

	lost, ok, err := GetDebtInSeasonLostByUserID(seasonID, loser.ID)
	if err != nil {
		t.Fatalf("GetDebtInSeasonLostByUserID returned error: %v", err)
	}
	if !ok || len(lost) != 1 {
		t.Errorf("lost debts: got ok=%v len=%d, want 1", ok, len(lost))
	}
}
