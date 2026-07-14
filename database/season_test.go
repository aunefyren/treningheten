package database

import (
	"testing"
	"time"

	"github.com/aunefyren/treningheten/models"

	"github.com/google/uuid"
)

// makeSeason inserts a season (Prize association omitted) and returns it. Enabled carries a
// `default: true` tag which GORM would flip back to true on a zero value, so it is forced via
// an explicit Update. Shared by the goal and wheelview tests.
func makeSeason(t *testing.T, name string, start, end time.Time, enabled bool) models.Season {
	t.Helper()
	season := models.Season{Name: name, Start: start, End: end, Enabled: enabled}
	season.ID = uuid.New()
	if err := Instance.Omit("Prize").Create(&season).Error; err != nil {
		t.Fatalf("failed to seed season: %v", err)
	}
	if err := Instance.Model(&models.Season{}).Where("id = ?", season.ID).Update("enabled", enabled).Error; err != nil {
		t.Fatalf("failed to set season enabled: %v", err)
	}
	return season
}

func TestCreateSeasonInDBAndGetSeasonByID(t *testing.T) {
	newTestDB(t)

	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2024, 3, 31, 0, 0, 0, 0, time.UTC)
	season := models.Season{Name: "Winter", Start: start, End: end, Enabled: true}
	season.ID = uuid.New()

	if err := CreateSeasonInDB(season); err != nil {
		t.Fatalf("CreateSeasonInDB returned error: %v", err)
	}

	found, err := GetSeasonByID(season.ID)
	if err != nil {
		t.Fatalf("GetSeasonByID returned error: %v", err)
	}
	if found == nil {
		t.Fatalf("expected to find season, got nil")
	}
	if found.Name != "Winter" {
		t.Errorf("season name: got %q, want %q", found.Name, "Winter")
	}

	missing, err := GetSeasonByID(uuid.New())
	if err != nil {
		t.Fatalf("GetSeasonByID(missing) returned error: %v", err)
	}
	if missing != nil {
		t.Errorf("expected nil for unknown season, got %v", missing)
	}
}

func TestGetSeasonByIDDisabled(t *testing.T) {
	newTestDB(t)

	season := makeSeason(t, "Hidden", time.Now(), time.Now().Add(24*time.Hour), false)

	found, err := GetSeasonByID(season.ID)
	if err != nil {
		t.Fatalf("GetSeasonByID returned error: %v", err)
	}
	if found != nil {
		t.Errorf("disabled season should not be returned, got %v", found)
	}
}

func TestVerifyUniqueSeasonName(t *testing.T) {
	newTestDB(t)

	makeSeason(t, "Taken", time.Now(), time.Now().Add(24*time.Hour), true)

	unique, err := VerifyUniqueSeasonName("Fresh")
	if err != nil {
		t.Fatalf("VerifyUniqueSeasonName(fresh) returned error: %v", err)
	}
	if !unique {
		t.Errorf("expected unused name to be unique")
	}

	taken, err := VerifyUniqueSeasonName("Taken")
	if err != nil {
		t.Fatalf("VerifyUniqueSeasonName(taken) returned error: %v", err)
	}
	if taken {
		t.Errorf("expected used name to not be unique")
	}
}

func TestGetAllEnabledSeasons(t *testing.T) {
	newTestDB(t)

	early := makeSeason(t, "Early", time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC), true)
	late := makeSeason(t, "Late", time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC), time.Date(2024, 7, 1, 0, 0, 0, 0, time.UTC), true)
	makeSeason(t, "Disabled", time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC), time.Date(2024, 4, 1, 0, 0, 0, 0, time.UTC), false)

	seasons, err := GetAllEnabledSeasons()
	if err != nil {
		t.Fatalf("GetAllEnabledSeasons returned error: %v", err)
	}
	if len(seasons) != 2 {
		t.Fatalf("got %d enabled seasons, want 2", len(seasons))
	}
	// Ordered by start desc: late first, early second.
	if seasons[0].ID != late.ID || seasons[1].ID != early.ID {
		t.Errorf("seasons not ordered by start desc: got %q then %q", seasons[0].Name, seasons[1].Name)
	}
}
