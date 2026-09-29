package controllers

import (
	"net/http"
	"testing"

	"github.com/aunefyren/treningheten/models"

	"github.com/google/uuid"
)

func TestActivityGoalSettings(t *testing.T) {
	h := newAPIHarness(t)
	_, token := h.user("settings@goal.test", false)

	listed := h.ok("GET", "/api/auth/activity-goal-settings", token, nil)
	settings := field(t, listed, "activity_goal_settings").([]any)
	if len(settings) == 0 {
		t.Fatal("no activity types listed")
	}
	for _, setting := range settings {
		if field(t, setting, "counts_toward_goal") != true {
			t.Fatal("an activity type defaults to not counting")
		}
	}

	actionID := uuid.MustParse(field(t, settings[0], "action_id").(string))
	updated := h.ok("PUT", "/api/auth/activity-goal-settings", token, models.ActivityGoalSettingUpdateRequest{ActionID: actionID, CountsTowardGoal: false})
	for _, setting := range field(t, updated, "activity_goal_settings").([]any) {
		if field(t, setting, "action_id") == actionID.String() && field(t, setting, "counts_toward_goal") != false {
			t.Error("the opt-out was not stored")
		}
	}
	// Flip it back (upsert on an existing row).
	h.ok("PUT", "/api/auth/activity-goal-settings", token, models.ActivityGoalSettingUpdateRequest{ActionID: actionID, CountsTowardGoal: true})

	h.expect(http.StatusNotFound, "PUT", "/api/auth/activity-goal-settings", token, models.ActivityGoalSettingUpdateRequest{ActionID: uuid.New()})
	h.expect(http.StatusBadRequest, "PUT", "/api/auth/activity-goal-settings", token, "{broken")

	// Another user's view is unaffected.
	_, otherToken := h.user("other@goal.test", false)
	for _, setting := range field(t, h.ok("GET", "/api/auth/activity-goal-settings", otherToken, nil), "activity_goal_settings").([]any) {
		if field(t, setting, "counts_toward_goal") != true {
			t.Error("one user's setting leaked to another")
		}
	}
}

func TestCustomActionsAndPrizes(t *testing.T) {
	h := newAPIHarness(t)
	_, adminToken := h.user("admin@catalog.test", true)
	_, token := h.user("user@catalog.test", false)

	created := h.expect(http.StatusCreated, "POST", "/api/auth/actions", token, models.ActionCreationRequest{
		Name: "Underwater Juggling", NorwegianName: "Undervannssjonglering", Description: "Hard", Type: "timing", BodyPart: "arms",
	})
	if field(t, created, "action", "name") != "Underwater Juggling" {
		t.Errorf("created action = %v", created)
	}
	for name, request := range map[string]models.ActionCreationRequest{
		"no names": {Type: "lifting"},
		"bad type": {Name: "Thing", Type: "flying"},
	} {
		if code := h.do("POST", "/api/auth/actions", token, request).Code; code != http.StatusBadRequest {
			t.Errorf("%s: status = %d, want 400", name, code)
		}
	}

	h.expect(http.StatusBadRequest, "POST", "/api/admin/prizes", adminToken, models.PrizeCreationRequest{Name: "", Quantity: 1})
	h.expect(http.StatusBadRequest, "POST", "/api/admin/prizes", adminToken, models.PrizeCreationRequest{Name: "Tiny", Quantity: 1})
	h.ok("POST", "/api/admin/prizes", adminToken, models.PrizeCreationRequest{Name: "Pizza night", Quantity: 1})
	h.expect(http.StatusBadRequest, "POST", "/api/admin/prizes", adminToken, models.PrizeCreationRequest{Name: "Pizza night", Quantity: 1})
	h.expect(http.StatusBadRequest, "POST", "/api/admin/prizes", adminToken, "{broken")
}
