package controllers

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

// TestValidGearType covers the gear-type vocabulary, including the normalisation applied
// before matching (the API accepts what a user types).
func TestValidGearType(t *testing.T) {
	valid := []string{"shoe", "bike", "other", "Shoe", "BIKE", "  other  ", "\tbike\n"}
	for _, gearType := range valid {
		if !validGearType(gearType) {
			t.Errorf("validGearType(%q) = false, want true", gearType)
		}
	}

	invalid := []string{"", "   ", "shoes", "bikes", "boat", "shoe bike", "oth er"}
	for _, gearType := range invalid {
		if validGearType(gearType) {
			t.Errorf("validGearType(%q) = true, want false", gearType)
		}
	}
}

// TestStravaGearTypeFromID covers mapping Strava's gear-id prefix onto a local gear type.
func TestStravaGearTypeFromID(t *testing.T) {
	tests := map[string]string{
		"b12345678": "bike",
		"g98765432": "shoe",
		"b":         "bike",
		"g":         "shoe",
		"x12345":    "other",
		"":          "other",
		"B12345678": "other", // Strava's ids are lowercase; an uppercase one is not ours to guess
		"12345":     "other",
	}

	for gearID, want := range tests {
		if got := stravaGearTypeFromID(gearID); got != want {
			t.Errorf("stravaGearTypeFromID(%q) = %q, want %q", gearID, got, want)
		}
	}
}

// TestPercentageOf covers the percentage helper used across the statistics endpoints,
// including the zero-denominator guard that keeps a NaN out of the JSON response.
func TestPercentageOf(t *testing.T) {
	tests := []struct {
		part, whole int64
		want        float64
	}{
		{1, 2, 50},
		{1, 3, 33.3},
		{2, 3, 66.7},
		{0, 10, 0},
		{10, 10, 100},
		{3, 0, 0},     // no denominator
		{3, -5, 0},    // negative denominator
		{15, 10, 150}, // over 100% is passed through, not clamped
	}

	for _, test := range tests {
		if got := percentageOf(test.part, test.whole); got != test.want {
			t.Errorf("percentageOf(%d, %d) = %v, want %v", test.part, test.whole, got, test.want)
		}
	}
}

// TestDaysLeftInWeekIncludingToday covers the Monday-Sunday week the goal system runs on:
// Monday has the full 7 days left, Sunday has 1 (today), and the Go Sunday=0 weekday is
// remapped rather than wrapping to 8.
func TestDaysLeftInWeekIncludingToday(t *testing.T) {
	// 2026-08-03 is a Monday.
	monday := time.Date(2026, 8, 3, 12, 0, 0, 0, time.UTC)

	want := []int{7, 6, 5, 4, 3, 2, 1}
	for offset, expected := range want {
		day := monday.AddDate(0, 0, offset)
		if got := daysLeftInWeekIncludingToday(day); got != expected {
			t.Errorf("%s: got %d days left, want %d", day.Weekday(), got, expected)
		}
	}

	// The time of day must not matter — it counts whole days.
	for _, hour := range []int{0, 1, 23} {
		day := time.Date(2026, 8, 5, hour, 30, 0, 0, time.UTC) // Wednesday
		if got := daysLeftInWeekIncludingToday(day); got != 5 {
			t.Errorf("Wednesday at %02d:30 gave %d days left, want 5", hour, got)
		}
	}
}

// TestMostCommonAction covers picking a session's headline activity type. The tie-break
// matters: map iteration order is random, so without a deterministic rule the same input
// would produce different answers between runs.
func TestMostCommonAction(t *testing.T) {
	if got := mostCommonAction(nil); got != nil {
		t.Errorf("mostCommonAction(nil) = %v, want nil", got)
	}
	if got := mostCommonAction([]uuid.UUID{}); got != nil {
		t.Errorf("mostCommonAction([]) = %v, want nil", got)
	}

	single := uuid.New()
	if got := mostCommonAction([]uuid.UUID{single}); got == nil || *got != single {
		t.Errorf("mostCommonAction of one id = %v, want %v", got, single)
	}

	// A clear majority wins regardless of position in the slice.
	major := uuid.New()
	minor := uuid.New()
	for _, input := range [][]uuid.UUID{
		{major, major, minor},
		{minor, major, major},
		{major, minor, major},
	} {
		if got := mostCommonAction(input); got == nil || *got != major {
			t.Errorf("mostCommonAction(%v) = %v, want %v", input, got, major)
		}
	}

	// A tie resolves to the lowest UUID, stably across repeated runs.
	a := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	b := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	c := uuid.MustParse("33333333-3333-3333-3333-333333333333")
	for i := 0; i < 100; i++ {
		got := mostCommonAction([]uuid.UUID{c, b, a})
		if got == nil || *got != a {
			t.Fatalf("tie-break returned %v, want the lowest id %v", got, a)
		}
	}
}
