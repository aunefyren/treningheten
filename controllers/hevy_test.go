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

func TestHevyProfileURL(t *testing.T) {
	for raw, want := range map[string]string{
		"https://hevy.com/user/someone": "https://hevy.com/user/someone",
		"  https://hevy.com/user/x  ":   "https://hevy.com/user/x",
	} {
		if got, ok := hevyProfileURL(raw); !ok || got != want {
			t.Errorf("hevyProfileURL(%q) = %q, %v; want %q, true", raw, got, ok, want)
		}
	}
	for _, raw := range []string{"", "javascript:alert(1)", "http://hevy.com/user/x", "https:///path", "hevy.com/user/x", "');alert(1);//"} {
		if got, ok := hevyProfileURL(raw); ok {
			t.Errorf("hevyProfileURL(%q) = %q, true; want rejected", raw, got)
		}
	}
}
