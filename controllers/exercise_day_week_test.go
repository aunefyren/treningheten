package controllers

import (
	"net/http"
	"testing"
	"time"

	"github.com/aunefyren/treningheten/database"
	"github.com/aunefyren/treningheten/models"

	"github.com/google/uuid"
)

func TestDayLoggableError(t *testing.T) {
	// A Wednesday at noon; its ISO week runs Monday 19 to Sunday 25 October 2026.
	now := time.Date(2026, time.October, 21, 12, 0, 0, 0, time.Local)

	tests := []struct {
		name    string
		date    time.Time
		wantErr string
	}{
		{"today", time.Date(2026, time.October, 21, 0, 0, 0, 0, time.Local), ""},
		{"today, later in the day", time.Date(2026, time.October, 21, 23, 0, 0, 0, time.Local), ""},
		{"Monday of the same week", time.Date(2026, time.October, 19, 0, 0, 0, 0, time.Local), ""},
		{"tomorrow", time.Date(2026, time.October, 22, 0, 0, 0, 0, time.Local), "You can't log workouts on days in the future."},
		{"Sunday of the same week", time.Date(2026, time.October, 25, 0, 0, 0, 0, time.Local), "You can't log workouts on days in the future."},
		{"Sunday of last week", time.Date(2026, time.October, 18, 0, 0, 0, 0, time.Local), "You can only add workouts to days in the current week."},
		{"Monday of next week", time.Date(2026, time.October, 26, 0, 0, 0, 0, time.Local), "You can only add workouts to days in the current week."},
		{"same week number, other year", time.Date(2025, time.October, 22, 0, 0, 0, 0, time.Local), "You can only add workouts to days in the current week."},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := dayLoggableError(test.date, now)
			got := ""
			if err != nil {
				got = err.Error()
			}
			if got != test.wantErr {
				t.Errorf("dayLoggableError(%s) = %q, want %q", test.date.Format("2006-01-02"), got, test.wantErr)
			}
		})
	}
}

// seedWeekDay inserts a day for the user at local midnight on the given date, holding
// `counted` workouts that count toward the goal and `uncounted` that don't.
func seedWeekDay(t *testing.T, userID uuid.UUID, date time.Time, counted int, uncounted int) models.ExerciseDay {
	t.Helper()

	day := models.ExerciseDay{
		Date:    time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.Local),
		Enabled: true,
		UserID:  &userID,
	}
	day.ID = uuid.New()
	if err := database.Instance.Omit("User", "Goal").Create(&day).Error; err != nil {
		t.Fatalf("failed to seed exercise day: %v", err)
	}

	for i := 0; i < counted+uncounted; i++ {
		exercise := models.Exercise{Enabled: true, IsOn: true, CountsTowardGoal: true, ExerciseDayID: day.ID}
		exercise.ID = uuid.New()
		if err := database.Instance.Omit("ExerciseDay").Create(&exercise).Error; err != nil {
			t.Fatalf("failed to seed exercise: %v", err)
		}
		// The column defaults to true, so a false has to be written after the insert.
		if i >= counted {
			if err := database.Instance.Model(&exercise).Update("counts_toward_goal", false).Error; err != nil {
				t.Fatalf("failed to mark exercise as not counting: %v", err)
			}
		}
	}

	return day
}

func todayAtMidnight() time.Time {
	now := time.Now()
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)
}

func weekFrom(t *testing.T, body map[string]any) map[string]any {
	t.Helper()
	week, ok := body["week"].(map[string]any)
	if !ok {
		t.Fatalf("response has no week object: %v", body)
	}
	return week
}

func TestExerciseDayWeekRejectsOthersAndBadIDs(t *testing.T) {
	h := newAPIHarness(t)
	owner, _ := h.user("owner@week.test", false)
	_, strangerToken := h.user("stranger@week.test", false)
	day := seedWeekDay(t, owner.ID, todayAtMidnight(), 1, 0)

	tests := []struct {
		name string
		path string
		want int
	}{
		{"another user's day", "/api/auth/exercise-days/" + day.ID.String() + "/week", http.StatusNotFound},
		{"unknown day", "/api/auth/exercise-days/" + uuid.New().String() + "/week", http.StatusNotFound},
		{"malformed ID", "/api/auth/exercise-days/not-a-uuid/week", http.StatusBadRequest},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			h.expect(test.want, "GET", test.path, strangerToken, nil)
		})
	}
}

func TestExerciseDayWeekCurrentWeek(t *testing.T) {
	h := newAPIHarness(t)
	user, token := h.user("current@week.test", false)
	today := todayAtMidnight()
	day := seedWeekDay(t, user.ID, today, 2, 1)

	week := weekFrom(t, h.ok("GET", "/api/auth/exercise-days/"+day.ID.String()+"/week", token, nil))

	if got := week["workouts"]; got != float64(2) {
		t.Errorf("workouts = %v, want 2 (the uncounted workout stays out)", got)
	}
	if got := week["workout_limit"]; got != float64(maxWorkoutsPerDay) {
		t.Errorf("workout_limit = %v, want %d", got, maxWorkoutsPerDay)
	}
	if got := week["loggable"]; got != true {
		t.Errorf("loggable = %v, want true for the current week", got)
	}

	days, _ := week["days"].([]any)
	if len(days) != 7 {
		t.Fatalf("days = %d, want 7", len(days))
	}
	for i, raw := range days {
		entry := raw.(map[string]any)
		date, err := time.Parse(time.RFC3339, entry["date"].(string))
		if err != nil {
			t.Fatalf("day %d: bad date %v", i, entry["date"])
		}
		local := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.Local)
		wantLoggable := !local.After(today)
		if entry["loggable"] != wantLoggable {
			t.Errorf("day %s: loggable = %v, want %v", local.Format("Mon"), entry["loggable"], wantLoggable)
		}

		if local.Equal(today) {
			if entry["id"] != day.ID.String() {
				t.Errorf("today: id = %v, want %s", entry["id"], day.ID)
			}
			if entry["workouts"] != float64(2) {
				t.Errorf("today: workouts = %v, want 2", entry["workouts"])
			}
		} else if entry["id"] != nil {
			t.Errorf("day %s: id = %v, want null where no row exists", local.Format("Mon"), entry["id"])
		}
	}

	if seasons, _ := week["seasons"].([]any); len(seasons) != 0 {
		t.Errorf("seasons = %v, want none without a goal", seasons)
	}
}

func TestExerciseDayWeekPastWeekIsSettled(t *testing.T) {
	h := newAPIHarness(t)
	user, token := h.user("past@week.test", false)
	day := seedWeekDay(t, user.ID, wednesdayOfWeeksAgo(2), 1, 0)

	week := weekFrom(t, h.ok("GET", "/api/auth/exercise-days/"+day.ID.String()+"/week", token, nil))

	if got := week["loggable"]; got != false {
		t.Errorf("loggable = %v, want false for a past week", got)
	}
	rows := 0
	for _, raw := range week["days"].([]any) {
		entry := raw.(map[string]any)
		if entry["loggable"] != false {
			t.Errorf("day %v: loggable = %v, want false", entry["date"], entry["loggable"])
		}
		if entry["id"] != nil {
			rows++
		}
	}
	if rows != 1 {
		t.Errorf("days with a row = %d, want 1", rows)
	}
}

func TestExerciseDayWeekSeasons(t *testing.T) {
	h := newAPIHarness(t)
	user, token := h.user("seasons@week.test", false)
	other, _ := h.user("other@week.test", false)

	spring := seedOngoingSeason(t, "Spring league", 3, 10)
	office := seedOngoingSeason(t, "Office cup", 3, 10)
	resting := seedOngoingSeason(t, "Winter series", 3, 10)
	notJoined := seedOngoingSeason(t, "Not joined", 3, 10)
	seedGoal(t, user.ID, spring, 4)
	seedGoal(t, user.ID, office, 2)
	restingGoal := seedGoal(t, user.ID, resting, 3)
	seedGoal(t, other.ID, notJoined, 3)

	date := wednesdayOfWeeksAgo(1)
	seedSickleave(t, restingGoal.ID, date, true)
	day := seedWeekDay(t, user.ID, date, 2, 1)

	// A goal joined after the week ended has no result for it.
	late := seedOngoingSeason(t, "Joined later", 3, 10)
	lateGoal := seedGoal(t, user.ID, late, 3)
	if err := database.Instance.Model(&lateGoal).Update("created_at", time.Now()).Error; err != nil {
		t.Fatalf("failed to move goal creation: %v", err)
	}

	week := weekFrom(t, h.ok("GET", "/api/auth/exercise-days/"+day.ID.String()+"/week", token, nil))

	type want struct {
		goal       float64
		completion float64
		sickLeave  bool
	}
	wanted := map[string]want{
		"Spring league": {4, 0.5, false},
		"Office cup":    {2, 1, false},
		"Winter series": {3, 2.0 / 3.0, true},
	}

	seasons := week["seasons"].([]any)
	if len(seasons) != len(wanted) {
		t.Fatalf("seasons = %v, want %d entries", seasons, len(wanted))
	}
	for _, raw := range seasons {
		entry := raw.(map[string]any)
		name := entry["name"].(string)
		w, ok := wanted[name]
		if !ok {
			t.Errorf("unexpected season %q", name)
			continue
		}
		t.Run(name, func(t *testing.T) {
			if entry["goal"] != w.goal {
				t.Errorf("goal = %v, want %v", entry["goal"], w.goal)
			}
			if entry["workouts"] != float64(2) {
				t.Errorf("workouts = %v, want 2", entry["workouts"])
			}
			if got := entry["completion"].(float64); got < w.completion-0.001 || got > w.completion+0.001 {
				t.Errorf("completion = %v, want %v", got, w.completion)
			}
			if entry["sick_leave"] != w.sickLeave {
				t.Errorf("sick_leave = %v, want %v", entry["sick_leave"], w.sickLeave)
			}
		})
	}
}
