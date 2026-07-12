package ollama

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gsaraiva2109/dietdaemon/core/ports"
)

// Compile-time interface check in test build too.
var _ ports.ModelAdapter = (*Adapter)(nil)

func TestEmbed(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/embeddings" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		// Verify request body shape.
		var req embedRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if req.Model != "nomic-embed-text" {
			t.Errorf("embed model = %q, want nomic-embed-text", req.Model)
		}
		if req.Prompt != "hello" {
			t.Errorf("prompt = %q, want hello", req.Prompt)
		}
		_ = json.NewEncoder(w).Encode(embedResponse{Embedding: []float64{0.1, 0.2, 0.3}})
	}))
	defer srv.Close()

	a := New(srv.URL, "nomic-embed-text", "llama3.1", 30*time.Second)
	vec, err := a.Embed(t.Context(), "hello")
	if err != nil {
		t.Fatalf("Embed: %v", err)
	}
	if len(vec) != 3 || vec[0] != 0.1 || vec[2] != 0.3 {
		t.Errorf("Embed = %v, want [0.1 0.2 0.3]", vec)
	}
}

func TestComplete(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/generate" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		// Verify request body shape including format:json.
		var req generateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if req.Model != "llama3.1" {
			t.Errorf("llm model = %q, want llama3.1", req.Model)
		}
		if req.Stream != false {
			t.Error("stream must be false")
		}
		if req.Format != "json" {
			t.Errorf("format = %q, want json", req.Format)
		}
		_ = json.NewEncoder(w).Encode(generateResponse{Response: "42 grams"})
	}))
	defer srv.Close()

	a := New(srv.URL, "nomic-embed-text", "llama3.1", 30*time.Second)
	got, err := a.Complete(t.Context(), "How many grams in an egg?")
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if got != "42 grams" {
		t.Errorf("Complete = %q, want %q", got, "42 grams")
	}
}

func TestEmbedHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	a := New(srv.URL, "nomic-embed-text", "llama3.1", 30*time.Second)
	if _, err := a.Embed(t.Context(), "text"); err == nil {
		t.Error("expected error on 500, got nil")
	}
}

func TestCompleteHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	a := New(srv.URL, "nomic-embed-text", "llama3.1", 30*time.Second)
	if _, err := a.Complete(t.Context(), "prompt"); err == nil {
		t.Error("expected error on 503, got nil")
	}
}

func TestContextCancellation(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Slow response — context should cancel before it arrives.
		<-r.Context().Done()
	}))
	defer srv.Close()

	a := New(srv.URL, "nomic-embed-text", "llama3.1", 30*time.Second)
	ctx, cancel := context.WithCancel(t.Context())
	cancel() // cancel immediately

	if _, err := a.Embed(ctx, "text"); err == nil {
		t.Error("expected error from cancelled context")
	}
	if _, err := a.Complete(ctx, "prompt"); err == nil {
		t.Error("expected error from cancelled context")
	}
}
