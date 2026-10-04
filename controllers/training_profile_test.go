package controllers

import (
	"net/http"
	"testing"
	"time"

	"github.com/aunefyren/treningheten/models"
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
