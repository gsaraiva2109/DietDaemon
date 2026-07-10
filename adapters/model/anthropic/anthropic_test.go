package anthropic

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gsaraiva2109/dietdaemon/core/ports"
)

// Compile-time interface check in test build too.
var _ ports.ModelAdapter = (*Adapter)(nil)

func TestComplete(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/messages" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if got := r.Header.Get("x-api-key"); got != "test-key" {
			t.Errorf("x-api-key = %q, want test-key", got)
		}
		if got := r.Header.Get("anthropic-version"); got != "2023-06-01" {
			t.Errorf("anthropic-version = %q, want 2023-06-01", got)
		}

		var req messagesRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if req.Model != "claude-haiku-4-5-20251001" {
			t.Errorf("model = %q, want claude-haiku-4-5-20251001", req.Model)
		}
		if len(req.Messages) != 1 {
			t.Fatalf("messages = %d, want 1", len(req.Messages))
		}
		content := req.Messages[0].Content
		if !strings.Contains(content, "How many grams in an egg?") {
			t.Errorf("prompt missing original text: %q", content)
		}
		if !strings.Contains(content, "Respond with ONLY valid JSON") {
			t.Errorf("prompt missing JSON instruction: %q", content)
		}

		json.NewEncoder(w).Encode(messagesResponse{
			Content: []contentBlock{{Type: "text", Text: "```json\n{\"food\":\"egg\"}\n```"}},
		})
	}))
	defer srv.Close()

	a := &Adapter{apiKey: "test-key", model: "claude-haiku-4-5-20251001", client: &http.Client{Timeout: 5 * time.Second}, baseURL: srv.URL}
	got, err := a.Complete(t.Context(), "How many grams in an egg?")
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if got != `{"food":"egg"}` {
		t.Errorf("Complete = %q, want %q", got, `{"food":"egg"}`)
	}
}

func TestCompleteHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":{"message":"invalid model"}}`))
	}))
	defer srv.Close()

	a := &Adapter{apiKey: "test-key", model: "claude-haiku-4-5-20251001", client: &http.Client{Timeout: 5 * time.Second}, baseURL: srv.URL}
	_, err := a.Complete(t.Context(), "prompt")
	if err == nil {
		t.Fatal("expected error on 400, got nil")
	}
	if !strings.Contains(err.Error(), "invalid model") {
		t.Errorf("error = %q, want it to include the response body detail", err.Error())
	}
}

func TestCompleteEmptyContent(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		json.NewEncoder(w).Encode(messagesResponse{Content: nil})
	}))
	defer srv.Close()

	a := &Adapter{apiKey: "test-key", model: "claude-haiku-4-5-20251001", client: &http.Client{Timeout: 5 * time.Second}, baseURL: srv.URL}
	if _, err := a.Complete(t.Context(), "prompt"); err == nil {
		t.Error("expected error on empty content, got nil")
	}
}

func TestEmbedNotSupported(t *testing.T) {
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
	}))
	defer srv.Close()

	a := &Adapter{apiKey: "test-key", model: "claude-haiku-4-5-20251001", client: &http.Client{Timeout: 5 * time.Second}, baseURL: srv.URL}
	_, err := a.Embed(t.Context(), "text")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "EMBED_ADAPTER=ollama") {
		t.Errorf("error = %q, want mention of EMBED_ADAPTER=ollama", err.Error())
	}
	if called {
		t.Error("Embed made an HTTP call, should not have")
	}
}

func TestContextCancellation(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Slow response — context should cancel before it arrives.
		<-r.Context().Done()
	}))
	defer srv.Close()

	a := &Adapter{apiKey: "test-key", model: "claude-haiku-4-5-20251001", client: &http.Client{Timeout: 5 * time.Second}, baseURL: srv.URL}
	ctx, cancel := context.WithCancel(t.Context())
	cancel() // cancel immediately

	if _, err := a.Complete(ctx, "prompt"); err == nil {
		t.Error("expected error from cancelled context")
	}
}
