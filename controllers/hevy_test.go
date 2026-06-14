package controllers

import "testing"

func TestHevyTypeToActionType(t *testing.T) {
	cases := map[string]string{
		// Strength / rep-based → lifting
		"weight_reps":         "lifting",
		"reps_only":           "lifting",
		"bodyweight_reps":     "lifting",
		"weighted_bodyweight": "lifting",
		"assisted_bodyweight": "lifting",
		// Time-based → timing
		"duration": "timing",
		// Distance-based → moving
		"distance_duration":     "moving",
		"weight_distance":       "moving",
		"short_distance_weight": "moving",
		// Unknown / empty default to lifting (Hevy is strength-centric)
		"":              "lifting",
		"something_new": "lifting",
		// Case/whitespace insensitive
		"  Duration ": "timing",
	}

	for input, expected := range cases {
		if got := hevyTypeToActionType(input); got != expected {
			t.Errorf("hevyTypeToActionType(%q) = %q, want %q", input, got, expected)
		}
	}
}
