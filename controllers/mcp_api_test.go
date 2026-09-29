package controllers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/aunefyren/treningheten/auth"
	"github.com/aunefyren/treningheten/database"
	"github.com/aunefyren/treningheten/files"
	"github.com/aunefyren/treningheten/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type bearerTransport struct{ token string }

func (b bearerTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	request = request.Clone(request.Context())
	request.Header.Set("Authorization", "Bearer "+b.token)
	return http.DefaultTransport.RoundTrip(request)
}

// mcpSession serves MCPHandler on an httptest server and connects the official client
// to it as the given token's user.
func mcpSession(t *testing.T, token string) *mcp.ClientSession {
	t.Helper()

	router := gin.New()
	router.Any("/mcp", MCPHandler())
	server := httptest.NewServer(router)
	t.Cleanup(server.Close)

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "v1"}, nil)
	session, err := client.Connect(t.Context(), &mcp.StreamableClientTransport{
		Endpoint:   server.URL + "/mcp",
		HTTPClient: &http.Client{Transport: bearerTransport{token: token}},
	}, nil)
	if err != nil {
		t.Fatalf("MCP connect: %v", err)
	}
	t.Cleanup(func() { _ = session.Close() })
	return session
}

// callTool calls a tool and decodes its structured output; wantError flips the check.
func callTool(t *testing.T, session *mcp.ClientSession, name string, args map[string]any, wantError bool) map[string]any {
	t.Helper()

	if args == nil {
		args = map[string]any{}
	}
	result, err := session.CallTool(t.Context(), &mcp.CallToolParams{Name: name, Arguments: args})
	if err != nil {
		t.Fatalf("%s: transport error: %v", name, err)
	}
	if result.IsError != wantError {
		text := ""
		for _, content := range result.Content {
			if textContent, isText := content.(*mcp.TextContent); isText {
				text += textContent.Text
			}
		}
		t.Fatalf("%s %v: IsError = %v, want %v (%s)", name, args, result.IsError, wantError, text)
	}
	decoded := map[string]any{}
	if result.StructuredContent != nil {
		encoded, _ := json.Marshal(result.StructuredContent)
		_ = json.Unmarshal(encoded, &decoded)
	}
	return decoded
}

func TestMCPToolsEndToEnd(t *testing.T) {
	h := newAPIHarness(t)
	enabled := true
	files.ConfigFile.MCPEnabled = &enabled

	fake := withStrava(t)
	gearID := "g55"
	fake.activities = []models.StravaGetActivitiesRequestReply{stravaActivity(2001, "Run", todayAt(6), &gearID, false)}

	_, adminToken := h.user("admin@mcp.test", true)
	user, token := h.user("agent@mcp.test", false)
	h.ok("POST", "/api/auth/users/"+user.ID.String()+"/strava", token, models.UserStravaCodeUpdateRequest{StravaCode: "code"})

	manual := createTodaySession(t, h, token)
	h.expect(http.StatusCreated, "POST", "/api/auth/weights", token, models.WeightValueCreationRequest{Date: time.Now(), Weight: 81.2})
	season := seedOngoingSeason(t, "Agent season", 1, 6)
	seedGoal(t, user.ID, season, 2)
	achievementID := "7f2d49ad-d056-415e-aa80-0ada6db7cc00"
	h.expect(http.StatusCreated, "POST", "/api/admin/users/"+user.ID.String()+"/achievement-delegations", adminToken,
		models.AchievementDelegationCreationRequest{AchievementID: uuid.MustParse(achievementID)})

	session := mcpSession(t, token)

	tools, err := session.ListTools(t.Context(), nil)
	if err != nil || len(tools.Tools) < 14 {
		t.Fatalf("tools = %v, err = %v", tools, err)
	}

	if who := callTool(t, session, "whoami", nil, false); who["email"] != "agent@mcp.test" {
		t.Errorf("whoami = %v", who)
	}
	if weights := callTool(t, session, "list_weights", map[string]any{"limit": 5}, false); len(weights["weights"].([]any)) != 1 {
		t.Errorf("list_weights = %v", weights)
	}
	if latest := callTool(t, session, "get_latest_weight", nil, false); latest["weight"] == nil {
		t.Error("get_latest_weight returned no weight")
	}

	// Workouts: the Strava run and the empty manual session are both findable.
	workouts := callTool(t, session, "list_workouts", nil, false)
	if workouts["total"] != float64(2) {
		t.Fatalf("list_workouts total = %v, want 2 (the run and the empty manual session)", workouts["total"])
	}
	callTool(t, session, "list_workouts", map[string]any{"action": "run", "has_distance": true, "sort": "distance", "order": "asc", "from": "2000-01-01", "to": time.Now().AddDate(0, 0, 1).Format(time.RFC3339), "limit": 500, "offset": -1}, false)
	callTool(t, session, "list_workouts", map[string]any{"sort": "vibes"}, true)
	callTool(t, session, "list_workouts", map[string]any{"order": "sideways"}, true)
	callTool(t, session, "list_workouts", map[string]any{"from": "yesterday"}, true)

	var runActivityID string
	for _, workout := range workouts["workouts"].([]any) {
		for _, activity := range field(t, workout, "activities").([]any) {
			if field(t, activity, "has_streams") == true {
				runActivityID = idOf(t, activity)
			}
		}
	}
	if runActivityID == "" {
		t.Fatal("no stream-backed activity in list_workouts")
	}

	activity := callTool(t, session, "get_activity", map[string]any{"activity_id": runActivityID,
		"include": []string{"segments", "zones", "analysis", "elevation", "route", "profile"}}, false)
	if field(t, activity, "activity", "source") != "strava" {
		t.Errorf("activity source = %v", field(t, activity, "activity", "source"))
	}
	callTool(t, session, "get_activity", map[string]any{"activity_id": runActivityID}, false)
	callTool(t, session, "get_activity", map[string]any{"activity_id": "nope"}, true)
	callTool(t, session, "get_activity", map[string]any{"activity_id": manual}, true) // a workout id

	streams := callTool(t, session, "get_activity_streams", map[string]any{"activity_id": runActivityID}, false)
	if streams["series"] == nil {
		t.Errorf("streams have no series: %v", streams)
	}
	callTool(t, session, "get_activity_streams", map[string]any{"activity_id": runActivityID, "from_seconds": 30, "to_seconds": 150, "resolution": 1, "max_points": 10}, false)
	// An id that isn't one of the user's stream-backed activities gets the soft "no streams" answer.
	if unknown := callTool(t, session, "get_activity_streams", map[string]any{"activity_id": uuid.NewString()}, false); unknown["has_streams"] != false {
		t.Errorf("unknown activity streams = %v, want has_streams=false", unknown)
	}

	soundtrack := callTool(t, session, "get_activity_soundtrack", map[string]any{"activity_id": manual}, false)
	if soundtrack["has_soundtrack"] != false {
		t.Errorf("soundtrack without media = %v, want has_soundtrack=false", soundtrack)
	}
	callTool(t, session, "get_activity_soundtrack", map[string]any{"activity_id": "nope"}, true)

	callTool(t, session, "get_statistics", nil, false)

	seasons := callTool(t, session, "list_seasons", map[string]any{"active_only": true}, false)
	if len(seasons["seasons"].([]any)) != 1 {
		t.Errorf("active seasons = %v", seasons)
	}
	if got := callTool(t, session, "get_season", map[string]any{"season_id": season.ID.String()}, false); got["joined"] != true {
		t.Errorf("get_season = %v, want joined", got)
	}
	callTool(t, session, "get_season", map[string]any{"season_id": "nope"}, true)

	achievements := callTool(t, session, "list_achievements", nil, false)
	if len(achievements["achievements"].([]any)) < 20 {
		t.Errorf("achievement catalog = %d entries", len(achievements["achievements"].([]any)))
	}
	if got := callTool(t, session, "get_achievement", map[string]any{"achievement_id": achievementID}, false); got["earned"] != true {
		t.Errorf("get_achievement = %v, want earned", got)
	}
	callTool(t, session, "get_achievement", map[string]any{"achievement_id": "nope"}, true)

	delegations := callTool(t, session, "list_achievement_delegations", map[string]any{"achievement_id": achievementID, "limit": 5}, false)
	delegationList := delegations["delegations"].([]any)
	if len(delegationList) == 0 {
		t.Fatalf("no delegations: %v", delegations)
	}
	callTool(t, session, "list_achievement_delegations", map[string]any{"achievement_id": "nope"}, true)
	callTool(t, session, "get_achievement_delegation", map[string]any{"delegation_id": idOf(t, delegationList[0])}, false)
	callTool(t, session, "get_achievement_delegation", map[string]any{"delegation_id": uuid.NewString()}, true)
	callTool(t, session, "get_achievement_delegation", map[string]any{"delegation_id": "nope"}, true)

	// Another user's MCP session can't see this user's data.
	_, otherToken := h.user("other@mcp.test", false)
	other := mcpSession(t, otherToken)
	callTool(t, other, "get_activity", map[string]any{"activity_id": runActivityID}, true)
	if total := callTool(t, other, "list_workouts", nil, false)["total"]; total != float64(0) {
		t.Errorf("another user's list_workouts total = %v, want 0", total)
	}
}

func TestMCPHandlerGates(t *testing.T) {
	newAPIHarness(t)
	router := gin.New()
	router.Any("/mcp", MCPHandler())
	post := func(token string) *httptest.ResponseRecorder {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest("POST", "/mcp", strings.NewReader(`{}`))
		if token != "" {
			request.Header.Set("Authorization", "Bearer "+token)
		}
		router.ServeHTTP(recorder, request)
		return recorder
	}

	disabled := false
	files.ConfigFile.MCPEnabled = &disabled
	if code := post("").Code; code != http.StatusNotFound {
		t.Errorf("disabled MCP: status = %d, want 404", code)
	}

	enabled := true
	files.ConfigFile.MCPEnabled = &enabled
	recorder := post("")
	if recorder.Code != http.StatusUnauthorized || !strings.HasPrefix(recorder.Header().Get("WWW-Authenticate"), "Bearer") {
		t.Errorf("no token: status = %d, challenge = %q", recorder.Code, recorder.Header().Get("WWW-Authenticate"))
	}

	user := makeControllerTestUser(t, "scopeless@mcp.test")
	noScope, _, err := auth.GenerateAccessToken(user.ID, false, "profile", "test")
	if err != nil {
		t.Fatal(err)
	}
	if code := post(noScope).Code; code != http.StatusForbidden {
		t.Errorf("token without API read scope: status = %d, want 403", code)
	}
	_ = database.Instance
}
