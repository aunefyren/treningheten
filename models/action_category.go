package models

import "strings"

// Action categories say what kind of training an action is, independent of how it is
// measured (Action.Type: lifting/moving/timing). Weight Training and Yoga are both measured
// by time, Plank by sets, but they are strength, mobility and strength respectively. The
// category drives the session headline (see SessionHeadline): the title, the feed's
// "did …" phrase and its logo.
const (
	ActionCategoryStrength = "strength"
	ActionCategoryCardio   = "cardio"
	ActionCategorySport    = "sport"
	ActionCategoryMobility = "mobility"
	ActionCategoryGeneral  = "general"
)

// ValidActionCategory reports whether category is one of the known action categories.
func ValidActionCategory(category string) bool {
	switch category {
	case ActionCategoryStrength, ActionCategoryCardio, ActionCategorySport, ActionCategoryMobility, ActionCategoryGeneral:
		return true
	}
	return false
}

// DefaultActionCategory is the category an action gets when nobody chose one, inferred from
// how it is measured: sets are strength, distance is cardio, and time could be anything.
func DefaultActionCategory(actionType string) string {
	switch actionType {
	case "lifting":
		return ActionCategoryStrength
	case "moving":
		return ActionCategoryCardio
	}
	return ActionCategoryGeneral
}

// Action equipment is what an exercise is performed with — a property of the exercise
// variant ("Bench Press (Dumbbell)" vs "(Barbell)"), so it lives on the Action rather than
// on each logged Operation. Empty means unknown or none.
const (
	ActionEquipmentBarbell    = "barbell"
	ActionEquipmentDumbbell   = "dumbbell"
	ActionEquipmentMachine    = "machine"
	ActionEquipmentCable      = "cable"
	ActionEquipmentSmith      = "smith_machine"
	ActionEquipmentKettlebell = "kettlebell"
	ActionEquipmentBand       = "band"
	ActionEquipmentSuspension = "suspension"
	ActionEquipmentBodyweight = "bodyweight"
	ActionEquipmentTrapBar    = "trap_bar"
	ActionEquipmentSandbag    = "sandbag"
)

// ValidActionEquipment reports whether equipment is empty (none/unknown) or one of the known
// action equipment values.
func ValidActionEquipment(equipment string) bool {
	switch equipment {
	case "", ActionEquipmentBarbell, ActionEquipmentDumbbell, ActionEquipmentMachine, ActionEquipmentCable,
		ActionEquipmentSmith, ActionEquipmentKettlebell, ActionEquipmentBand, ActionEquipmentSuspension,
		ActionEquipmentBodyweight, ActionEquipmentTrapBar, ActionEquipmentSandbag:
		return true
	}
	return false
}

// hevyTitleEquipment maps the bracketed qualifier Hevy puts in exercise titles onto the
// equipment vocabulary. Qualifiers that are not equipment ("Weighted", "Assisted", grips,
// directions) are deliberately absent.
var hevyTitleEquipment = map[string]string{
	"barbell":        ActionEquipmentBarbell,
	"dumbbell":       ActionEquipmentDumbbell,
	"machine":        ActionEquipmentMachine,
	"machine plates": ActionEquipmentMachine,
	"pec deck":       ActionEquipmentMachine,
	"cable":          ActionEquipmentCable,
	"smith machine":  ActionEquipmentSmith,
	"smith":          ActionEquipmentSmith,
	"kettlebell":     ActionEquipmentKettlebell,
	"band":           ActionEquipmentBand,
	"suspension":     ActionEquipmentSuspension,
	"bodyweight":     ActionEquipmentBodyweight,
	"trap bar":       ActionEquipmentTrapBar,
	"sandbag":        ActionEquipmentSandbag,
}

// HevyTitleEquipment reads the equipment out of a Hevy exercise title such as
// "Bench Press (Dumbbell)". Returns "" when the title has no bracketed equipment.
func HevyTitleEquipment(title string) string {
	open := strings.LastIndex(title, "(")
	end := strings.LastIndex(title, ")")
	if open < 0 || end < open {
		return ""
	}
	qualifier := strings.ToLower(strings.TrimSpace(title[open+1 : end]))
	return hevyTitleEquipment[qualifier]
}

// HevyActionCategory maps a Hevy template onto an action category. Hevy is a strength
// tracker, so everything is strength except its cardio entries (running, rowing machine,
// spinning, …, by primary muscle group) and its "other" group (stretching), which is
// mobility.
func HevyActionCategory(primaryMuscleGroup string) string {
	switch strings.ToLower(strings.TrimSpace(primaryMuscleGroup)) {
	case "cardio":
		return ActionCategoryCardio
	case "other":
		return ActionCategoryMobility
	}
	return ActionCategoryStrength
}
