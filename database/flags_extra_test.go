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
