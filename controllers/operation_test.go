package controllers

import (
	"testing"

	"github.com/aunefyren/treningheten/models"
	"github.com/google/uuid"
)

func TestValidEquipment(t *testing.T) {
	for _, equipment := range []string{"barbells", "dumbbells", "bands", "rope", "bench", "treadmill", "machine"} {
		t.Run(equipment, func(t *testing.T) {
			if !validEquipment(equipment) {
				t.Errorf("validEquipment(%q) = false, want true", equipment)
			}
		})
	}
	for _, equipment := range []string{"", "anvil", "Barbells", " bench"} {
		t.Run("invalid "+equipment, func(t *testing.T) {
			if validEquipment(equipment) {
				t.Errorf("validEquipment(%q) = true, want false", equipment)
			}
		})
	}
}

func TestApplyActionToOperation(t *testing.T) {
	action := models.Action{Name: "Running", Type: "moving"}
	action.ID = uuid.New()
	operation := models.Operation{Type: "lifting"}

	applyActionToOperation(&operation, action)

	if operation.ActionID == nil || *operation.ActionID != action.ID {
		t.Errorf("ActionID = %v, want %s", operation.ActionID, action.ID)
	}
	if operation.Action == nil || operation.Action.Name != "Running" {
		t.Errorf("Action = %v, want the Running action", operation.Action)
	}
	if operation.Type != "moving" {
		t.Errorf("Type = %q, want the action's type %q", operation.Type, "moving")
	}
}
