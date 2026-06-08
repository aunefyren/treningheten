package utilities

import (
	"regexp"
	"strings"
	"unicode/utf8"
)

var hexColorRegex = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)

// ValidHexColor reports whether s is a 6-digit hex color like "#4363d8".
func ValidHexColor(s string) bool {
	return hexColorRegex.MatchString(s)
}

// ValidWheelEmoji reports whether s is acceptable as a wheel emoji: short, and
// containing at least one non-ASCII (symbol/emoji) rune. This blocks plain text and
// long strings while allowing single emoji, variation selectors, flags and ZWJ
// sequences (e.g. family emoji).
func ValidWheelEmoji(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return false
	}
	// A complex ZWJ/flag sequence can be several runes / dozens of bytes; cap both.
	if len(s) > 64 || utf8.RuneCountInString(s) > 8 {
		return false
	}
	hasSymbol := false
	for _, r := range s {
		if r > 0x2000 {
			hasSymbol = true
			break
		}
	}
	return hasSymbol
}
