package web

import "strings"

// IsSPARoute reports whether path is declared by the dashboard router. Keep
// this list aligned with web/src/App.tsx so direct navigation gets the SPA only
// for routes the client can render.
func IsSPARoute(path string) bool {
	path = strings.Trim(path, "/")
	if path == "" {
		return true
	}

	parts := strings.Split(path, "/")
	for _, route := range spaRoutes {
		if len(route) != len(parts) {
			continue
		}
		matched := true
		for i, part := range parts {
			if route[i] != ":" && route[i] != part {
				matched = false
				break
			}
		}
		if matched {
			return true
		}
	}
	return false
}

var spaRoutes = [][]string{
	{"login"},
	{"register"},
	{"auth", "callback"},
	{"verify-email"},
	{"forgot-password"},
	{"reset-password"},
	{"magic"},
	{"shared", ":"},
	{"chat"},
	{"log"},
	{"history"},
	{"history", ":"},
	{"trends"},
	{"summary"},
	{"settings"},
	{"settings", "security"},
	{"settings", "link-bot"},
	{"settings", "aliases"},
	{"settings", "aliases", "pending"},
	{"settings", "precedence"},
	{"settings", "nudges"},
	{"settings", "backup"},
	{"settings", "ai-key"},
	{"settings", "assistant"},
	{"settings", "deleted-chats"},
	{"settings", "hevy-import"},
	{"foods"},
	{"templates"},
	{"body"},
	{"body", ":"},
	{"goals"},
}
