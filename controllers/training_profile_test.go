package controllers

import (
	"net/http"
	"testing"
	"time"

	"github.com/aunefyren/treningheten/database"
	"github.com/aunefyren/treningheten/models"

	"github.com/google/uuid"
)

func TestValidateTrainingProfile(t *testing.T) {
	now := time.Date(2026, 10, 4, 12, 0, 0, 0, time.UTC)
	adult := now.AddDate(-30, 0, 0)
	child := now.AddDate(-5, 0, 0)

	tests := []struct {
		name    string
		request models.UserTrainingProfileRequest
		want    string
	}{
		{"empty clears everything", models.UserTrainingProfileRequest{}, ""},
		{"resting above max", models.UserTrainingProfileRequest{MaxHeartrate: intPtr(110), RestingHeartrate: intPtr(115)}, "Your resting heart rate must be below your maximum heart rate."},
		{"full profile", models.UserTrainingProfileRequest{BirthDate: &adult, MaxHeartrate: intPtr(190), RestingHeartrate: intPtr(50)}, ""},
		{"range bounds", models.UserTrainingProfileRequest{MaxHeartrate: intPtr(240), RestingHeartrate: intPtr(25)}, ""},
		{"too young", models.UserTrainingProfileRequest{BirthDate: &child}, "Your birth date must be more than thirteen years ago."},
		{"max too low", models.UserTrainingProfileRequest{MaxHeartrate: intPtr(99)}, "Your maximum heart rate must be between 100 and 240 bpm."},
		{"max too high", models.UserTrainingProfileRequest{MaxHeartrate: intPtr(241)}, "Your maximum heart rate must be between 100 and 240 bpm."},
		{"resting too low", models.UserTrainingProfileRequest{RestingHeartrate: intPtr(24)}, "Your resting heart rate must be between 25 and 120 bpm."},
		{"resting too high", models.UserTrainingProfileRequest{RestingHeartrate: intPtr(121)}, "Your resting heart rate must be between 25 and 120 bpm."},
		{"resting equals max", models.UserTrainingProfileRequest{MaxHeartrate: intPtr(110), RestingHeartrate: intPtr(110)}, "Your resting heart rate must be below your maximum heart rate."},
		{"olympiatoppen without resting", models.UserTrainingProfileRequest{HRZoneSystem: systemPtr(models.HRZoneSystemOlympiatoppen)}, ""},
		{"percent of max", models.UserTrainingProfileRequest{HRZoneSystem: systemPtr(models.HRZoneSystemPercentMax), RestingHeartrate: intPtr(50)}, ""},
		{"reserve with resting", models.UserTrainingProfileRequest{HRZoneSystem: systemPtr(models.HRZoneSystemReserve), RestingHeartrate: intPtr(50)}, ""},
		{"reserve without resting", models.UserTrainingProfileRequest{HRZoneSystem: systemPtr(models.HRZoneSystemReserve)}, "Heart-rate reserve zones need a resting heart rate."},
		{"unknown zone system", models.UserTrainingProfileRequest{HRZoneSystem: systemPtr("zones-by-vibes")}, "Unknown heart-rate zone system."},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := validateTrainingProfile(test.request, now); got != test.want {
				t.Errorf("validateTrainingProfile() = %q, want %q", got, test.want)
			}
		})
	}
}

func TestTrainingProfileAPI(t *testing.T) {
	h := newAPIHarness(t)
	me, token := userWithPassword(h, "me@training.test", "Password123")
	_, otherToken := h.user("other@training.test", false)
	userPath := "/api/auth/users/" + me.ID.String()
	profilePath := userPath + "/training-profile"
	birth := time.Now().AddDate(-30, 0, 0)

	readProfile := func(t *testing.T) map[string]any {
		t.Helper()
		return field(t, h.ok("GET", userPath, token, nil), "user").(map[string]any)
	}

	t.Run("saves every field", func(t *testing.T) {
		h.ok("PUT", profilePath, token, models.UserTrainingProfileRequest{BirthDate: &birth, MaxHeartrate: intPtr(190), RestingHeartrate: intPtr(50), HRZoneSystem: systemPtr(models.HRZoneSystemOlympiatoppen)})
		user := readProfile(t)
		if user["max_heartrate"] != float64(190) || user["resting_heartrate"] != float64(50) || user["birth_date"] == nil || user["hr_zone_system"] != models.HRZoneSystemOlympiatoppen {
			t.Errorf("profile after save = max %v, resting %v, birth %v, system %v", user["max_heartrate"], user["resting_heartrate"], user["birth_date"], user["hr_zone_system"])
		}
	})

	t.Run("rejects invalid values and keeps the old ones", func(t *testing.T) {
		h.expect(http.StatusBadRequest, "PUT", profilePath, token, models.UserTrainingProfileRequest{MaxHeartrate: intPtr(150), RestingHeartrate: intPtr(150)})
		h.expect(http.StatusBadRequest, "PUT", profilePath, token, "not json")
		h.expect(http.StatusBadRequest, "PUT", profilePath, token, models.UserTrainingProfileRequest{MaxHeartrate: intPtr(190), HRZoneSystem: systemPtr(models.HRZoneSystemReserve)})
		if user := readProfile(t); user["max_heartrate"] != float64(190) || user["hr_zone_system"] != models.HRZoneSystemOlympiatoppen {
			t.Errorf("max_heartrate/system = %v/%v after a rejected save, want 190/olympiatoppen", user["max_heartrate"], user["hr_zone_system"])
		}
	})

	t.Run("account update leaves the profile alone", func(t *testing.T) {
		h.ok("POST", userPath, token, map[string]any{
			"email": "me@training.test", "password_old": "Password123", "max_heartrate": nil, "resting_heartrate": nil, "birth_date": nil,
		})
		if user := readProfile(t); user["max_heartrate"] != float64(190) || user["resting_heartrate"] != float64(50) || user["birth_date"] == nil {
			t.Errorf("account update changed the training profile: %v", user)
		}
	})

	t.Run("path id is ignored", func(t *testing.T) {
		h.ok("PUT", profilePath, otherToken, models.UserTrainingProfileRequest{MaxHeartrate: intPtr(200)})
		if user := readProfile(t); user["max_heartrate"] != float64(190) {
			t.Errorf("max_heartrate = %v after another user's save, want it untouched (190)", user["max_heartrate"])
		}
	})

	t.Run("null clears", func(t *testing.T) {
		h.ok("PUT", profilePath, token, models.UserTrainingProfileRequest{})
		user := readProfile(t)
		if user["max_heartrate"] != nil || user["resting_heartrate"] != nil || user["birth_date"] != nil || user["hr_zone_system"] != nil {
			t.Errorf("profile after clearing = max %v, resting %v, birth %v, system %v", user["max_heartrate"], user["resting_heartrate"], user["birth_date"], user["hr_zone_system"])
		}
	})
}

func TestHRZoneSystemsAPI(t *testing.T) {
	h := newAPIHarness(t)
	_, token := h.user("zones@training.test", false)

	systems := field(t, h.ok("GET", "/api/auth/hr-zone-systems", token, nil), "hr_zone_systems").([]any)
	if len(systems) != len(hrZoneSystems) {
		t.Fatalf("got %d systems, want %d", len(systems), len(hrZoneSystems))
	}
	for i, raw := range systems {
		system := raw.(map[string]any)
		if system["key"] != hrZoneSystems[i].Key || system["uses_resting"] != hrZoneSystems[i].UsesResting {
			t.Errorf("system %d = %v, want key %q uses_resting %v", i, system, hrZoneSystems[i].Key, hrZoneSystems[i].UsesResting)
		}
		if len(system["bounds"].([]any)) != len(hrZoneSystems[i].Bounds) || len(system["codes"].([]any)) != len(hrZoneSystems[i].Codes) {
			t.Errorf("system %q bounds/codes not serialised in full: %v", hrZoneSystems[i].Key, system)
		}
	}

	h.expect(http.StatusUnauthorized, "GET", "/api/auth/hr-zone-systems", "", nil)
}

// sportActionID returns the seeded activity-type action with the given name.
func sportActionID(t *testing.T, name string) uuid.UUID {
	t.Helper()
	sports, err := database.GetActivityTypeActions()
	if err != nil {
		t.Fatalf("activity types: %v", err)
	}
	for _, sport := range sports {
		if sport.Name == name {
			return sport.ID
		}
	}
	t.Fatalf("no seeded activity type %q", name)
	return uuid.Nil
}

// seedHRActivity seeds a day → session → operation of the given sport carrying a steady
// 150 bpm Strava stream and the stored max_heartrate rollup. A disabled operation is seeded
// only when enabled is false. Returns the day and operation ids.
func seedHRActivity(t *testing.T, userID uuid.UUID, actionID uuid.UUID, maxHeartrate int, enabled bool) (uuid.UUID, uuid.UUID) {
	t.Helper()
	day := models.ExerciseDay{Date: time.Now(), Enabled: true, UserID: &userID}
	day.ID = uuid.New()
	if err := database.Instance.Omit("User", "Goal").Create(&day).Error; err != nil {
		t.Fatalf("seed day: %v", err)
	}
	exercise := models.Exercise{Enabled: true, IsOn: true, ExerciseDayID: day.ID}
	exercise.ID = uuid.New()
	if err := database.Instance.Omit("ExerciseDay").Create(&exercise).Error; err != nil {
		t.Fatalf("seed exercise: %v", err)
	}
	operation := models.Operation{Enabled: true, ExerciseID: exercise.ID, ActionID: &actionID, Type: "moving", DistanceUnit: "km", WeightUnit: "kg", MaxHeartrate: &maxHeartrate}
	operation.ID = uuid.New()
	if err := database.Instance.Omit("Exercise", "Action", "Gear").Create(&operation).Error; err != nil {
		t.Fatalf("seed operation: %v", err)
	}
	if !enabled {
		if err := database.Instance.Model(&models.Operation{}).Where("id = ?", operation.ID).Update("enabled", false).Error; err != nil {
			t.Fatalf("disable operation: %v", err)
		}
	}
	set := models.OperationSet{Enabled: true, OperationID: operation.ID, StravaID: sp("4242"), StravaStreams: &models.StravaStreamsJSON{StravaActivityStreams: *steadyRun(100, 3.0, 150)}}
	set.ID = uuid.New()
	if err := database.Instance.Omit("Operation").Create(&set).Error; err != nil {
		t.Fatalf("seed set: %v", err)
	}
	return day.ID, operation.ID
}

func TestSportMaxHeartratesAPI(t *testing.T) {
	h := newAPIHarness(t)
	me, token := h.user("sport@training.test", false)
	other, _ := h.user("other-sport@training.test", false)
	profilePath := "/api/auth/users/" + me.ID.String() + "/training-profile"
	run, ride := sportActionID(t, "Run"), sportActionID(t, "Bicycling")

	runDay, runOperation := seedHRActivity(t, me.ID, run, 182, true)
	seedHRActivity(t, me.ID, run, 210, false)    // deleted activity: not observed
	seedHRActivity(t, other.ID, ride, 199, true) // someone else's ride: not observed

	// sportEntry finds one sport in GET /sport-heartrates.
	sportEntry := func(t *testing.T, actionID uuid.UUID) map[string]any {
		t.Helper()
		for _, raw := range field(t, h.ok("GET", "/api/auth/sport-heartrates", token, nil), "sport_heartrates").([]any) {
			if entry := raw.(map[string]any); entry["action_id"] == actionID.String() {
				return entry
			}
		}
		t.Fatalf("sport %s missing from sport-heartrates", actionID)
		return nil
	}
	// runZones reads the HR anchor of the run on the exercise-day endpoint (web card) and
	// through the MCP summary path.
	runZones := func(t *testing.T) (float64, any) {
		t.Helper()
		day := field(t, h.ok("GET", "/api/auth/exercise-days/"+runDay.String(), token, nil), "exercise_day").(map[string]any)
		summary := field(t, day, "exercises").([]any)[0].(map[string]any)["operations"].([]any)[0].(map[string]any)["stream_summary"].(map[string]any)
		mcp, err := assembleActivityStreamSummary(me.ID, runOperation)
		if err != nil || mcp == nil {
			t.Fatalf("MCP summary: %v %v", mcp, err)
		}
		if float64(mcp.HRMaxBpm) != summary["hr_max_bpm"] || mcp.HRMaxBasis != summary["hr_max_basis"] {
			t.Fatalf("MCP anchor %d/%q disagrees with web anchor %v/%v", mcp.HRMaxBpm, mcp.HRMaxBasis, summary["hr_max_bpm"], summary["hr_max_basis"])
		}
		return summary["hr_max_bpm"].(float64), summary["hr_max_basis"]
	}

	t.Run("lists observed max per sport before anything is set", func(t *testing.T) {
		if entry := sportEntry(t, run); entry["max_heartrate"] != nil || entry["observed_max_heartrate"] != float64(182) {
			t.Errorf("run = %v, want no max and observed 182", entry)
		}
		if entry := sportEntry(t, ride); entry["max_heartrate"] != nil || entry["observed_max_heartrate"] != nil {
			t.Errorf("ride = %v, want neither set", entry)
		}
	})

	t.Run("saves sport maxes and anchors that sport's zones", func(t *testing.T) {
		h.ok("PUT", profilePath, token, models.UserTrainingProfileRequest{
			MaxHeartrate:       intPtr(198),
			SportMaxHeartrates: []models.SportMaxHeartrate{{ActionID: run, MaxHeartrate: 190}, {ActionID: ride, MaxHeartrate: 184}},
		})
		if entry := sportEntry(t, run); entry["max_heartrate"] != float64(190) {
			t.Errorf("run max = %v, want 190", entry["max_heartrate"])
		}
		if entry := sportEntry(t, ride); entry["max_heartrate"] != float64(184) {
			t.Errorf("ride max = %v, want 184", entry["max_heartrate"])
		}
		if max, basis := runZones(t); max != 190 || basis != "sport_max" {
			t.Errorf("run zones anchored to %v/%v, want 190/sport_max", max, basis)
		}
	})

	t.Run("replaces the set as sent", func(t *testing.T) {
		h.ok("PUT", profilePath, token, models.UserTrainingProfileRequest{
			MaxHeartrate:       intPtr(198),
			SportMaxHeartrates: []models.SportMaxHeartrate{{ActionID: ride, MaxHeartrate: 186}},
		})
		if entry := sportEntry(t, run); entry["max_heartrate"] != nil {
			t.Errorf("run max = %v after it was left out, want cleared", entry["max_heartrate"])
		}
		if entry := sportEntry(t, ride); entry["max_heartrate"] != float64(186) {
			t.Errorf("ride max = %v, want 186", entry["max_heartrate"])
		}
		if max, basis := runZones(t); max != 198 || basis != "max" {
			t.Errorf("run zones anchored to %v/%v, want the global 198/max", max, basis)
		}
	})

	t.Run("rejects invalid sport maxes and keeps the old ones", func(t *testing.T) {
		invalid := [][]models.SportMaxHeartrate{
			{{ActionID: uuid.New(), MaxHeartrate: 190}},
			{{ActionID: run, MaxHeartrate: 190}, {ActionID: run, MaxHeartrate: 191}},
			{{ActionID: run, MaxHeartrate: 90}},
		}
		for _, entries := range invalid {
			h.expect(http.StatusBadRequest, "PUT", profilePath, token, models.UserTrainingProfileRequest{SportMaxHeartrates: entries})
		}
		h.expect(http.StatusBadRequest, "PUT", profilePath, token, models.UserTrainingProfileRequest{
			RestingHeartrate: intPtr(120), SportMaxHeartrates: []models.SportMaxHeartrate{{ActionID: run, MaxHeartrate: 115}},
		})
		if entry := sportEntry(t, ride); entry["max_heartrate"] != float64(186) {
			t.Errorf("ride max = %v after rejected saves, want 186", entry["max_heartrate"])
		}
	})

	t.Run("null clears", func(t *testing.T) {
		h.ok("PUT", profilePath, token, models.UserTrainingProfileRequest{})
		if entry := sportEntry(t, ride); entry["max_heartrate"] != nil {
			t.Errorf("ride max = %v after clearing, want nil", entry["max_heartrate"])
		}
	})

	h.expect(http.StatusUnauthorized, "GET", "/api/auth/sport-heartrates", "", nil)
}
