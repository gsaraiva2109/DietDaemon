package openai

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

func TestCompleteHappyPath(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer sk-test" {
			t.Errorf("Authorization = %q, want %q", got, "Bearer sk-test")
		}

		var req chatRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if req.Model != "gpt-4o-mini" {
			t.Errorf("model = %q, want gpt-4o-mini", req.Model)
		}
		if len(req.Messages) != 1 || req.Messages[0].Role != "user" || req.Messages[0].Content != "How many grams in an egg?" {
			t.Errorf("messages = %+v, unexpected", req.Messages)
		}
		if req.ResponseFormat == nil || req.ResponseFormat.Type != "json_object" {
			t.Errorf("response_format = %+v, want json_object", req.ResponseFormat)
		}

		json.NewEncoder(w).Encode(chatResponse{Choices: []struct {
			Message chatMessage `json:"message"`
		}{{Message: chatMessage{Role: "assistant", Content: `{"food":"egg"}`}}}})
	}))
	defer srv.Close()

	a := New(srv.URL, "sk-test", "gpt-4o-mini", 30*time.Second)
	got, err := a.Complete(t.Context(), "How many grams in an egg?")
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if got != `{"food":"egg"}` {
		t.Errorf("Complete = %q, want %q", got, `{"food":"egg"}`)
	}
}

func TestCompleteEmptyAPIKeyOmitsHeader(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := r.Header["Authorization"]; ok {
			t.Errorf("Authorization header should not be set, got %q", r.Header.Get("Authorization"))
		}
		json.NewEncoder(w).Encode(chatResponse{Choices: []struct {
			Message chatMessage `json:"message"`
		}{{Message: chatMessage{Role: "assistant", Content: `{"food":"egg"}`}}}})
	}))
	defer srv.Close()

	a := New(srv.URL, "", "gpt-4o-mini", 30*time.Second)
	if _, err := a.Complete(t.Context(), "prompt"); err != nil {
		t.Fatalf("Complete: %v", err)
	}
}

func TestCompleteStripsFence(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		json.NewEncoder(w).Encode(chatResponse{Choices: []struct {
			Message chatMessage `json:"message"`
		}{{Message: chatMessage{Role: "assistant", Content: "```json\n{\"food\":\"egg\"}\n```"}}}})
	}))
	defer srv.Close()

	a := New(srv.URL, "sk-test", "gpt-4o-mini", 30*time.Second)
	got, err := a.Complete(t.Context(), "prompt")
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if got != `{"food":"egg"}` {
		t.Errorf("Complete = %q, want fence-stripped %q", got, `{"food":"egg"}`)
	}
}

func TestCompleteHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	a := New(srv.URL, "sk-test", "gpt-4o-mini", 30*time.Second)
	if _, err := a.Complete(t.Context(), "prompt"); err == nil {
		t.Error("expected error on 500, got nil")
	}
}

func TestCompleteEmptyChoices(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		json.NewEncoder(w).Encode(chatResponse{Choices: nil})
	}))
	defer srv.Close()

	a := New(srv.URL, "sk-test", "gpt-4o-mini", 30*time.Second)
	if _, err := a.Complete(t.Context(), "prompt"); err == nil {
		t.Error("expected error on empty choices, got nil")
	}
}

func TestEmbedNotSupported(t *testing.T) {
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
	}))
	defer srv.Close()

	a := New(srv.URL, "sk-test", "gpt-4o-mini", 30*time.Second)
	_, err := a.Embed(t.Context(), "text")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if got := err.Error(); !strings.Contains(got, "EMBED_ADAPTER=ollama") {
		t.Errorf("error = %q, want it to mention EMBED_ADAPTER=ollama", got)
	}
	if called {
		t.Error("Embed made an HTTP call, it should not")
	}
}

func TestContextCancellation(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Slow response — context should cancel before it arrives.
		<-r.Context().Done()
	}))
	defer srv.Close()

	a := New(srv.URL, "sk-test", "gpt-4o-mini", 30*time.Second)
	ctx, cancel := context.WithCancel(t.Context())
	cancel() // cancel immediately

	if _, err := a.Complete(ctx, "prompt"); err == nil {
		t.Error("expected error from cancelled context")
	}
}
