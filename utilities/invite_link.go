package utilities

import (
	"net/url"
	"strings"
)

// InviteLink is the register link that fills in an invite code for the recipient
// (/register?invite=CODE; web/js/register.js reads it). It returns "" when the app's external
// URL isn't configured, since a relative link is no use to someone outside the app.
func InviteLink(externalURL string, code string) string {
	base := strings.TrimRight(strings.TrimSpace(externalURL), "/")
	if base == "" {
		return ""
	}
	return base + "/register?invite=" + url.QueryEscape(code)
}
