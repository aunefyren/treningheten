package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
)

// Activity tag slugs. Treningheten owns this vocabulary. Strava's public API can
// only supply the StravaManaged subset (derived from `commute` + `workout_type`);
// the remaining tags exist only in Strava's internal app and are therefore set
// manually by the user inside Treningheten.
const (
	TagRace      = "race"
	TagLongRun   = "long-run"
	TagWorkout   = "workout"
	TagCommute   = "commute"
	TagForACause = "for-a-cause"
	TagRecovery  = "recovery"
	TagWithPet   = "with-pet"
	TagWithKid   = "with-kid"
)

// ValidTags is the full controlled vocabulary, in display order.
var ValidTags = []string{
	TagRace, TagLongRun, TagWorkout, TagCommute,
	TagForACause, TagRecovery, TagWithPet, TagWithKid,
}

// StravaManagedTags are the tags Strava can derive and therefore owns on each
// sync. Tags outside this set are user-controlled and preserved across syncs.
var StravaManagedTags = []string{TagRace, TagLongRun, TagWorkout, TagCommute}

// IsValidTag reports whether tag is part of the controlled vocabulary.
func IsValidTag(tag string) bool {
	for _, t := range ValidTags {
		if t == tag {
			return true
		}
	}
	return false
}

// IsStravaManagedTag reports whether tag is one Strava derives (and thus owns).
func IsStravaManagedTag(tag string) bool {
	for _, t := range StravaManagedTags {
		if t == tag {
			return true
		}
	}
	return false
}

// TagList is a string slice persisted as a JSON column (mirrors StravaStreamsJSON).
type TagList []string

func (t TagList) Value() (driver.Value, error) {
	if t == nil {
		return nil, nil
	}
	return json.Marshal(t)
}

func (t *TagList) Scan(value interface{}) error {
	if value == nil {
		*t = nil
		return nil
	}

	var b []byte
	switch v := value.(type) {
	case []byte:
		b = v
	case string:
		b = []byte(v)
	default:
		return errors.New("failed to cast tags value to []byte")
	}

	if len(b) == 0 || string(b) == "null" {
		*t = nil
		return nil
	}

	return json.Unmarshal(b, t)
}
