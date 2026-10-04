package controllers

import (
	"math"
	"testing"
	"time"

	"github.com/aunefyren/treningheten/models"
)

func systemPtr(key string) *string { return &key }

func mustHRZoneSystem(t *testing.T, key string) models.HRZoneSystem {
	t.Helper()
	system, ok := hrZoneSystemByKey(key)
	if !ok {
		t.Fatalf("zone system %q is not registered", key)
	}
	return system
}

// steadyHR is a four-second stream holding one heart rate, for bucketing tests.
func steadyHR(bpm int) (*models.StravaActivityStreams, []int) {
	times := []int{0, 1, 2, 3}
	return &models.StravaActivityStreams{
		Heartrate: intStream([]int{bpm, bpm, bpm, bpm}),
		Time:      intStream(times),
	}, times
}

func TestHRZoneSystemsAreWellFormed(t *testing.T) {
	seen := map[string]bool{}
	for _, system := range hrZoneSystems {
		t.Run(system.Key, func(t *testing.T) {
			if system.Key == "" || system.Label == "" || system.Description == "" {
				t.Fatalf("system is missing key/label/description: %+v", system)
			}
			if seen[system.Key] {
				t.Fatalf("duplicate key %q", system.Key)
			}
			seen[system.Key] = true
			if len(system.Codes) != len(system.Bounds)+1 || len(system.Names) != len(system.Bounds)+1 {
				t.Fatalf("%d bounds need %d codes and names, got %d/%d", len(system.Bounds), len(system.Bounds)+1, len(system.Codes), len(system.Names))
			}
			for i, b := range system.Bounds {
				if b <= 0 || b >= 1 || (i > 0 && b <= system.Bounds[i-1]) {
					t.Fatalf("bounds must rise strictly within (0,1): %v", system.Bounds)
				}
			}
		})
	}
	for _, key := range []string{models.HRZoneSystemPercentMax, models.HRZoneSystemReserve, models.HRZoneSystemOlympiatoppen} {
		if !seen[key] {
			t.Errorf("key constant %q has no registered system", key)
		}
	}
	if defaultHRZoneSystem().Key != models.HRZoneSystemPercentMax {
		t.Errorf("default system = %q, want percent_max", defaultHRZoneSystem().Key)
	}
}

func TestHRZoneSystemByKey(t *testing.T) {
	if _, ok := hrZoneSystemByKey("nope"); ok {
		t.Fatal("unknown key should not resolve")
	}
	if system, ok := hrZoneSystemByKey(models.HRZoneSystemOlympiatoppen); !ok || system.Codes[0] != "I-1" {
		t.Fatalf("olympiatoppen lookup = %+v/%v", system, ok)
	}
}

func TestEffectiveHRZoneSystem(t *testing.T) {
	tests := []struct {
		name string
		user models.User
		want string
	}{
		{"nothing set", models.User{}, models.HRZoneSystemPercentMax},
		{"legacy resting implies reserve", models.User{RestingHeartrate: intPtr(50)}, models.HRZoneSystemReserve},
		{"legacy zero resting ignored", models.User{RestingHeartrate: intPtr(0)}, models.HRZoneSystemPercentMax},
		{"explicit choice beats legacy rule", models.User{RestingHeartrate: intPtr(50), HRZoneSystem: systemPtr(models.HRZoneSystemPercentMax)}, models.HRZoneSystemPercentMax},
		{"explicit olympiatoppen", models.User{HRZoneSystem: systemPtr(models.HRZoneSystemOlympiatoppen)}, models.HRZoneSystemOlympiatoppen},
		{"unknown stored key falls back", models.User{HRZoneSystem: systemPtr("retired")}, models.HRZoneSystemPercentMax},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := effectiveHRZoneSystem(tc.user).Key; got != tc.want {
				t.Fatalf("effectiveHRZoneSystem = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestComputeHRZones(t *testing.T) {
	tests := []struct {
		name       string
		hr         []int
		anchor     hrAnchor
		wantBasis  string
		wantMaxBpm int
		wantSystem string
		wantZone   int // 1-based zone expected to hold ~all time
	}{
		{
			name:       "age based tempo",
			hr:         []int{150, 150, 150, 150},
			anchor:     hrAnchor{MaxBpm: 200, Basis: "age"}, // 150/200 = 75% -> zone 3
			wantBasis:  "age",
			wantMaxBpm: 200,
			wantSystem: models.HRZoneSystemPercentMax,
			wantZone:   3,
		},
		{
			name:       "observed fallback",
			hr:         []int{100, 200, 200, 200},
			anchor:     hrAnchor{}, // observed max 200
			wantBasis:  "observed",
			wantMaxBpm: 200,
			wantSystem: models.HRZoneSystemPercentMax,
			wantZone:   5, // 200/200 = 100% -> top zone holds most time
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			times := []int{0, 1, 2, 3}
			zones, applied := computeHRZones(&models.StravaActivityStreams{
				Heartrate: intStream(tc.hr),
				Time:      intStream(times),
			}, times, tc.anchor)
			if applied.Basis != tc.wantBasis || applied.MaxBpm != tc.wantMaxBpm || applied.System.Key != tc.wantSystem {
				t.Fatalf("applied = %q/%d/%q, want %q/%d/%q", applied.Basis, applied.MaxBpm, applied.System.Key, tc.wantBasis, tc.wantMaxBpm, tc.wantSystem)
			}
			if len(zones) != 5 {
				t.Fatalf("got %d zones, want 5", len(zones))
			}
			var pct float64
			for _, z := range zones {
				pct += z.Percent
			}
			if math.Abs(pct-100) > 0.5 {
				t.Fatalf("zone percents sum to %v, want ~100", pct)
			}
			if zones[tc.wantZone-1].Percent < 50 {
				t.Fatalf("expected most time in zone %d, got %+v", tc.wantZone, zones)
			}
		})
	}
}

func TestComputeHRZones_NoData(t *testing.T) {
	tests := []struct {
		name    string
		streams *models.StravaActivityStreams
		times   []int
	}{
		{"no heart-rate channel", &models.StravaActivityStreams{}, nil},
		{"only zero readings", &models.StravaActivityStreams{Heartrate: intStream([]int{0, 0})}, []int{0, 1}},
		{"empty heart-rate channel", &models.StravaActivityStreams{Heartrate: intStream([]int{})}, nil},
		{"no elapsed time", &models.StravaActivityStreams{Heartrate: intStream([]int{0, 150})}, []int{5, 5}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Both with a configured max and with none (the observed-peak path).
			for _, anchor := range []hrAnchor{{MaxBpm: 190}, {}} {
				zones, applied := computeHRZones(tc.streams, tc.times, anchor)
				if zones != nil || applied.MaxBpm != 0 || applied.System.Key != "" {
					t.Fatalf("max %d: got zones %+v / anchor %+v, want none", anchor.MaxBpm, zones, applied)
				}
			}
		})
	}
}

func TestComputeHRZones_Reserve(t *testing.T) {
	// Reserve (Karvonen): rest 50, max 200 -> 150 bpm reserve. Boundaries: 60% = 50+0.6*150
	// = 140, 70% = 155. A steady 150 bpm sits between them, in zone 2. (On a plain %-max
	// model, 150/200 = 75% would instead land in zone 3 — this asserts the reserve
	// boundaries are actually applied.)
	streams, times := steadyHR(150)
	anchor := hrAnchor{MaxBpm: 200, RestBpm: 50, Basis: "max", System: mustHRZoneSystem(t, models.HRZoneSystemReserve)}
	zones, applied := computeHRZones(streams, times, anchor)
	if applied.Basis != "max" || applied.MaxBpm != 200 || applied.RestBpm != 50 || applied.System.Key != models.HRZoneSystemReserve {
		t.Fatalf("applied = %+v, want max/200/rest 50/reserve", applied)
	}
	if zones[1].Percent < 50 {
		t.Fatalf("expected most reserve time in zone 2, got %+v", zones)
	}
	// Zone 1 lower bound is the resting HR under reserve, not 0.
	if zones[0].MinBpm != 50 {
		t.Fatalf("zone 1 min = %d, want 50 (resting HR)", zones[0].MinBpm)
	}
}

func TestComputeHRZones_ReserveFallsBackWithoutUsableRest(t *testing.T) {
	reserve := mustHRZoneSystem(t, models.HRZoneSystemReserve)
	tests := []struct {
		name   string
		anchor hrAnchor
	}{
		{"no resting HR", hrAnchor{MaxBpm: 200, System: reserve}},
		{"resting at max", hrAnchor{MaxBpm: 200, RestBpm: 200, System: reserve}},
		{"resting above observed peak", hrAnchor{RestBpm: 180, System: reserve}}, // peak 150
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			streams, times := steadyHR(150)
			zones, applied := computeHRZones(streams, times, tc.anchor)
			if applied.System.Key != models.HRZoneSystemPercentMax || applied.RestBpm != 0 {
				t.Fatalf("applied = %q rest %d, want percent_max with no rest", applied.System.Key, applied.RestBpm)
			}
			if zones[0].MinBpm != 0 {
				t.Fatalf("zone 1 min = %d, want 0 on %%-of-max", zones[0].MinBpm)
			}
		})
	}
}

func TestComputeHRZones_PercentMaxDropsRest(t *testing.T) {
	// A resting HR on file must not leak into a %-of-max system's bounds or output.
	streams, times := steadyHR(150)
	zones, applied := computeHRZones(streams, times, hrAnchor{MaxBpm: 200, RestBpm: 50, System: mustHRZoneSystem(t, models.HRZoneSystemPercentMax)})
	if applied.RestBpm != 0 || zones[0].MinBpm != 0 || zones[0].MaxBpm != 120 {
		t.Fatalf("applied rest %d, zone 1 = %+v; want rest 0 and 0–120", applied.RestBpm, zones[0])
	}
}

func TestComputeHRZones_Olympiatoppen(t *testing.T) {
	// Max 200 -> I-scale edges at 144 / 164 / 174 / 184 bpm.
	olt := mustHRZoneSystem(t, models.HRZoneSystemOlympiatoppen)
	tests := []struct {
		bpm      int
		wantZone int
	}{
		{90, 1}, // under the nominal 55% floor still counts as I-1
		{143, 1},
		{144, 2},
		{163, 2},
		{164, 3},
		{174, 4},
		{184, 5},
		{205, 5},
	}
	for _, tc := range tests {
		t.Run(olt.Codes[tc.wantZone-1], func(t *testing.T) {
			streams, times := steadyHR(tc.bpm)
			zones, applied := computeHRZones(streams, times, hrAnchor{MaxBpm: 200, Basis: "max", System: olt})
			if applied.System.Key != models.HRZoneSystemOlympiatoppen {
				t.Fatalf("system = %q, want olympiatoppen", applied.System.Key)
			}
			if zones[tc.wantZone-1].Percent != 100 {
				t.Fatalf("%d bpm: expected all time in zone %d, got %+v", tc.bpm, tc.wantZone, zones)
			}
		})
	}

	streams, times := steadyHR(150)
	zones, _ := computeHRZones(streams, times, hrAnchor{MaxBpm: 200, System: olt})
	wantBounds := [][2]int{{0, 144}, {144, 164}, {164, 174}, {174, 184}, {184, 0}}
	for i, z := range zones {
		if z.Code != olt.Codes[i] || z.Name != olt.Names[i] || z.Zone != i+1 {
			t.Errorf("zone %d labelled %q/%q/%d, want %q/%q/%d", i, z.Code, z.Name, z.Zone, olt.Codes[i], olt.Names[i], i+1)
		}
		if z.MinBpm != wantBounds[i][0] || z.MaxBpm != wantBounds[i][1] {
			t.Errorf("%s bounds = %d–%d, want %d–%d", z.Code, z.MinBpm, z.MaxBpm, wantBounds[i][0], wantBounds[i][1])
		}
	}
}

func TestComputeHRZones_ZeroSystemUsesDefault(t *testing.T) {
	streams, times := steadyHR(150)
	zones, applied := computeHRZones(streams, times, hrAnchor{MaxBpm: 200})
	if applied.System.Key != models.HRZoneSystemPercentMax || zones[2].Code != "Z3" {
		t.Fatalf("zero-value system: applied %q, zone 3 code %q; want percent_max/Z3", applied.System.Key, zones[2].Code)
	}
}

func TestResolveUserHR(t *testing.T) {
	now := time.Date(2026, 7, 21, 0, 0, 0, 0, time.UTC)
	birth := timePtr(time.Date(1996, 1, 1, 0, 0, 0, 0, time.UTC)) // age 30 -> 190
	tests := []struct {
		name              string
		user              models.User
		wantMax, wantRest int
		wantBasis         string
		wantSystem        string
	}{
		{"nothing set", models.User{}, 0, 0, "", models.HRZoneSystemPercentMax},
		{"age only", models.User{BirthDate: birth}, 190, 0, "age", models.HRZoneSystemPercentMax},
		{"observed only", models.User{ObservedMaxHeartrate: intPtr(185)}, 185, 0, "observed_max", models.HRZoneSystemPercentMax},
		{"observed beats age", models.User{BirthDate: birth, ObservedMaxHeartrate: intPtr(196)}, 196, 0, "observed_max", models.HRZoneSystemPercentMax},
		{"explicit beats observed", models.User{MaxHeartrate: intPtr(198), ObservedMaxHeartrate: intPtr(196)}, 198, 0, "max", models.HRZoneSystemPercentMax},
		{"explicit max wins", models.User{BirthDate: birth, MaxHeartrate: intPtr(198)}, 198, 0, "max", models.HRZoneSystemPercentMax},
		{"reserve inputs", models.User{MaxHeartrate: intPtr(198), RestingHeartrate: intPtr(48)}, 198, 48, "max", models.HRZoneSystemReserve},
		{"observed zero ignored", models.User{BirthDate: birth, ObservedMaxHeartrate: intPtr(0)}, 190, 0, "age", models.HRZoneSystemPercentMax},
		{"chosen system carried", models.User{MaxHeartrate: intPtr(198), HRZoneSystem: systemPtr(models.HRZoneSystemOlympiatoppen)}, 198, 0, "max", models.HRZoneSystemOlympiatoppen},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := resolveUserHR(tc.user, now)
			if got.MaxBpm != tc.wantMax || got.RestBpm != tc.wantRest || got.Basis != tc.wantBasis || got.System.Key != tc.wantSystem {
				t.Fatalf("got %d/%d/%q/%q, want %d/%d/%q/%q", got.MaxBpm, got.RestBpm, got.Basis, got.System.Key, tc.wantMax, tc.wantRest, tc.wantBasis, tc.wantSystem)
			}
		})
	}
}

func TestResolveUserHR_AgeUsesActivityDate(t *testing.T) {
	// Same birth date, two different activity dates → the age-based max reflects the age
	// *at the activity*, so an old activity's zones don't drift as the athlete ages.
	user := models.User{BirthDate: timePtr(time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC))}
	if got := resolveUserHR(user, time.Date(2010, 6, 1, 0, 0, 0, 0, time.UTC)); got.MaxBpm != 200 || got.Basis != "age" {
		t.Fatalf("age 20 activity: max=%d basis=%q, want 200/age", got.MaxBpm, got.Basis)
	}
	if got := resolveUserHR(user, time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)); got.MaxBpm != 186 || got.Basis != "age" {
		t.Fatalf("age 34 activity: max=%d basis=%q, want 186/age", got.MaxBpm, got.Basis)
	}
}

func TestHRMaxFromBirthDate(t *testing.T) {
	now := time.Date(2026, 7, 21, 0, 0, 0, 0, time.UTC)
	tests := []struct {
		name  string
		birth *time.Time
		want  int
	}{
		{"nil", nil, 0},
		{"age 30", timePtr(time.Date(1996, 1, 1, 0, 0, 0, 0, time.UTC)), 190},
		{"birthday not yet this year", timePtr(time.Date(1996, 12, 31, 0, 0, 0, 0, time.UTC)), 191},
		{"implausible future", timePtr(time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)), 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := hrMaxFromBirthDate(tc.birth, now); got != tc.want {
				t.Fatalf("hrMaxFromBirthDate = %d, want %d", got, tc.want)
			}
		})
	}
}

func TestSummarizeStreams_HRZoneFields(t *testing.T) {
	reserve := mustHRZoneSystem(t, models.HRZoneSystemReserve)
	olt := mustHRZoneSystem(t, models.HRZoneSystemOlympiatoppen)
	tests := []struct {
		name       string
		anchor     hrAnchor
		wantSystem string
		wantRest   int
		wantCode   string
	}{
		{"reserve reports rest", hrAnchor{MaxBpm: 190, RestBpm: 50, Basis: "max", System: reserve}, models.HRZoneSystemReserve, 50, "Z1"},
		{"olympiatoppen omits rest", hrAnchor{MaxBpm: 190, RestBpm: 50, Basis: "max", System: olt}, models.HRZoneSystemOlympiatoppen, 0, "I-1"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s := SummarizeStreams(steadyRun(100, 3.0, 150), "km", tc.anchor)
			if s.HRZoneSystem != tc.wantSystem || s.HRRestBpm != tc.wantRest || s.HRMaxBasis != "max" || s.HRMaxBpm != 190 {
				t.Fatalf("summary zone fields = %q/rest %d/%q/%d", s.HRZoneSystem, s.HRRestBpm, s.HRMaxBasis, s.HRMaxBpm)
			}
			if len(s.HRZones) != 5 || s.HRZones[0].Code != tc.wantCode {
				t.Fatalf("zones = %+v, want five starting at %s", s.HRZones, tc.wantCode)
			}
		})
	}

	// No heart rate → no zone fields at all.
	noHR := steadyRun(100, 3.0, 150)
	noHR.Heartrate = nil
	if s := SummarizeStreams(noHR, "km", hrAnchor{MaxBpm: 190, System: olt}); s.HRZones != nil || s.HRZoneSystem != "" || s.HRMaxBpm != 0 {
		t.Fatalf("summary without HR carries zone fields: %+v", s)
	}
}
