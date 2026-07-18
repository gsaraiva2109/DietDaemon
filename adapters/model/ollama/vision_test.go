package ollama

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestExtractLabel(t *testing.T) {
	img := []byte("fake-jpeg-bytes")
	wantB64 := base64.StdEncoding.EncodeToString(img)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/generate" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		var req visionRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if req.Model != "llava" {
			t.Errorf("model = %q, want llava", req.Model)
		}
		if !strings.Contains(req.Prompt, "nutrition facts label") {
			t.Errorf("prompt missing labelextract prompt: %q", req.Prompt)
		}
		if len(req.Images) != 1 || req.Images[0] != wantB64 {
			t.Errorf("images = %v, want [%s]", req.Images, wantB64)
		}
		if req.Images[0] != wantB64 || strings.HasPrefix(req.Images[0], "data:") {
			t.Errorf("images[0] must be bare base64, no data: prefix")
		}
		if req.Stream {
			t.Error("stream must be false")
		}
		if req.Format != "json" {
			t.Errorf("format = %q, want json", req.Format)
		}

		_ = json.NewEncoder(w).Encode(generateResponse{
			Response: `{"name":"Oats","basis_grams":100,"calories":389,"protein_g":16.9,"carbs_g":66.3,"fat_g":6.9,"fiber_g":10.6,"low_confidence_fields":[],"unreadable":false}`,
		})
	}))
	defer srv.Close()

	a := New(srv.URL, "nomic-embed-text", "llama3.1", 30*time.Second)
	a.SetVisionModel("llava")
	draft, err := a.ExtractLabel(t.Context(), img, "image/jpeg")
	if err != nil {
		t.Fatalf("ExtractLabel: %v", err)
	}
	if draft.Name == nil || *draft.Name != "Oats" {
		t.Errorf("Name = %v, want Oats", draft.Name)
	}
	if draft.BasisGrams == nil || *draft.BasisGrams != 100 {
		t.Errorf("BasisGrams = %v, want 100", draft.BasisGrams)
	}
}

func TestExtractLabelUnreadable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(generateResponse{
			Response: `{"name":null,"basis_grams":null,"calories":null,"protein_g":null,"carbs_g":null,"fat_g":null,"fiber_g":null,"low_confidence_fields":[],"unreadable":true}`,
		})
	}))
	defer srv.Close()

	a := New(srv.URL, "", "", 30*time.Second)
	a.SetVisionModel("llava")
	draft, err := a.ExtractLabel(t.Context(), []byte("img"), "image/jpeg")
	if err != nil {
		t.Fatalf("ExtractLabel: %v", err)
	}
	if !draft.Unreadable {
		t.Error("Unreadable = false, want true")
	}
	if draft.Name != nil {
		t.Errorf("Name = %v, want nil", draft.Name)
	}
}

func TestExtractLabelHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	a := New(srv.URL, "", "", 30*time.Second)
	a.SetVisionModel("llava")
	if _, err := a.ExtractLabel(t.Context(), []byte("img"), "image/jpeg"); err == nil {
		t.Error("expected error on 503, got nil")
	}
}
