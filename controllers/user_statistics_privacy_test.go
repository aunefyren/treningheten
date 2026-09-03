package controllers

import (
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/aunefyren/treningheten/auth"
	"github.com/aunefyren/treningheten/database"
	"github.com/aunefyren/treningheten/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// seedStatisticsSession inserts one session (with a single distance set) under its own
// exercise day, so each session is an independent operation in the statistics roll-up.
func seedStatisticsSession(t *testing.T, userID uuid.UUID, actionID uuid.UUID, date time.Time, distance float64, private bool) {
	t.Helper()

	day := models.ExerciseDay{Date: date, Enabled: true, UserID: &userID}
	day.ID = uuid.New()
	if err := database.Instance.Omit("User", "Goal").Create(&day).Error; err != nil {
		t.Fatalf("failed to seed exercise day: %v", err)
	}

	exercise := models.Exercise{Enabled: true, IsOn: true, ExerciseDayID: day.ID, Private: private}
	exercise.ID = uuid.New()
	if err := database.Instance.Omit("ExerciseDay").Create(&exercise).Error; err != nil {
		t.Fatalf("failed to seed exercise: %v", err)
	}

	operation := models.Operation{
		Enabled: true, ExerciseID: exercise.ID, ActionID: &actionID,
		Type: "moving", DistanceUnit: "km", WeightUnit: "kg",
	}
	operation.ID = uuid.New()
	if err := database.Instance.Omit("Exercise", "Action").Create(&operation).Error; err != nil {
		t.Fatalf("failed to seed operation: %v", err)
	}

	set := models.OperationSet{Enabled: true, OperationID: operation.ID, Distance: &distance}
	set.ID = uuid.New()
	if err := database.Instance.Omit("Operation").Create(&set).Error; err != nil {
		t.Fatalf("failed to seed operation set: %v", err)
	}
}

func seedStatisticsAction(t *testing.T, name string) uuid.UUID {
	t.Helper()

	action := models.Action{Enabled: true, Name: name, StravaName: name, Type: "moving"}
	action.ID = uuid.New()
	if err := database.Instance.Create(&action).Error; err != nil {
		t.Fatalf("failed to seed action: %v", err)
	}
	return action.ID
}

// getUserStatistics drives APIGetUserStatistics as `caller` against `target`.
func getUserStatistics(t *testing.T, callerID uuid.UUID, targetID uuid.UUID) *httptest.ResponseRecorder {
	t.Helper()

	token, _, err := auth.GenerateAccessToken(callerID, false, models.ScopeAPI, "test-client")
	if err != nil {
		t.Fatalf("failed to mint access token: %v", err)
	}

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest("GET", "/", nil)
	context.Request.Header.Set("Authorization", "Bearer "+token)
	context.Params = gin.Params{{Key: "user_id", Value: targetID.String()}}

	APIGetUserStatistics(context)
	return recorder
}

type statisticsResponse struct {
	Data models.UserStatisticsReply `json:"data"`
}

func decodeStatistics(t *testing.T, recorder *httptest.ResponseRecorder) models.UserStatisticsReply {
	t.Helper()

	if recorder.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", recorder.Code, recorder.Body.String())
	}
	var response statisticsResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to decode statistics: %v (body: %s)", err, recorder.Body.String())
	}
	return response.Data
}

// A private session still counts toward the totals — it counts toward the weekly goal and
// the season leaderboard, so excluding it would make the profile disagree with them — but
// it must never become the published "top", which names one session and its window.
func TestUserStatisticsTopsExcludePrivateSessions(t *testing.T) {
	newControllerTestDB(t)

	owner := createTestUser(t, "tops-owner@example.com", "Owner")
	viewer := createTestUser(t, "tops-viewer@example.com", "Viewer")
	actionID := seedStatisticsAction(t, "Running")

	recent := time.Now().AddDate(0, 0, -2)
	seedStatisticsSession(t, owner.ID, actionID, recent, 5, false)
	seedStatisticsSession(t, owner.ID, actionID, recent.AddDate(0, 0, -1), 7, false)
	// The longest run of the lot, but hidden.
	seedStatisticsSession(t, owner.ID, actionID, recent.AddDate(0, 0, -2), 42, true)

	statistics := decodeStatistics(t, getUserStatistics(t, viewer.ID, owner.ID))

	if got := statistics.ActivityStatistics.PastMonth.Tops.Distance; got != 7 {
		t.Errorf("past-month top distance = %v, want 7 (the private 42 km session must not surface)", got)
	}
	if got := statistics.ActivityStatistics.AllTime.Tops.Distance; got != 7 {
		t.Errorf("all-time top distance = %v, want 7", got)
	}

	// The session itself still counts, so the profile agrees with the leaderboard.
	if got := statistics.ActivityStatistics.AllTime.Sums.Operations; got != 3 {
		t.Errorf("all-time operations = %d, want 3 (private sessions still count)", got)
	}
	if got := statistics.ActivityStatistics.AllTime.Sums.Distance; got != 54 {
		t.Errorf("all-time distance sum = %v, want 54 (private sessions still count)", got)
	}
	if got := statistics.ExercisesAllTime; got != 3 {
		t.Errorf("exercises all time = %d, want 3", got)
	}
}

// A window below the floor is withheld whole, not just its averages. With one session the
// window's "distance", "best" and "total time" are that one workout said three ways — and
// visibly so, since distance and best are then the same number. The profile here has years
// of history, so the breakdown is headlined and reported; it is the thin recent month that
// drops out.
func TestUserStatisticsWindowWithheldBelowSampleFloor(t *testing.T) {
	newControllerTestDB(t)

	owner := createTestUser(t, "avg-owner@example.com", "Owner")
	viewer := createTestUser(t, "avg-viewer@example.com", "Viewer")
	actionID := seedStatisticsAction(t, "Walking")

	// Enough history to clear the all-time floor.
	for i := 0; i < userStatisticsMinSampleSize; i++ {
		seedStatisticsSession(t, owner.ID, actionID, time.Now().AddDate(0, 0, -90-i), 10, false)
	}
	// A single session this month — the case from the reported screenshot.
	seedStatisticsSession(t, owner.ID, actionID, time.Now().AddDate(0, 0, -2), 6, false)

	statistics := decodeStatistics(t, getUserStatistics(t, viewer.ID, owner.ID))

	if statistics.ActivityStatistics.PastMonth != nil {
		t.Errorf("expected the thin past-month window to be withheld, got distance %v / best %v",
			statistics.ActivityStatistics.PastMonth.Sums.Distance,
			statistics.ActivityStatistics.PastMonth.Tops.Distance)
	}

	// The windows that clear the floor are reported in full, averages included.
	if statistics.ActivityStatistics.AllTime == nil {
		t.Fatal("expected the all-time window to be reported")
	}
	if statistics.ActivityStatistics.AllTime.Averages.Distance == nil {
		t.Fatal("expected an all-time average once the sample floor is met")
	}
	// 3 history sessions at 10 km plus the single 6 km one.
	if got := *statistics.ActivityStatistics.AllTime.Averages.Distance; got != 9 {
		t.Errorf("all-time average distance = %v, want 9", got)
	}
	// The headline survives — a thin month must not cost the profile its breakdown.
	if statistics.ActivityStatistics.Action == nil {
		t.Error("expected the breakdown to keep its headline activity")
	}
}

// When even the all-time window is below the floor there is nothing to build a breakdown
// from, so the whole block goes — including any window that might otherwise have qualified.
// This is reachable with a headline pick: three activities of three different types clear
// the pick's floor while the most common of them has only one session.
func TestUserStatisticsNoBreakdownWhenAllTimeBelowFloor(t *testing.T) {
	newControllerTestDB(t)

	owner := createTestUser(t, "spread-owner@example.com", "Owner")
	viewer := createTestUser(t, "spread-viewer@example.com", "Viewer")

	for _, name := range []string{"Running", "Walking", "Cycling"} {
		actionID := seedStatisticsAction(t, name)
		seedStatisticsSession(t, owner.ID, actionID, time.Now().AddDate(0, 0, -2), 10, false)
	}

	statistics := decodeStatistics(t, getUserStatistics(t, viewer.ID, owner.ID))

	if statistics.ActivityStatistics.Action != nil {
		t.Errorf("expected no headline activity, got %q", statistics.ActivityStatistics.Action.Name)
	}
	if statistics.ActivityStatistics.AllTime != nil {
		t.Error("expected no all-time window when the headline activity has too few sessions")
	}
	if statistics.ActivityStatistics.PastMonth != nil {
		t.Error("expected no past-month window when there is no breakdown")
	}
	// The session counts are independent of the breakdown and stay on the page.
	if got := statistics.ExercisesAllTime; got != 3 {
		t.Errorf("exercises all time = %d, want 3", got)
	}
}

// publishWindow is the floor itself, exercised directly across the boundary.
func TestPublishWindow(t *testing.T) {
	cases := []struct {
		name        string
		operations  int64
		wantPublish bool
	}{
		{"empty window", 0, false},
		{"one below the floor", userStatisticsMinSampleSize - 1, false},
		{"exactly the floor", userStatisticsMinSampleSize, true},
		{"above the floor", userStatisticsMinSampleSize + 10, true},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			window := models.UserStatisticsCompilation{}
			window.Sums.Operations = testCase.operations
			window.Sums.Distance = 10 * float64(testCase.operations)
			window.Sums.Time = 60 * testCase.operations
			window.Sums.Weight = 5 * float64(testCase.operations)

			published := publishWindow(window)

			if !testCase.wantPublish {
				if published != nil {
					t.Fatalf("expected no window for %d operations, got one", testCase.operations)
				}
				return
			}

			if published == nil {
				t.Fatalf("expected a window for %d operations, got nil", testCase.operations)
			}
			if published.Averages.Distance == nil || published.Averages.Time == nil || published.Averages.Weight == nil {
				t.Fatalf("expected averages to be set for %d operations", testCase.operations)
			}
			if *published.Averages.Distance != 10 {
				t.Errorf("average distance = %v, want 10", *published.Averages.Distance)
			}
			if *published.Averages.Time != 60 {
				t.Errorf("average time = %v, want 60", *published.Averages.Time)
			}
			if *published.Averages.Weight != 5 {
				t.Errorf("average weight = %v, want 5", *published.Averages.Weight)
			}
		})
	}
}

// Below the floor there is nothing honest to headline — a pick made from one or two
// activities describes a workout, not a person — so the whole activity block is omitted and
// the client renders an explicit empty state instead.
func TestUserStatisticsNoHeadlineActionBelowSampleFloor(t *testing.T) {
	newControllerTestDB(t)

	owner := createTestUser(t, "headline-owner@example.com", "Owner")
	viewer := createTestUser(t, "headline-viewer@example.com", "Viewer")
	actionID := seedStatisticsAction(t, "Running")

	recent := time.Now().AddDate(0, 0, -2)
	for i := 0; i < userStatisticsMinSampleSize-1; i++ {
		seedStatisticsSession(t, owner.ID, actionID, recent.AddDate(0, 0, -i), 10, false)
	}

	statistics := decodeStatistics(t, getUserStatistics(t, viewer.ID, owner.ID))

	if statistics.ActivityStatistics.Action != nil {
		t.Errorf("expected no headline activity below the floor, got %q", statistics.ActivityStatistics.Action.Name)
	}
	// Nothing may be reported for a section that has no subject — this used to fall through
	// to a nameless block built from actionless operations.
	if statistics.ActivityStatistics.AllTime != nil {
		t.Errorf("expected no all-time window when there is no headline activity, got %d operations",
			statistics.ActivityStatistics.AllTime.Sums.Operations)
	}
	// The session counts are independent of the breakdown and stay on the page.
	if got := statistics.ExercisesAllTime; got != userStatisticsMinSampleSize-1 {
		t.Errorf("exercises all time = %d, want %d", got, userStatisticsMinSampleSize-1)
	}
	// The client renders the threshold in its empty state, so it has to come back.
	if got := statistics.MinimumSampleSize; got != userStatisticsMinSampleSize {
		t.Errorf("minimum sample size = %d, want %d", got, userStatisticsMinSampleSize)
	}
}

// A thin recent window falls back to the all-time pick rather than blanking the section:
// the block carries month/year/all-time together, so suppressing it outright would take
// years of statistics off the page to hide a quiet month.
func TestUserStatisticsHeadlineFallsBackToAllTime(t *testing.T) {
	newControllerTestDB(t)

	owner := createTestUser(t, "fallback-owner@example.com", "Owner")
	viewer := createTestUser(t, "fallback-viewer@example.com", "Viewer")
	runningID := seedStatisticsAction(t, "Running")
	liftingID := seedStatisticsAction(t, "Lifting")

	// One recent lift — on its own that would headline the profile "Lifting".
	seedStatisticsSession(t, owner.ID, liftingID, time.Now().AddDate(0, 0, -2), 1, false)
	// A long history of running, all outside the one-month recent window.
	for i := 0; i < userStatisticsMinSampleSize; i++ {
		seedStatisticsSession(t, owner.ID, runningID, time.Now().AddDate(0, 0, -90-i), 10, false)
	}

	statistics := decodeStatistics(t, getUserStatistics(t, viewer.ID, owner.ID))

	if statistics.ActivityStatistics.Action == nil {
		t.Fatal("expected the all-time pick to headline the section")
	}
	if got := statistics.ActivityStatistics.Action.Name; got != "Running" {
		t.Errorf("headline activity = %q, want Running (one recent lift must not headline)", got)
	}
	if got := statistics.ActivityStatistics.AllTime.Sums.Operations; got != int64(userStatisticsMinSampleSize) {
		t.Errorf("all-time operations = %d, want %d (only the headline activity counts)", got, userStatisticsMinSampleSize)
	}
	// The recent window holds no running at all, so it is withheld rather than reported
	// as a row of zeroes.
	if statistics.ActivityStatistics.PastMonth != nil {
		t.Error("expected the empty past-month window to be withheld")
	}
}

// chooseHeadlineAction is the pick itself, exercised directly across both boundaries.
func TestChooseHeadlineAction(t *testing.T) {
	recentSport := uuid.New()
	historicSport := uuid.New()

	repeat := func(id uuid.UUID, count int) []uuid.UUID {
		ids := make([]uuid.UUID, 0, count)
		for i := 0; i < count; i++ {
			ids = append(ids, id)
		}
		return ids
	}

	t.Run("prefers the recent window once it clears the floor", func(t *testing.T) {
		got := chooseHeadlineAction(
			repeat(recentSport, userStatisticsMinSampleSize),
			repeat(historicSport, 50),
		)
		if got == nil || *got != recentSport {
			t.Errorf("got %v, want the recent pick %v", got, recentSport)
		}
	})

	t.Run("falls back to all time when the recent window is thin", func(t *testing.T) {
		got := chooseHeadlineAction(
			repeat(recentSport, userStatisticsMinSampleSize-1),
			append(repeat(historicSport, userStatisticsMinSampleSize), recentSport),
		)
		if got == nil || *got != historicSport {
			t.Errorf("got %v, want the all-time pick %v", got, historicSport)
		}
	})

	t.Run("no pick when neither window clears the floor", func(t *testing.T) {
		if got := chooseHeadlineAction(repeat(recentSport, 1), repeat(recentSport, 1)); got != nil {
			t.Errorf("got %v, want nil", got)
		}
	})

	t.Run("no pick with nothing logged", func(t *testing.T) {
		if got := chooseHeadlineAction(nil, nil); got != nil {
			t.Errorf("got %v, want nil", got)
		}
	})
}
