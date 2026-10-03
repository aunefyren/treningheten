package database

import (
	"testing"
	"time"

	"github.com/aunefyren/treningheten/models"
)

func TestSetExerciseIsOn(t *testing.T) {
	newTestDB(t)
	user := makeTestUser(t, "ison@example.com", nil)
	exerciseID := seedExerciseWithID(t, seedDay(t, user.ID, time.Now(), true))

	if err := SetExerciseIsOn(exerciseID, false); err != nil {
		t.Fatal(err)
	}
	var exercise models.Exercise
	Instance.Where("id = ?", exerciseID).First(&exercise)
	if exercise.IsOn {
		t.Error("is_on is still true")
	}
}

// The Hevy baseline only advances while the user is still connected, so a finished
// background sync can't mark a disconnected account as synced.
func TestSetHevyLastSyncForUserOnlyWhileConnected(t *testing.T) {
	newTestDB(t)
	user := makeTestUser(t, "hevysync@example.com", nil)
	at := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)

	if err := SetHevyLastSyncForUser(user.ID, at); err != nil {
		t.Fatal(err)
	}
	stored, _ := GetAllUserInformation(user.ID)
	if stored.HevyLastSync != nil {
		t.Error("a baseline was recorded for a user with no Hevy key")
	}

	key := "encrypted"
	Instance.Model(&models.User{}).Where("id = ?", user.ID).Update("hevy_api_key", &key)
	if err := SetHevyLastSyncForUser(user.ID, at); err != nil {
		t.Fatal(err)
	}
	stored, _ = GetAllUserInformation(user.ID)
	if stored.HevyLastSync == nil || !stored.HevyLastSync.Equal(at) {
		t.Errorf("baseline = %v, want %v", stored.HevyLastSync, at)
	}
}

// TestSetStravaCredentialsForUserWritesOnlyItsColumns covers that storing a refreshed Strava
// token touches nothing else on the row: the sync holds a stale copy of the user, and a
// whole-row save would revert a profile change made while it ran.
func TestSetStravaCredentialsForUserWritesOnlyItsColumns(t *testing.T) {
	tests := []struct {
		name     string
		stravaID *string
		wantID   string
	}{
		{name: "first connection records the athlete id", stravaID: strPtr("4242"), wantID: "4242"},
		{name: "a refresh keeps the stored athlete id", stravaID: nil, wantID: "1111"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			newTestDB(t)
			user := makeTestUser(t, "stravacreds@example.com", func(u *models.User) {
				u.StravaID = strPtr("1111")
			})

			// A profile change made while the sync was running.
			if err := Instance.Model(&models.User{}).Where("id = ?", user.ID).Update("first_name", "Renamed").Error; err != nil {
				t.Fatal(err)
			}

			if err := SetStravaCredentialsForUser(user.ID, "r:ciphertext", test.stravaID); err != nil {
				t.Fatal(err)
			}

			stored, err := GetAllUserInformation(user.ID)
			if err != nil {
				t.Fatal(err)
			}
			if stored.StravaCode == nil || *stored.StravaCode != "r:ciphertext" {
				t.Errorf("strava code = %v, want r:ciphertext", stored.StravaCode)
			}
			if stored.StravaID == nil || *stored.StravaID != test.wantID {
				t.Errorf("strava id = %v, want %s", stored.StravaID, test.wantID)
			}
			if stored.FirstName != "Renamed" {
				t.Errorf("first name = %q, the credential write reverted a concurrent profile change", stored.FirstName)
			}
		})
	}
}

// Nothing listens on port 1, so both network backends fail fast with an error rather
// than hanging or leaving a half-initialised Instance behind silently.
func TestConnectReportsUnreachableServers(t *testing.T) {
	previous := Instance
	t.Cleanup(func() { Instance = previous })

	for _, dbType := range []string{"mysql", "postgres"} {
		if err := Connect(dbType, "UTC", "user", "pass", "127.0.0.1", 1, "treningheten", false, ""); err == nil {
			t.Errorf("Connect(%s) to a closed port succeeded", dbType)
		}
	}
	if err := CreateTable("user", "pass", "127.0.0.1", 1, "treningheten"); err == nil {
		t.Error("CreateTable against a closed port succeeded")
	}
}
