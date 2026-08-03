package models

import (
	"encoding/json"
	"testing"
)

// TestIsValidTag covers the controlled vocabulary gate applied to user-supplied tags.
func TestIsValidTag(t *testing.T) {
	for _, tag := range ValidTags {
		if !IsValidTag(tag) {
			t.Errorf("IsValidTag(%q) = false for a tag in ValidTags", tag)
		}
	}

	for _, tag := range []string{"", "Race", "RACE", " race", "race ", "long_run", "unknown", "commuter"} {
		if IsValidTag(tag) {
			t.Errorf("IsValidTag(%q) = true, want false", tag)
		}
	}
}

// TestIsStravaManagedTag covers the split that decides which tags a Strava sync may
// overwrite. Getting this wrong either erases a user's manual tags on every sync or lets
// Strava-derived tags go stale, so the two sets are asserted explicitly.
func TestIsStravaManagedTag(t *testing.T) {
	managed := map[string]bool{
		TagRace: true, TagLongRun: true, TagWorkout: true, TagCommute: true,
	}

	for _, tag := range ValidTags {
		if got := IsStravaManagedTag(tag); got != managed[tag] {
			t.Errorf("IsStravaManagedTag(%q) = %v, want %v", tag, got, managed[tag])
		}
	}

	// Every Strava-managed tag must itself be a valid tag.
	for _, tag := range StravaManagedTags {
		if !IsValidTag(tag) {
			t.Errorf("Strava-managed tag %q is not in the valid vocabulary", tag)
		}
	}

	if IsStravaManagedTag("not-a-tag") {
		t.Errorf("IsStravaManagedTag accepted an unknown tag")
	}
}

// TestTagListValueScanRoundTrip covers the JSON column mapping: what Value writes must be
// what Scan reads back, for both the driver's []byte and string forms.
func TestTagListValueScanRoundTrip(t *testing.T) {
	original := TagList{TagRace, TagCommute}

	value, err := original.Value()
	if err != nil {
		t.Fatalf("Value returned error: %v", err)
	}
	encoded, ok := value.([]byte)
	if !ok {
		t.Fatalf("Value returned %T, want []byte", value)
	}

	var fromBytes TagList
	if err := fromBytes.Scan(encoded); err != nil {
		t.Fatalf("Scan([]byte) returned error: %v", err)
	}
	if len(fromBytes) != 2 || fromBytes[0] != TagRace || fromBytes[1] != TagCommute {
		t.Errorf("round-tripped %v, want %v", fromBytes, original)
	}

	// Some drivers hand back a string instead.
	var fromString TagList
	if err := fromString.Scan(string(encoded)); err != nil {
		t.Fatalf("Scan(string) returned error: %v", err)
	}
	if len(fromString) != 2 {
		t.Errorf("Scan(string) produced %v, want %v", fromString, original)
	}
}

// TestTagListValueNil covers that an unset tag list stores as SQL NULL rather than an empty
// JSON array, and that an explicitly empty list is stored as such.
func TestTagListValueNil(t *testing.T) {
	var unset TagList
	value, err := unset.Value()
	if err != nil {
		t.Fatalf("Value returned error: %v", err)
	}
	if value != nil {
		t.Errorf("Value of a nil TagList = %v, want nil", value)
	}

	empty := TagList{}
	value, err = empty.Value()
	if err != nil {
		t.Fatalf("Value returned error: %v", err)
	}
	if string(value.([]byte)) != "[]" {
		t.Errorf("Value of an empty TagList = %s, want []", value)
	}
}

// TestTagListScanEdgeCases covers the column values Scan has to survive: NULL, an empty
// column, the literal "null", a wrong type, and malformed JSON.
func TestTagListScanEdgeCases(t *testing.T) {
	nulls := []interface{}{nil, []byte(nil), []byte(""), []byte("null"), "null", ""}
	for _, value := range nulls {
		tags := TagList{TagRace}
		if err := tags.Scan(value); err != nil {
			t.Errorf("Scan(%#v) returned error: %v", value, err)
		}
		if len(tags) != 0 {
			t.Errorf("Scan(%#v) left %v, want an empty list", value, tags)
		}
	}

	var tags TagList
	if err := tags.Scan(42); err == nil {
		t.Errorf("Scan accepted an int column value")
	}
	if err := tags.Scan([]byte("{not json")); err == nil {
		t.Errorf("Scan accepted malformed JSON")
	}
}

// TestTagListJSONShape covers that a TagList serialises as a plain JSON array in API
// responses — the frontend reads it directly.
func TestTagListJSONShape(t *testing.T) {
	encoded, err := json.Marshal(TagList{TagWithPet, TagRecovery})
	if err != nil {
		t.Fatalf("json.Marshal returned error: %v", err)
	}
	if string(encoded) != `["with-pet","recovery"]` {
		t.Errorf("marshalled to %s, want a plain array of slugs", encoded)
	}
}
