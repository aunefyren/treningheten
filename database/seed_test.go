package database

import (
	"testing"

	"github.com/aunefyren/treningheten/models"

	"github.com/google/uuid"
)

func TestSeedActions(t *testing.T) {
	newTestDB(t)

	SeedActions()

	actions, err := GetAllEnabledActions()
	if err != nil {
		t.Fatalf("GetAllEnabledActions returned error: %v", err)
	}
	if len(actions) != len(defaultActions) {
		t.Fatalf("seeded %d actions, want %d", len(actions), len(defaultActions))
	}

	// A curated action is looked up by its stable name.
	run, err := GetActionByName("Run")
	if err != nil {
		t.Fatalf("expected seeded Run action: %v", err)
	}
	if run.StravaName != "Run" {
		t.Errorf("Run strava name: got %q, want %q", run.StravaName, "Run")
	}

	// Re-running is idempotent (matches on ID) — the count does not grow.
	SeedActions()
	again, err := GetAllEnabledActions()
	if err != nil {
		t.Fatalf("GetAllEnabledActions returned error: %v", err)
	}
	if len(again) != len(defaultActions) {
		t.Errorf("after re-seed: %d actions, want %d (idempotent)", len(again), len(defaultActions))
	}
}

func TestSeedOAuthClients(t *testing.T) {
	newTestDB(t)

	SeedOAuthClients()

	client, err := GetOAuthClientByClientID(models.FirstPartyClientID)
	if err != nil {
		t.Fatalf("expected first-party OAuth client after seeding: %v", err)
	}
	if !client.FirstParty || !client.Public {
		t.Errorf("first-party client flags: firstParty=%v public=%v", client.FirstParty, client.Public)
	}
	firstID := client.ID

	// Idempotent: a second run leaves the same single client in place.
	SeedOAuthClients()
	reloaded, err := GetOAuthClientByClientID(models.FirstPartyClientID)
	if err != nil {
		t.Fatalf("GetOAuthClientByClientID returned error: %v", err)
	}
	if reloaded.ID != firstID {
		t.Errorf("re-seed created a new client: %v != %v", reloaded.ID, firstID)
	}
}

func TestSeedHevyActions(t *testing.T) {
	newTestDB(t)

	// Pre-seed the curated Run action (an overlap target) without a Hevy id, so the
	// merge branch attaches template "AC1BB830" to it rather than seeding a duplicate.
	run := models.Action{Name: "Run", Type: "moving", StravaName: "Run", Enabled: true}
	run.ID = uuid.MustParse("a3fe89d4-bbb9-40af-9bd1-e72ec448705c")
	if _, err := CreateActionInDB(run); err != nil {
		t.Fatalf("failed to seed curated Run action: %v", err)
	}

	SeedHevyActions()

	// The curated Run action now carries the Hevy running template id (merge path).
	mergedRun, err := GetActionByID(run.ID)
	if err != nil {
		t.Fatalf("GetActionByID(run) returned error: %v", err)
	}
	if mergedRun.HevyTemplateID == nil || *mergedRun.HevyTemplateID != "AC1BB830" {
		t.Errorf("expected Run to be merged with Hevy id AC1BB830, got %v", mergedRun.HevyTemplateID)
	}

	// Standalone catalog actions were seeded too.
	after, err := GetAllEnabledActions()
	if err != nil {
		t.Fatalf("GetAllEnabledActions returned error: %v", err)
	}
	if len(after) <= 1 {
		t.Fatalf("expected standalone Hevy actions to be seeded, got %d total", len(after))
	}

	// Re-running is idempotent (dedup on HevyTemplateID) — the count does not grow.
	SeedHevyActions()
	again, err := GetAllEnabledActions()
	if err != nil {
		t.Fatalf("GetAllEnabledActions returned error: %v", err)
	}
	if len(again) != len(after) {
		t.Errorf("after re-seed: %d actions, want %d (idempotent)", len(again), len(after))
	}
}
