package api

import (
	"bytes"
	"context"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gsaraiva2109/dietdaemon/core/types"
)

// fakeCompletionAdapter returns a pre-programmed completion response/error,
// mirroring fakeVisionAdapter in handler_food_ocr_test.go.
type fakeCompletionAdapter struct {
	response string
	err      error

	calledPrompt string
}

func (f *fakeCompletionAdapter) Complete(_ context.Context, prompt string) (string, error) {
	f.calledPrompt = prompt
	if f.err != nil {
		return "", f.err
	}
	return f.response, nil
}

func (f *fakeCompletionAdapter) Embed(_ context.Context, _ string) ([]float32, error) {
	return nil, nil
}

const validPlanDraftResponse = `{"plan_name":"Plano","day_types":[{"name":"Dia único","targets":{"Calories":2000,"Protein":150,"Carbs":200,"Fat":60,"Fiber":25},"water_goal_ml":2500,"slots":[],"low_confidence_fields":[]}],"unreadable":false,"notes":null}`

func doExtractPlanFromText(h *Handler, text string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/plans/extract/text", strings.NewReader(text))
	req.Header.Set("Content-Type", "text/plain")
	req.Header.Set("Authorization", "Bearer test-api-key")

	rec := httptest.NewRecorder()
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	mux.ServeHTTP(rec, req)
	return rec
}

func TestHandleExtractPlanFromText(t *testing.T) {
	adapter := &fakeCompletionAdapter{response: validPlanDraftResponse}
	h := newHandler(&fakeMealStore{}, &fakeMealLogger{})
	h.completionAdapter = adapter

	rec := doExtractPlanFromText(h, "Dia único: 2000 kcal, 150g proteína...")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
	got := decodeJSON[types.PlanDraft](t, rec)
	if got.PlanName == nil || *got.PlanName != "Plano" {
		t.Errorf("PlanName = %v, want Plano", got.PlanName)
	}
	if len(got.DayTypes) != 1 || got.DayTypes[0].Name != "Dia único" {
		t.Errorf("DayTypes = %+v, want one day type named Dia único", got.DayTypes)
	}
	if !strings.Contains(adapter.calledPrompt, "Dia único: 2000 kcal") {
		t.Errorf("Complete prompt does not contain the pasted text: %q", adapter.calledPrompt)
	}
}

func TestHandleExtractPlanFromTextEmptyBody(t *testing.T) {
	h := newHandler(&fakeMealStore{}, &fakeMealLogger{})
	h.completionAdapter = &fakeCompletionAdapter{}

	rec := doExtractPlanFromText(h, "")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", rec.Code, rec.Body.String())
	}
}

func TestHandleExtractPlanFromTextOversized(t *testing.T) {
	h := newHandler(&fakeMealStore{}, &fakeMealLogger{})
	h.completionAdapter = &fakeCompletionAdapter{response: validPlanDraftResponse}

	rec := doExtractPlanFromText(h, strings.Repeat("a", maxPlanTextChars+1))
	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want 413; body=%s", rec.Code, rec.Body.String())
	}
}

func TestHandleExtractPlanFromTextAdapterError(t *testing.T) {
	h := newHandler(&fakeMealStore{}, &fakeMealLogger{})
	h.completionAdapter = &fakeCompletionAdapter{err: context.DeadlineExceeded}

	rec := doExtractPlanFromText(h, "some plan text")
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500; body=%s", rec.Code, rec.Body.String())
	}
}

func TestHandleExtractPlanFromTextUnparseableResponse(t *testing.T) {
	h := newHandler(&fakeMealStore{}, &fakeMealLogger{})
	h.completionAdapter = &fakeCompletionAdapter{response: "this is not json"}

	rec := doExtractPlanFromText(h, "some plan text")
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500; body=%s", rec.Code, rec.Body.String())
	}
}

// ---------------------------------------------------------------------------
// handleExtractPlanFromImage (issue #194)
// ---------------------------------------------------------------------------

func doExtractPlanFromImage(h *Handler, fileContent []byte, fileName string) *httptest.ResponseRecorder {
	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	if fileName != "" {
		part, _ := w.CreateFormFile("file", fileName)
		_, _ = part.Write(fileContent)
	}
	_ = w.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/plans/extract/image", &body)
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set("Authorization", "Bearer test-api-key")

	rec := httptest.NewRecorder()
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	mux.ServeHTTP(rec, req)
	return rec
}

func TestHandleExtractPlanFromImageDisabled(t *testing.T) {
	h := newHandler(&fakeMealStore{}, &fakeMealLogger{})
	// h.visionAdapter left nil: plan photo extraction not configured.

	rec := doExtractPlanFromImage(h, testPNGBytes, "plan.png")
	if rec.Code != http.StatusNotImplemented {
		t.Fatalf("status = %d, want 501; body=%s", rec.Code, rec.Body.String())
	}
}

func TestHandleExtractPlanFromImageOversized(t *testing.T) {
	h := newHandler(&fakeMealStore{}, &fakeMealLogger{})
	h.visionAdapter = &fakeVisionAdapter{}

	rec := doExtractPlanFromImage(h, bytes.Repeat([]byte("a"), 6<<20), "plan.png")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", rec.Code, rec.Body.String())
	}
}

func TestHandleExtractPlanFromImageAdapterError(t *testing.T) {
	h := newHandler(&fakeMealStore{}, &fakeMealLogger{})
	h.visionAdapter = &fakeVisionAdapter{err: context.DeadlineExceeded}

	rec := doExtractPlanFromImage(h, testPNGBytes, "plan.png")
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500; body=%s", rec.Code, rec.Body.String())
	}
}

func TestHandleExtractPlanFromImage(t *testing.T) {
	planName := "Plano"
	adapter := &fakeVisionAdapter{planDraft: types.PlanDraft{
		PlanName: &planName,
		DayTypes: []types.PlanDraftDayType{{Name: "Dia único"}},
	}}
	h := newHandler(&fakeMealStore{}, &fakeMealLogger{})
	h.visionAdapter = adapter

	rec := doExtractPlanFromImage(h, testPNGBytes, "plan.png")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
	got := decodeJSON[types.PlanDraft](t, rec)
	if got.PlanName == nil || *got.PlanName != "Plano" {
		t.Errorf("PlanName = %v, want Plano", got.PlanName)
	}
	if len(got.DayTypes) != 1 || got.DayTypes[0].Name != "Dia único" {
		t.Errorf("DayTypes = %+v, want one day type named Dia único", got.DayTypes)
	}
	if adapter.calledMime != "image/png" {
		t.Errorf("ExtractPlan mimeType = %q, want image/png", adapter.calledMime)
	}
	if adapter.calledLen != len(testPNGBytes) {
		t.Errorf("ExtractPlan image len = %d, want %d", adapter.calledLen, len(testPNGBytes))
	}
}
