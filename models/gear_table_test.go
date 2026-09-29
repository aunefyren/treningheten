package models

import "testing"

func TestGearTableNameIsUncountable(t *testing.T) {
	if name := (Gear{}).TableName(); name != "gear" {
		t.Errorf("TableName = %q, want gear", name)
	}
}
