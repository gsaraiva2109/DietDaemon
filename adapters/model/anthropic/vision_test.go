package anthropic

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
		if r.URL.Path != "/v1/messages" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if got := r.Header.Get("x-api-key"); got != "test-key" {
			t.Errorf("x-api-key = %q, want test-key", got)
		}

		var req visionRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if len(req.Messages) != 1 || len(req.Messages[0].Content) != 2 {
			t.Fatalf("unexpected message shape: %+v", req.Messages)
		}
		imgBlock := req.Messages[0].Content[0]
		if imgBlock.Type != "image" || imgBlock.Source == nil {
			t.Fatalf("content[0] = %+v, want image block", imgBlock)
		}
		if imgBlock.Source.MediaType != "image/jpeg" {
			t.Errorf("media_type = %q, want image/jpeg", imgBlock.Source.MediaType)
		}
		if imgBlock.Source.Data != wantB64 {
			t.Errorf("image data mismatch")
		}
		textBlock := req.Messages[0].Content[1]
		if textBlock.Type != "text" || !strings.Contains(textBlock.Text, "nutrition facts label") {
			t.Errorf("content[1] = %+v, want the labelextract prompt", textBlock)
		}

		_ = json.NewEncoder(w).Encode(messagesResponse{
			Content: []contentBlock{{Type: "text", Text: `{"name":"Oats","basis_grams":100,"calories":389,"protein_g":16.9,"carbs_g":66.3,"fat_g":6.9,"fiber_g":10.6,"low_confidence_fields":["fiber_g"],"unreadable":false}`}},
		})
	}))
	defer srv.Close()

	a := &Adapter{apiKey: "test-key", model: "claude-haiku-4-5-20251001", client: &http.Client{Timeout: 5 * time.Second}, baseURL: srv.URL}
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
	if len(draft.LowConfidenceFields) != 1 || draft.LowConfidenceFields[0] != "fiber_g" {
		t.Errorf("LowConfidenceFields = %v, want [fiber_g]", draft.LowConfidenceFields)
	}
	if draft.Unreadable {
		t.Error("Unreadable = true, want false")
	}
}

func TestExtractLabelHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":{"message":"invalid image"}}`))
	}))
	defer srv.Close()

	a := &Adapter{apiKey: "test-key", model: "claude-haiku-4-5-20251001", client: &http.Client{Timeout: 5 * time.Second}, baseURL: srv.URL}
	_, err := a.ExtractLabel(t.Context(), []byte("img"), "image/jpeg")
	if err == nil {
		t.Fatal("expected error on 400, got nil")
	}
	if !strings.Contains(err.Error(), "invalid image") {
		t.Errorf("error = %q, want it to include the response body detail", err.Error())
	}
}
