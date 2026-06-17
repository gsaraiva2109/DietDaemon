package ollama

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestEmbed(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/embeddings" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(embedResponse{Embedding: []float64{0.1, 0.2, 0.3}})
	}))
	defer srv.Close()

	a := New(srv.URL, "nomic-embed-text")
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
		json.NewEncoder(w).Encode(generateResponse{Response: "42 grams"})
	}))
	defer srv.Close()

	a := New(srv.URL, "llama3.2")
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

	a := New(srv.URL, "nomic-embed-text")
	if _, err := a.Embed(t.Context(), "text"); err == nil {
		t.Error("expected error on 500, got nil")
	}
}

func TestCompleteHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	a := New(srv.URL, "llama3.2")
	if _, err := a.Complete(t.Context(), "prompt"); err == nil {
		t.Error("expected error on 503, got nil")
	}
}
