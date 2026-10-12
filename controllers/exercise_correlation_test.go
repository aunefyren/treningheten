package controllers

import (
	"net/http"
	"reflect"
	"testing"
	"time"

	"github.com/aunefyren/treningheten/database"
	"github.com/aunefyren/treningheten/models"

	"github.com/google/uuid"
)

// correlationSession builds an ExerciseObject for the planner tests. Sessions are enabled
// unless a test says otherwise; the GORM getters only ever return enabled rows.
func correlationSession(id uuid.UUID, isOn bool, countsTowardGoal bool, operations int, strava bool, hevy bool) models.ExerciseObject {
	session := models.ExerciseObject{
		Enabled:          true,
		IsOn:             isOn,
		CountsTowardGoal: countsTowardGoal,
		Operations:       make([]models.OperationObject, operations),
		StravaID:         []string{},
	}
	session.ID = id
	if strava {
		session.StravaID = []string{"123"}
	}
	if hevy {
		session.HevyWorkoutID = stringPtr("hevy-workout")
	}
	return session
}

func TestPlanSessionCorrelation(t *testing.T) {
	emptyOn, emptyOff, manualOn, manualOff, stravaOn, stravaOff, walkOn, hevyOn, hevyOff, offNotCounting :=
		uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New()

	tests := []struct {
		name      string
		exercises []models.ExerciseObject
		target    int
		want      sessionCorrelationPlan
	}{
		{
			name:   "already at target",
			target: 1, exercises: []models.ExerciseObject{correlationSession(emptyOn, true, true, 0, false, false)},
			want: sessionCorrelationPlan{},
		},
		{
			// The production bug: a Strava walk that is on but doesn't count must not
			// satisfy a +1, so a new session is created.
			name:   "non-counting session does not satisfy +1",
			target: 1, exercises: []models.ExerciseObject{correlationSession(walkOn, true, false, 1, true, false)},
			want: sessionCorrelationPlan{create: 1},
		},
		{
			name:   "empty day creates sessions",
			target: 2, exercises: []models.ExerciseObject{},
			want: sessionCorrelationPlan{create: 2},
		},
		{
			name:   "reuses an off empty manual session before creating",
			target: 2, exercises: []models.ExerciseObject{
				correlationSession(emptyOff, false, true, 0, false, false),
				correlationSession(stravaOn, true, true, 1, true, false),
			},
			want: sessionCorrelationPlan{turnOn: []uuid.UUID{emptyOff}},
		},
		{
			name:   "never restores deleted sessions with activities or imports",
			target: 1, exercises: []models.ExerciseObject{
				correlationSession(manualOff, false, true, 2, false, false),
				correlationSession(stravaOff, false, true, 1, true, false),
				correlationSession(hevyOff, false, true, 0, false, true),
				correlationSession(offNotCounting, false, false, 0, false, false),
			},
			want: sessionCorrelationPlan{create: 1},
		},
		{
			name:   "lowering switches off empty, then manual, then imported",
			target: 1, exercises: []models.ExerciseObject{
				correlationSession(stravaOn, true, true, 1, true, false),
				correlationSession(manualOn, true, true, 2, false, false),
				correlationSession(emptyOn, true, true, 0, false, false),
			},
			want: sessionCorrelationPlan{turnOff: []uuid.UUID{emptyOn, manualOn}},
		},
		{
			name:   "lowering to zero reaches imports last and leaves non-counting sessions alone",
			target: 0, exercises: []models.ExerciseObject{
				correlationSession(hevyOn, true, true, 1, false, true),
				correlationSession(walkOn, true, false, 1, true, false),
				correlationSession(manualOn, true, true, 1, false, false),
			},
			want: sessionCorrelationPlan{turnOff: []uuid.UUID{manualOn, hevyOn}},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := planSessionCorrelation(test.exercises, test.target)
			if !reflect.DeepEqual(got, test.want) {
				t.Errorf("planSessionCorrelation() = %+v, want %+v", got, test.want)
			}
		})
	}
}

func TestSessionRemovalRank(t *testing.T) {
	tests := []struct {
		name    string
		session models.ExerciseObject
		want    int
	}{
		{"empty manual", correlationSession(uuid.New(), true, true, 0, false, false), 0},
		{"manual with activities", correlationSession(uuid.New(), true, true, 1, false, false), 1},
		{"strava", correlationSession(uuid.New(), true, true, 1, true, false), 2},
		{"hevy", correlationSession(uuid.New(), true, true, 0, false, true), 2},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := sessionRemovalRank(test.session); got != test.want {
				t.Errorf("sessionRemovalRank() = %d, want %d", got, test.want)
			}
		})
	}
}

func TestIsReusableEmptySession(t *testing.T) {
	disabled := correlationSession(uuid.New(), false, true, 0, false, false)
	disabled.Enabled = false

	tests := []struct {
		name    string
		session models.ExerciseObject
		want    bool
	}{
		{"off empty manual counting", correlationSession(uuid.New(), false, true, 0, false, false), true},
		{"already on", correlationSession(uuid.New(), true, true, 0, false, false), false},
		{"not counting", correlationSession(uuid.New(), false, false, 0, false, false), false},
		{"has activities", correlationSession(uuid.New(), false, true, 1, false, false), false},
		{"imported", correlationSession(uuid.New(), false, true, 0, true, false), false},
		{"disabled", disabled, false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := isReusableEmptySession(test.session); got != test.want {
				t.Errorf("isReusableEmptySession() = %v, want %v", got, test.want)
			}
		})
	}
}

// TestRegisterWeekWithNonCountingSession is the end-to-end regression for the front-page
// calendar: today holds a session that is on but doesn't count (an excluded Strava walk), so
// the calendar shows 0. A +1 must add a counting session instead of being absorbed by the
// walk, a -1 must not switch the walk off, and a second +1 reuses the session it switched off.
func TestRegisterWeekWithNonCountingSession(t *testing.T) {
	h := newAPIHarness(t)
	_, token := h.user("calendar@workout.test", false)

	day := h.ok("GET", "/api/auth/exercise-days/week?today=true", token, nil)
	dayID := mustUUID(t, idOf(t, day, "exercise"))

	now := time.Now()
	walk := models.Exercise{ExerciseDayID: dayID, Enabled: true, IsOn: true, Time: &now}
	walk.ID = uuid.New()
	if err := database.CreateExerciseForExerciseDayInDatabase(walk); err != nil {
		t.Fatalf("failed to seed walk: %v", err)
	}
	if err := database.SetExerciseCountsTowardGoal(walk.ID, false); err != nil {
		t.Fatalf("failed to exclude walk: %v", err)
	}

	todayIndex := (int(now.Weekday()) + 6) % 7
	registerToday := func(interval int) (exerciseInterval int, sessions []any) {
		t.Helper()
		days := currentWeekDays(0)
		days[todayIndex].ExerciseInterval = interval
		response := h.expect(http.StatusCreated, "POST", "/api/auth/exercises/week", token, models.WeekCreationRequest{Days: days, TimeZone: "UTC"})
		today := field(t, response, "week", "days").([]any)[todayIndex]
		return int(field(t, today, "exercise_interval").(float64)), field(t, today, "exercises").([]any)
	}
	isOn := func(sessions []any, id uuid.UUID) bool {
		t.Helper()
		for _, session := range sessions {
			if field(t, session, "id").(string) == id.String() {
				return field(t, session, "is_on").(bool)
			}
		}
		t.Fatalf("session %s not found", id)
		return false
	}

	interval, sessions := registerToday(1)
	if interval != 1 || len(sessions) != 2 {
		t.Fatalf("after +1: interval = %d, sessions = %d, want 1 and 2", interval, len(sessions))
	}

	interval, sessions = registerToday(0)
	if interval != 0 {
		t.Errorf("after -1: interval = %d, want 0", interval)
	}
	if !isOn(sessions, walk.ID) {
		t.Errorf("lowering the count switched off the non-counting walk")
	}

	interval, sessions = registerToday(1)
	if interval != 1 || len(sessions) != 2 {
		t.Errorf("after second +1: interval = %d, sessions = %d, want 1 and 2 (reuse, not create)", interval, len(sessions))
	}
}

// TestCreateExerciseFromBuilderPayload pins the exact JSON the builder and front page send
// to POST /exercises. They once sent "on" instead of "is_on", so every new session was
// stored switched off and rendered as the text "null".
func TestCreateExerciseFromBuilderPayload(t *testing.T) {
	h := newAPIHarness(t)
	_, token := h.user("builder@workout.test", false)

	day := h.ok("GET", "/api/auth/exercise-days/week?today=true", token, nil)
	created := h.expect(http.StatusCreated, "POST", "/api/auth/exercises", token, map[string]any{
		"exercise_day_id": idOf(t, day, "exercise"),
		"is_on":           true,
		"note":            "",
		"duration":        nil,
	})
	if !field(t, created, "exercise", "is_on").(bool) {
		t.Errorf("a session created with the builder payload was stored switched off")
	}
}

// TestCorrelateExerciseWithExerciseDayFaults fails each database operation the correlation
// issues in turn — for a turn-off, a turn-on and a create plan — and checks a failure is
// reported rather than swallowed.
func TestCorrelateExerciseWithExerciseDayFaults(t *testing.T) {
	scenarios := []struct {
		name   string
		seed   func(t *testing.T, dayID uuid.UUID)
		target int
	}{
		{"turn off", func(t *testing.T, dayID uuid.UUID) { seedCorrelationSession(t, dayID, true) }, 0},
		{"turn on", func(t *testing.T, dayID uuid.UUID) { seedCorrelationSession(t, dayID, false) }, 1},
		{"create", func(t *testing.T, dayID uuid.UUID) {}, 1},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			setup := func() (*faultInjector, uuid.UUID) {
				newControllerTestDB(t)
				user := createTestUser(t, "correlate-"+uuid.NewString()+"@example.com", "Correlate")
				day := models.ExerciseDay{Date: time.Now(), Enabled: true, UserID: &user.ID}
				day.ID = uuid.New()
				if err := database.CreateExerciseDayInDB(day); err != nil {
					t.Fatalf("failed to create exercise day: %v", err)
				}
				scenario.seed(t, day.ID)
				return installFaultInjector(t, database.Instance), day.ID
			}

			injector, dayID := setup()
			injector.arm(0)
			if err := CorrelateExerciseWithExerciseDay(dayID, scenario.target); err != nil {
				t.Fatalf("CorrelateExerciseWithExerciseDay returned error without a fault: %v", err)
			}
			operations := injector.disarm()

			reported := 0
			for failAt := 1; failAt <= operations; failAt++ {
				injector, dayID := setup()
				injector.arm(failAt)
				if err := CorrelateExerciseWithExerciseDay(dayID, scenario.target); err != nil {
					reported++
				}
				injector.disarm()
			}
			if reported == 0 {
				t.Errorf("no injected fault out of %d was reported", operations)
			}
		})
	}
}

// seedCorrelationSession stores an empty, goal-counting manual session on the day.
func seedCorrelationSession(t *testing.T, dayID uuid.UUID, isOn bool) {
	t.Helper()
	session := models.Exercise{ExerciseDayID: dayID, Enabled: true, CountsTowardGoal: true}
	session.ID = uuid.New()
	if err := database.CreateExerciseForExerciseDayInDatabase(session); err != nil {
		t.Fatalf("failed to seed session: %v", err)
	}
	if err := database.SetExerciseIsOn(session.ID, isOn); err != nil {
		t.Fatalf("failed to set session state: %v", err)
	}
}

// TestCorrelateExerciseWithExerciseDayMissingDay covers a day id with no row behind it.
func TestCorrelateExerciseWithExerciseDayMissingDay(t *testing.T) {
	newControllerTestDB(t)
	if err := CorrelateExerciseWithExerciseDay(uuid.New(), 1); err == nil {
		t.Errorf("CorrelateExerciseWithExerciseDay on a missing day returned no error")
	}
}

// TestCreateSessionForExerciseDay checks a calendar +1 is a switched-on, counting session,
// timed now on today and at the day's own date otherwise.
func TestCreateSessionForExerciseDay(t *testing.T) {
	newControllerTestDB(t)
	user := createTestUser(t, "createsession@example.com", "Create")
	now := time.Now()

	tests := []struct {
		name     string
		date     time.Time
		wantTime time.Time
	}{
		{"today is timed now", now, now},
		{"earlier day is timed at its date", now.AddDate(0, 0, -1), now.AddDate(0, 0, -1)},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			day := models.ExerciseDay{Date: test.date, Enabled: true, UserID: &user.ID}
			day.ID = uuid.New()
			if err := database.CreateExerciseDayInDB(day); err != nil {
				t.Fatalf("failed to create exercise day: %v", err)
			}
			if err := createSessionForExerciseDay(day, now); err != nil {
				t.Fatalf("createSessionForExerciseDay returned error: %v", err)
			}

			sessions, err := database.GetExerciseByExerciseDayID(day.ID)
			if err != nil || len(sessions) != 1 {
				t.Fatalf("sessions = %d, err = %v, want 1", len(sessions), err)
			}
			session := sessions[0]
			if !session.IsOn || !session.CountsTowardGoal {
				t.Errorf("session is_on = %v, counts_toward_goal = %v, want both true", session.IsOn, session.CountsTowardGoal)
			}
			// Within a second: MySQL and Postgres store timestamps at lower precision.
			if session.Time == nil || session.Time.Sub(test.wantTime).Abs() > time.Second {
				t.Errorf("session time = %v, want %v", session.Time, test.wantTime)
			}
		})
	}
}
