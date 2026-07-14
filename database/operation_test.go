package database

import (
	"testing"
	"time"

	"github.com/aunefyren/treningheten/models"

	"github.com/google/uuid"
)

// seedOperationStravaSet inserts an operation set carrying a Strava id and returns its ID.
func seedOperationStravaSet(t *testing.T, operationID uuid.UUID, stravaID string) uuid.UUID {
	t.Helper()
	set := models.OperationSet{OperationID: operationID, Enabled: true, StravaID: &stravaID}
	set.ID = uuid.New()
	insertRow(t, &set)
	return set.ID
}

// seedChain builds an enabled user → day → session → operation tree and returns the pieces
// the operation queries walk. The operation is linked to the given action (may be nil).
func seedChain(t *testing.T, email string, actionID *uuid.UUID) (models.User, models.Exercise, models.Operation) {
	t.Helper()
	user := makeTestUser(t, email, nil)
	day := makeDay(t, user.ID, time.Now())
	session := makeSession(t, day.ID, time.Now())
	op := makeOperation(t, session.ID, actionID)
	return user, session, op
}

func TestGetOperationsByExerciseID(t *testing.T) {
	newTestDB(t)

	_, session, op := seedChain(t, "opsbyex@example.com", nil)

	// A disabled operation under the same session must be excluded.
	disabled := models.Operation{ExerciseID: session.ID, WeightUnit: "kg", DistanceUnit: "km"}
	disabled.ID = uuid.New()
	insertRow(t, &disabled)
	disableRow(t, &models.Operation{}, disabled.ID)

	ops, err := GetOperationsByExerciseID(session.ID)
	if err != nil {
		t.Fatalf("GetOperationsByExerciseID returned error: %v", err)
	}
	if len(ops) != 1 || ops[0].ID != op.ID {
		t.Errorf("got %d ops, want 1 (enabled only)", len(ops))
	}
}

func TestGetOperationSetsByOperationID(t *testing.T) {
	newTestDB(t)

	_, _, op := seedChain(t, "setsbyop@example.com", nil)
	makeSet(t, op.ID, f64Ptr(5), nil, nil, nil)
	makeSet(t, op.ID, f64Ptr(3), nil, nil, nil)

	disabled := models.OperationSet{OperationID: op.ID}
	disabled.ID = uuid.New()
	insertRow(t, &disabled)
	disableRow(t, &models.OperationSet{}, disabled.ID)

	sets, err := GetOperationSetsByOperationID(op.ID)
	if err != nil {
		t.Fatalf("GetOperationSetsByOperationID returned error: %v", err)
	}
	if len(sets) != 2 {
		t.Errorf("got %d sets, want 2 (enabled only)", len(sets))
	}
}

func TestGetOperationsAndSetsByUserID(t *testing.T) {
	newTestDB(t)

	user, _, op := seedChain(t, "opsbyuser@example.com", nil)
	makeSet(t, op.ID, f64Ptr(1), nil, nil, nil)
	makeSet(t, op.ID, f64Ptr(2), nil, nil, nil)

	// A second, unrelated user's tree must not bleed in.
	_, _, otherOp := seedChain(t, "opsother@example.com", nil)
	makeSet(t, otherOp.ID, f64Ptr(9), nil, nil, nil)

	ops, err := GetOperationsByUserID(user.ID)
	if err != nil {
		t.Fatalf("GetOperationsByUserID returned error: %v", err)
	}
	if len(ops) != 1 || ops[0].ID != op.ID {
		t.Errorf("got %d ops for user, want 1", len(ops))
	}

	sets, err := GetOperationSetsByUserID(user.ID)
	if err != nil {
		t.Fatalf("GetOperationSetsByUserID returned error: %v", err)
	}
	if len(sets) != 2 {
		t.Errorf("got %d sets for user, want 2", len(sets))
	}
}

func TestGetOperationByIDAndUserID(t *testing.T) {
	newTestDB(t)

	user, _, op := seedChain(t, "opbyid@example.com", nil)
	stranger := makeTestUser(t, "opstranger@example.com", nil)

	found, err := GetOperationByIDAndUserID(op.ID, user.ID)
	if err != nil {
		t.Fatalf("GetOperationByIDAndUserID returned error: %v", err)
	}
	if found.ID != op.ID {
		t.Errorf("got operation %v, want %v", found.ID, op.ID)
	}

	// Scoped to the owner: another user cannot read it.
	if _, err := GetOperationByIDAndUserID(op.ID, stranger.ID); err == nil {
		t.Errorf("expected error reading another user's operation")
	}
}

func TestGetOperationSetsAndSetByIDForUser(t *testing.T) {
	newTestDB(t)

	user, _, op := seedChain(t, "setbyid@example.com", nil)
	set := models.OperationSet{OperationID: op.ID, Enabled: true, Distance: f64Ptr(4)}
	set.ID = uuid.New()
	insertRow(t, &set)

	sets, err := GetOperationSetsByOperationIDAndUserID(op.ID, user.ID)
	if err != nil {
		t.Fatalf("GetOperationSetsByOperationIDAndUserID returned error: %v", err)
	}
	if len(sets) != 1 {
		t.Errorf("got %d sets, want 1", len(sets))
	}

	one, err := GetOperationSetByIDAndUserID(set.ID, user.ID)
	if err != nil {
		t.Fatalf("GetOperationSetByIDAndUserID returned error: %v", err)
	}
	if one.ID != set.ID {
		t.Errorf("got set %v, want %v", one.ID, set.ID)
	}
}

func TestCreateAndUpdateOperationAndSet(t *testing.T) {
	newTestDB(t)

	user := makeTestUser(t, "createop@example.com", nil)
	day := makeDay(t, user.ID, time.Now())
	session := makeSession(t, day.ID, time.Now())

	op := models.Operation{Enabled: true, ExerciseID: session.ID, Type: "running", WeightUnit: "kg", DistanceUnit: "km"}
	op.ID = uuid.New()
	created, err := CreateOperationInDB(op)
	if err != nil {
		t.Fatalf("CreateOperationInDB returned error: %v", err)
	}

	created.Note = strPtr("felt great")
	updated, err := UpdateOperationInDB(created)
	if err != nil {
		t.Fatalf("UpdateOperationInDB returned error: %v", err)
	}
	if updated.Note == nil || *updated.Note != "felt great" {
		t.Errorf("operation note not persisted: %v", updated.Note)
	}

	set := models.OperationSet{Enabled: true, OperationID: created.ID, Repetitions: f64Ptr(10)}
	set.ID = uuid.New()
	createdSet, err := CreateOperationSetInDB(set)
	if err != nil {
		t.Fatalf("CreateOperationSetInDB returned error: %v", err)
	}

	createdSet.Weight = f64Ptr(60)
	updatedSet, err := UpdateOperationSetInDB(createdSet)
	if err != nil {
		t.Fatalf("UpdateOperationSetInDB returned error: %v", err)
	}
	if updatedSet.Weight == nil || *updatedSet.Weight != 60 {
		t.Errorf("set weight not persisted: %v", updatedSet.Weight)
	}
}

func TestActionCRUDAndLookups(t *testing.T) {
	newTestDB(t)

	action := models.Action{Enabled: true, Name: "Running", NorwegianName: "Løping", Type: "cardio", StravaName: "Run", HevyTemplateID: strPtr("tmpl-1")}
	action.ID = uuid.New()
	created, err := CreateActionInDB(action)
	if err != nil {
		t.Fatalf("CreateActionInDB returned error: %v", err)
	}

	all, err := GetAllEnabledActions()
	if err != nil {
		t.Fatalf("GetAllEnabledActions returned error: %v", err)
	}
	if len(all) != 1 {
		t.Errorf("got %d actions, want 1", len(all))
	}

	byID, err := GetActionByID(created.ID)
	if err != nil {
		t.Fatalf("GetActionByID returned error: %v", err)
	}
	if byID.Name != "Running" {
		t.Errorf("action name: got %q, want %q", byID.Name, "Running")
	}
	if _, err := GetActionByID(uuid.New()); err == nil {
		t.Errorf("expected error for unknown action id")
	}

	// Name lookup is case-insensitive and matches either the English or Norwegian name.
	if _, err := GetActionByName("running"); err != nil {
		t.Errorf("GetActionByName(english) returned error: %v", err)
	}
	if _, err := GetActionByName("LØPING"); err != nil {
		t.Errorf("GetActionByName(norwegian) returned error: %v", err)
	}
	if _, err := GetActionByName("swimming"); err == nil {
		t.Errorf("expected error for unknown action name")
	}

	// Strava-name lookup is case-insensitive; misses return (nil, nil).
	byStrava, err := GetActionByStravaName("run")
	if err != nil {
		t.Fatalf("GetActionByStravaName returned error: %v", err)
	}
	if byStrava == nil || byStrava.ID != created.ID {
		t.Errorf("expected to find action by strava name")
	}
	miss, err := GetActionByStravaName("ride")
	if err != nil || miss != nil {
		t.Errorf("expected (nil, nil) for unknown strava name, got %v %v", miss, err)
	}

	// Hevy template lookup.
	byHevy, err := GetActionByHevyTemplateID("tmpl-1")
	if err != nil {
		t.Fatalf("GetActionByHevyTemplateID returned error: %v", err)
	}
	if byHevy == nil || byHevy.ID != created.ID {
		t.Errorf("expected to find action by hevy template id")
	}
	missHevy, err := GetActionByHevyTemplateID("tmpl-none")
	if err != nil || missHevy != nil {
		t.Errorf("expected (nil, nil) for unknown hevy template, got %v %v", missHevy, err)
	}

	created.BodyPart = "legs"
	if _, err := UpdateActionInDB(created); err != nil {
		t.Fatalf("UpdateActionInDB returned error: %v", err)
	}
	reloaded, _ := GetActionByID(created.ID)
	if reloaded.BodyPart != "legs" {
		t.Errorf("action body part not persisted: %q", reloaded.BodyPart)
	}
}

func TestStravaOperationLookups(t *testing.T) {
	newTestDB(t)

	user, session, op := seedChain(t, "stravaops@example.com", nil)
	setID := seedOperationStravaSet(t, op.ID, "778899")

	// Operation lookup via its set's Strava id, scoped to exercise + user.
	foundOp, err := GetOperationByStravaIDAndUserIDAndExerciseID(user.ID, 778899, session.ID)
	if err != nil {
		t.Fatalf("GetOperationByStravaIDAndUserIDAndExerciseID returned error: %v", err)
	}
	if foundOp == nil || foundOp.ID != op.ID {
		t.Errorf("expected to find operation by strava id")
	}

	// Set lookup with operation id.
	foundSet, err := GetOperationSetByStravaIDAndUserIDAndOperationID(user.ID, 778899, op.ID)
	if err != nil {
		t.Fatalf("GetOperationSetByStravaIDAndUserIDAndOperationID returned error: %v", err)
	}
	if foundSet == nil || foundSet.ID != setID {
		t.Errorf("expected to find set by strava id + operation id")
	}

	// Set lookup without operation id.
	foundSet2, err := GetOperationSetByStravaIDAndUserID(user.ID, 778899)
	if err != nil {
		t.Fatalf("GetOperationSetByStravaIDAndUserID returned error: %v", err)
	}
	if foundSet2 == nil || foundSet2.ID != setID {
		t.Errorf("expected to find set by strava id alone")
	}

	// Unknown Strava id → (nil, nil).
	missing, err := GetOperationSetByStravaIDAndUserID(user.ID, 111)
	if err != nil || missing != nil {
		t.Errorf("expected (nil, nil) for unknown strava id, got %v %v", missing, err)
	}

	// The set carries a Strava id, so it appears in the Strava-only listing.
	stravaSets, err := GetStravaOperationSetsByUserID(user.ID)
	if err != nil {
		t.Fatalf("GetStravaOperationSetsByUserID returned error: %v", err)
	}
	if len(stravaSets) != 1 {
		t.Errorf("got %d strava sets, want 1", len(stravaSets))
	}
}

func TestGetActionsDoneUsingUserID(t *testing.T) {
	newTestDB(t)

	user := makeTestUser(t, "actionsdone@example.com", nil)
	run := makeAction(t, "Run", "cardio")
	lift := makeAction(t, "Lift", "strength")

	day := makeDay(t, user.ID, time.Now())
	session := makeSession(t, day.ID, time.Now())

	// Two operations of the same action must collapse to one distinct action.
	makeOperation(t, session.ID, &run.ID)
	makeOperation(t, session.ID, &run.ID)
	makeOperation(t, session.ID, &lift.ID)

	actions, err := GetActionsDoneUsingUserID(user.ID)
	if err != nil {
		t.Fatalf("GetActionsDoneUsingUserID returned error: %v", err)
	}
	if len(actions) != 2 {
		t.Errorf("got %d distinct actions, want 2", len(actions))
	}
}
