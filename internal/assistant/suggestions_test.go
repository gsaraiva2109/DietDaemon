package assistant

import (
	"context"
	"strings"
	"testing"

	"github.com/gsaraiva2109/dietdaemon/core/ports"
)

func TestExtractSuggestions_ValidBlock(t *testing.T) {
	text := "Here are some options for you.\n\n```suggestions\n[\"Yes, log it\", \"Actually, use /suggest instead\"]\n```"
	cleaned, suggestions := ExtractSuggestions(text)

	if len(suggestions) != 2 {
		t.Fatalf("got %d suggestions, want 2", len(suggestions))
	}
	if suggestions[0] != "Yes, log it" {
		t.Errorf("suggestions[0] = %q, want 'Yes, log it'", suggestions[0])
	}
	if suggestions[1] != "Actually, use /suggest instead" {
		t.Errorf("suggestions[1] = %q, want 'Actually, use /suggest instead'", suggestions[1])
	}
	if strings.Contains(cleaned, "```suggestions") {
		t.Errorf("cleaned text still contains fenced block: %q", cleaned)
	}
	if !strings.Contains(cleaned, "Here are some options for you.") {
		t.Errorf("cleaned text missing original content: %q", cleaned)
	}
}

func TestExtractSuggestions_NoBlock(t *testing.T) {
	text := "Your daily summary: 1800 kcal consumed, 200 remaining."
	cleaned, suggestions := ExtractSuggestions(text)

	if suggestions != nil {
		t.Errorf("suggestions = %v, want nil", suggestions)
	}
	if cleaned != text {
		t.Errorf("cleaned = %q, want original text unchanged", cleaned)
	}
}

func TestExtractSuggestions_MalformedJSON(t *testing.T) {
	text := "Pick one:\n\n```suggestions\n{not an array}\n```"
	cleaned, suggestions := ExtractSuggestions(text)

	if suggestions != nil {
		t.Errorf("suggestions = %v, want nil for malformed JSON", suggestions)
	}
	if cleaned != text {
		t.Errorf("cleaned = %q, want original text unchanged on malformed JSON", cleaned)
	}
}

func TestExtractSuggestions_EmptyArray(t *testing.T) {
	text := "Options:\n\n```suggestions\n[]\n```"
	cleaned, suggestions := ExtractSuggestions(text)

	if suggestions != nil {
		t.Errorf("suggestions = %v, want nil for empty array", suggestions)
	}
	if cleaned != text {
		t.Errorf("cleaned = %q, want original text unchanged for empty array", cleaned)
	}
}

func TestExtractSuggestions_TrailingWhitespace(t *testing.T) {
	text := "Choose:\n\n```suggestions\n[\"Option A\", \"Option B\"]\n```\n\n"
	cleaned, suggestions := ExtractSuggestions(text)

	if len(suggestions) != 2 {
		t.Fatalf("got %d suggestions, want 2", len(suggestions))
	}
	if suggestions[0] != "Option A" || suggestions[1] != "Option B" {
		t.Errorf("suggestions = %v, want [Option A, Option B]", suggestions)
	}
	if strings.Contains(cleaned, "```") {
		t.Errorf("cleaned text still contains fence: %q", cleaned)
	}
}

func TestExtractSuggestions_SingleOption(t *testing.T) {
	text := "How about:\n\n```suggestions\n[\"Yes, please\"]\n```"
	cleaned, suggestions := ExtractSuggestions(text)

	if len(suggestions) != 1 {
		t.Fatalf("got %d suggestions, want 1", len(suggestions))
	}
	if suggestions[0] != "Yes, please" {
		t.Errorf("suggestions[0] = %q, want 'Yes, please'", suggestions[0])
	}
	if strings.Contains(cleaned, "```") {
		t.Errorf("cleaned text still contains fence: %q", cleaned)
	}
}

func TestExtractSuggestions_BlockNotAtEnd(t *testing.T) {
	// Block in the middle of text — should NOT be extracted (trailing-only).
	text := "```suggestions\n[\"Mid-text option\"]\n```\n\nMore text after the block."
	cleaned, suggestions := ExtractSuggestions(text)

	if suggestions != nil {
		t.Errorf("suggestions = %v, want nil for mid-text block", suggestions)
	}
	if cleaned != text {
		t.Errorf("cleaned = %q, want original text unchanged", cleaned)
	}
}

func TestExtractSuggestions_NotJSONArray(t *testing.T) {
	// Valid JSON but not a string array (object instead).
	text := "```suggestions\n{\"key\": \"value\"}\n```"
	cleaned, suggestions := ExtractSuggestions(text)

	if suggestions != nil {
		t.Errorf("suggestions = %v, want nil for non-array JSON", suggestions)
	}
	if cleaned != text {
		t.Errorf("cleaned = %q, want original text unchanged", cleaned)
	}
}

func TestExtractSuggestions_IntArray(t *testing.T) {
	text := "```suggestions\n[1, 2, 3]\n```"
	cleaned, suggestions := ExtractSuggestions(text)

	if suggestions != nil {
		t.Errorf("suggestions = %v, want nil for int array", suggestions)
	}
	if cleaned != text {
		t.Errorf("cleaned = %q, want original text unchanged", cleaned)
	}
}

// ---------------------------------------------------------------------------
// Integration: suggestions event emitted before done in router loop
// ---------------------------------------------------------------------------

func TestRouterSuggestionsEvent(t *testing.T) {
	adapter := &fakeChatAdapter{
		rounds: [][]ports.ChatEvent{
			{
				{Kind: "text-delta", Text: "I can log that for you.\n\n"},
				{Kind: "text-delta", Text: "```suggestions\n[\"Yes, log it\", \"No, cancel\"]\n```"},
				{Kind: "done"},
			},
		},
	}
	r := New(adapter, nil, nil)
	ch := r.Run(context.Background(), "u1", "system", nil, "log 200g chicken")

	events := collectEvents(ch)
	if len(events) != 4 {
		t.Fatalf("got %d events, want 4 (2 text-delta + suggestions + done)", len(events))
	}

	// Event 0: first text-delta
	if events[0].Kind != "text-delta" {
		t.Errorf("events[0].Kind = %q, want text-delta", events[0].Kind)
	}

	// Event 1: second text-delta (fenced block streams as raw text)
	if events[1].Kind != "text-delta" {
		t.Errorf("events[1].Kind = %q, want text-delta", events[1].Kind)
	}

	// Event 2: suggestions (emitted before done)
	if events[2].Kind != "suggestions" {
		t.Fatalf("events[2].Kind = %q, want suggestions", events[2].Kind)
	}
	if len(events[2].Suggestions) != 2 {
		t.Fatalf("got %d suggestions, want 2", len(events[2].Suggestions))
	}
	if events[2].Suggestions[0] != "Yes, log it" {
		t.Errorf("suggestions[0] = %q, want 'Yes, log it'", events[2].Suggestions[0])
	}
	if events[2].Suggestions[1] != "No, cancel" {
		t.Errorf("suggestions[1] = %q, want 'No, cancel'", events[2].Suggestions[1])
	}

	// Event 3: done (after suggestions)
	if events[3].Kind != "done" {
		t.Fatalf("events[3].Kind = %q, want done (must be AFTER suggestions)", events[3].Kind)
	}
}

func TestRouterSuggestionsEvent_NoSuggestionsBlock(t *testing.T) {
	adapter := &fakeChatAdapter{
		rounds: [][]ports.ChatEvent{
			{
				{Kind: "text-delta", Text: "Your status looks good today!"},
				{Kind: "done"},
			},
		},
	}
	r := New(adapter, nil, nil)
	ch := r.Run(context.Background(), "u1", "system", nil, "how am I doing?")

	events := collectEvents(ch)
	if len(events) != 2 {
		t.Fatalf("got %d events, want 2 (text-delta + done)", len(events))
	}

	// No suggestions event should appear.
	for _, e := range events {
		if e.Kind == "suggestions" {
			t.Errorf("unexpected suggestions event: %+v", e)
		}
	}

	if events[0].Kind != "text-delta" {
		t.Errorf("events[0].Kind = %q, want text-delta", events[0].Kind)
	}
	if events[1].Kind != "done" {
		t.Errorf("events[1].Kind = %q, want done", events[1].Kind)
	}
}
