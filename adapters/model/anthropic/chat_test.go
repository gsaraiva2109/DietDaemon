package anthropic

import (
	"encoding/json"
	"testing"

	"github.com/gsaraiva2109/dietdaemon/core/ports"
)

// TestToWireMessagesToolRoundTrip guards the bug where a "tool" role message
// was passed straight through to Anthropic (which rejects any role other than
// user/assistant) and the assistant's tool_use block was dropped from history
// entirely — Anthropic requires a tool_result's tool_use_id to match a
// tool_use block in the immediately preceding assistant turn.
func TestToWireMessagesToolRoundTrip(t *testing.T) {
	msgs := []ports.ChatMessage{
		{Role: "user", Content: "what should I eat"},
		{
			Role:      "assistant",
			Content:   "",
			ToolCalls: []ports.ToolCallEvent{{ID: "tc_1", Name: "suggest", Args: "breakfast"}},
		},
		{Role: "tool", Content: "eat oatmeal", ToolCallID: "tc_1"},
	}

	wire := toWireMessages(msgs)
	if len(wire) != 3 {
		t.Fatalf("got %d messages, want 3", len(wire))
	}

	for _, m := range wire {
		if m.Role != "user" && m.Role != "assistant" {
			t.Fatalf("message role %q is invalid for Anthropic (only user/assistant allowed)", m.Role)
		}
	}

	assistantMsg := wire[1]
	if assistantMsg.Role != "assistant" {
		t.Fatalf("wire[1].Role = %q, want assistant", assistantMsg.Role)
	}
	if len(assistantMsg.Content) != 1 || assistantMsg.Content[0].Type != "tool_use" {
		t.Fatalf("wire[1].Content = %+v, want single tool_use block", assistantMsg.Content)
	}
	if assistantMsg.Content[0].ID != "tc_1" || assistantMsg.Content[0].Name != "suggest" {
		t.Errorf("tool_use block = %+v, want ID tc_1 Name suggest", assistantMsg.Content[0])
	}
	var input struct {
		Args string `json:"args"`
	}
	if err := json.Unmarshal(assistantMsg.Content[0].Input, &input); err != nil || input.Args != "breakfast" {
		t.Errorf("tool_use input = %s, want {\"args\":\"breakfast\"}", assistantMsg.Content[0].Input)
	}

	toolResultMsg := wire[2]
	if toolResultMsg.Role != "user" {
		t.Fatalf("wire[2].Role = %q, want user (tool_result must be a user turn)", toolResultMsg.Role)
	}
	if len(toolResultMsg.Content) != 1 || toolResultMsg.Content[0].Type != "tool_result" {
		t.Fatalf("wire[2].Content = %+v, want single tool_result block", toolResultMsg.Content)
	}
	if toolResultMsg.Content[0].ToolUseID != "tc_1" {
		t.Errorf("tool_result.tool_use_id = %q, want tc_1 (must match the preceding tool_use block)", toolResultMsg.Content[0].ToolUseID)
	}
	if toolResultMsg.Content[0].Content != "eat oatmeal" {
		t.Errorf("tool_result.content = %q, want %q", toolResultMsg.Content[0].Content, "eat oatmeal")
	}
}
