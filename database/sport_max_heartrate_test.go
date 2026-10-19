package database

import (
	"testing"
	"time"

	"github.com/aunefyren/treningheten/models"

	"github.com/google/uuid"
)

func TestSaveTrainingProfileInDB(t *testing.T) {
	newTestDB(t)
	user := makeTestUser(t, "sport@db.test", nil)
	other := makeTestUser(t, "other@db.test", nil)
	run, ride := uuid.New(), uuid.New()

	maxes := func(t *testing.T, userID uuid.UUID) map[uuid.UUID]int {
		t.Helper()
		rows, err := GetSportMaxHeartratesForUserID(userID)
		if err != nil {
			t.Fatalf("GetSportMaxHeartratesForUserID: %v", err)
		}
		got := map[uuid.UUID]int{}
		for _, row := range rows {
			got[row.ActionID] = row.MaxHeartrate
		}
		return got
	}

	if err := SaveTrainingProfileInDB(other, []models.SportMaxHeartrate{{ActionID: run, MaxHeartrate: 200}}); err != nil {
		t.Fatalf("save other: %v", err)
	}

	tests := []struct {
		name  string
		saved []models.SportMaxHeartrate
		want  map[uuid.UUID]int
	}{
		{"first save", []models.SportMaxHeartrate{{ActionID: run, MaxHeartrate: 195}, {ActionID: ride, MaxHeartrate: 186}}, map[uuid.UUID]int{run: 195, ride: 186}},
		{"replaces the set, same sport again", []models.SportMaxHeartrate{{ActionID: ride, MaxHeartrate: 188}}, map[uuid.UUID]int{ride: 188}},
		{"empty clears", nil, map[uuid.UUID]int{}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			user.MaxHeartrate = intPtr(190)
			if err := SaveTrainingProfileInDB(user, tc.saved); err != nil {
				t.Fatalf("SaveTrainingProfileInDB: %v", err)
			}
			got := maxes(t, user.ID)
			if len(got) != len(tc.want) {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
			for actionID, want := range tc.want {
				if got[actionID] != want {
					t.Fatalf("got %v, want %v", got, tc.want)
				}
			}
			stored, err := GetAllUserInformation(user.ID)
			if err != nil || stored.MaxHeartrate == nil || *stored.MaxHeartrate != 190 {
				t.Fatalf("user row not saved with the profile: %v %v", stored.MaxHeartrate, err)
			}
		})
	}

	if got := maxes(t, other.ID); got[run] != 200 || len(got) != 1 {
		t.Fatalf("another user's maxes changed: %v", got)
	}
}

func TestSaveTrainingProfileInDB_RollsBack(t *testing.T) {
	newTestDB(t)
	user := makeTestUser(t, "rollback@db.test", nil)
	run := uuid.New()
	if err := SaveTrainingProfileInDB(user, []models.SportMaxHeartrate{{ActionID: run, MaxHeartrate: 195}}); err != nil {
		t.Fatalf("seed: %v", err)
	}

	// A duplicate sport breaks the unique index mid-transaction: nothing may change.
	user.MaxHeartrate = intPtr(180)
	err := SaveTrainingProfileInDB(user, []models.SportMaxHeartrate{{ActionID: run, MaxHeartrate: 190}, {ActionID: run, MaxHeartrate: 191}})
	if err == nil {
		t.Fatal("duplicate sport should fail")
	}
	rows, _ := GetSportMaxHeartratesForUserID(user.ID)
	if len(rows) != 1 || rows[0].MaxHeartrate != 195 {
		t.Fatalf("sport maxes after a failed save = %+v, want the old 195 kept", rows)
	}
	if stored, _ := GetAllUserInformation(user.ID); stored.MaxHeartrate != nil {
		t.Fatalf("user max = %v after a failed save, want unchanged (nil)", *stored.MaxHeartrate)
	}
}

func TestGetObservedMaxHeartratesByActionForUserID(t *testing.T) {
	newTestDB(t)
	user := makeTestUser(t, "observed@db.test", nil)
	other := makeTestUser(t, "observed-other@db.test", nil)
	run, ride, swim := uuid.New(), uuid.New(), uuid.New()

	// seed adds one activity; the flags knock out one level of the enabled chain.
	type activity struct {
		userID                         uuid.UUID
		actionID                       *uuid.UUID
		maxHeartrate                   *int
		dayOff, sessionOff, opOff, off bool
	}
	seed := func(a activity) {
		t.Helper()
		day := models.ExerciseDay{Date: time.Now(), Enabled: true, UserID: &a.userID}
		day.ID = uuid.New()
		if err := Instance.Omit("User", "Goal").Create(&day).Error; err != nil {
			t.Fatalf("seed day: %v", err)
		}
		exercise := models.Exercise{Enabled: true, IsOn: true, ExerciseDayID: day.ID}
		exercise.ID = uuid.New()
		if err := Instance.Omit("ExerciseDay").Create(&exercise).Error; err != nil {
			t.Fatalf("seed exercise: %v", err)
		}
		operation := models.Operation{Enabled: true, ExerciseID: exercise.ID, ActionID: a.actionID, Type: "moving", DistanceUnit: "km", WeightUnit: "kg", MaxHeartrate: a.maxHeartrate}
		operation.ID = uuid.New()
		if err := Instance.Omit("Exercise", "Action", "Gear").Create(&operation).Error; err != nil {
			t.Fatalf("seed operation: %v", err)
		}
		// Flags are applied after Create: GORM skips false zero values on columns with a
		// default:true tag.
		if a.dayOff {
			Instance.Model(&models.ExerciseDay{}).Where("id = ?", day.ID).Update("enabled", false)
		}
		if a.sessionOff {
			Instance.Model(&models.Exercise{}).Where("id = ?", exercise.ID).Update("enabled", false)
		}
		if a.off {
			Instance.Model(&models.Exercise{}).Where("id = ?", exercise.ID).Update("is_on", false)
		}
		if a.opOff {
			Instance.Model(&models.Operation{}).Where("id = ?", operation.ID).Update("enabled", false)
		}
	}

	seed(activity{userID: user.ID, actionID: &run, maxHeartrate: intPtr(181)})
	seed(activity{userID: user.ID, actionID: &run, maxHeartrate: intPtr(193)})
	seed(activity{userID: user.ID, actionID: &run, maxHeartrate: intPtr(220), opOff: true})
	seed(activity{userID: user.ID, actionID: &run, maxHeartrate: intPtr(221), sessionOff: true})
	seed(activity{userID: user.ID, actionID: &run, maxHeartrate: intPtr(222), dayOff: true})
	seed(activity{userID: user.ID, actionID: &run, maxHeartrate: intPtr(223), off: true})
	seed(activity{userID: user.ID, actionID: &ride, maxHeartrate: intPtr(178)})
	seed(activity{userID: user.ID, actionID: &swim})                             // no HR data
	seed(activity{userID: user.ID, maxHeartrate: intPtr(199)})                   // no action
	seed(activity{userID: other.ID, actionID: &ride, maxHeartrate: intPtr(205)}) // someone else

	got, err := GetObservedMaxHeartratesByActionForUserID(user.ID)
	if err != nil {
		t.Fatalf("GetObservedMaxHeartratesByActionForUserID: %v", err)
	}
	if len(got) != 2 || got[run] != 193 || got[ride] != 178 {
		t.Fatalf("observed = %v, want run 193 and ride 178 only", got)
	}
}
