package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gsaraiva2109/dietdaemon/internal/config"
)

// fakeFoodImportRunner returns pre-programmed results/errors, mirroring
// fakeVisionAdapter in handler_food_ocr_test.go.
type fakeFoodImportRunner struct {
	importRows int
	importErr  error

	repairChecked, repairFixed int
	repairErr                  error

	backfillEmbedded, backfillFailed int
	backfillErr                      error

	calledSource string
	calledMax    int
}

func (f *fakeFoodImportRunner) ImportSource(_ context.Context, source string, maxRows int) (int, error) {
	f.calledSource = source
	f.calledMax = maxRows
	return f.importRows, f.importErr
}

func (f *fakeFoodImportRunner) RepairSource(_ context.Context, source string) (int, int, error) {
	f.calledSource = source
	return f.repairChecked, f.repairFixed, f.repairErr
}

func (f *fakeFoodImportRunner) BackfillEmbeddings(_ context.Context) (int, int, error) {
	return f.backfillEmbedded, f.backfillFailed, f.backfillErr
}

func doAdminRequest(h *Handler, path, token string, body any) *httptest.ResponseRecorder {
	var buf bytes.Buffer
	if body != nil {
		_ = json.NewEncoder(&buf).Encode(body)
	}
	req := httptest.NewRequest(http.MethodPost, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	rec := httptest.NewRecorder()
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	mux.ServeHTTP(rec, req)
	return rec
}

const adminTestToken = "test-admin-token"

func newAdminTestHandler(runner FoodImportRunner) *Handler {
	h := newHandler(&fakeMealStore{}, &fakeMealLogger{})
	h.cfg = &config.Config{APIAuthToken: adminTestToken}
	h.foodImportRunner = runner
	return h
}

func TestAdminFoodImport_RunnerNil503(t *testing.T) {
	h := newHandler(&fakeMealStore{}, &fakeMealLogger{})
	h.cfg = &config.Config{APIAuthToken: adminTestToken}
	// h.foodImportRunner left nil.

	for _, path := range []string{
		"/api/v1/admin/food-import/run",
		"/api/v1/admin/food-import/repair",
		"/api/v1/admin/food-import/backfill-embeddings",
	} {
		rec := doAdminRequest(h, path, adminTestToken, map[string]string{"source": "taco"})
		if rec.Code != http.StatusServiceUnavailable {
			t.Errorf("%s: status = %d, want 503; body=%s", path, rec.Code, rec.Body.String())
		}
	}
}

func TestAdminFoodImport_TokenUnset503(t *testing.T) {
	h := newHandler(&fakeMealStore{}, &fakeMealLogger{})
	h.cfg = &config.Config{} // APIAuthToken == ""
	h.foodImportRunner = &fakeFoodImportRunner{}

	rec := doAdminRequest(h, "/api/v1/admin/food-import/run", "anything", map[string]string{"source": "taco"})
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503; body=%s", rec.Code, rec.Body.String())
	}
}

func TestAdminFoodImport_WrongToken401(t *testing.T) {
	h := newAdminTestHandler(&fakeFoodImportRunner{})

	rec := doAdminRequest(h, "/api/v1/admin/food-import/run", "wrong-token", map[string]string{"source": "taco"})
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401; body=%s", rec.Code, rec.Body.String())
	}
}

func TestAdminFoodImport_MissingToken401(t *testing.T) {
	h := newAdminTestHandler(&fakeFoodImportRunner{})

	rec := doAdminRequest(h, "/api/v1/admin/food-import/run", "", map[string]string{"source": "taco"})
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401; body=%s", rec.Code, rec.Body.String())
	}
}

func TestAdminFoodImport_Run200(t *testing.T) {
	runner := &fakeFoodImportRunner{importRows: 42}
	h := newAdminTestHandler(runner)

	rec := doAdminRequest(h, "/api/v1/admin/food-import/run", adminTestToken, map[string]any{"source": "taco", "max_rows": 100})
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
	got := decodeJSON[map[string]int](t, rec)
	if got["rows"] != 42 {
		t.Errorf("rows = %d, want 42", got["rows"])
	}
	if runner.calledSource != "taco" || runner.calledMax != 100 {
		t.Errorf("runner called with source=%q maxRows=%d, want taco/100", runner.calledSource, runner.calledMax)
	}
}

func TestAdminFoodImport_RunMissingSource400(t *testing.T) {
	h := newAdminTestHandler(&fakeFoodImportRunner{})

	rec := doAdminRequest(h, "/api/v1/admin/food-import/run", adminTestToken, map[string]string{})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", rec.Code, rec.Body.String())
	}
}

func TestAdminFoodImport_Repair200(t *testing.T) {
	runner := &fakeFoodImportRunner{repairChecked: 10, repairFixed: 3}
	h := newAdminTestHandler(runner)

	rec := doAdminRequest(h, "/api/v1/admin/food-import/repair", adminTestToken, map[string]string{"source": "taco"})
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
	got := decodeJSON[map[string]int](t, rec)
	if got["checked"] != 10 || got["fixed"] != 3 {
		t.Errorf("body = %+v, want checked=10 fixed=3", got)
	}
	if runner.calledSource != "taco" {
		t.Errorf("runner called with source=%q, want taco", runner.calledSource)
	}
}

func TestAdminFoodImport_RepairMissingSource400(t *testing.T) {
	h := newAdminTestHandler(&fakeFoodImportRunner{})

	rec := doAdminRequest(h, "/api/v1/admin/food-import/repair", adminTestToken, map[string]string{})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", rec.Code, rec.Body.String())
	}
}

func TestAdminFoodImport_BackfillEmbeddings200(t *testing.T) {
	runner := &fakeFoodImportRunner{backfillEmbedded: 7, backfillFailed: 1}
	h := newAdminTestHandler(runner)

	rec := doAdminRequest(h, "/api/v1/admin/food-import/backfill-embeddings", adminTestToken, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
	got := decodeJSON[map[string]int](t, rec)
	if got["embedded"] != 7 || got["failed"] != 1 {
		t.Errorf("body = %+v, want embedded=7 failed=1", got)
	}
}
