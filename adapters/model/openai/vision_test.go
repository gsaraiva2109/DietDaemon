package openai

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
	wantDataURI := "data:image/jpeg;base64," + base64.StdEncoding.EncodeToString(img)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer sk-test" {
			t.Errorf("Authorization = %q, want Bearer sk-test", got)
		}

		var req visionRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if len(req.Messages) != 1 || len(req.Messages[0].Content) != 2 {
			t.Fatalf("unexpected message shape: %+v", req.Messages)
		}
		textPart := req.Messages[0].Content[0]
		if textPart.Type != "text" || !strings.Contains(textPart.Text, "nutrition facts label") {
			t.Errorf("content[0] = %+v, want the labelextract prompt", textPart)
		}
		imgPart := req.Messages[0].Content[1]
		if imgPart.Type != "image_url" || imgPart.ImageURL == nil || imgPart.ImageURL.URL != wantDataURI {
			t.Errorf("content[1] = %+v, want image_url %q", imgPart, wantDataURI)
		}
		if req.ResponseFormat == nil || req.ResponseFormat.Type != "json_object" {
			t.Errorf("response_format = %+v, want json_object", req.ResponseFormat)
		}

		_ = json.NewEncoder(w).Encode(chatResponse{Choices: []struct {
			Message chatMessage `json:"message"`
		}{{Message: chatMessage{Role: "assistant", Content: `{"name":"Oats","basis_grams":100,"calories":389,"protein_g":null,"carbs_g":null,"fat_g":null,"fiber_g":null,"low_confidence_fields":[],"unreadable":false}`}}}})
	}))
	defer srv.Close()

	a := New(srv.URL, "sk-test", "gpt-4o-mini", 30*time.Second)
	draft, err := a.ExtractLabel(t.Context(), img, "image/jpeg")
	if err != nil {
		t.Fatalf("ExtractLabel: %v", err)
	}
	if draft.Name == nil || *draft.Name != "Oats" {
		t.Errorf("Name = %v, want Oats", draft.Name)
	}
	if draft.Calories == nil || *draft.Calories != 389 {
		t.Errorf("Calories = %v, want 389", draft.Calories)
	}
	if draft.ProteinG != nil {
		t.Errorf("ProteinG = %v, want nil", draft.ProteinG)
	}
}

func TestExtractLabelHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":{"message":"unsupported image type"}}`))
	}))
	defer srv.Close()

	a := New(srv.URL, "sk-test", "gpt-4o-mini", 30*time.Second)
	_, err := a.ExtractLabel(t.Context(), []byte("img"), "image/jpeg")
	if err == nil {
		t.Fatal("expected error on 400, got nil")
	}
	if !strings.Contains(err.Error(), "unsupported image type") {
		t.Errorf("error = %q, want it to include the response body detail", err.Error())
	}
}
