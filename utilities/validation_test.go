package utilities

import "testing"

func TestValidHexColor(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want bool
	}{
		{"valid lowercase", "#4363d8", true},
		{"valid uppercase", "#AABBCC", true},
		{"valid mixed case", "#aAbBcC", true},
		{"all digits", "#123456", true},
		{"missing hash", "4363d8", false},
		{"three digit short form", "#abc", false},
		{"too long", "#1234567", false},
		{"non-hex char", "#12345g", false},
		{"empty", "", false},
		{"hash only", "#", false},
		{"trailing space", "#4363d8 ", false},
		{"named color", "red", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ValidHexColor(tt.in); got != tt.want {
				t.Errorf("ValidHexColor(%q) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}

func TestValidWheelEmoji(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want bool
	}{
		{"single emoji", "🎉", true},
		{"emoji with variation selector", "❤️", true},
		{"family zwj sequence", "👨‍👩‍👧‍👦", true},
		{"flag", "🇳🇴", true},
		{"trimmed surrounding space", "  🎯  ", true},
		{"empty", "", false},
		{"whitespace only", "   ", false},
		{"plain ascii letter", "a", false},
		{"plain ascii word", "spin", false},
		{"digit", "7", false},
		{"just over rune limit", "🎉🎊🎉🎊🎉🎊🎉🎊🎉", false},
		{"well over rune limit", "🎉🎉🎉🎉🎉🎉🎉🎉🎉🎉🎉🎉🎉🎉🎉🎉🎉", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ValidWheelEmoji(tt.in); got != tt.want {
				t.Errorf("ValidWheelEmoji(%q) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}
