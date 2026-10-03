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

// seasonGoalUserIDs returns the user ids on a season's goal list as the API serves it.
func seasonGoalUserIDs(t *testing.T, h *apiHarness, seasonID uuid.UUID, token string) []string {
	t.Helper()
	season := h.ok("GET", "/api/auth/seasons/"+seasonID.String(), token, nil)
	goals, _ := field(t, season, "season", "goals").([]any)
	userIDs := []string{}
	for _, goal := range goals {
		userIDs = append(userIDs, field(t, goal, "user", "id").(string))
	}
	return userIDs
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

// TestAdminDisableUserFlow disables a member who is in a finished, an ongoing and an
// upcoming season: their access stops, they leave the unfinished seasons, the finished
// season still renders them, and re-enabling restores access without rejoining seasons.
func TestAdminDisableUserFlow(t *testing.T) {
	h := newAPIHarness(t)

	admin, adminToken := h.user("admin@disable.test", true)
	member, memberToken := h.user("member@disable.test", false)
	peer, peerToken := h.user("peer@disable.test", false)

	now := time.Now()
	finished := seedSeason(t, "Finished", now.AddDate(0, -3, 0), now.AddDate(0, -1, 0), true)
	ongoing := seedOngoingSeason(t, "Ongoing", 1, 4)
	upcoming := seedSeason(t, "Upcoming", now.AddDate(0, 1, 0), now.AddDate(0, 3, 0), true)
	for _, season := range []models.Season{finished, ongoing, upcoming} {
		seedGoal(t, member.ID, season, 2)
		seedGoal(t, peer.ID, season, 2)
	}

	// Non-admins can neither list nor toggle.
	h.expect(http.StatusForbidden, "GET", "/api/admin/users", memberToken, nil)
	h.expect(http.StatusForbidden, "PUT", "/api/admin/users/"+peer.ID.String()+"/enabled", memberToken, models.UserEnabledRequest{Enabled: boolPtr(false)})

	// Request validation.
	h.expect(http.StatusBadRequest, "PUT", "/api/admin/users/"+member.ID.String()+"/enabled", adminToken, map[string]any{})
	h.expect(http.StatusBadRequest, "PUT", "/api/admin/users/"+member.ID.String()+"/enabled", adminToken, "{broken")
	h.expect(http.StatusBadRequest, "PUT", "/api/admin/users/nope/enabled", adminToken, models.UserEnabledRequest{Enabled: boolPtr(false)})
	h.expect(http.StatusNotFound, "PUT", "/api/admin/users/"+uuid.NewString()+"/enabled", adminToken, models.UserEnabledRequest{Enabled: boolPtr(false)})
	h.expect(http.StatusBadRequest, "PUT", "/api/admin/users/"+admin.ID.String()+"/enabled", adminToken, models.UserEnabledRequest{Enabled: boolPtr(false)})

	// Disable the member.
	disabled := h.ok("PUT", "/api/admin/users/"+member.ID.String()+"/enabled", adminToken, models.UserEnabledRequest{Enabled: boolPtr(false)})
	if enabled := field(t, disabled, "user", "enabled"); enabled != false {
		t.Errorf("disabled user enabled: got %v, want false", enabled)
	}
	if message := field(t, disabled, "message").(string); !strings.Contains(message, "2 season(s)") {
		t.Errorf("disable message: got %q, want it to report 2 seasons", message)
	}
	if _, found := disabled["user"].(map[string]any)["password"]; found {
		t.Errorf("admin user response leaks the password field: %v", disabled["user"])
	}

	// Access stops immediately for the existing token.
	h.expect(http.StatusForbidden, "GET", "/api/auth/seasons", memberToken, nil)

	// Out of the unfinished seasons, still in the finished one; the peer is untouched and
	// every season still renders for them.
	for _, season := range []models.Season{ongoing, upcoming} {
		userIDs := seasonGoalUserIDs(t, h, season.ID, peerToken)
		if containsString(userIDs, member.ID.String()) {
			t.Errorf("season %s: disabled member still has a goal", season.Name)
		}
		if !containsString(userIDs, peer.ID.String()) {
			t.Errorf("season %s: peer lost their goal", season.Name)
		}
	}
	if userIDs := seasonGoalUserIDs(t, h, finished.ID, peerToken); !containsString(userIDs, member.ID.String()) {
		t.Errorf("finished season no longer shows the disabled member: %v", userIDs)
	}

	// The admin list still includes them, flagged.
	listed := h.ok("GET", "/api/admin/users", adminToken, nil)
	found := false
	for _, user := range field(t, listed, "users").([]any) {
		if idOf(t, user) == member.ID.String() {
			found = true
			if enabled := field(t, user, "enabled"); enabled != false {
				t.Errorf("listed member enabled: got %v, want false", enabled)
			}
			if email := field(t, user, "email"); email != "member@disable.test" {
				t.Errorf("listed member email: got %v", email)
			}
		}
	}
	if !found {
		t.Fatalf("disabled member missing from admin user list")
	}

	// Disabling again is idempotent.
	again := h.ok("PUT", "/api/admin/users/"+member.ID.String()+"/enabled", adminToken, models.UserEnabledRequest{Enabled: boolPtr(false)})
	if message := field(t, again, "message").(string); !strings.Contains(message, "0 season(s)") {
		t.Errorf("repeat disable message: got %q, want 0 seasons", message)
	}

	// Re-enable: access is back, seasons are not.
	enabled := h.ok("PUT", "/api/admin/users/"+member.ID.String()+"/enabled", adminToken, models.UserEnabledRequest{Enabled: boolPtr(true)})
	if value := field(t, enabled, "user", "enabled"); value != true {
		t.Errorf("re-enabled user enabled: got %v, want true", value)
	}
	h.ok("GET", "/api/auth/seasons", memberToken, nil)
	if userIDs := seasonGoalUserIDs(t, h, ongoing.ID, peerToken); containsString(userIDs, member.ID.String()) {
		t.Errorf("re-enabling put the member back into the ongoing season")
	}

	// An admin may re-enable (a no-op) themselves; only disabling self is refused.
	h.ok("PUT", "/api/admin/users/"+admin.ID.String()+"/enabled", adminToken, models.UserEnabledRequest{Enabled: boolPtr(true)})
}

// TestHistoryRendersDisabledUsers pins the read paths that must keep working when a user
// in them is disabled: debts, wheel views, invites and exercise days.
func TestHistoryRendersDisabledUsers(t *testing.T) {
	newControllerTestDB(t)

	user := createTestUser(t, "gone@history.test", "Gone")
	if err := database.SetUserEnabled(user.ID, false); err != nil {
		t.Fatal(err)
	}
	season := seedSeason(t, "History", time.Now().AddDate(0, -2, 0), time.Now().AddDate(0, -1, 0), true)

	debtObject, err := ConvertDebtToDebtObject(models.Debt{LoserID: user.ID, WinnerID: &user.ID, SeasonID: season.ID})
	if err != nil {
		t.Fatalf("debt conversion failed for disabled users: %v", err)
	}
	if debtObject.Loser.FirstName != "Gone" || debtObject.Winner == nil || debtObject.Winner.FirstName != "Gone" {
		t.Errorf("disabled debt users not rendered by name: loser %+v, winner %+v", debtObject.Loser, debtObject.Winner)
	}

	goalObject, err := ConvertGoalToGoalObject(models.Goal{UserID: user.ID})
	if err != nil {
		t.Fatalf("goal conversion failed for a disabled user: %v", err)
	}
	if goalObject.User.ID != user.ID {
		t.Errorf("goal user: got %v, want %v", goalObject.User.ID, user.ID)
	}

	inviteObject, err := ConvertInviteToInviteObject(models.Invite{RecipientID: &user.ID})
	if err != nil || inviteObject.Recipient == nil || inviteObject.Recipient.ID != user.ID {
		t.Errorf("invite conversion for a disabled recipient: got (%+v, %v)", inviteObject.Recipient, err)
	}
}
