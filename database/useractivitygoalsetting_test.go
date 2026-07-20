package database

import (
	"testing"

	"github.com/aunefyren/treningheten/models"

	"github.com/google/uuid"
)

func TestGetActivityTypeActions(t *testing.T) {
	newTestDB(t)

	// Two activity-type actions (with a Strava name) and two that must be excluded.
	run := models.Action{Name: "Run", Type: "cardio", Enabled: true, StravaName: "Run"}
	run.ID = uuid.New()
	insertRow(t, &run)
	walk := models.Action{Name: "Walk", Type: "cardio", Enabled: true, StravaName: "Walk"}
	walk.ID = uuid.New()
	insertRow(t, &walk)
	bench := models.Action{Name: "Bench press", Type: "strength", Enabled: true} // no strava_name
	bench.ID = uuid.New()
	insertRow(t, &bench)
	disabled := models.Action{Name: "Old", Type: "cardio", Enabled: false, StravaName: "Old"}
	disabled.ID = uuid.New()
	insertRow(t, &disabled)
	// Action.Enabled carries a default:true tag, so GORM ignores the false zero value on
	// Create — force the column off explicitly to exercise the enabled filter.
	if err := Instance.Model(&models.Action{}).Where("id = ?", disabled.ID).Update("enabled", false).Error; err != nil {
		t.Fatalf("failed to disable action: %v", err)
	}

	actions, err := GetActivityTypeActions()
	if err != nil {
		t.Fatalf("GetActivityTypeActions returned error: %v", err)
	}
	if len(actions) != 2 {
		t.Fatalf("got %d activity-type actions, want 2 (enabled + strava_name only)", len(actions))
	}
	// Ordered by name asc: Run, Walk.
	if actions[0].Name != "Run" || actions[1].Name != "Walk" {
		t.Errorf("unexpected order/content: %q, %q", actions[0].Name, actions[1].Name)
	}
}

func TestUpsertAndGetActivityGoalSettings(t *testing.T) {
	newTestDB(t)
	user := makeTestUser(t, "goalset@example.com", nil)
	run := makeAction(t, "Run", "cardio")
	walk := makeAction(t, "Walk", "cardio")

	// No settings yet → nil, no error.
	none, err := GetActivityGoalSettingForUserAndAction(user.ID, run.ID)
	if err != nil {
		t.Fatalf("GetActivityGoalSettingForUserAndAction returned error: %v", err)
	}
	if none != nil {
		t.Errorf("expected no setting, got %+v", none)
	}

	// Create a "doesn't count" row — the false must persist despite the default:true tag.
	if err := UpsertActivityGoalSettingInDB(user.ID, walk.ID, false); err != nil {
		t.Fatalf("upsert (create) returned error: %v", err)
	}
	got, err := GetActivityGoalSettingForUserAndAction(user.ID, walk.ID)
	if err != nil || got == nil {
		t.Fatalf("expected setting, err=%v got=%v", err, got)
	}
	if got.CountsTowardGoal {
		t.Errorf("CountsTowardGoal = true, want false (create must persist false)")
	}

	// Upsert the same (user, action) updates in place rather than inserting a duplicate.
	if err := UpsertActivityGoalSettingInDB(user.ID, walk.ID, true); err != nil {
		t.Fatalf("upsert (update) returned error: %v", err)
	}
	updated, err := GetActivityGoalSettingForUserAndAction(user.ID, walk.ID)
	if err != nil || updated == nil {
		t.Fatalf("expected setting after update, err=%v got=%v", err, updated)
	}
	if !updated.CountsTowardGoal {
		t.Errorf("CountsTowardGoal = false, want true after update")
	}

	// A second action for the same user; get-for-user returns both.
	if err := UpsertActivityGoalSettingInDB(user.ID, run.ID, false); err != nil {
		t.Fatalf("upsert second action returned error: %v", err)
	}
	all, err := GetActivityGoalSettingsForUserID(user.ID)
	if err != nil {
		t.Fatalf("GetActivityGoalSettingsForUserID returned error: %v", err)
	}
	if len(all) != 2 {
		t.Errorf("got %d settings, want 2", len(all))
	}
}

func TestMigrateStravaIgnoreWalksToGoalSettings(t *testing.T) {
	newTestDB(t) // runs Migrate() (and this backfill) once on empty data — a no-op

	ignorer := makeTestUser(t, "ignorer@example.com", nil)
	keeper := makeTestUser(t, "keeper@example.com", nil)
	// Force the legacy column on for the ignorer (explicit update sidesteps the default:false).
	if err := Instance.Model(&models.User{}).Where("id = ?", ignorer.ID).Update("strava_walks", true).Error; err != nil {
		t.Fatalf("failed to set strava_walks: %v", err)
	}

	walking := models.Action{Name: "Walking", Type: "cardio", Enabled: true, StravaName: "Walk"}
	walking.ID = uuid.New()
	insertRow(t, &walking)

	migrateStravaIgnoreWalksToGoalSettings()

	// The ignorer gets a Walking → doesn't-count setting; the keeper gets nothing.
	ign, err := GetActivityGoalSettingForUserAndAction(ignorer.ID, walking.ID)
	if err != nil || ign == nil {
		t.Fatalf("expected migrated setting for ignorer, err=%v got=%v", err, ign)
	}
	if ign.CountsTowardGoal {
		t.Errorf("migrated Walking setting should not count")
	}
	keep, err := GetActivityGoalSettingForUserAndAction(keeper.ID, walking.ID)
	if err != nil {
		t.Fatalf("GetActivityGoalSettingForUserAndAction(keeper) error: %v", err)
	}
	if keep != nil {
		t.Errorf("keeper should have no setting, got %+v", keep)
	}

	// The flag is cleared, so a re-run is a no-op (self-limiting).
	var reloaded models.User
	if err := Instance.Where("id = ?", ignorer.ID).First(&reloaded).Error; err != nil {
		t.Fatalf("failed to reload ignorer: %v", err)
	}
	if reloaded.StravaIgnoreWalks == nil || *reloaded.StravaIgnoreWalks {
		t.Errorf("strava_walks not cleared for migrated user: %v", reloaded.StravaIgnoreWalks)
	}
}
