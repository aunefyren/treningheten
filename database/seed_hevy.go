package database

import (
	_ "embed"
	"encoding/json"
	"strings"
	"time"

	"github.com/aunefyren/treningheten/logger"
	"github.com/aunefyren/treningheten/models"

	"github.com/google/uuid"
)

// The official Hevy exercise catalog, exported from the Hevy API. Refresh it from
// scripts/hevyseed and copy the result here to update the built-in catalog.
//
//go:embed data/hevy_templates.json
var hevyTemplatesJSON []byte

// hevyActionNamespace gives seeded Hevy Actions deterministic (v5) UUIDs keyed by their
// template id, so a fresh install produces the same ids every time and re-seeds never
// duplicate.
var hevyActionNamespace = uuid.MustParse("7e9f2c3a-1d4b-4e8a-9c6f-2a5b8d3e1f00")

// hevyOverlapOverrides maps a Hevy template id onto an existing curated Action, so the
// catalog merges into the curated vocabulary instead of duplicating it. Only clean 1:1
// matches belong here; ambiguous ones (e.g. Hevy's single generic "Skiing" vs the
// curated Alpine/Cross-Country split, or "Boxing"/"Climbing" with no curated equivalent)
// are intentionally left out so they seed as standalone Actions.
var hevyOverlapOverrides = map[string]uuid.UUID{
	"AC1BB830": uuid.MustParse("a3fe89d4-bbb9-40af-9bd1-e72ec448705c"), // Running        -> Run
	"D8F7F851": uuid.MustParse("9a9230b0-82d1-4419-9cbe-c8ffe99cb62f"), // Cycling        -> Bicycling
	"B60A678F": uuid.MustParse("21a4666d-d1ee-40a2-802c-f09ef3bce895"), // Swimming       -> Swim
	"33EDD7DB": uuid.MustParse("59211f92-3207-46d0-9e2a-33fe27ead513"), // Walking        -> Walking
	"1C34A172": uuid.MustParse("1ec8f181-475a-4544-8902-676c9dbb3bfc"), // Hiking         -> Hike
	"0222DB42": uuid.MustParse("a460eb20-0fd9-4ca4-af8e-40a880fe6dfd"), // Rowing Machine -> Rowing
	"3303376C": uuid.MustParse("067a3de0-0171-43cc-b95e-99da5e07e546"), // Elliptical Tr. -> Elliptical
	"8C9D2928": uuid.MustParse("6afc4e91-ed77-4f30-a38a-d10ea2aeba94"), // Yoga           -> Yoga
	"EC2510CD": uuid.MustParse("9be08d4e-a5ec-49df-9c5b-718cd5bbf46e"), // Pilates        -> Pilates
	"C6C9B8A0": uuid.MustParse("e3b0849b-6cf0-4f07-a8a1-48ee84cbc83d"), // Plank          -> Plank
}

// SeedHevyActions builds the official Hevy exercise catalog into the Action table from
// the embedded snapshot. The dedup authority is HevyTemplateID (the same key the live
// import path uses), so it is idempotent and never duplicates an Action a sync already
// created. Overlap templates attach their id to the matching curated Action; the rest
// seed as new Actions with deterministic UUIDs. Run it after SeedActions so the curated
// overlap targets already exist.
func SeedHevyActions() {
	var templates []models.HevyExerciseTemplate
	if err := json.Unmarshal(hevyTemplatesJSON, &templates); err != nil {
		logger.Log.Printf("failed to parse embedded Hevy templates: %v", err)
		return
	}

	existing, err := GetAllEnabledActions()
	if err != nil {
		logger.Log.Printf("could not fetch existing actions for Hevy seed: %v", err)
		return
	}

	byTemplateID := make(map[string]bool)
	byID := make(map[uuid.UUID]models.Action)
	for _, a := range existing {
		if a.HevyTemplateID != nil && *a.HevyTemplateID != "" {
			byTemplateID[*a.HevyTemplateID] = true
		}
		byID[a.ID] = a
	}

	seeded, merged := 0, 0
	for _, template := range templates {
		if template.IsCustom || strings.TrimSpace(template.ID) == "" {
			continue
		}

		// Already mapped (seeded before, or auto-created by a live sync) — skip.
		if byTemplateID[template.ID] {
			continue
		}

		// Overlap: attach the template to its curated Action instead of duplicating.
		if curatedID, ok := hevyOverlapOverrides[template.ID]; ok {
			if curated, found := byID[curatedID]; found {
				if curated.HevyTemplateID == nil || *curated.HevyTemplateID == "" {
					tid := template.ID
					curated.HevyTemplateID = &tid
					if _, err := UpdateActionInDB(curated); err != nil {
						logger.Log.Printf("failed to attach Hevy id %s to action %q: %v", template.ID, curated.Name, err)
						continue
					}
					byTemplateID[template.ID] = true
					merged++
				}
				continue
			}
			// Curated target missing — fall through and seed it standalone.
			logger.Log.Printf("Hevy overlap target %s missing for template %s (%s); seeding standalone", curatedID, template.ID, template.Title)
		}

		// New catalog Action with a deterministic UUID.
		action := template.ToAction()
		action.ID = uuid.NewSHA1(hevyActionNamespace, []byte(template.ID))
		action.CreatedAt = time.Now()
		if _, err := CreateActionInDB(action); err != nil {
			logger.Log.Printf("failed to seed Hevy action %q: %v", action.Name, err)
			continue
		}
		byTemplateID[template.ID] = true
		seeded++
	}

	logger.Log.Printf("Hevy catalog seed: %d new action(s), %d merged into curated action(s)", seeded, merged)
}
