package controllers

import "testing"

func TestOperationTakesGear(t *testing.T) {
	tests := map[string]bool{"moving": true, "timing": true, "lifting": false, "": false}
	for operationType, want := range tests {
		if got := operationTakesGear(operationType); got != want {
			t.Errorf("operationTakesGear(%q) = %v, want %v", operationType, got, want)
		}
	}
}
