package controllers

import (
	"fmt"
	"testing"

	"github.com/aunefyren/treningheten/database"
	"github.com/aunefyren/treningheten/models"
)

// importHevyWorkout imports one workout for a fresh user and returns its activities as the
// builder reads them (through the Convert layer, so in display order).
func importHevyWorkout(t *testing.T, workout models.HevyWorkout, templates map[string]models.HevyExerciseTemplate) []models.OperationObject {
	t.Helper()
	user := createTestUser(t, "import@hevy.test", "Hevy")
	if err := HevySyncWorkoutForUser(user, workout, templates); err != nil {
		t.Fatalf("import: %v", err)
	}
	exercise, err := database.GetExerciseForUserWithHevyWorkoutID(user.ID, workout.ID)
	if err != nil || exercise == nil {
		t.Fatalf("imported session not found: %v", err)
	}
	operations, err := database.GetOperationsByExerciseID(exercise.ID)
	if err != nil {
		t.Fatalf("GetOperationsByExerciseID: %v", err)
	}
	operationObjects, err := ConvertOperationsToOperationObjects(operations)
	if err != nil {
		t.Fatalf("ConvertOperationsToOperationObjects: %v", err)
	}
	return operationObjects
}

func operationSetObjects(t *testing.T, operation models.OperationObject) []models.OperationSetObject {
	t.Helper()
	sets, err := database.GetOperationSetsByOperationID(operation.ID)
	if err != nil {
		t.Fatalf("GetOperationSetsByOperationID: %v", err)
	}
	setObjects, err := ConvertOperationSetsToOperationSetObjects(sets)
	if err != nil {
		t.Fatalf("ConvertOperationSetsToOperationSetObjects: %v", err)
	}
	return setObjects
}

// TestHevyImportKeepsHevyOrder pins that activities and their sets come back in the order
// Hevy lists them. They used to share one CreatedAt, the only sort key, so they came back
// in whatever order the database and the sort left them.
func TestHevyImportKeepsHevyOrder(t *testing.T) {
	newControllerTestDB(t)
	database.SeedActions()

	const exerciseCount, setCount = 20, 20
	workout := hevyWorkout("hevy-order", todayAt(60), "Long session")
	workout.Exercises = nil
	templates := map[string]models.HevyExerciseTemplate{}
	for i := 0; i < exerciseCount; i++ {
		templateID := fmt.Sprintf("tmpl-custom-%02d", i)
		title := fmt.Sprintf("Custom move %02d", i)
		templates[templateID] = models.HevyExerciseTemplate{ID: templateID, Title: title, Type: "reps_only", IsCustom: true}
		exercise := models.HevyWorkoutExercise{Index: i, Title: title, ExerciseTemplateID: templateID}
		for j := 0; j < setCount; j++ {
			reps := float64(j + 1)
			exercise.Sets = append(exercise.Sets, models.HevyWorkoutSet{Index: j, Type: "normal", Reps: &reps})
		}
		workout.Exercises = append(workout.Exercises, exercise)
	}

	operations := importHevyWorkout(t, workout, templates)
	if len(operations) != exerciseCount {
		t.Fatalf("got %d activities, want %d", len(operations), exerciseCount)
	}
	for i, operation := range operations {
		if want := workout.Exercises[i].Title; operation.Note == nil || *operation.Note != want {
			t.Fatalf("activity %d = %v, want %q", i, operation.Note, want)
		}
		if i > 0 && !operation.CreatedAt.After(operations[i-1].CreatedAt) {
			t.Errorf("activity %d CreatedAt %v is not after the previous one's %v", i, operation.CreatedAt, operations[i-1].CreatedAt)
		}
		for j, set := range operationSetObjects(t, operation) {
			if set.Repetitions == nil || *set.Repetitions != float64(j+1) {
				t.Fatalf("activity %d set %d has %v reps, want %d", i, j, set.Repetitions, j+1)
			}
		}
	}
}

// TestHevyImportStairMachines covers Hevy's floors_duration and steps_duration exercises:
// they are timed activities, and the set's custom_metric is the floor or step count.
func TestHevyImportStairMachines(t *testing.T) {
	newControllerTestDB(t)
	database.SeedActions()

	duration, floors, steps := 900.0, 42.0, 1800.0
	templates := map[string]models.HevyExerciseTemplate{
		"4377A52B": {ID: "4377A52B", Title: "Stair Machine (Floors)", Type: "floors_duration", PrimaryMuscleGroup: "cardio"},
		"4377A52C": {ID: "4377A52C", Title: "Stair Machine (Steps)", Type: "steps_duration", PrimaryMuscleGroup: "cardio"},
	}
	workout := hevyWorkout("hevy-stairs", todayAt(60), "Stairs")
	workout.Exercises = []models.HevyWorkoutExercise{
		{Index: 0, Title: "Stair Machine (Floors)", ExerciseTemplateID: "4377A52B", Sets: []models.HevyWorkoutSet{
			{Index: 0, Type: "normal", DurationSeconds: &duration, CustomMetric: &floors},
		}},
		{Index: 1, Title: "Stair Machine (Steps)", ExerciseTemplateID: "4377A52C", Sets: []models.HevyWorkoutSet{
			{Index: 0, Type: "normal", DurationSeconds: &duration, CustomMetric: &steps},
		}},
	}

	operations := importHevyWorkout(t, workout, templates)
	if len(operations) != 2 {
		t.Fatalf("got %d activities, want 2", len(operations))
	}
	cases := []struct{ wantFloors, wantSteps *float64 }{{&floors, nil}, {nil, &steps}}
	for i, tc := range cases {
		operation := operations[i]
		if operation.Type != "timing" || operation.Action == nil || operation.Action.Type != "timing" {
			t.Errorf("activity %d type = %q (action %v), want timing", i, operation.Type, operation.Action)
		}
		sets := operationSetObjects(t, operation)
		if len(sets) != 1 {
			t.Fatalf("activity %d has %d sets, want 1", i, len(sets))
		}
		set := sets[0]
		if set.Time == nil || *set.Time != int64(duration) {
			t.Errorf("activity %d time = %v, want %v", i, set.Time, duration)
		}
		if !equalFloatPtr(set.Floors, tc.wantFloors) || !equalFloatPtr(set.Steps, tc.wantSteps) {
			t.Errorf("activity %d floors/steps = %v/%v, want %v/%v", i, set.Floors, set.Steps, tc.wantFloors, tc.wantSteps)
		}
	}
}

// TestHevyImportMatchesCustomExerciseByName covers a custom Hevy exercise named like an
// existing action: it becomes that action. One with no namesake keeps its title on the
// activity, without an action.
func TestHevyImportMatchesCustomExerciseByName(t *testing.T) {
	newControllerTestDB(t)
	database.SeedActions()

	padel, err := database.GetEnabledActionByName("Padel")
	if err != nil || padel == nil {
		t.Fatalf("curated Padel action not seeded: %v", err)
	}

	duration := 3600.0
	templates := map[string]models.HevyExerciseTemplate{
		"tmpl-padel":  {ID: "tmpl-padel", Title: "padel", Type: "duration", IsCustom: true},
		"tmpl-custom": {ID: "tmpl-custom", Title: "My Secret Move", Type: "duration", IsCustom: true},
	}
	workout := hevyWorkout("hevy-custom", todayAt(60), "Padel night")
	workout.Exercises = []models.HevyWorkoutExercise{
		{Index: 0, Title: "padel", ExerciseTemplateID: "tmpl-padel", Notes: "won", Sets: []models.HevyWorkoutSet{
			{Index: 0, Type: "normal", DurationSeconds: &duration},
		}},
		{Index: 1, Title: "My Secret Move", ExerciseTemplateID: "tmpl-custom", Sets: []models.HevyWorkoutSet{
			{Index: 0, Type: "normal", DurationSeconds: &duration},
		}},
	}

	operations := importHevyWorkout(t, workout, templates)
	if len(operations) != 2 {
		t.Fatalf("got %d activities, want 2", len(operations))
	}

	matched := operations[0]
	if matched.Action == nil || matched.Action.ID != padel.ID {
		t.Errorf("custom \"padel\" action = %v, want the curated Padel", matched.Action)
	}
	if matched.Note != nil || matched.Description == nil || *matched.Description != "won" {
		t.Errorf("matched activity note/description = %v/%v, want nil/\"won\"", matched.Note, matched.Description)
	}
	if matched.Type != padel.Type {
		t.Errorf("matched activity type = %q, want the action's %q", matched.Type, padel.Type)
	}

	unmatched := operations[1]
	if unmatched.Action != nil {
		t.Errorf("custom \"My Secret Move\" got action %q, want none", unmatched.Action.Name)
	}
	if unmatched.Note == nil || *unmatched.Note != "My Secret Move" || unmatched.Type != "timing" {
		t.Errorf("unmatched activity note/type = %v/%q, want \"My Secret Move\"/timing", unmatched.Note, unmatched.Type)
	}
}

func equalFloatPtr(a, b *float64) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}
