package openai

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gsaraiva2109/dietdaemon/core/ports"
)

// TestExtractArgsEmptyValue guards the bug where a legitimately empty args
// value (no-arg commands like /help emit {"args":""}) was misread as a parse
// failure and the raw JSON blob leaked through as the command's argument.
func TestExtractArgsEmptyValue(t *testing.T) {
	if got := extractArgs(`{"args": ""}`); got != "" {
		t.Errorf("extractArgs(empty args) = %q, want empty string", got)
	}
	if got := extractArgs(`{"args": "grilled chicken"}`); got != "grilled chicken" {
		t.Errorf("extractArgs = %q, want %q", got, "grilled chicken")
	}
	if got := extractArgs(`not json`); got != "not json" {
		t.Errorf("extractArgs(invalid json) = %q, want raw fallback", got)
	}
}

func TestStreamChatHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":{"message":"model does not support tools"}}`))
	}))
	defer srv.Close()

	c := NewChatAdapter(srv.URL, "sk-test", "deepseek-chat", 5*time.Second)
	_, err := c.StreamChat(t.Context(), ports.ChatRequest{Messages: []ports.ChatMessage{{Role: "user", Content: "hi"}}})
	if err == nil {
		t.Fatal("expected error on 400, got nil")
	}
	if !strings.Contains(err.Error(), "model does not support tools") {
		t.Errorf("error = %q, want it to include the response body detail", err.Error())
	}
}
