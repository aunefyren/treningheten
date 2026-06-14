package database

import (
	"encoding/json"
	"testing"

	"github.com/aunefyren/treningheten/models"

	"github.com/google/uuid"
)

// loadEmbeddedHevyTemplates parses the embedded catalog or fails the test.
func loadEmbeddedHevyTemplates(t *testing.T) []models.HevyExerciseTemplate {
	t.Helper()
	var templates []models.HevyExerciseTemplate
	if err := json.Unmarshal(hevyTemplatesJSON, &templates); err != nil {
		t.Fatalf("embedded Hevy templates do not parse: %v", err)
	}
	if len(templates) == 0 {
		t.Fatal("embedded Hevy templates are empty")
	}
	return templates
}

// Every override key must reference a template that actually exists in the catalog,
// otherwise the merge silently never fires.
func TestHevyOverlapOverridesExistInCatalog(t *testing.T) {
	templates := loadEmbeddedHevyTemplates(t)

	catalogIDs := make(map[string]bool, len(templates))
	for _, tpl := range templates {
		catalogIDs[tpl.ID] = true
	}

	for hevyID := range hevyOverlapOverrides {
		if !catalogIDs[hevyID] {
			t.Errorf("override references Hevy template id %q which is not in the catalog", hevyID)
		}
	}
}

// The deterministic ids of seeded (non-override) Actions must be unique and must not
// collide with a curated overlap target — a collision would drop or overwrite an Action.
func TestHevySeedActionIDsAreUniqueAndDistinctFromOverrides(t *testing.T) {
	templates := loadEmbeddedHevyTemplates(t)

	overrideTargets := make(map[uuid.UUID]bool, len(hevyOverlapOverrides))
	for _, target := range hevyOverlapOverrides {
		overrideTargets[target] = true
	}

	seen := make(map[uuid.UUID]string)
	for _, tpl := range templates {
		if tpl.IsCustom || tpl.ID == "" {
			continue
		}
		if _, isOverride := hevyOverlapOverrides[tpl.ID]; isOverride {
			continue
		}

		id := uuid.NewSHA1(hevyActionNamespace, []byte(tpl.ID))
		if prev, dup := seen[id]; dup {
			t.Errorf("deterministic id collision: %q and %q both map to %s", prev, tpl.ID, id)
		}
		seen[id] = tpl.ID

		if overrideTargets[id] {
			t.Errorf("seeded action id for template %q collides with a curated override target %s", tpl.ID, id)
		}
	}
}

// ToAction should carry the template id through and classify the type, so the seeder
// and live import produce the same dedup key.
func TestHevyTemplateToActionCarriesID(t *testing.T) {
	tpl := models.HevyExerciseTemplate{ID: "ABCD1234", Title: "Test Lift", Type: "weight_reps", PrimaryMuscleGroup: "chest"}
	action := tpl.ToAction()

	if action.HevyTemplateID == nil || *action.HevyTemplateID != "ABCD1234" {
		t.Errorf("ToAction dropped the template id: %v", action.HevyTemplateID)
	}
	if action.Type != "lifting" {
		t.Errorf("ToAction type = %q, want lifting", action.Type)
	}
	if action.Name != "Test Lift" || action.NorwegianName != "Test Lift" {
		t.Errorf("ToAction names = %q/%q, want Test Lift", action.Name, action.NorwegianName)
	}
}
