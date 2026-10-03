package controllers

import (
	"net/http"
	"testing"

	"github.com/aunefyren/treningheten/database"
	"github.com/aunefyren/treningheten/models"

	"github.com/google/uuid"
)

func TestOperationSetOwnershipAndStravaEdges(t *testing.T) {
	h := newAPIHarness(t)
	_, adminToken := h.user("admin@edges.test", true)
	_, token := h.user("owner@edges.test", false)
	_, strangerToken := h.user("stranger@edges.test", false)

	sessionID := createTodaySession(t, h, token)
	operation := h.expect(http.StatusCreated, "POST", "/api/auth/operations", token, models.OperationCreationRequest{
		ExerciseID: uuid.MustParse(sessionID), Type: "lifting", WeightUnit: "kg", DistanceUnit: "km",
	})
	operationID := idOf(t, operation, "operation")
	set := h.expect(http.StatusCreated, "POST", "/api/auth/operation-sets", token, models.OperationSetCreationRequest{
		OperationID: uuid.MustParse(operationID), Repetitions: float64Ptr(5), Weight: float64Ptr(100),
	})
	setID := idOf(t, set, "operation_set")

	// Someone else can't delete the set; bad ids are client errors.
	// Listing is scoped to the caller: a stranger gets an empty list, not the owner's sets.
	if sets := field(t, h.ok("GET", "/api/auth/operation-sets?operation_id="+operationID, strangerToken, nil), "operation_sets").([]any); len(sets) != 0 {
		t.Errorf("stranger sees %d of the owner's sets", len(sets))
	}
	// Twice: the lookup used to return an empty set as a hit, and saving it inserted a
	// row with a nil id — 200 the first time, a duplicate-key 500 after.
	for attempt := 1; attempt <= 2; attempt++ {
		h.expect(http.StatusNotFound, "DELETE", "/api/auth/operation-sets/"+setID, strangerToken, nil)
		h.expect(http.StatusNotFound, "DELETE", "/api/auth/operation-sets/"+uuid.NewString(), token, nil)
		h.expect(http.StatusNotFound, "PUT", "/api/auth/operation-sets/"+setID, strangerToken, models.OperationSetUpdateRequest{Repetitions: float64Ptr(1)})
	}
	var nilSets int64
	database.Instance.Model(&models.OperationSet{}).Where("id = ?", uuid.Nil).Count(&nilSets)
	if nilSets != 0 {
		t.Errorf("%d operation sets with a nil id were written", nilSets)
	}
	h.expect(http.StatusBadRequest, "DELETE", "/api/auth/operation-sets/nope", token, nil)
	h.expect(http.StatusBadRequest, "PUT", "/api/auth/operation-sets/nope", token, models.OperationSetUpdateRequest{})
	h.expect(http.StatusBadRequest, "DELETE", "/api/auth/operations/nope", token, nil)
	h.expect(http.StatusBadRequest, "GET", "/api/auth/operations/nope", token, nil)

	// Strava re-sync of a set: refused while Strava is off, and for a manual set.
	h.expect(http.StatusBadRequest, "POST", "/api/auth/operation-sets/"+setID+"/strava-sync", token, nil)
	withStrava(t)
	h.expect(http.StatusBadRequest, "POST", "/api/auth/operation-sets/"+setID+"/strava-sync", token, nil)
	h.expect(http.StatusBadRequest, "POST", "/api/auth/operation-sets/nope/strava-sync", token, nil)

	// Admin bulk Strava sync validates its targets.
	h.expect(http.StatusBadRequest, "POST", "/api/admin/strava/sync-activities-for-users", adminToken, "{broken")
	h.expect(http.StatusBadRequest, "POST", "/api/admin/strava/sync-activities-for-users", adminToken, models.StravaSyncActivitiesForUsersRequest{UserIDs: []string{"nope"}})
	h.expect(http.StatusNotFound, "POST", "/api/admin/strava/sync-activities-for-users", adminToken, models.StravaSyncActivitiesForUsersRequest{UserIDs: []string{uuid.NewString()}})

	h.ok("DELETE", "/api/auth/operation-sets/"+setID, token, nil)
}

func TestGearSelectionAndActivityVisibilityEdges(t *testing.T) {
	h := newAPIHarness(t)
	owner, token := h.user("gearowner@edges.test", false)
	_, otherToken := h.user("gearother@edges.test", false)

	sessionID := createTodaySession(t, h, token)
	gearPath := "/api/auth/exercises/" + sessionID + "/gear"

	h.expect(http.StatusBadRequest, "PUT", "/api/auth/exercises/nope/gear", token, map[string]any{"gear_id": nil})
	h.expect(http.StatusBadRequest, "PUT", gearPath, token, "{broken")
	h.expect(http.StatusBadRequest, "PUT", gearPath, token, map[string]any{"gear_id": uuid.NewString()})
	h.expect(http.StatusNotFound, "PUT", "/api/auth/exercises/"+uuid.NewString()+"/gear", token, map[string]any{"gear_id": nil})
	// Clearing gear on a session with no activities is a no-op success.
	h.ok("PUT", gearPath, token, map[string]any{"gear_id": nil})

	// Another user's gear can't be attached.
	othersGear := h.expect(http.StatusCreated, "POST", "/api/auth/gear", otherToken, models.GearCreationRequest{Name: "Borrowed", Type: "bike"})
	h.expect(http.StatusBadRequest, "PUT", gearPath, token, map[string]any{"gear_id": idOf(t, othersGear, "gear")})
	h.expect(http.StatusNotFound, "PUT", "/api/auth/gear/"+idOf(t, othersGear, "gear"), token, models.GearUpdateRequest{Nickname: stringPtr("mine now")})
	h.expect(http.StatusBadRequest, "PUT", "/api/auth/gear/nope", token, models.GearUpdateRequest{})

	// A user who turned sharing off hides their activity page from others.
	h.ok("GET", "/api/auth/users/"+owner.ID.String()+"/activities", otherToken, nil)
	h.expect(http.StatusBadRequest, "GET", "/api/auth/users/nope/activities", otherToken, nil)
}
