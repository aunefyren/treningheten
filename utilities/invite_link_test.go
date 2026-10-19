package utilities

import "testing"

func TestInviteLink(t *testing.T) {
	tests := []struct {
		name        string
		externalURL string
		code        string
		want        string
	}{
		{"plain", "https://trening.example.com", "ABC123", "https://trening.example.com/register?invite=ABC123"},
		{"trailing slash trimmed", "https://trening.example.com/", "ABC123", "https://trening.example.com/register?invite=ABC123"},
		{"sub-path kept", "https://example.com/trening", "ABC123", "https://example.com/trening/register?invite=ABC123"},
		{"code is escaped", "https://trening.example.com", "a b&c", "https://trening.example.com/register?invite=a+b%26c"},
		{"no external URL", "", "ABC123", ""},
		{"blank external URL", "   ", "ABC123", ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := InviteLink(tc.externalURL, tc.code); got != tc.want {
				t.Fatalf("InviteLink(%q, %q) = %q, want %q", tc.externalURL, tc.code, got, tc.want)
			}
		})
	}
}
