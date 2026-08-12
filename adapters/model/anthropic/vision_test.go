package anthropic

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gsaraiva2109/dietdaemon/core/types"
)

func checkExtractLabelRequest(t *testing.T, req visionRequest, wantB64 string) {
	t.Helper()
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
}

func checkExtractLabelDraft(t *testing.T, draft types.NutritionLabelDraft) {
	t.Helper()
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
		checkExtractLabelRequest(t, req, wantB64)

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
	checkExtractLabelDraft(t, draft)
}

func checkExtractPlanRequest(t *testing.T, req visionRequest, pages []types.PlanImagePage) {
	t.Helper()
	if len(req.Messages) != 1 || len(req.Messages[0].Content) != len(pages)+1 {
		t.Fatalf("unexpected message shape: %+v", req.Messages)
	}
	if req.MaxTokens != 4096 {
		t.Errorf("MaxTokens = %d, want 4096", req.MaxTokens)
	}
	for i, page := range pages {
		imgBlock := req.Messages[0].Content[i]
		if imgBlock.Type != "image" || imgBlock.Source == nil {
			t.Fatalf("content[%d] = %+v, want image block", i, imgBlock)
		}
		if want := base64.StdEncoding.EncodeToString(page.Data); imgBlock.Source.Data != want {
			t.Errorf("content[%d] image data mismatch", i)
		}
	}
	textBlock := req.Messages[0].Content[len(pages)]
	if textBlock.Type != "text" || !strings.Contains(textBlock.Text, "carb-cycling") {
		t.Errorf("last content block = %+v, want the planextract photo prompt", textBlock)
	}
}

func TestExtractPlan(t *testing.T) {
	pages := []types.PlanImagePage{
		{Data: []byte("fake-jpeg-page-1"), MimeType: "image/jpeg"},
		{Data: []byte("fake-jpeg-page-2"), MimeType: "image/jpeg"},
		{Data: []byte("fake-jpeg-page-3"), MimeType: "image/jpeg"},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/messages" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		var req visionRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		checkExtractPlanRequest(t, req, pages)

		_ = json.NewEncoder(w).Encode(messagesResponse{
			Content: []contentBlock{{Type: "text", Text: `{"plan_name":"Plano","day_types":[{"name":"Dia único","targets":{"Calories":2000,"Protein":150,"Carbs":200,"Fat":60,"Fiber":25},"water_goal_ml":2500,"slots":[],"low_confidence_fields":[]}],"unreadable":false,"notes":null}`}},
		})
	}))
	defer srv.Close()

	a := &Adapter{apiKey: "test-key", model: "claude-haiku-4-5-20251001", client: &http.Client{Timeout: 5 * time.Second}, baseURL: srv.URL}
	draft, err := a.ExtractPlan(t.Context(), pages)
	if err != nil {
		t.Fatalf("ExtractPlan: %v", err)
	}
	if draft.PlanName == nil || *draft.PlanName != "Plano" {
		t.Errorf("PlanName = %v, want Plano", draft.PlanName)
	}
	if len(draft.DayTypes) != 1 || draft.DayTypes[0].Name != "Dia único" {
		t.Errorf("DayTypes = %+v, want one day type named Dia único", draft.DayTypes)
	}
	if draft.Unreadable {
		t.Error("Unreadable = true, want false")
	}
}

func TestExtractPlanHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":{"message":"invalid image"}}`))
	}))
	defer srv.Close()

	a := &Adapter{apiKey: "test-key", model: "claude-haiku-4-5-20251001", client: &http.Client{Timeout: 5 * time.Second}, baseURL: srv.URL}
	_, err := a.ExtractPlan(t.Context(), []types.PlanImagePage{{Data: []byte("img"), MimeType: "image/jpeg"}})
	if err == nil {
		t.Fatal("expected error on 400, got nil")
	}
	if !strings.Contains(err.Error(), "invalid image") {
		t.Errorf("error = %q, want it to include the response body detail", err.Error())
	}
}

func checkExtractMenuRequest(t *testing.T, req visionRequest, wantB64 string) {
	t.Helper()
	if len(req.Messages) != 1 || len(req.Messages[0].Content) != 2 {
		t.Fatalf("unexpected message shape: %+v", req.Messages)
	}
	imgBlock := req.Messages[0].Content[0]
	if imgBlock.Type != "image" || imgBlock.Source == nil {
		t.Fatalf("content[0] = %+v, want image block", imgBlock)
	}
	if imgBlock.Source.Data != wantB64 {
		t.Errorf("image data mismatch")
	}
	textBlock := req.Messages[0].Content[1]
	if textBlock.Type != "text" || !strings.Contains(textBlock.Text, "restaurant menu") {
		t.Errorf("content[1] = %+v, want the menuextract prompt", textBlock)
	}
}

func TestExtractMenu(t *testing.T) {
	img := []byte("fake-jpeg-bytes")
	wantB64 := base64.StdEncoding.EncodeToString(img)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/messages" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		var req visionRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		checkExtractMenuRequest(t, req, wantB64)

		_ = json.NewEncoder(w).Encode(messagesResponse{
			Content: []contentBlock{{Type: "text", Text: `{"dishes":[{"name":"Frango à parmegiana","description":"Peito empanado, molho de tomate"}],"unreadable":false}`}},
		})
	}))
	defer srv.Close()

	a := &Adapter{apiKey: "test-key", model: "claude-haiku-4-5-20251001", client: &http.Client{Timeout: 5 * time.Second}, baseURL: srv.URL}
	draft, err := a.ExtractMenu(t.Context(), img, "image/jpeg")
	if err != nil {
		t.Fatalf("ExtractMenu: %v", err)
	}
	if len(draft.Dishes) != 1 || draft.Dishes[0].Name != "Frango à parmegiana" {
		t.Errorf("Dishes = %+v, want one dish named Frango à parmegiana", draft.Dishes)
	}
	if draft.Unreadable {
		t.Error("Unreadable = true, want false")
	}
}

func TestExtractMenuHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":{"message":"invalid image"}}`))
	}))
	defer srv.Close()

	a := &Adapter{apiKey: "test-key", model: "claude-haiku-4-5-20251001", client: &http.Client{Timeout: 5 * time.Second}, baseURL: srv.URL}
	_, err := a.ExtractMenu(t.Context(), []byte("img"), "image/jpeg")
	if err == nil {
		t.Fatal("expected error on 400, got nil")
	}
	if !strings.Contains(err.Error(), "invalid image") {
		t.Errorf("error = %q, want it to include the response body detail", err.Error())
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
