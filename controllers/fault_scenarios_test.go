package controllers

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/aunefyren/treningheten/database"
	"github.com/aunefyren/treningheten/models"

	"github.com/google/uuid"
)

// faultWorld is a small, realistic dataset every fault scenario starts from: two members
// of a running season (one also admin), a logged session with an activity and a set,
// gear, a weight, news, an invite and an unchosen debt.
type faultWorld struct {
	admin, member                   models.User
	adminToken, memberToken         string
	season                          models.Season
	adminGoal, memberGoal           models.Goal
	dayID, sessionID, operationID   string
	setID, gearID, weightID, newsID string
	actionID                        uuid.UUID
	actionName                      string
	debtID                          uuid.UUID
}

func buildFaultWorld(t *testing.T, h *apiHarness) faultWorld {
	t.Helper()
	w := faultWorld{}
	w.admin, w.adminToken = h.user("admin@fault.test", true)
	w.member, w.memberToken = h.user("member@fault.test", false)

	w.season = seedOngoingSeason(t, "Fault season", 2, 6)
	w.adminGoal = seedGoal(t, w.admin.ID, w.season, 1)
	w.memberGoal = seedGoal(t, w.member.ID, w.season, 2)
	seedExerciseDayWithExercises(t, w.admin.ID, wednesdayOfWeeksAgo(1), 1)

	day := h.ok("GET", "/api/auth/exercise-days/week?today=true", w.memberToken, nil)
	w.dayID = idOf(t, day, "exercise")
	session := h.expect(http.StatusCreated, "POST", "/api/auth/exercises", w.memberToken, models.ExerciseCreationRequest{
		ExerciseDayID: uuid.MustParse(w.dayID), Note: "fault session", IsOn: true, Duration: int64Ptr(1800),
	})
	w.sessionID = idOf(t, session, "exercise")

	actions := field(t, h.ok("GET", "/api/auth/actions", w.memberToken, nil), "actions").([]any)
	w.actionID = uuid.MustParse(idOf(t, actions[0]))
	w.actionName = field(t, actions[0], "name").(string)

	operation := h.expect(http.StatusCreated, "POST", "/api/auth/operations", w.memberToken, models.OperationCreationRequest{
		ExerciseID: uuid.MustParse(w.sessionID), Action: &w.actionID, Type: "moving", WeightUnit: "kg", DistanceUnit: "km",
	})
	w.operationID = idOf(t, operation, "operation")
	set := h.expect(http.StatusCreated, "POST", "/api/auth/operation-sets", w.memberToken, models.OperationSetCreationRequest{
		OperationID: uuid.MustParse(w.operationID), Distance: float64Ptr(5), Time: int64Ptr(1500),
	})
	w.setID = idOf(t, set, "operation_set")

	gear := h.expect(http.StatusCreated, "POST", "/api/auth/gear", w.memberToken, models.GearCreationRequest{Name: "Shoe", Type: "shoe"})
	w.gearID = idOf(t, gear, "gear")
	weight := h.expect(http.StatusCreated, "POST", "/api/auth/weights", w.memberToken, models.WeightValueCreationRequest{Date: time.Now(), Weight: 80})
	w.weightID = idOf(t, weight, "weight")
	news := h.expect(http.StatusCreated, "POST", "/api/admin/news", w.adminToken, models.NewsCreationRequest{Title: "Hello", Body: "Body text", Date: time.Now()})
	w.newsID = idOf(t, field(t, news, "news").([]any)[0])

	debt := models.Debt{Date: wednesdayOfWeeksAgo(1), SeasonID: w.season.ID, LoserID: w.member.ID, Enabled: true}
	debt.ID = uuid.New()
	if err := database.Instance.Omit("Season", "Loser", "Winner").Create(&debt).Error; err != nil {
		t.Fatal(err)
	}
	w.debtID = debt.ID
	return w
}

// scenario is a shorthand for a sweep that starts from the fault world.
func scenario(name string, want int, request func(t *testing.T, h *apiHarness, w faultWorld) faultRequest) faultScenario {
	return faultScenario{name: name, wantStatus: want, setup: func(t *testing.T, h *apiHarness) faultRequest {
		return request(t, h, buildFaultWorld(t, h))
	}}
}

func faultScenarios() []faultScenario {
	get := func(path string, member bool) func(t *testing.T, h *apiHarness, w faultWorld) faultRequest {
		return func(t *testing.T, h *apiHarness, w faultWorld) faultRequest {
			token := w.adminToken
			if member {
				token = w.memberToken
			}
			return faultRequest{"GET", expandPath(path, w), token, nil, nil}
		}
	}
	send := func(method, path string, member bool, body any) func(t *testing.T, h *apiHarness, w faultWorld) faultRequest {
		return func(t *testing.T, h *apiHarness, w faultWorld) faultRequest {
			token := w.adminToken
			if member {
				token = w.memberToken
			}
			if fields, isMap := body.(map[string]any); isMap {
				expanded := map[string]any{}
				for key, value := range fields {
					if text, isText := value.(string); isText {
						value = expandPath(text, w)
					}
					expanded[key] = value
				}
				return faultRequest{method, expandPath(path, w), token, expanded, nil}
			}
			return faultRequest{method, expandPath(path, w), token, body, nil}
		}
	}

	return []faultScenario{
		// Seasons and goals
		scenario("seasons", 200, get("/api/auth/seasons", true)),
		scenario("seasons potential", 200, get("/api/auth/seasons?potential=true", true)),
		scenario("seasons countdown", 200, get("/api/auth/seasons?countdown=true", true)),
		scenario("season", 200, get("/api/auth/seasons/{season}", true)),
		scenario("season weeks", 200, get("/api/auth/seasons/{season}/weeks", true)),
		scenario("season weeks personal", 200, get("/api/auth/seasons/{season}/weeks-personal", true)),
		scenario("ongoing seasons", 200, get("/api/auth/seasons/get-on-going", true)),
		scenario("leaderboard", 200, get("/api/auth/seasons/{season}/leaderboard", true)),
		scenario("season activities", 200, get("/api/auth/seasons/{season}/activities", true)),
		scenario("goals", 201, get("/api/auth/goals", true)),
		scenario("register goal", 201, func(t *testing.T, h *apiHarness, w faultWorld) faultRequest {
			_, token := h.user("joiner@fault.test", false)
			return faultRequest{"POST", "/api/auth/goals", token, models.GoalCreationRequest{ExerciseInterval: 2, Competing: true, SeasonID: w.season.ID}, nil}
		}),
		scenario("sick leave", 200, func(t *testing.T, h *apiHarness, w faultWorld) faultRequest {
			// Joining through the API creates the season's sick-leave allowance.
			_, token := h.user("sickly@fault.test", false)
			h.expect(http.StatusCreated, "POST", "/api/auth/goals", token, models.GoalCreationRequest{ExerciseInterval: 1, Competing: true, SeasonID: w.season.ID})
			return faultRequest{"POST", "/api/auth/sickleave/" + w.season.ID.String(), token, nil, nil}
		}),
		scenario("create season", 201, func(t *testing.T, h *apiHarness, w faultWorld) faultRequest {
			start := nextMonday(2)
			return faultRequest{"POST", "/api/admin/seasons", w.adminToken, models.SeasonCreationRequest{
				Name: "Next season", Start: start, End: start.AddDate(0, 0, 13), Prize: w.season.PrizeID, TimeZone: "UTC",
			}, nil}
		}),

		// Exercise days, sessions, week
		scenario("exercise day today", 200, get("/api/auth/exercise-days/week?today=true", true)),
		scenario("exercise days", 200, get("/api/auth/exercise-days", true)),
		scenario("exercise days by goal", 200, get("/api/auth/exercise-days?goal={memberGoal}", true)),
		scenario("exercise day", 201, get("/api/auth/exercise-days/{day}", true)),
		scenario("update exercise day", 201, send("POST", "/api/auth/exercise-days/{day}", true, models.ExerciseDayUpdateRequest{Note: "updated"})),
		scenario("admin exercise days", 200, get("/api/admin/exercise-days", false)),
		scenario("get week", 200, get("/api/auth/exercises/week", true)),
		scenario("register week", 201, send("POST", "/api/auth/exercises/week", false, models.WeekCreationRequest{Days: currentWeekDays(1), TimeZone: "UTC"})),
		scenario("create session", 201, send("POST", "/api/auth/exercises", true, map[string]any{"exercise_day_id": "{day}", "note": "n", "is_on": false})),
		scenario("update session", 200, send("PUT", "/api/auth/exercises/{session}", true, models.ExerciseUpdateRequest{
			Note: "u", IsOn: true, Time: time.Now().Format(time.RFC3339), CountsTowardGoal: boolPtr(false), Private: boolPtr(true),
		})),
		scenario("session gear", 200, send("PUT", "/api/auth/exercises/{session}/gear", true, map[string]any{"gear_id": "{gear}"})),

		// Activities and sets
		scenario("operations", 200, get("/api/auth/operations", true)),
		scenario("operation", 200, get("/api/auth/operations/{operation}", true)),
		scenario("create operation", 201, send("POST", "/api/auth/operations", true, map[string]any{"exercise_id": "{session}", "type": "lifting", "weight_unit": "kg", "distance_unit": "km"})),
		scenario("update operation", 200, func(t *testing.T, h *apiHarness, w faultWorld) faultRequest {
			return faultRequest{"PUT", "/api/auth/operations/" + w.operationID, w.memberToken, models.OperationUpdateRequest{
				Action: w.actionName, Type: "moving", WeightUnit: "kg", DistanceUnit: "km", Tags: &[]string{"race"}, Description: stringPtr("d"), GearID: stringPtr(w.gearID),
			}, nil}
		}),
		scenario("delete operation", 200, send("DELETE", "/api/auth/operations/{operation}", true, nil)),
		scenario("operation sets", 200, get("/api/auth/operation-sets?operation_id={operation}", true)),
		scenario("all operation sets", 200, get("/api/auth/operation-sets", true)),
		scenario("create set", 201, send("POST", "/api/auth/operation-sets", true, map[string]any{"operation_id": "{operation}", "distance": 3, "time": 900})),
		scenario("update set", 200, send("PUT", "/api/auth/operation-sets/{set}", true, models.OperationSetUpdateRequest{Distance: float64Ptr(6), MovingTime: int64Ptr(1600)})),
		scenario("delete set", 200, send("DELETE", "/api/auth/operation-sets/{set}", true, nil)),
		scenario("actions", 200, get("/api/auth/actions?experienced=true", true)),
		scenario("create action", 201, send("POST", "/api/auth/actions", true, models.ActionCreationRequest{Name: "Fault move", Type: "timing"})),
		scenario("action statistics", 200, func(t *testing.T, h *apiHarness, w faultWorld) faultRequest {
			now := time.Now().UTC()
			return faultRequest{"GET", "/api/auth/actions/" + w.actionID.String() + "/statistics?start=" + now.AddDate(0, -1, 0).Format("2006-01-02T15:04:05Z") + "&end=" + now.AddDate(0, 0, 1).Format("2006-01-02T15:04:05Z"), w.memberToken, nil, nil}
		}),
		scenario("activity goal settings", 200, get("/api/auth/activity-goal-settings", true)),
		scenario("set activity goal setting", 200, func(t *testing.T, h *apiHarness, w faultWorld) faultRequest {
			return faultRequest{"PUT", "/api/auth/activity-goal-settings", w.memberToken, models.ActivityGoalSettingUpdateRequest{ActionID: w.actionID, CountsTowardGoal: false}, nil}
		}),

		// Feeds and statistics
		scenario("activity feed", 200, get("/api/auth/activities?q=fault&sort=date", true)),
		scenario("shared activities", 200, get("/api/auth/activities/shared", false)),
		scenario("user activities", 200, get("/api/auth/users/{member}/activities", false)),
		scenario("user statistics", 200, get("/api/auth/users/{member}/statistics", true)),
		scenario("other user statistics", 200, get("/api/auth/users/{member}/statistics", false)),

		// Gear, weights
		scenario("gear", 200, get("/api/auth/gear", true)),
		scenario("create gear", 201, send("POST", "/api/auth/gear", true, models.GearCreationRequest{Name: "Bike", Type: "bike"})),
		scenario("update gear", 200, send("PUT", "/api/auth/gear/{gear}", true, models.GearUpdateRequest{Name: stringPtr("Shoe 2"), IsPrimary: boolPtr(true)})),
		scenario("delete gear", 200, send("DELETE", "/api/auth/gear/{gear}", true, nil)),
		scenario("weights", 200, get("/api/auth/weights", true)),
		scenario("weight", 200, get("/api/auth/weights/{weight}", true)),
		scenario("create weight", 201, send("POST", "/api/auth/weights", true, models.WeightValueCreationRequest{Date: time.Now(), Weight: 79})),
		scenario("delete weight", 200, send("DELETE", "/api/auth/weights/{weight}", true, nil)),

		// Users
		scenario("users", 200, get("/api/auth/users", true)),
		scenario("user", 200, get("/api/auth/users/{member}", true)),
		scenario("public user", 200, get("/api/auth/users/{member}", false)),
		scenario("patch user", 200, send("PATCH", "/api/auth/users/{member}", true, models.UserPartialUpdateRequest{SundayAlert: boolPtr(true), WheelColor: stringPtr("#123456")})),

		// Debts and achievements
		scenario("unchosen debt", 200, get("/api/auth/debts/unchosen", true)),
		scenario("debt", 200, get("/api/auth/debts/{debt}", true)),
		scenario("choose winner", 200, send("POST", "/api/auth/debts/{debt}/choose", true, nil)),
		scenario("debt overview", 200, get("/api/auth/debts", true)),
		scenario("generate debt", 200, send("POST", "/api/admin/debts", false, map[string]any{"date": wednesdayOfWeeksAgo(1)})),
		scenario("achievements", 200, get("/api/auth/achievements", true)),
		scenario("achievements of user", 200, get("/api/auth/achievements?user={admin}", true)),
		scenario("give achievement", 201, send("POST", "/api/admin/users/{member}/achievement-delegations", false, map[string]any{"achievement_id": "7f2d49ad-d056-415e-aa80-0ada6db7cc00"})),

		// News, invites, prizes, PATs, admin
		scenario("news", 201, get("/api/auth/news", true)),
		scenario("news post", 201, get("/api/auth/news/{news}", true)),
		scenario("create news", 201, send("POST", "/api/admin/news", false, models.NewsCreationRequest{Title: "More news", Body: "Body text", Date: time.Now()})),
		scenario("delete news", 201, send("DELETE", "/api/admin/news/{news}", false, nil)),
		scenario("create invite", 201, send("POST", "/api/admin/invites", false, nil)),
		scenario("invites", 200, get("/api/admin/invites", false)),
		scenario("prizes", 200, get("/api/admin/prizes", false)),
		scenario("create prize", 200, send("POST", "/api/admin/prizes", false, models.PrizeCreationRequest{Name: "Fault prize", Quantity: 1})),
		scenario("create pat", 201, send("POST", "/api/auth/pats", true, models.PATCreationRequest{Name: "ci", Scope: "api:read", ExpiresInDays: 10})),
		scenario("pats", 200, get("/api/auth/pats", true)),
		scenario("server info", 200, get("/api/admin/server-info", false)),
		scenario("admin stats", 200, get("/api/admin/stats", false)),
	}
}

// expandPath substitutes {name} placeholders with the world's ids (in paths and, via
// JSON bodies built as maps, in request fields).
func expandPath(path string, w faultWorld) string {
	replacements := map[string]string{
		"{season}": w.season.ID.String(), "{memberGoal}": w.memberGoal.ID.String(), "{day}": w.dayID,
		"{session}": w.sessionID, "{operation}": w.operationID, "{set}": w.setID, "{gear}": w.gearID,
		"{weight}": w.weightID, "{news}": w.newsID, "{debt}": w.debtID.String(),
		"{member}": w.member.ID.String(), "{admin}": w.admin.ID.String(),
	}
	for placeholder, value := range replacements {
		path = replaceAll(path, placeholder, value)
	}
	return path
}

func TestFaultInjectionSweep(t *testing.T) {
	if testing.Short() {
		t.Skip("fault sweep replays every scenario once per database operation")
	}
	for _, scenario := range append(append(faultScenarios(), moreFaultScenarios()...), richFaultScenarios()...) {
		scenario := scenario
		t.Run(scenario.name, func(t *testing.T) { sweepFaults(t, scenario) })
	}

	swallowed.mutex.Lock()
	defer swallowed.mutex.Unlock()
	for name, count := range swallowed.counts {
		t.Logf("answered 2xx with a database operation failing: %-28s ×%d", name, count)
	}
}

func replaceAll(text, old, new string) string { return strings.ReplaceAll(text, old, new) }
