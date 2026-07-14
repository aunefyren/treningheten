package database

import (
	"testing"
	"time"

	"github.com/aunefyren/treningheten/models"

	"github.com/google/uuid"
)

// mysqlTimestamp mirrors utilities.TimeToMySQLTimestamp (which cannot be imported here due
// to an import cycle) so the query date matches the format callers actually pass.
func mysqlTimestamp(pointInTime time.Time) string {
	return pointInTime.Format("2006-01-02 15:04:05.000")
}

// makeGoal inserts a goal (User/Season associations omitted) and returns it. Enabled carries a
// `default: true` tag, so it is forced via an explicit Update.
func makeGoal(t *testing.T, userID, seasonID uuid.UUID, enabled bool) models.Goal {
	t.Helper()
	goal := models.Goal{UserID: userID, SeasonID: seasonID, Enabled: enabled, Competing: true, ExerciseInterval: 3}
	goal.ID = uuid.New()
	if err := Instance.Omit("User", "Season").Create(&goal).Error; err != nil {
		t.Fatalf("failed to seed goal: %v", err)
	}
	if err := Instance.Model(&models.Goal{}).Where("id = ?", goal.ID).Update("enabled", enabled).Error; err != nil {
		t.Fatalf("failed to set goal enabled: %v", err)
	}
	return goal
}

func TestCreateGoalInDBAndGetGoalUsingGoalID(t *testing.T) {
	newTestDB(t)

	user := makeTestUser(t, "goaluser@example.com", nil)
	season := makeSeason(t, "GoalSeason", time.Now(), time.Now().Add(30*24*time.Hour), true)

	goal := models.Goal{UserID: user.ID, SeasonID: season.ID, Enabled: true, Competing: true, ExerciseInterval: 4}
	goal.ID = uuid.New()

	id, err := CreateGoalInDB(goal)
	if err != nil {
		t.Fatalf("CreateGoalInDB returned error: %v", err)
	}
	if id != goal.ID {
		t.Errorf("CreateGoalInDB returned id %v, want %v", id, goal.ID)
	}

	found, err := GetGoalUsingGoalID(id)
	if err != nil {
		t.Fatalf("GetGoalUsingGoalID returned error: %v", err)
	}
	if found.ExerciseInterval != 4 {
		t.Errorf("exercise interval: got %d, want 4", found.ExerciseInterval)
	}

	if _, err := GetGoalUsingGoalID(uuid.New()); err == nil {
		t.Errorf("expected error for unknown goal id")
	}
}

func TestVerifyUserGoalInSeason(t *testing.T) {
	newTestDB(t)

	user := makeTestUser(t, "verifygoal@example.com", nil)
	season := makeSeason(t, "VerifySeason", time.Now(), time.Now().Add(30*24*time.Hour), true)
	goal := makeGoal(t, user.ID, season.ID, true)

	has, goalID, err := VerifyUserGoalInSeason(user.ID, season.ID)
	if err != nil {
		t.Fatalf("VerifyUserGoalInSeason returned error: %v", err)
	}
	if !has || goalID != goal.ID {
		t.Errorf("expected goal found, got has=%v id=%v", has, goalID)
	}

	// Different season → no goal.
	has, _, err = VerifyUserGoalInSeason(user.ID, uuid.New())
	if err != nil {
		t.Fatalf("VerifyUserGoalInSeason(other season) returned error: %v", err)
	}
	if has {
		t.Errorf("expected no goal in unrelated season")
	}
}

func TestGetGoalsFromWithinSeasonAndGetGoalFromUser(t *testing.T) {
	newTestDB(t)

	userA := makeTestUser(t, "gwsA@example.com", nil)
	userB := makeTestUser(t, "gwsB@example.com", nil)
	season := makeSeason(t, "WithinSeason", time.Now(), time.Now().Add(30*24*time.Hour), true)

	makeGoal(t, userA.ID, season.ID, true)
	makeGoal(t, userB.ID, season.ID, true)
	// A disabled goal must be excluded.
	makeGoal(t, userA.ID, season.ID, false)

	goals, err := GetGoalsFromWithinSeason(season.ID)
	if err != nil {
		t.Fatalf("GetGoalsFromWithinSeason returned error: %v", err)
	}
	if len(goals) != 2 {
		t.Errorf("got %d goals in season, want 2", len(goals))
	}

	one, err := GetGoalFromUserWithinSeason(season.ID, userB.ID)
	if err != nil {
		t.Fatalf("GetGoalFromUserWithinSeason returned error: %v", err)
	}
	if one == nil || one.UserID != userB.ID {
		t.Errorf("expected user B's goal, got %v", one)
	}

	// User with no goal in the season → nil.
	none, err := GetGoalFromUserWithinSeason(season.ID, uuid.New())
	if err != nil {
		t.Fatalf("GetGoalFromUserWithinSeason(missing) returned error: %v", err)
	}
	if none != nil {
		t.Errorf("expected nil goal for user without one, got %v", none)
	}
}

func TestDisableGoalInDBUsingGoalID(t *testing.T) {
	newTestDB(t)

	user := makeTestUser(t, "disablegoal@example.com", nil)
	season := makeSeason(t, "DisableSeason", time.Now(), time.Now().Add(30*24*time.Hour), true)
	goal := makeGoal(t, user.ID, season.ID, true)

	if err := DisableGoalInDBUsingGoalID(goal.ID); err != nil {
		t.Fatalf("DisableGoalInDBUsingGoalID returned error: %v", err)
	}

	// Disabled goals are filtered out by GetGoalUsingGoalID.
	if _, err := GetGoalUsingGoalID(goal.ID); err == nil {
		t.Errorf("expected disabled goal to be unfindable")
	}
}

func TestGetGoalsForUserUsingUserID(t *testing.T) {
	newTestDB(t)

	user := makeTestUser(t, "goalsforuser@example.com", nil)
	enabledSeason := makeSeason(t, "EnabledS", time.Now(), time.Now().Add(30*24*time.Hour), true)
	disabledSeason := makeSeason(t, "DisabledS", time.Now(), time.Now().Add(30*24*time.Hour), false)

	makeGoal(t, user.ID, enabledSeason.ID, true)
	// Goal in a disabled season is excluded by the join.
	makeGoal(t, user.ID, disabledSeason.ID, true)

	goals, err := GetGoalsForUserUsingUserID(user.ID)
	if err != nil {
		t.Fatalf("GetGoalsForUserUsingUserID returned error: %v", err)
	}
	if len(goals) != 1 {
		t.Errorf("got %d goals, want 1 (disabled-season goal excluded)", len(goals))
	}
}

func TestGetActiveGoalsForUserIDAndDate(t *testing.T) {
	newTestDB(t)

	user := makeTestUser(t, "activegoal@example.com", nil)
	start := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)
	season := makeSeason(t, "ActiveSeason", start, end, true)
	makeGoal(t, user.ID, season.ID, true)

	inRange := mysqlTimestamp(time.Date(2024, 6, 15, 12, 0, 0, 0, time.UTC))
	active, err := GetActiveGoalsForUserIDAndDate(user.ID, inRange)
	if err != nil {
		t.Fatalf("GetActiveGoalsForUserIDAndDate(in range) returned error: %v", err)
	}
	if len(active) != 1 {
		t.Errorf("in-range date: got %d active goals, want 1", len(active))
	}

	afterEnd := mysqlTimestamp(time.Date(2025, 6, 15, 12, 0, 0, 0, time.UTC))
	none, err := GetActiveGoalsForUserIDAndDate(user.ID, afterEnd)
	if err != nil {
		t.Fatalf("GetActiveGoalsForUserIDAndDate(after end) returned error: %v", err)
	}
	if len(none) != 0 {
		t.Errorf("out-of-range date: got %d active goals, want 0", len(none))
	}
}
