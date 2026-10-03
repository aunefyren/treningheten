package controllers

import "testing"

func TestValidBodyWeight(t *testing.T) {
	tests := []struct {
		name   string
		weight float64
		want   bool
	}{
		{"zero", 0, false},
		{"negative", -1, false},
		{"tiny positive", 0.1, true},
		{"typical", 80.5, true},
		{"upper bound", maxBodyWeightKg, true},
		{"just above bound", maxBodyWeightKg + 0.01, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := validBodyWeight(tt.weight); got != tt.want {
				t.Errorf("validBodyWeight(%v) = %v, want %v", tt.weight, got, tt.want)
			}
		})
	}
}
