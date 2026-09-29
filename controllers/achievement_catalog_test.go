package controllers

import (
	"testing"

	"github.com/aunefyren/treningheten/database"
	"github.com/aunefyren/treningheten/models"

	"github.com/google/uuid"
)

func TestValidateAchievementsSeedsTheCatalogIdempotently(t *testing.T) {
	newControllerTestDB(t)

	if exists, err := CheckIfAchievementsExist(); err != nil || exists {
		t.Fatalf("fresh DB: exists = %v, err = %v; want false, nil", exists, err)
	}
	if err := ValidateAchievements(); err != nil {
		t.Fatalf("ValidateAchievements: %v", err)
	}
	var first int64
	database.Instance.Model(&models.Achievement{}).Count(&first)
	if first < 20 {
		t.Fatalf("achievements seeded = %d, want the full catalog", first)
	}

	// Running it again (every startup does) updates in place rather than duplicating.
	if err := ValidateAchievements(); err != nil {
		t.Fatalf("second ValidateAchievements: %v", err)
	}
	var second int64
	database.Instance.Model(&models.Achievement{}).Count(&second)
	if second != first {
		t.Errorf("achievements after re-run = %d, want %d", second, first)
	}
	if exists, err := CheckIfAchievementsExist(); err != nil || !exists {
		t.Errorf("after seeding: exists = %v, err = %v", exists, err)
	}
}

func TestMergeAchievements(t *testing.T) {
	shared, onlyOld, onlyNew := uuid.New(), uuid.New(), uuid.New()

	old := []models.Achievement{
		{Name: "Old name", Category: "Old", AchievementOrder: 1},
		{Name: "Retired"},
	}
	old[0].ID, old[1].ID = shared, onlyOld
	old[0].CreatedAt = old[0].CreatedAt.AddDate(2020, 0, 0)

	fresh := []models.Achievement{{Name: "New name", Category: "New", AchievementOrder: 5}, {Name: "Brand new"}}
	fresh[0].ID, fresh[1].ID = shared, onlyNew

	merged, err := MergeAchievements(old, fresh)
	if err != nil {
		t.Fatal(err)
	}
	if len(merged) != 3 {
		t.Fatalf("merged = %d, want 3 (updated, added, kept)", len(merged))
	}
	byID := map[uuid.UUID]models.Achievement{}
	for _, achievement := range merged {
		byID[achievement.ID] = achievement
	}
	if updated := byID[shared]; updated.Name != "New name" || updated.AchievementOrder != 5 || updated.CreatedAt != old[0].CreatedAt {
		t.Errorf("shared achievement = %+v, want new fields on the old row", updated)
	}
	if _, kept := byID[onlyOld]; !kept {
		t.Error("an achievement only in the DB was dropped")
	}
	if _, added := byID[onlyNew]; !added {
		t.Error("a new achievement was not added")
	}
}
