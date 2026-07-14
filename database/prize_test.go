package database

import (
	"testing"

	"github.com/aunefyren/treningheten/models"

	"github.com/google/uuid"
)

// makePrize inserts a prize and returns it.
func makePrize(t *testing.T, name string, quantity int) models.Prize {
	t.Helper()
	prize := models.Prize{Name: name, Quantity: quantity, Enabled: true}
	prize.ID = uuid.New()
	if err := CreatePrizeInDB(prize); err != nil {
		t.Fatalf("failed to create prize: %v", err)
	}
	return prize
}

func TestGetPrizeByID(t *testing.T) {
	newTestDB(t)

	prize := makePrize(t, "Trophy", 1)

	found, ok, err := GetPrizeByID(prize.ID)
	if err != nil {
		t.Fatalf("GetPrizeByID returned error: %v", err)
	}
	if !ok {
		t.Fatalf("expected to find prize")
	}
	if found.Name != "Trophy" {
		t.Errorf("prize name: got %q, want %q", found.Name, "Trophy")
	}

	_, ok, err = GetPrizeByID(uuid.New())
	if err != nil {
		t.Fatalf("GetPrizeByID(missing) returned error: %v", err)
	}
	if ok {
		t.Errorf("expected unknown prize to report not found")
	}
}

func TestGetPrizes(t *testing.T) {
	t.Run("with prizes", func(t *testing.T) {
		newTestDB(t)
		makePrize(t, "Medal", 1)
		makePrize(t, "Cup", 2)

		prizes, ok, err := GetPrizes()
		if err != nil {
			t.Fatalf("GetPrizes returned error: %v", err)
		}
		if !ok || len(prizes) != 2 {
			t.Errorf("got ok=%v len=%d, want ok=true len=2", ok, len(prizes))
		}
	})

	t.Run("no prizes", func(t *testing.T) {
		newTestDB(t)
		prizes, ok, err := GetPrizes()
		if err != nil {
			t.Fatalf("GetPrizes returned error: %v", err)
		}
		if ok || len(prizes) != 0 {
			t.Errorf("got ok=%v len=%d, want ok=false len=0", ok, len(prizes))
		}
	})
}

func TestGetPrizeByNameAndQuantity(t *testing.T) {
	newTestDB(t)

	makePrize(t, "Belt", 3)

	found, ok, err := GetPrizeByNameAndQuantity("Belt", 3)
	if err != nil {
		t.Fatalf("GetPrizeByNameAndQuantity returned error: %v", err)
	}
	if !ok || found.Name != "Belt" {
		t.Errorf("expected to find Belt/3, got ok=%v name=%q", ok, found.Name)
	}

	// Same name, different quantity must not match.
	_, ok, err = GetPrizeByNameAndQuantity("Belt", 4)
	if err != nil {
		t.Fatalf("GetPrizeByNameAndQuantity(mismatch) returned error: %v", err)
	}
	if ok {
		t.Errorf("expected quantity mismatch to report not found")
	}
}
