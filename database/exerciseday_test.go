package database

import (
	"testing"
	"time"

	"github.com/aunefyren/treningheten/models"

	"github.com/google/uuid"
)

// makeDayFor inserts an exercise day for the given user and (optional) goal and returns it.
func makeDayFor(t *testing.T, userID uuid.UUID, goalID *uuid.UUID, date time.Time) models.ExerciseDay {
	t.Helper()
	day := models.ExerciseDay{Date: date, Enabled: true, UserID: &userID, GoalID: goalID}
	day.ID = uuid.New()
	insertRow(t, &day)
	return day
}

// dayBounds returns the start/end string bounds a whole-day query uses for a given date.
func dayBounds(date time.Time) (string, string) {
	return date.Format("2006-01-02") + " 00:00:00.000", date.Format("2006-01-02") + " 23:59:59"
}

func TestCreateAndGetExerciseDayByID(t *testing.T) {
	newTestDB(t)

	user := makeTestUser(t, "edcreate@example.com", nil)
	day := models.ExerciseDay{Date: time.Now(), Enabled: true, UserID: &user.ID}
	day.ID = uuid.New()

	id, err := CreateExerciseDayForGoalInDatabase(day)
	if err != nil {
		t.Fatalf("CreateExerciseDayForGoalInDatabase returned error: %v", err)
	}
	if id != day.ID {
		t.Errorf("returned id %v, want %v", id, day.ID)
	}

	found, err := GetExerciseDayByID(id)
	if err != nil {
		t.Fatalf("GetExerciseDayByID returned error: %v", err)
	}
	if found == nil || found.ID != id {
		t.Errorf("expected to find exercise day, got %v", found)
	}

	// Unknown id → no error and an empty (zero-ID) record. GetExerciseDayByID leaves the
	// destination untouched on no rows, so callers see a zero struct rather than nil.
	missing, err := GetExerciseDayByID(uuid.New())
	if err != nil {
		t.Fatalf("GetExerciseDayByID(missing) returned error: %v", err)
	}
	if missing != nil && missing.ID != uuid.Nil {
		t.Errorf("expected empty record for unknown exercise day, got %v", missing)
	}
}

func TestCreateExerciseDayInDBAndGetAll(t *testing.T) {
	newTestDB(t)

	user := makeTestUser(t, "edall@example.com", nil)
	day := models.ExerciseDay{Date: time.Now(), Enabled: true, UserID: &user.ID}
	day.ID = uuid.New()
	if err := CreateExerciseDayInDB(day); err != nil {
		t.Fatalf("CreateExerciseDayInDB returned error: %v", err)
	}

	all, err := GetAllExerciseDays()
	if err != nil {
		t.Fatalf("GetAllExerciseDays returned error: %v", err)
	}
	if len(all) != 1 {
		t.Errorf("got %d exercise days, want 1", len(all))
	}
}

func TestGetExerciseDayByDateLookups(t *testing.T) {
	newTestDB(t)

	user := makeTestUser(t, "eddate@example.com", nil)
	season := makeSeason(t, "EDSeason", time.Now(), time.Now().Add(30*24*time.Hour), true)
	goal := makeGoal(t, user.ID, season.ID, true)
	date := time.Date(2024, 5, 10, 8, 0, 0, 0, time.UTC)
	makeDayFor(t, user.ID, &goal.ID, date)

	byGoal, err := GetExerciseDayByGoalAndDate(goal.ID, date)
	if err != nil || byGoal == nil {
		t.Fatalf("GetExerciseDayByGoalAndDate: err=%v day=%v", err, byGoal)
	}
	byUser, err := GetExerciseDayByUserIDAndDate(user.ID, date)
	if err != nil || byUser == nil {
		t.Fatalf("GetExerciseDayByUserIDAndDate: err=%v day=%v", err, byUser)
	}
	byDateGoal, err := GetExerciseDayByDateAndGoal(goal.ID, date)
	if err != nil || byDateGoal == nil {
		t.Fatalf("GetExerciseDayByDateAndGoal: err=%v day=%v", err, byDateGoal)
	}
	byDateUser, err := GetExerciseDayByDateAndUserID(user.ID, date)
	if err != nil || byDateUser == nil {
		t.Fatalf("GetExerciseDayByDateAndUserID: err=%v day=%v", err, byDateUser)
	}

	// A different day → no result.
	otherDay := time.Date(2024, 5, 11, 8, 0, 0, 0, time.UTC)
	none, err := GetExerciseDayByUserIDAndDate(user.ID, otherDay)
	if err != nil {
		t.Fatalf("GetExerciseDayByUserIDAndDate(other) returned error: %v", err)
	}
	if none != nil {
		t.Errorf("expected nil for a date with no day, got %v", none)
	}
}

func TestGetExerciseDayByIDAndUserID(t *testing.T) {
	newTestDB(t)

	user := makeTestUser(t, "edbyiduser@example.com", nil)
	stranger := makeTestUser(t, "edstranger@example.com", nil)
	day := makeDayFor(t, user.ID, nil, time.Now())

	found, err := GetExerciseDayByIDAndUserID(day.ID, user.ID)
	if err != nil {
		t.Fatalf("GetExerciseDayByIDAndUserID returned error: %v", err)
	}
	if found == nil || found.ID != day.ID {
		t.Errorf("expected to find day for owner, got %v", found)
	}

	// Scoped: another user cannot read it (nil, nil).
	other, err := GetExerciseDayByIDAndUserID(day.ID, stranger.ID)
	if err != nil {
		t.Fatalf("GetExerciseDayByIDAndUserID(stranger) returned error: %v", err)
	}
	if other != nil {
		t.Errorf("expected nil reading another user's day, got %v", other)
	}
}

func TestUpdateExerciseDaySaveAndNote(t *testing.T) {
	newTestDB(t)

	user := makeTestUser(t, "edupdate@example.com", nil)
	season := makeSeason(t, "EDUpdSeason", time.Now(), time.Now().Add(30*24*time.Hour), true)
	goal := makeGoal(t, user.ID, season.ID, true)
	date := time.Date(2024, 7, 1, 8, 0, 0, 0, time.UTC)
	day := makeDayFor(t, user.ID, &goal.ID, date)

	// Save-based updates.
	day.Note = "saved note"
	if _, err := UpdateExerciseDayInDatabase(day); err != nil {
		t.Fatalf("UpdateExerciseDayInDatabase returned error: %v", err)
	}
	day.Note = "second note"
	if _, err := UpdateExerciseDayInDB(day); err != nil {
		t.Fatalf("UpdateExerciseDayInDB returned error: %v", err)
	}

	// Targeted note update by goal + date range.
	start, end := dayBounds(date)
	if err := UpdateExerciseDayNoteInDatabase(goal.ID, start, end, "range note"); err != nil {
		t.Fatalf("UpdateExerciseDayNoteInDatabase returned error: %v", err)
	}
	reloaded, err := GetExerciseDayByID(day.ID)
	if err != nil || reloaded == nil {
		t.Fatalf("failed to reload day: err=%v", err)
	}
	if reloaded.Note != "range note" {
		t.Errorf("note: got %q, want %q", reloaded.Note, "range note")
	}

	// No matching day → error (RowsAffected != 1).
	if err := UpdateExerciseDayNoteInDatabase(uuid.New(), start, end, "x"); err == nil {
		t.Errorf("expected error updating note for unknown goal")
	}
}

func TestGetExerciseDaysBetweenDates(t *testing.T) {
	newTestDB(t)

	user := makeTestUser(t, "edbetween@example.com", nil)
	season := makeSeason(t, "EDBetSeason", time.Now(), time.Now().Add(30*24*time.Hour), true)
	goal := makeGoal(t, user.ID, season.ID, true)

	rangeStart := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	rangeEnd := time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC)
	makeDayFor(t, user.ID, &goal.ID, time.Date(2024, 1, 10, 8, 0, 0, 0, time.UTC)) // in range
	makeDayFor(t, user.ID, &goal.ID, time.Date(2024, 2, 10, 8, 0, 0, 0, time.UTC)) // out of range

	byGoal, err := GetExerciseDaysBetweenDatesUsingDates(goal.ID, rangeStart, rangeEnd)
	if err != nil {
		t.Fatalf("GetExerciseDaysBetweenDatesUsingDates returned error: %v", err)
	}
	if len(byGoal) != 1 {
		t.Errorf("by goal: got %d days, want 1", len(byGoal))
	}

	byUser, err := GetExerciseDaysBetweenDatesUsingDatesAndUserID(user.ID, rangeStart, rangeEnd)
	if err != nil {
		t.Fatalf("GetExerciseDaysBetweenDatesUsingDatesAndUserID returned error: %v", err)
	}
	if len(byUser) != 1 {
		t.Errorf("by user: got %d days, want 1", len(byUser))
	}
}

func TestGetExerciseDaysForUserAndGoal(t *testing.T) {
	newTestDB(t)

	user := makeTestUser(t, "eduseryear@example.com", nil)
	season := makeSeason(t, "EDYearSeason", time.Now(), time.Now().Add(30*24*time.Hour), true)
	goal := makeGoal(t, user.ID, season.ID, true)

	makeDayFor(t, user.ID, &goal.ID, time.Date(2024, 6, 15, 8, 0, 0, 0, time.UTC))
	makeDayFor(t, user.ID, &goal.ID, time.Date(2024, 8, 20, 8, 0, 0, 0, time.UTC))
	makeDayFor(t, user.ID, &goal.ID, time.Date(2023, 3, 5, 8, 0, 0, 0, time.UTC))

	allForUser, err := GetExerciseDaysForUserUsingUserID(user.ID)
	if err != nil {
		t.Fatalf("GetExerciseDaysForUserUsingUserID returned error: %v", err)
	}
	if len(allForUser) != 3 {
		t.Errorf("all days: got %d, want 3", len(allForUser))
	}

	byGoal, err := GetExerciseDaysForUserUsingUserIDAndGoalID(user.ID, goal.ID)
	if err != nil {
		t.Fatalf("GetExerciseDaysForUserUsingUserIDAndGoalID returned error: %v", err)
	}
	if len(byGoal) != 3 {
		t.Errorf("by goal: got %d days, want 3", len(byGoal))
	}
}

func TestGetAllEnabledExerciseDays(t *testing.T) {
	newTestDB(t)

	user := makeTestUser(t, "edenabled@example.com", nil)
	enabledSeason := makeSeason(t, "EnabledEDSeason", time.Now(), time.Now().Add(30*24*time.Hour), true)
	disabledSeason := makeSeason(t, "DisabledEDSeason", time.Now(), time.Now().Add(30*24*time.Hour), false)
	enabledGoal := makeGoal(t, user.ID, enabledSeason.ID, true)
	disabledSeasonGoal := makeGoal(t, user.ID, disabledSeason.ID, true)

	makeDayFor(t, user.ID, &enabledGoal.ID, time.Now())
	// Day under a goal whose season is disabled → excluded by the join.
	makeDayFor(t, user.ID, &disabledSeasonGoal.ID, time.Now())

	days, err := GetAllEnabledExerciseDays()
	if err != nil {
		t.Fatalf("GetAllEnabledExerciseDays returned error: %v", err)
	}
	if len(days) != 1 {
		t.Errorf("got %d fully-enabled days, want 1", len(days))
	}
}

func TestGetValidExercisesBetweenDatesUsingDatesByUserID(t *testing.T) {
	newTestDB(t)

	user := makeTestUser(t, "edvalid@example.com", nil)
	rangeStart := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	rangeEnd := time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC)

	day := makeDayFor(t, user.ID, nil, time.Date(2024, 1, 10, 8, 0, 0, 0, time.UTC))
	makeSession(t, day.ID, time.Now())   // valid, on
	seedExercise(t, day.ID, true, false) // off → excluded
	seedExercise(t, day.ID, false, true) // disabled → excluded

	// A day out of range.
	outDay := makeDayFor(t, user.ID, nil, time.Date(2024, 2, 10, 8, 0, 0, 0, time.UTC))
	makeSession(t, outDay.ID, time.Now())

	valid, err := GetValidExercisesBetweenDatesUsingDatesByUserID(user.ID, rangeStart, rangeEnd)
	if err != nil {
		t.Fatalf("GetValidExercisesBetweenDatesUsingDatesByUserID returned error: %v", err)
	}
	if len(valid) != 1 {
		t.Errorf("got %d valid exercises, want 1", len(valid))
	}
}

func TestGetExerciseDaysForSharingUsersUsingDates(t *testing.T) {
	newTestDB(t)

	sharing := makeTestUser(t, "edsharing@example.com", func(u *models.User) { u.ShareActivities = boolPtr(true) })
	private := makeTestUser(t, "edprivate@example.com", func(u *models.User) { u.ShareActivities = boolPtr(false) })

	rangeStart := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	rangeEnd := time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC)
	inRange := time.Date(2024, 1, 15, 8, 0, 0, 0, time.UTC)

	makeDayFor(t, sharing.ID, nil, inRange)
	makeDayFor(t, private.ID, nil, inRange) // excluded: not sharing

	days, err := GetExerciseDaysForSharingUsersUsingDates(rangeStart, rangeEnd)
	if err != nil {
		t.Fatalf("GetExerciseDaysForSharingUsersUsingDates returned error: %v", err)
	}
	if len(days) != 1 {
		t.Errorf("got %d shared days, want 1 (private user excluded)", len(days))
	}
}
