package assistant

import (
	"encoding/json"
	"regexp"
	"strings"
)

// reSuggestions matches a trailing fenced ```suggestions block, tolerant of
// surrounding whitespace and newlines. The block must appear at the very end
// of the text — the model is prompted to emit it as the last thing before
// ending its turn, and requiring trailing-only keeps extraction simple and
// avoids false positives mid-text.
var reSuggestions = regexp.MustCompile("(?s)\n```suggestions\\s*\\n(.*?)```\\s*$")

// ExtractSuggestions looks for a trailing fenced ```suggestions block at the
// end of text, parses it as a JSON array of strings, and returns the text
// with the block stripped plus the parsed options. If no block is present or
// it doesn't parse as expected, returns the original text unchanged and a nil
// slice — this must never turn into an error for the whole turn, most turns
// simply won't have the block.
func ExtractSuggestions(text string) (cleaned string, suggestions []string) {
	loc := reSuggestions.FindStringSubmatchIndex(text)
	if loc == nil {
		return text, nil
	}

	// Extract the raw JSON inside the fence (capture group 1).
	raw := text[loc[2]:loc[3]]

	var opts []string
	if err := json.Unmarshal([]byte(raw), &opts); err != nil {
		return text, nil
	}

	// Filter out empty strings and validate types (json.Unmarshal into
	// []string already rejects non-string array elements, but guard
	// against an empty array as well).
	if len(opts) == 0 {
		return text, nil
	}

	// Remove the fenced block from the text (loc[0] is start of the
	// leading newline before ```, loc[1] is end of the closing ```).
	cleaned = strings.TrimRight(text[:loc[0]], "\n")

	return cleaned, opts
}
