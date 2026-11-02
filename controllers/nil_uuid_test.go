package controllers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/aunefyren/treningheten/database"
	"github.com/aunefyren/treningheten/files"
	"github.com/aunefyren/treningheten/models"

	"github.com/google/uuid"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// GORM drops zero-valued fields from struct conditions, so Where(&models.Gear{GormModel:
// models.GormModel{ID: id}}) with the nil UUID (which uuid.Parse accepts) silently loses
// its id filter and matches the first or every row. The data layer uses explicit column
// conditions instead (TestNoStructConditions in database/ guards the source); these tests
// guard the behaviour: every route and MCP tool must treat the nil UUID exactly like any
// other unknown id.

const nilUUIDVictimMarker = "NILUUIDVICTIM"

// seedNilUUIDWorld gives a user one row of every user-owned kind, all in a shared season,
// so a lost id condition has someone else's data to match.
func seedNilUUIDWorld(t *testing.T, user models.User, season models.Season, marker string) {
	t.Helper()

	create := func(value any) {
		if err := database.Instance.Create(value).Error; err != nil {
			t.Fatalf("failed to seed %T: %v", value, err)
		}
	}

	goal := models.Goal{SeasonID: season.ID, UserID: user.ID, ExerciseInterval: 3, Competing: true, Enabled: true}
	goal.ID = uuid.New()
	create(&goal)

	userID, goalID := user.ID, goal.ID
	day := models.ExerciseDay{Date: time.Now(), Note: marker + " day", Enabled: true, UserID: &userID, GoalID: &goalID}
	day.ID = uuid.New()
	create(&day)

	exercise := models.Exercise{Note: marker + " exercise", Enabled: true, IsOn: true, ExerciseDayID: day.ID, CountsTowardGoal: true}
	exercise.ID = uuid.New()
	create(&exercise)

	gear := models.Gear{UserID: user.ID, Name: marker + " shoe", Type: "shoe", Enabled: true}
	gear.ID = uuid.New()
	create(&gear)

	var action models.Action
	if err := database.Instance.Where("enabled = ?", true).Order("id").First(&action).Error; err != nil {
		t.Fatalf("failed to load an action: %v", err)
	}
	note, actionID, gearID := marker+" operation", action.ID, gear.ID
	operation := models.Operation{Enabled: true, ExerciseID: exercise.ID, ActionID: &actionID, GearID: &gearID,
		Type: "lifting", WeightUnit: "kg", DistanceUnit: "km", Note: &note}
	operation.ID = uuid.New()
	create(&operation)

	repetitions := 5.0
	set := models.OperationSet{OperationID: operation.ID, Enabled: true, Repetitions: &repetitions}
	set.ID = uuid.New()
	create(&set)

	weight := models.WeightValue{Enabled: true, Date: time.Now(), Weight: 80, UserID: user.ID}
	weight.ID = uuid.New()
	create(&weight)

	sickleave := models.Sickleave{Enabled: true, GoalID: goal.ID}
	sickleave.ID = uuid.New()
	create(&sickleave)

	debt := models.Debt{Date: time.Now().AddDate(0, 0, -7), SeasonID: season.ID, LoserID: user.ID, Enabled: true}
	debt.ID = uuid.New()
	create(&debt)

	pat := models.PersonalAccessToken{UserID: user.ID, Name: marker + " pat", TokenHash: uuid.NewString(),
		Scope: models.ScopeAPI, ExpiresAt: time.Now().AddDate(0, 1, 0)}
	pat.ID = uuid.New()
	create(&pat)

	subscription := models.Subscription{Enabled: true, UserID: user.ID, Endpoint: "https://push.test/" + marker, Auth: marker}
	subscription.ID = uuid.New()
	create(&subscription)

	var achievement models.Achievement
	if err := database.Instance.Order("id").First(&achievement).Error; err != nil {
		t.Fatalf("failed to load an achievement: %v", err)
	}
	delegation := models.AchievementDelegation{Enabled: true, UserID: user.ID, AchievementID: achievement.ID, GivenAt: time.Now()}
	delegation.ID = uuid.New()
	create(&delegation)
}

// newNilUUIDWorld seeds a victim, an attacker and an admin who each own a full set of rows.
func newNilUUIDWorld(t *testing.T, h *apiHarness) (victim models.User, attackerToken string, adminToken string) {
	t.Helper()

	prize := models.Prize{Name: "prize", Quantity: 1, Enabled: true}
	prize.ID = uuid.New()
	if err := database.Instance.Create(&prize).Error; err != nil {
		t.Fatal(err)
	}
	joinAnytime := true
	season := models.Season{Name: "season", Start: time.Now().AddDate(0, 0, -30), End: time.Now().AddDate(0, 0, 30),
		PrizeID: prize.ID, JoinAnytime: &joinAnytime, Enabled: true}
	season.ID = uuid.New()
	if err := database.Instance.Create(&season).Error; err != nil {
		t.Fatal(err)
	}
	news := models.News{Title: "news", Body: "body", Enabled: true, Date: time.Now()}
	news.ID = uuid.New()
	if err := database.Instance.Create(&news).Error; err != nil {
		t.Fatal(err)
	}

	victim, _ = h.user("victim-"+strings.ToLower(nilUUIDVictimMarker)+"@nil.test", false)
	attacker, attackerToken := h.user("attacker@nil.test", false)
	admin, adminToken := h.user("admin@nil.test", true)
	seedNilUUIDWorld(t, victim, season, nilUUIDVictimMarker)
	seedNilUUIDWorld(t, attacker, season, "ATTACKER")
	seedNilUUIDWorld(t, admin, season, "ADMIN")
	return victim, attackerToken, adminToken
}

// tableSnapshot is every row of every table, serialized, to spot which tables a request wrote.
func tableSnapshot(t *testing.T) map[string]string {
	t.Helper()

	quiesceBackgroundWork(t)
	tables, err := database.Instance.Migrator().GetTables()
	if err != nil {
		t.Fatalf("failed to list tables: %v", err)
	}
	snapshot := map[string]string{}
	for _, table := range tables {
		var rows []map[string]any
		if err := database.Instance.Table(table).Find(&rows).Error; err != nil {
			t.Fatalf("failed to read %s: %v", table, err)
		}
		lines := make([]string, 0, len(rows))
		for _, row := range rows {
			encoded, _ := json.Marshal(row)
			lines = append(lines, string(encoded))
		}
		sort.Strings(lines)
		snapshot[table] = strings.Join(lines, "\n")
	}
	return snapshot
}

func changedTables(before, after map[string]string) string {
	changed := []string{}
	for table, rows := range after {
		if before[table] != rows {
			changed = append(changed, table)
		}
	}
	sort.Strings(changed)
	return strings.Join(changed, ",")
}

// nilUUIDRequestBody fills every id-shaped key a handler might read with id. The free-text
// fields are unique per request, so a create route doesn't collide with its own previous
// call (a duplicate prize name, a re-used push endpoint) and answer differently for that
// reason alone.
func nilUUIDRequestBody(id string) string {
	unique := uuid.NewString()
	body := map[string]any{
		"name": "probe " + unique, "note": "probe " + unique, "title": "probe " + unique, "body": "probe " + unique,
		"subscription": map[string]any{"endpoint": "https://push.test/" + unique}, "date": time.Now().Format(time.RFC3339),
		"users": []string{id}, "user_ids": []string{id},
	}
	for _, key := range []string{"id", "user", "user_id", "target_user", "season", "season_id", "goal", "goal_id",
		"exercise_day", "exercise_day_id", "exercise", "exercise_id", "operation", "operation_id", "action", "action_id",
		"gear_id", "achievement_id", "prize", "prize_id"} {
		body[key] = id
	}
	encoded, _ := json.Marshal(body)
	return string(encoded)
}

// Every route must answer a nil-UUID id — in the path, the query and the body — exactly as
// it answers an id that matches nothing: same status, nobody else's data, no extra writes.
func TestRoutesTreatTheNilUUIDAsUnknown(t *testing.T) {
	h := newAPIHarness(t)
	victim, attackerToken, adminToken := newNilUUIDWorld(t, h)

	send := func(method, routePath, token, id string) *httptest.ResponseRecorder {
		path := routePath
		for _, segment := range strings.Split(routePath, "/") {
			if strings.HasPrefix(segment, ":") {
				path = strings.Replace(path, segment, id, 1)
			}
		}
		path += "?action_id=" + id + "&operation_id=" + id + "&user=" + id + "&goal=" + id
		request := httptest.NewRequest(method, path, bytes.NewBufferString(nilUUIDRequestBody(id)))
		request.Header.Set("Content-Type", "application/json")
		if token != "" {
			request.Header.Set("Authorization", "Bearer "+token)
		}
		return serve(h, request)
	}
	leaks := func(body string) bool {
		return strings.Contains(body, nilUUIDVictimMarker) || strings.Contains(body, victim.ID.String())
	}

	callers := []struct{ name, token string }{{"anonymous", ""}, {"user", attackerToken}, {"admin", adminToken}}
	requests := 0
	before := tableSnapshot(t)
	for _, route := range h.router.Routes() {
		external := false
		for _, marker := range externalRoutes {
			external = external || strings.Contains(route.Path, marker)
		}
		if external {
			continue
		}
		for _, caller := range callers {
			guarded := strings.HasPrefix(route.Path, "/api/auth") || strings.HasPrefix(route.Path, "/api/admin")
			if (caller.name == "anonymous" && guarded) || (caller.name == "user" && strings.HasPrefix(route.Path, "/api/admin")) {
				continue
			}
			name := fmt.Sprintf("%s %s as %s", route.Method, route.Path, caller.name)

			random := send(route.Method, route.Path, caller.token, uuid.NewString())
			afterRandom := tableSnapshot(t)
			nilID := send(route.Method, route.Path, caller.token, uuid.Nil.String())
			afterNil := tableSnapshot(t)
			requests++

			if random.Code != nilID.Code {
				t.Errorf("%s: status %d for an unknown id but %d for the nil UUID; body: %s", name, random.Code, nilID.Code, nilID.Body.String())
			}
			if leaks(nilID.Body.String()) && !leaks(random.Body.String()) {
				t.Errorf("%s: the nil UUID returned another user's data: %s", name, nilID.Body.String())
			}
			if wrote, wroteForRandom := changedTables(afterRandom, afterNil), changedTables(before, afterRandom); wrote != wroteForRandom {
				t.Errorf("%s: the nil UUID wrote tables [%s], an unknown id wrote [%s]", name, wrote, wroteForRandom)
			}
			before = afterNil
		}
	}
	if requests < 150 {
		t.Errorf("only %d requests sent; did the route table shrink?", requests)
	}
}

// The MCP tools take ids as strings and uuid.Parse them, so they must refuse the nil UUID
// the same way as any id the caller doesn't own.
func TestMCPToolsTreatTheNilUUIDAsUnknown(t *testing.T) {
	h := newAPIHarness(t)
	enabled := true
	previous := files.ConfigFile.MCPEnabled
	files.ConfigFile.MCPEnabled = &enabled
	restoreAfterBackgroundWork(t, func() { files.ConfigFile.MCPEnabled = previous })

	victim, attackerToken, _ := newNilUUIDWorld(t, h)
	session := mcpSession(t, attackerToken)

	call := func(tool, argument, id string) (bool, string) {
		result, err := session.CallTool(t.Context(), &mcp.CallToolParams{Name: tool, Arguments: map[string]any{argument: id}})
		if err != nil {
			t.Fatalf("%s: transport error: %v", tool, err)
		}
		text := ""
		for _, content := range result.Content {
			if textContent, isText := content.(*mcp.TextContent); isText {
				text += textContent.Text
			}
		}
		return result.IsError, strings.ReplaceAll(text, id, "<id>")
	}

	cases := []struct{ tool, argument string }{
		{"get_activity", "activity_id"},
		{"get_activity_streams", "activity_id"},
		{"get_activity_soundtrack", "activity_id"},
		{"get_season", "season_id"},
		{"get_achievement", "achievement_id"},
		{"list_achievement_delegations", "achievement_id"},
		{"get_achievement_delegation", "delegation_id"},
		{"get_personal_records", "action"},
		{"get_activity_type_statistics", "action"},
		{"list_workouts", "action"},
	}
	for _, c := range cases {
		t.Run(c.tool, func(t *testing.T) {
			randomFailed, randomText := call(c.tool, c.argument, uuid.NewString())
			nilFailed, nilText := call(c.tool, c.argument, uuid.Nil.String())
			if randomFailed != nilFailed || randomText != nilText {
				t.Errorf("unknown id → (error=%v) %s\nnil UUID → (error=%v) %s", randomFailed, randomText, nilFailed, nilText)
			}
			if strings.Contains(nilText, nilUUIDVictimMarker) || strings.Contains(nilText, victim.ID.String()) {
				t.Errorf("the nil UUID returned another user's data: %s", nilText)
			}
		})
	}
}
