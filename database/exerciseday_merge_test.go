package database

import (
	"testing"
	"time"

	"github.com/aunefyren/treningheten/models"

	"github.com/google/uuid"
)

// makeDayCreatedAt inserts an enabled exercise day with an explicit creation time, so
// tests control which of two duplicates is the oldest.
func makeDayCreatedAt(t *testing.T, userID uuid.UUID, goalID *uuid.UUID, date time.Time, note string, createdAt time.Time) models.ExerciseDay {
	t.Helper()
	day := models.ExerciseDay{Date: date, Note: note, Enabled: true, UserID: &userID, GoalID: goalID}
	day.ID = uuid.New()
	day.CreatedAt = createdAt
	insertRow(t, &day)
	return day
}

func makeSessionOn(t *testing.T, dayID uuid.UUID) models.Exercise {
	t.Helper()
	exercise := models.Exercise{ExerciseDayID: dayID, Enabled: true, IsOn: true, CountsTowardGoal: true}
	exercise.ID = uuid.New()
	insertRow(t, &exercise)
	return exercise
}

func loadDay(t *testing.T, id uuid.UUID) models.ExerciseDay {
	t.Helper()
	var day models.ExerciseDay
	if err := Instance.Where("id = ?", id).First(&day).Error; err != nil {
		t.Fatalf("failed to load day %s: %v", id, err)
	}
	return day
}

func dayOfSession(t *testing.T, id uuid.UUID) uuid.UUID {
	t.Helper()
	var exercise models.Exercise
	if err := Instance.Where("id = ?", id).First(&exercise).Error; err != nil {
		t.Fatalf("failed to load exercise %s: %v", id, err)
	}
	return exercise.ExerciseDayID
}

func TestMergeDuplicateExerciseDays(t *testing.T) {
	newTestDB(t)

	user := makeTestUser(t, "merge@example.com", nil)
	other := makeTestUser(t, "merge-other@example.com", nil)
	season := makeSeason(t, "MergeSeason", time.Now(), time.Now().Add(30*24*time.Hour), true)
	goal := makeGoal(t, user.ID, season.ID, true)

	date := time.Date(2026, 9, 28, 0, 0, 0, 0, time.UTC)
	base := time.Date(2026, 9, 28, 9, 0, 0, 0, time.UTC)

	keeper := makeDayCreatedAt(t, user.ID, nil, date, "Legs", base)
	duplicate := makeDayCreatedAt(t, user.ID, &goal.ID, date, "Felt strong", base.Add(time.Hour))
	keeperSession := makeSessionOn(t, keeper.ID)
	movedSession := makeSessionOn(t, duplicate.ID)

	// Not duplicates: another user on the same date, and the same user on another date.
	otherUserDay := makeDayCreatedAt(t, other.ID, nil, date, "", base)
	nextDay := makeDayCreatedAt(t, user.ID, nil, date.AddDate(0, 0, 1), "", base)

	mergeDuplicateExerciseDays()

	if got := dayOfSession(t, movedSession.ID); got != keeper.ID {
		t.Errorf("duplicate's session is on day %s, want the keeper %s", got, keeper.ID)
	}
	if got := dayOfSession(t, keeperSession.ID); got != keeper.ID {
		t.Errorf("keeper's session moved to %s", got)
	}

	merged := loadDay(t, keeper.ID)
	if !merged.Enabled {
		t.Error("keeper was disabled")
	}
	if merged.Note != "Legs\nFelt strong" {
		t.Errorf("keeper note = %q, want both notes", merged.Note)
	}
	if merged.GoalID == nil || *merged.GoalID != goal.ID {
		t.Errorf("keeper goal = %v, want the duplicate's goal %s", merged.GoalID, goal.ID)
	}
	if loadDay(t, duplicate.ID).Enabled {
		t.Error("duplicate is still enabled")
	}
	for _, untouched := range []models.ExerciseDay{otherUserDay, nextDay} {
		if !loadDay(t, untouched.ID).Enabled {
			t.Errorf("non-duplicate day %s was disabled", untouched.ID)
		}
	}

	// A second run finds nothing left to merge and changes nothing.
	mergeDuplicateExerciseDays()
	if again := loadDay(t, keeper.ID); again.Note != merged.Note || !again.Enabled {
		t.Errorf("second run changed the keeper: %+v", again)
	}
}

func TestGroupDuplicateExerciseDays(t *testing.T) {
	userA, userB := uuid.New(), uuid.New()
	monday := time.Date(2026, 9, 28, 0, 0, 0, 0, time.UTC)
	day := func(user *uuid.UUID, date time.Time) models.ExerciseDay {
		d := models.ExerciseDay{Date: date, UserID: user}
		d.ID = uuid.New()
		return d
	}

	first, second := day(&userA, monday), day(&userA, monday.Add(8*time.Hour))
	days := []models.ExerciseDay{first, day(&userB, monday), second, day(&userA, monday.AddDate(0, 0, 1)), day(nil, monday)}

	groups := groupDuplicateExerciseDays(days)
	if len(groups) != 1 {
		t.Fatalf("groups = %d, want 1", len(groups))
	}
	if len(groups[0]) != 2 || groups[0][0].ID != first.ID || groups[0][1].ID != second.ID {
		t.Errorf("group = %v, want [first second] in input order", groups[0])
	}
}

func TestMergeExerciseDayNotes(t *testing.T) {
	tests := []struct {
		name, note, addition, want string
	}{
		{"both empty", "", "", ""},
		{"empty addition", "Legs", "  ", "Legs"},
		{"empty note", "", "Legs", "Legs"},
		{"repeat", "Legs", " Legs ", "Legs"},
		{"append", "Legs", "Arms", "Legs\nArms"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := mergeExerciseDayNotes(tt.note, tt.addition); got != tt.want {
				t.Errorf("mergeExerciseDayNotes(%q, %q) = %q, want %q", tt.note, tt.addition, got, tt.want)
			}
		})
	}
}

// With duplicates present every single-day lookup returns the oldest day, so callers that
// "find or create" a day keep reusing it instead of adding yet another one.
func TestExerciseDayLookupsPreferOldestDuplicate(t *testing.T) {
	newTestDB(t)

	user := makeTestUser(t, "oldest@example.com", nil)
	season := makeSeason(t, "OldestSeason", time.Now(), time.Now().Add(30*24*time.Hour), true)
	goal := makeGoal(t, user.ID, season.ID, true)
	date := time.Date(2026, 9, 28, 0, 0, 0, 0, time.UTC)
	base := time.Date(2026, 9, 28, 9, 0, 0, 0, time.UTC)

	// Newer inserted first: Find into a struct takes the first row, and an unordered query
	// returns rows in insertion order, so without the ordering the newer day would win.
	makeDayCreatedAt(t, user.ID, &goal.ID, date, "", base.Add(time.Hour))
	oldest := makeDayCreatedAt(t, user.ID, &goal.ID, date, "", base)

	lookups := map[string]func() (*models.ExerciseDay, error){
		"GetExerciseDayByGoalAndDate":   func() (*models.ExerciseDay, error) { return GetExerciseDayByGoalAndDate(goal.ID, date) },
		"GetExerciseDayByUserIDAndDate": func() (*models.ExerciseDay, error) { return GetExerciseDayByUserIDAndDate(user.ID, date) },
		"GetExerciseDayByDateAndGoal":   func() (*models.ExerciseDay, error) { return GetExerciseDayByDateAndGoal(goal.ID, date) },
		"GetExerciseDayByDateAndUserID": func() (*models.ExerciseDay, error) { return GetExerciseDayByDateAndUserID(user.ID, date) },
	}
	for name, lookup := range lookups {
		t.Run(name, func(t *testing.T) {
			day, err := lookup()
			if err != nil {
				t.Fatalf("error: %v", err)
			}
			if day == nil || day.ID != oldest.ID {
				t.Errorf("got %v, want the oldest day %s", day, oldest.ID)
			}
		})
	}
}
