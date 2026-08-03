package models

import "testing"

// TestHevyActionType covers mapping Hevy's template type onto the local action-type
// vocabulary. It is shared by the live import and the catalog seeder, so a change here
// silently reclassifies every exercise in the catalog.
func TestHevyActionType(t *testing.T) {
	tests := map[string]string{
		"duration":              "timing",
		"distance_duration":     "moving",
		"weight_distance":       "moving",
		"short_distance_weight": "moving",

		// Everything else — including the unrecognised — is lifting.
		"weight_reps":         "lifting",
		"reps_only":           "lifting",
		"bodyweight_reps":     "lifting",
		"weighted_bodyweight": "lifting",
		"assisted_bodyweight": "lifting",
		"":                    "lifting",
		"something_new":       "lifting",

		// Input is normalised before matching.
		"DURATION":          "timing",
		"  duration  ":      "timing",
		"Distance_Duration": "moving",
	}

	for hevyType, want := range tests {
		if got := HevyActionType(hevyType); got != want {
			t.Errorf("HevyActionType(%q) = %q, want %q", hevyType, got, want)
		}
	}
}

// TestHevyTemplateToAction covers building a catalog Action from a Hevy template: fields are
// trimmed, the type is mapped, and the template id is attached (it is the dedup key both the
// seeder and the live import rely on). The ID is deliberately left for the caller to set.
func TestHevyTemplateToAction(t *testing.T) {
	template := HevyExerciseTemplate{
		ID:                 "  AC1BB830 ",
		Title:              "  Bench Press (Barbell)  ",
		Type:               "weight_reps",
		PrimaryMuscleGroup: " chest ",
	}

	action := template.ToAction()

	if !action.Enabled {
		t.Errorf("action is not enabled")
	}
	if action.Name != "Bench Press (Barbell)" {
		t.Errorf("name = %q, want the trimmed title", action.Name)
	}
	if action.NorwegianName != action.Name {
		t.Errorf("norwegian name = %q, want it to mirror the title", action.NorwegianName)
	}
	if action.Type != "lifting" {
		t.Errorf("type = %q, want lifting", action.Type)
	}
	if action.BodyPart != "chest" {
		t.Errorf("body part = %q, want the trimmed muscle group", action.BodyPart)
	}
	if action.HevyTemplateID == nil || *action.HevyTemplateID != "AC1BB830" {
		t.Errorf("hevy template id = %v, want the trimmed id", action.HevyTemplateID)
	}
	if action.HasLogo {
		t.Errorf("has logo = true, want false for an imported catalog entry")
	}
	if action.StravaName != "" {
		t.Errorf("strava name = %q, want empty — a Hevy template carries no Strava mapping", action.StravaName)
	}
	// The ID is the caller's to assign (random on import, deterministic in the seeder).
	if action.ID.String() != "00000000-0000-0000-0000-000000000000" {
		t.Errorf("ToAction assigned an ID (%v); that is the caller's job", action.ID)
	}
}

// TestHevyTemplateToActionCardio covers a duration/distance template mapping onto a
// non-lifting action type, so a Hevy treadmill run isn't filed as a lift.
func TestHevyTemplateToActionCardio(t *testing.T) {
	template := HevyExerciseTemplate{ID: "33EDD7DB", Title: "Walking", Type: "distance_duration"}

	if action := template.ToAction(); action.Type != "moving" {
		t.Errorf("type = %q, want moving", action.Type)
	}
}
