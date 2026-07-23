package api

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gsaraiva2109/dietdaemon/core/types"
)

// --- handleCreateLinkCode ---

func TestHandleCreateLinkCode(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "POST", "/api/v1/bot/link-code", map[string]string{"platform": "telegram"}, nil)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	got := decodeJSON[map[string]string](t, rec)
	if len(got["code"]) != 6 {
		t.Errorf("code = %q, want length 6", got["code"])
	}
}

func TestHandleCreateLinkCodeMissingPlatform(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "POST", "/api/v1/bot/link-code", map[string]string{"platform": ""}, nil)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("missing platform expected 400, got %d", rec.Code)
	}
}

func TestHandleCreateLinkCodeInvalidJSON(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{})

	req := httptest.NewRequest("POST", "/api/v1/bot/link-code", strings.NewReader("not json"))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer test-api-key")
	rec := httptest.NewRecorder()
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("invalid JSON expected 400, got %d", rec.Code)
	}
}

func TestHandleCreateLinkCodeStoreError(t *testing.T) {
	store := newFakeMealStore()
	store.createLinkingCodeErr = errors.New("db down")
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "POST", "/api/v1/bot/link-code", map[string]string{"platform": "telegram"}, nil)
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", rec.Code)
	}
}

func TestHandleCreateLinkCodeUnauthorized(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "POST", "/api/v1/bot/link-code", map[string]string{"platform": "telegram"}, map[string]string{"Authorization": ""})
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

// --- handleCompleteLink ---

func TestHandleCompleteLink(t *testing.T) {
	store := newFakeMealStore()
	store.linkingCode = types.LinkingCode{Code: "ABC123", UserID: "test-user", Platform: "telegram"}
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "POST", "/api/v1/bot/link", map[string]string{"code": "ABC123"}, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	got := decodeJSON[map[string]string](t, rec)
	if got["status"] != "linked" {
		t.Errorf("status = %q, want linked", got["status"])
	}
}

func TestHandleCompleteLinkMissingCode(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "POST", "/api/v1/bot/link", map[string]string{"code": ""}, nil)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("missing code expected 400, got %d", rec.Code)
	}
}

func TestHandleCompleteLinkInvalidJSON(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{})

	req := httptest.NewRequest("POST", "/api/v1/bot/link", strings.NewReader("not json"))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer test-api-key")
	rec := httptest.NewRecorder()
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("invalid JSON expected 400, got %d", rec.Code)
	}
}

func TestHandleCompleteLinkInvalidCode(t *testing.T) {
	store := newFakeMealStore()
	store.linkingCodeErr = types.ErrNotFound
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "POST", "/api/v1/bot/link", map[string]string{"code": "BADCOD"}, nil)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("invalid/expired code expected 400, got %d", rec.Code)
	}
}

func TestHandleCompleteLinkWrongUser(t *testing.T) {
	store := newFakeMealStore()
	store.linkingCode = types.LinkingCode{Code: "ABC123", UserID: "other-user"}
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "POST", "/api/v1/bot/link", map[string]string{"code": "ABC123"}, nil)
	if rec.Code != http.StatusForbidden {
		t.Errorf("code belonging to another user expected 403, got %d", rec.Code)
	}
}

func TestHandleCompleteLinkConsumeError(t *testing.T) {
	store := newFakeMealStore()
	store.linkingCode = types.LinkingCode{Code: "ABC123", UserID: "test-user"}
	store.consumeLinkingCodeErr = errors.New("db down")
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "POST", "/api/v1/bot/link", map[string]string{"code": "ABC123"}, nil)
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", rec.Code)
	}
}

func TestHandleCompleteLinkUnauthorized(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "POST", "/api/v1/bot/link", map[string]string{"code": "ABC123"}, map[string]string{"Authorization": ""})
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

// --- handleStreamLinkCode ---
//
// The endpoint polls on a hardcoded 1s ticker and waits on a real
// time.Timer for expiry, neither of which is injectable from tests. Per
// issue #126 guidance, we cover the auth/ownership/validation error paths
// synchronously and use a pre-canceled request context to exercise the
// happy-path setup (headers + 200) without blocking on the real ticker.
// ponytail: no fake clock plumbed through; add one only if a bug ships here.

func TestHandleStreamLinkCodeInvalidCode(t *testing.T) {
	store := newFakeMealStore()
	store.linkingCodeErr = types.ErrNotFound
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "GET", "/api/v1/bot/link-code/BADCOD/stream", nil, nil)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("invalid/expired code expected 400, got %d", rec.Code)
	}
}

func TestHandleStreamLinkCodeWrongUser(t *testing.T) {
	store := newFakeMealStore()
	store.linkingCode = types.LinkingCode{Code: "ABC123", UserID: "other-user"}
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "GET", "/api/v1/bot/link-code/ABC123/stream", nil, nil)
	if rec.Code != http.StatusForbidden {
		t.Errorf("code belonging to another user expected 403, got %d", rec.Code)
	}
}

func TestHandleStreamLinkCodeUnauthorized(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "GET", "/api/v1/bot/link-code/ABC123/stream", nil, map[string]string{"Authorization": ""})
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestHandleStreamLinkCodeOpensSSEThenClientDisconnects(t *testing.T) {
	store := newFakeMealStore()
	store.linkingCode = types.LinkingCode{Code: "ABC123", UserID: "test-user", ExpiresAt: "2026-06-17 12:00:00"}
	h := newHandler(store, &fakeMealLogger{})

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // simulate the client disconnecting immediately after the stream opens.

	req := httptest.NewRequest("GET", "/api/v1/bot/link-code/ABC123/stream", nil).WithContext(ctx)
	req.Header.Set("Authorization", "Bearer test-api-key")
	rec := httptest.NewRecorder()
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/event-stream") {
		t.Errorf("Content-Type = %q, want text/event-stream", ct)
	}
}
