package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHighRiskHandlersRejectUnavailableOrMalformedRequests(t *testing.T) {
	h := newHandler(newFakeMealStore(), &fakeMealLogger{})
	for name, tc := range map[string]struct {
		want int
		run  func(*httptest.ResponseRecorder)
	}{
		"backup unavailable": {http.StatusServiceUnavailable, func(rec *httptest.ResponseRecorder) {
			h.handleRunBackupNow(rec, httptest.NewRequest(http.MethodPost, "/", nil), "user-1")
		}},
		"export missing dates": {http.StatusBadRequest, func(rec *httptest.ResponseRecorder) {
			h.handleExportMeals(rec, httptest.NewRequest(http.MethodGet, "/", nil), "user-1")
		}},
		"hevy key unavailable": {http.StatusServiceUnavailable, func(rec *httptest.ResponseRecorder) {
			h.handleSetHevyKey(rec, httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"key":"x"}`)), "user-1")
		}},
		"hevy import unavailable": {http.StatusServiceUnavailable, func(rec *httptest.ResponseRecorder) {
			h.handleImportHevy(rec, httptest.NewRequest(http.MethodPost, "/", nil), "user-1")
		}},
		"link missing platform": {http.StatusBadRequest, func(rec *httptest.ResponseRecorder) {
			h.handleCreateLinkCode(rec, httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{}`)), "user-1")
		}},
		"photo missing file": {http.StatusBadRequest, func(rec *httptest.ResponseRecorder) {
			h.handleUploadPhoto(rec, httptest.NewRequest(http.MethodPost, "/", nil), "user-1")
		}},
		"ai key unavailable": {http.StatusServiceUnavailable, func(rec *httptest.ResponseRecorder) {
			h.handleSetAIKey(rec, httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"provider":"openai","key":"x"}`)), "user-1")
		}},
		"sleep malformed": {http.StatusBadRequest, func(rec *httptest.ResponseRecorder) {
			h.handleLogSleep(rec, httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{`)), "user-1")
		}},
		"streak empty history": {http.StatusOK, func(rec *httptest.ResponseRecorder) {
			h.handleStreak(rec, httptest.NewRequest(http.MethodGet, "/", nil), "user-1")
		}},
	} {
		t.Run(name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			tc.run(rec)
			if rec.Code != tc.want {
				t.Fatalf("status = %d, want %d: %s", rec.Code, tc.want, rec.Body.String())
			}
		})
	}
}
