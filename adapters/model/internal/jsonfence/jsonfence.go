// Package jsonfence strips markdown code fences that cloud LLMs (Anthropic,
// OpenAI-compatible backends without a reliable JSON-only mode) sometimes wrap
// JSON responses in despite being asked for raw JSON.
package jsonfence

import "strings"

// Strip removes a leading/trailing ``` or ```json fence and surrounding
// whitespace from raw. If raw has no fence, it is returned trimmed and
// otherwise unchanged.
func Strip(raw string) string {
	s := strings.TrimSpace(raw)
	if !strings.HasPrefix(s, "```") {
		return s
	}
	s = strings.TrimPrefix(s, "```")
	if nl := strings.IndexByte(s, '\n'); nl != -1 && strings.TrimSpace(s[:nl]) != "" {
		// First line after the fence is a language tag (e.g. "json"), not content.
		s = s[nl+1:]
	}
	s = strings.TrimSuffix(strings.TrimSpace(s), "```")
	return strings.TrimSpace(s)
}
