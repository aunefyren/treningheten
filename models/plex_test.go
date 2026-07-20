package models

import (
	"encoding/json"
	"testing"
)

// Plex sends librarySectionID inconsistently as a bare number or a quoted string
// (and sometimes not at all), so PlexFlexString must accept every shape without
// failing the whole history decode.
func TestPlexFlexStringUnmarshal(t *testing.T) {
	cases := map[string]string{
		`{"librarySectionID": "3"}`:  "3", // quoted string
		`{"librarySectionID": 3}`:    "3", // bare number
		`{"librarySectionID": null}`: "",  // explicit null
		`{}`:                         "",  // absent
	}
	for raw, want := range cases {
		var item PlexHistoryMetadata
		if err := json.Unmarshal([]byte(raw), &item); err != nil {
			t.Fatalf("unmarshal %s: %v", raw, err)
		}
		if string(item.LibrarySectionID) != want {
			t.Errorf("%s → LibrarySectionID = %q, want %q", raw, item.LibrarySectionID, want)
		}
	}
}
