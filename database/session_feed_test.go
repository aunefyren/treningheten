package database

import (
	"testing"
	"time"

	"github.com/aunefyren/treningheten/models"

	"github.com/google/uuid"
)

// makeBareSession inserts a session with no operations at all — what the front page's "+"
// button and the week calendar create. The feed used to be rooted in Operation, so these were
// invisible; they are the reason the feed is session-rooted.
func makeBareSession(t *testing.T, dayID uuid.UUID, at time.Time, duration *int64) models.Exercise {
	t.Helper()
	session := models.Exercise{ExerciseDayID: dayID, Enabled: true, IsOn: true, Time: &at, Duration: duration}
	session.ID = uuid.New()
	insertRow(t, &session)
	return session
}

func sessionByID(sessions []models.SessionFeedItem, id uuid.UUID) *models.SessionFeedItem {
	for i := range sessions {
		if sessions[i].ExerciseID == id {
			return &sessions[i]
		}
	}
	return nil
}

func TestSessionFeedIncludesSessionsWithoutActivities(t *testing.T) {
	newTestDB(t)
	user := makeTestUser(t, "bare@test.dev", nil)
	run := makeAction(t, "Run", "cardio")

	day := makeDay(t, user.ID, time.Date(2025, 5, 4, 0, 0, 0, 0, time.UTC))
	logged := makeSession(t, day.ID, time.Date(2025, 5, 4, 9, 0, 0, 0, time.UTC))
	loggedOp := makeOperation(t, logged.ID, &run.ID)
	makeSet(t, loggedOp.ID, f64Ptr(10), nil, nil, durPtr(3000))

	bare := makeBareSession(t, day.ID, time.Date(2025, 5, 4, 18, 0, 0, 0, time.UTC), durPtr(2700))

	sessions, total, err := GetSessionFeedForUser(user.ID, models.ActivityFeedFilter{Sort: "date", Order: "desc", Limit: 30})
	if err != nil {
		t.Fatalf("session feed error: %v", err)
	}
	if total != 2 || len(sessions) != 2 {
		t.Fatalf("both sessions should appear: got total %d / len %d", total, len(sessions))
	}

	bareItem := sessionByID(sessions, bare.ID)
	if bareItem == nil {
		t.Fatalf("the session with no activities is missing from the feed")
	}
	if bareItem.ActivityCount != 0 || len(bareItem.Activities) != 0 {
		t.Errorf("bare session should carry no activities, got %d", bareItem.ActivityCount)
	}
	// Its own duration is the only metric it has; the cascade must surface it.
	if bareItem.DurationSeconds != 2700 {
		t.Errorf("bare session duration: got %d, want 2700", bareItem.DurationSeconds)
	}

	loggedItem := sessionByID(sessions, logged.ID)
	if loggedItem == nil || loggedItem.ActivityCount != 1 {
		t.Fatalf("logged session should carry its one activity")
	}
	if loggedItem.Activities[0].ActionName != "Run" {
		t.Errorf("nested activity action: %q", loggedItem.Activities[0].ActionName)
	}
}

func TestSessionFeedAggregatesAcrossActivities(t *testing.T) {
	newTestDB(t)
	user := makeTestUser(t, "agg@test.dev", nil)
	run := makeAction(t, "Run", "cardio")
	lift := makeAction(t, "Lifting", "strength")

	day := makeDay(t, user.ID, time.Date(2025, 4, 12, 0, 0, 0, 0, time.UTC))
	session := makeSession(t, day.ID, time.Date(2025, 4, 12, 18, 0, 0, 0, time.UTC))

	runOp := makeOperation(t, session.ID, &run.ID)
	makeSet(t, runOp.ID, f64Ptr(8), nil, nil, durPtr(2400))
	liftOp := makeOperation(t, session.ID, &lift.ID)
	makeSet(t, liftOp.ID, nil, f64Ptr(60), f64Ptr(10), durPtr(600))
	makeSet(t, liftOp.ID, nil, f64Ptr(80), f64Ptr(8), durPtr(600))

	sessions, total, err := GetSessionFeedForUser(user.ID, models.ActivityFeedFilter{Sort: "date", Order: "desc", Limit: 30})
	if err != nil {
		t.Fatalf("session feed error: %v", err)
	}
	// One workout, not three exercise rows — the count is measured in sessions now.
	if total != 1 || len(sessions) != 1 {
		t.Fatalf("expected 1 session, got total %d / len %d", total, len(sessions))
	}

	item := sessions[0]
	if item.Distance != 8 {
		t.Errorf("session distance: %v", item.Distance)
	}
	if item.DurationSeconds != 3600 {
		t.Errorf("session duration should sum every set: got %d, want 3600", item.DurationSeconds)
	}
	if item.TopWeight != 80 {
		t.Errorf("session top weight: %v", item.TopWeight)
	}
	if item.Repetitions != 18 {
		t.Errorf("session reps: %v", item.Repetitions)
	}
	if item.SetCount != 3 {
		t.Errorf("session set count: %d", item.SetCount)
	}
	if item.ActivityCount != 2 || len(item.Activities) != 2 {
		t.Fatalf("session should carry both activities, got %d", item.ActivityCount)
	}
	// The nested rows keep their own per-activity metrics.
	byName := map[string]models.ActivityFeedItem{}
	for _, activity := range item.Activities {
		byName[activity.ActionName] = activity
	}
	if byName["Run"].Distance != 8 || byName["Lifting"].TopWeight != 80 {
		t.Errorf("nested activity metrics wrong: %+v", byName)
	}
	if byName["Lifting"].SessionActivityCount != 2 {
		t.Errorf("nested SessionActivityCount: %d, want 2", byName["Lifting"].SessionActivityCount)
	}
}

func TestSessionFeedFilterMatchesSessionButKeepsEveryActivity(t *testing.T) {
	newTestDB(t)
	user := makeTestUser(t, "filter@test.dev", nil)
	run := makeAction(t, "Run", "cardio")
	lift := makeAction(t, "Lifting", "strength")

	// A session that mixes a run and a lift, plus a lift-only session on another day.
	day1 := makeDay(t, user.ID, time.Date(2025, 4, 12, 0, 0, 0, 0, time.UTC))
	mixed := makeSession(t, day1.ID, time.Date(2025, 4, 12, 18, 0, 0, 0, time.UTC))
	mixedRun := makeOperation(t, mixed.ID, &run.ID)
	makeSet(t, mixedRun.ID, f64Ptr(5), nil, nil, durPtr(1800))
	makeOperation(t, mixed.ID, &lift.ID)

	day2 := makeDay(t, user.ID, time.Date(2025, 4, 13, 0, 0, 0, 0, time.UTC))
	liftOnly := makeSession(t, day2.ID, time.Date(2025, 4, 13, 7, 0, 0, 0, time.UTC))
	makeOperation(t, liftOnly.ID, &lift.ID)

	sessions, total, err := GetSessionFeedForUser(user.ID, models.ActivityFeedFilter{
		ActionID: &run.ID, Sort: "date", Order: "desc", Limit: 30,
	})
	if err != nil {
		t.Fatalf("session feed error: %v", err)
	}
	// Filtering by Run keeps only the session containing a run…
	if total != 1 || len(sessions) != 1 {
		t.Fatalf("expected 1 matching session, got total %d / len %d", total, len(sessions))
	}
	if sessions[0].ExerciseID != mixed.ID {
		t.Fatalf("matched the wrong session")
	}
	// …but the card still describes the whole workout: both activities, not just the run.
	if sessions[0].ActivityCount != 2 {
		t.Errorf("a filtered session must still list all its activities, got %d", sessions[0].ActivityCount)
	}
}

func TestSessionFeedDurationCascadePrefersSessionDuration(t *testing.T) {
	newTestDB(t)
	user := makeTestUser(t, "cascade@test.dev", nil)
	run := makeAction(t, "Run", "cardio")

	// A session that recorded its own duration, longer than its sets add up to. The session's
	// own number wins (same cascade as resolveSessionWindow) — and drives the duration sort.
	day := makeDay(t, user.ID, time.Date(2025, 5, 4, 0, 0, 0, 0, time.UTC))
	stated := makeBareSession(t, day.ID, time.Date(2025, 5, 4, 9, 0, 0, 0, time.UTC), durPtr(5400))
	statedOp := makeOperation(t, stated.ID, &run.ID)
	makeSet(t, statedOp.ID, f64Ptr(4), nil, nil, durPtr(1200))

	day2 := makeDay(t, user.ID, time.Date(2025, 5, 5, 0, 0, 0, 0, time.UTC))
	summed := makeSession(t, day2.ID, time.Date(2025, 5, 5, 9, 0, 0, 0, time.UTC))
	summedOp := makeOperation(t, summed.ID, &run.ID)
	makeSet(t, summedOp.ID, f64Ptr(9), nil, nil, durPtr(3000))

	sessions, _, err := GetSessionFeedForUser(user.ID, models.ActivityFeedFilter{
		Sort: "duration", Order: "desc", Limit: 30,
	})
	if err != nil {
		t.Fatalf("session feed error: %v", err)
	}
	if len(sessions) != 2 {
		t.Fatalf("expected 2 sessions, got %d", len(sessions))
	}
	if sessions[0].ExerciseID != stated.ID {
		t.Errorf("the session with the stated 5400s duration should sort first")
	}
	if sessions[0].DurationSeconds != 5400 {
		t.Errorf("stated duration should win over the summed sets: got %d", sessions[0].DurationSeconds)
	}
	if sessions[1].DurationSeconds != 3000 {
		t.Errorf("summed duration: got %d, want 3000", sessions[1].DurationSeconds)
	}
}

func TestSessionFeedPagesBySession(t *testing.T) {
	newTestDB(t)
	user := makeTestUser(t, "page@test.dev", nil)
	run := makeAction(t, "Run", "cardio")

	// Three sessions; the middle one holds three activities, so an operation-rooted page of
	// two would have returned a different slice than a session-rooted one.
	for i := 0; i < 3; i++ {
		day := makeDay(t, user.ID, time.Date(2025, 5, 1+i, 0, 0, 0, 0, time.UTC))
		session := makeSession(t, day.ID, time.Date(2025, 5, 1+i, 9, 0, 0, 0, time.UTC))
		activities := 1
		if i == 1 {
			activities = 3
		}
		for j := 0; j < activities; j++ {
			makeOperation(t, session.ID, &run.ID)
		}
	}

	sessions, total, err := GetSessionFeedForUser(user.ID, models.ActivityFeedFilter{
		Sort: "date", Order: "desc", Limit: 2, Offset: 0,
	})
	if err != nil {
		t.Fatalf("session feed error: %v", err)
	}
	if total != 3 {
		t.Errorf("total should count sessions, not activities: got %d, want 3", total)
	}
	if len(sessions) != 2 {
		t.Fatalf("limit should be measured in sessions: got %d", len(sessions))
	}
	if sessions[1].ActivityCount != 3 {
		t.Errorf("the multi-activity session should carry all 3: got %d", sessions[1].ActivityCount)
	}

	page2, _, err := GetSessionFeedForUser(user.ID, models.ActivityFeedFilter{
		Sort: "date", Order: "desc", Limit: 2, Offset: 2,
	})
	if err != nil {
		t.Fatalf("session feed error: %v", err)
	}
	if len(page2) != 1 {
		t.Fatalf("second page should hold the remaining session, got %d", len(page2))
	}
}
