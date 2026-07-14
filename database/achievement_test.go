package database

import (
	"testing"
	"time"

	"github.com/aunefyren/treningheten/models"

	"github.com/google/uuid"
)

// makeAchievement inserts an achievement and returns it. When enabled is false the row is
// forced disabled after insert (the `default: true` tag would otherwise keep it enabled).
func makeAchievement(t *testing.T, name, category string, order int, enabled bool) models.Achievement {
	t.Helper()
	achievement := models.Achievement{Name: name, Description: "desc", Category: category, AchievementOrder: order, Enabled: enabled}
	achievement.ID = uuid.New()
	created, err := RegisterAchievementInDB(achievement)
	if err != nil {
		t.Fatalf("RegisterAchievementInDB returned error: %v", err)
	}
	if !enabled {
		disableRow(t, &models.Achievement{}, achievement.ID)
	}
	return created
}

// makeDelegation grants an achievement to a user and returns the delegation.
func makeDelegation(t *testing.T, userID, achievementID uuid.UUID, seen bool) models.AchievementDelegation {
	t.Helper()
	delegation := models.AchievementDelegation{UserID: userID, AchievementID: achievementID, GivenAt: time.Now(), Enabled: true, Seen: seen}
	delegation.ID = uuid.New()
	created, err := RegisterAchievementDelegationInDB(delegation)
	if err != nil {
		t.Fatalf("RegisterAchievementDelegationInDB returned error: %v", err)
	}
	return created
}

func TestAchievementCRUDAndExistence(t *testing.T) {
	newTestDB(t)

	exists, err := CheckIfAchievementsExistsInDB()
	if err != nil {
		t.Fatalf("CheckIfAchievementsExistsInDB returned error: %v", err)
	}
	if exists {
		t.Errorf("expected no achievements initially")
	}

	enabled := makeAchievement(t, "First", "cat", 1, true)
	disabled := makeAchievement(t, "Hidden", "cat", 2, false)

	exists, err = CheckIfAchievementsExistsInDB()
	if err != nil {
		t.Fatalf("CheckIfAchievementsExistsInDB returned error: %v", err)
	}
	if !exists {
		t.Errorf("expected achievements to exist after registering")
	}

	// GetAllAchievements includes the disabled one; GetAllEnabledAchievements does not.
	all, err := GetAllAchievements()
	if err != nil {
		t.Fatalf("GetAllAchievements returned error: %v", err)
	}
	if len(all) != 2 {
		t.Errorf("GetAllAchievements: got %d, want 2", len(all))
	}
	enabledOnly, err := GetAllEnabledAchievements()
	if err != nil {
		t.Fatalf("GetAllEnabledAchievements returned error: %v", err)
	}
	if len(enabledOnly) != 1 || enabledOnly[0].ID != enabled.ID {
		t.Errorf("GetAllEnabledAchievements: got %d, want 1 (enabled only)", len(enabledOnly))
	}

	// GetAchievementByID: enabled hit, disabled/unknown → empty record.
	byID, err := GetAchievementByID(enabled.ID)
	if err != nil {
		t.Fatalf("GetAchievementByID returned error: %v", err)
	}
	if byID.ID != enabled.ID {
		t.Errorf("GetAchievementByID returned wrong achievement")
	}
	hiddenLookup, err := GetAchievementByID(disabled.ID)
	if err != nil {
		t.Fatalf("GetAchievementByID(disabled) returned error: %v", err)
	}
	if hiddenLookup.ID != uuid.Nil {
		t.Errorf("expected empty record for disabled achievement, got %v", hiddenLookup.ID)
	}

	// SaveAchievementInDB persists an edit.
	enabled.Name = "Renamed"
	if _, err := SaveAchievementInDB(enabled); err != nil {
		t.Fatalf("SaveAchievementInDB returned error: %v", err)
	}
	reloaded, _ := GetAchievementByID(enabled.ID)
	if reloaded.Name != "Renamed" {
		t.Errorf("achievement name: got %q, want %q", reloaded.Name, "Renamed")
	}
}

func TestGetAllEnabledAchievementsOrdering(t *testing.T) {
	newTestDB(t)

	// Ordering is: category DESC, then achievement_order ASC.
	makeAchievement(t, "A2", "A", 2, true)
	makeAchievement(t, "A1", "A", 1, true)
	makeAchievement(t, "B1", "B", 1, true)

	got, err := GetAllEnabledAchievements()
	if err != nil {
		t.Fatalf("GetAllEnabledAchievements returned error: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("got %d achievements, want 3", len(got))
	}
	wantNames := []string{"B1", "A1", "A2"}
	for i, name := range wantNames {
		if got[i].Name != name {
			t.Errorf("position %d: got %q, want %q", i, got[i].Name, name)
		}
	}
}

func TestAchievementDelegations(t *testing.T) {
	newTestDB(t)

	user := makeTestUser(t, "deleg@example.com", nil)
	ach1 := makeAchievement(t, "Ach1", "cat", 1, true)
	ach2 := makeAchievement(t, "Ach2", "cat", 2, true)
	ach3 := makeAchievement(t, "Ach3Disabled", "cat", 3, false)

	// ach1 granted twice, ach2 once, ach3 once (but ach3 is disabled → excluded by joins).
	deleg1 := makeDelegation(t, user.ID, ach1.ID, false)
	makeDelegation(t, user.ID, ach1.ID, false)
	makeDelegation(t, user.ID, ach2.ID, false)
	makeDelegation(t, user.ID, ach3.ID, false)

	delegated, ok, err := GetDelegatedAchievementsByUserID(user.ID)
	if err != nil {
		t.Fatalf("GetDelegatedAchievementsByUserID returned error: %v", err)
	}
	if !ok || len(delegated) != 3 {
		t.Errorf("delegated: got ok=%v len=%d, want ok=true len=3 (disabled achievement excluded)", ok, len(delegated))
	}

	distinct, ok, err := GetDistinctDelegatedAchievementsByUserID(user.ID)
	if err != nil {
		t.Fatalf("GetDistinctDelegatedAchievementsByUserID returned error: %v", err)
	}
	if !ok || len(distinct) != 2 {
		t.Errorf("distinct: got ok=%v len=%d, want ok=true len=2", ok, len(distinct))
	}

	forAch1, err := GetAchievementDelegationByAchievementIDAndUserID(user.ID, ach1.ID)
	if err != nil {
		t.Fatalf("GetAchievementDelegationByAchievementIDAndUserID returned error: %v", err)
	}
	if len(forAch1) != 2 {
		t.Errorf("ach1 delegations: got %d, want 2", len(forAch1))
	}

	// Single delegation lookup, scoped by user.
	one, found, err := GetAchievementDelegationByIDAndUserID(deleg1.ID, user.ID)
	if err != nil {
		t.Fatalf("GetAchievementDelegationByIDAndUserID returned error: %v", err)
	}
	if !found || one.ID != deleg1.ID {
		t.Errorf("expected to find delegation for owner")
	}
	_, found, err = GetAchievementDelegationByIDAndUserID(deleg1.ID, uuid.New())
	if err != nil {
		t.Fatalf("GetAchievementDelegationByIDAndUserID(stranger) returned error: %v", err)
	}
	if found {
		t.Errorf("expected not found for a different user")
	}

	// VerifyIfAchievedByUser: true for a granted+enabled achievement, false for the disabled one.
	achieved, err := VerifyIfAchievedByUser(ach1.ID, user.ID)
	if err != nil {
		t.Fatalf("VerifyIfAchievedByUser returned error: %v", err)
	}
	if !achieved {
		t.Errorf("expected ach1 to be achieved")
	}
	achievedDisabled, err := VerifyIfAchievedByUser(ach3.ID, user.ID)
	if err != nil {
		t.Fatalf("VerifyIfAchievedByUser(disabled) returned error: %v", err)
	}
	if achievedDisabled {
		t.Errorf("expected disabled achievement to not count as achieved")
	}
}

func TestSetAchievementsToSeenForUser(t *testing.T) {
	newTestDB(t)

	user := makeTestUser(t, "seen@example.com", nil)
	ach := makeAchievement(t, "SeenAch", "cat", 1, true)

	makeDelegation(t, user.ID, ach.ID, false)
	makeDelegation(t, user.ID, ach.ID, false)

	updates, err := SetAchievementsToSeenForUser(user.ID)
	if err != nil {
		t.Fatalf("SetAchievementsToSeenForUser returned error: %v", err)
	}
	if updates != 2 {
		t.Errorf("first run: marked %d seen, want 2", updates)
	}

	// Running again marks nothing (all already seen).
	updates, err = SetAchievementsToSeenForUser(user.ID)
	if err != nil {
		t.Fatalf("SetAchievementsToSeenForUser (second) returned error: %v", err)
	}
	if updates != 0 {
		t.Errorf("second run: marked %d seen, want 0", updates)
	}
}
