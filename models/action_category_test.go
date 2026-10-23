package models

import "testing"

func TestValidActionCategory(t *testing.T) {
	for _, category := range []string{ActionCategoryStrength, ActionCategoryCardio, ActionCategorySport, ActionCategoryMobility, ActionCategoryGeneral} {
		if !ValidActionCategory(category) {
			t.Errorf("ValidActionCategory(%q) = false, want true", category)
		}
	}
	for _, category := range []string{"", "vibes", "Strength", SessionCategoryMixed} {
		if ValidActionCategory(category) {
			t.Errorf("ValidActionCategory(%q) = true, want false", category)
		}
	}
}

func TestDefaultActionCategory(t *testing.T) {
	tests := map[string]string{
		"lifting": ActionCategoryStrength,
		"moving":  ActionCategoryCardio,
		"timing":  ActionCategoryGeneral,
		"":        ActionCategoryGeneral,
	}
	for actionType, want := range tests {
		t.Run(actionType, func(t *testing.T) {
			if got := DefaultActionCategory(actionType); got != want {
				t.Fatalf("DefaultActionCategory(%q) = %q, want %q", actionType, got, want)
			}
		})
	}
}

func TestValidActionEquipment(t *testing.T) {
	for _, equipment := range []string{"", ActionEquipmentBarbell, ActionEquipmentDumbbell, ActionEquipmentMachine, ActionEquipmentCable,
		ActionEquipmentSmith, ActionEquipmentKettlebell, ActionEquipmentBand, ActionEquipmentSuspension,
		ActionEquipmentBodyweight, ActionEquipmentTrapBar, ActionEquipmentSandbag} {
		if !ValidActionEquipment(equipment) {
			t.Errorf("ValidActionEquipment(%q) = false, want true", equipment)
		}
	}
	// The legacy per-operation values are not action equipment.
	for _, equipment := range []string{"anvil", "barbells", "treadmill", "rope"} {
		if ValidActionEquipment(equipment) {
			t.Errorf("ValidActionEquipment(%q) = true, want false", equipment)
		}
	}
}

func TestHevyTitleEquipment(t *testing.T) {
	tests := []struct {
		title string
		want  string
	}{
		{"Bench Press (Dumbbell)", ActionEquipmentDumbbell},
		{"Squat (Barbell)", ActionEquipmentBarbell},
		{"Chest Fly (Machine)", ActionEquipmentMachine},
		{"Seated Row (Machine Plates)", ActionEquipmentMachine},
		{"Butterfly (Pec Deck)", ActionEquipmentMachine},
		{"Triceps Pushdown (Cable)", ActionEquipmentCable},
		{"Squat (Smith Machine)", ActionEquipmentSmith},
		{"Bench Press (Smith)", ActionEquipmentSmith},
		{"Swing (Kettlebell)", ActionEquipmentKettlebell},
		{"Face Pull (band)", ActionEquipmentBand},
		{"Row (Suspension)", ActionEquipmentSuspension},
		{"Lunge (Bodyweight)", ActionEquipmentBodyweight},
		{"Deadlift (Trap bar)", ActionEquipmentTrapBar},
		{"Walking Lunge (Sandbag)", ActionEquipmentSandbag},
		{"Pull Up (Weighted)", ""}, // a qualifier, not equipment
		{"Back Extension (Hyperextension)", ""},
		{"Plank", ""},
		{"Odd )title(", ""},
	}
	for _, tc := range tests {
		t.Run(tc.title, func(t *testing.T) {
			if got := HevyTitleEquipment(tc.title); got != tc.want {
				t.Fatalf("HevyTitleEquipment(%q) = %q, want %q", tc.title, got, tc.want)
			}
		})
	}
}

func TestHevyActionCategory(t *testing.T) {
	tests := map[string]string{
		"cardio":    ActionCategoryCardio,
		" Cardio ":  ActionCategoryCardio,
		"other":     ActionCategoryMobility,
		"chest":     ActionCategoryStrength,
		"full_body": ActionCategoryStrength,
		"":          ActionCategoryStrength,
	}
	for group, want := range tests {
		t.Run(group, func(t *testing.T) {
			if got := HevyActionCategory(group); got != want {
				t.Fatalf("HevyActionCategory(%q) = %q, want %q", group, got, want)
			}
		})
	}
}

func TestHevyToActionCategoryAndEquipment(t *testing.T) {
	action := HevyExerciseTemplate{ID: "T1", Title: "Bench Press (Dumbbell)", Type: "weight_reps", PrimaryMuscleGroup: "chest"}.ToAction()
	if action.Category != ActionCategoryStrength || action.Equipment != ActionEquipmentDumbbell {
		t.Fatalf("category/equipment = %q/%q, want strength/dumbbell", action.Category, action.Equipment)
	}
}
