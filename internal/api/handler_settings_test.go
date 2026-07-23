package api

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gsaraiva2109/dietdaemon/core/types"
	"github.com/gsaraiva2109/dietdaemon/internal/auth"
	"github.com/gsaraiva2109/dietdaemon/internal/config"
)

// newHandlerWithAIKeyCfg mirrors newHandler but wires a *config.Config with
// AIKeyEncKey set, since handleSetAIKey/handleDeleteAIKey's encrypt path is
// gated on it (see handler_settings.go's h.cfg nil check).
func newHandlerWithAIKeyCfg(store MealStore, logger MealLogger) *Handler {
	authStore := newFakeAuthStore()
	cfg := &config.Config{AIKeyEncKey: []byte("0123456789abcdef0123456789abcdef")[:32]}
	return New(store, logger, time.UTC, nil, cfg,
		WithAuth(authStore, authStore, authStore, authStore, authStore, authStore, nil, "DietDaemon", AuthConfig{
			SessionCfg: auth.SessionConfig{
				IdleTTL:     1 * time.Hour,
				AbsoluteTTL: 24 * time.Hour,
				RememberTTL: 72 * time.Hour,
			},
			LockoutCfg:       auth.DefaultLockoutConfig(),
			RegistrationMode: types.RegistrationOpen,
			CookieSecure:     false,
		}),
	)
}

// --- handleGetAIKey ---

func TestHandleGetAIKeyFound(t *testing.T) {
	store := newFakeMealStore()
	store.aiKeyFound = true
	store.aiKeyProvider = "anthropic"
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "GET", "/api/v1/settings/ai-key", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	got := decodeJSON[aiKeyStatus](t, rec)
	if !got.HasKey || got.Provider != "anthropic" {
		t.Errorf("unexpected status: %+v", got)
	}
}

func TestHandleGetAIKeyNotSet(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "GET", "/api/v1/settings/ai-key", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	got := decodeJSON[aiKeyStatus](t, rec)
	if got.HasKey {
		t.Errorf("expected has_key=false, got %+v", got)
	}
}

func TestHandleGetAIKeyStoreError(t *testing.T) {
	store := newFakeMealStore()
	store.aiKeyErr = errors.New("db down")
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "GET", "/api/v1/settings/ai-key", nil, nil)
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", rec.Code)
	}
}

func TestHandleGetAIKeyUnauthorized(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "GET", "/api/v1/settings/ai-key", nil, map[string]string{"Authorization": ""})
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

// --- handleSetAIKey ---

func TestHandleSetAIKeyNotConfigured(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{}) // no cfg -> AIKeyEncKey unset

	body := map[string]string{"provider": "anthropic", "key": "sk-test"}
	rec := doRequest(h, "POST", "/api/v1/settings/ai-key", body, nil)
	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503 when AI_KEY_ENC_KEY unset, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleSetAIKey(t *testing.T) {
	store := newFakeMealStore()
	h := newHandlerWithAIKeyCfg(store, &fakeMealLogger{})

	body := map[string]string{"provider": "anthropic", "key": "sk-test"}
	rec := doRequest(h, "POST", "/api/v1/settings/ai-key", body, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	got := decodeJSON[map[string]string](t, rec)
	if got["status"] != "ok" {
		t.Errorf("status = %q, want ok", got["status"])
	}
}

func TestHandleSetAIKeyInvalidProvider(t *testing.T) {
	store := newFakeMealStore()
	h := newHandlerWithAIKeyCfg(store, &fakeMealLogger{})

	body := map[string]string{"provider": "gemini", "key": "sk-test"}
	rec := doRequest(h, "POST", "/api/v1/settings/ai-key", body, nil)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("invalid provider expected 400, got %d", rec.Code)
	}
}

func TestHandleSetAIKeyMissingKey(t *testing.T) {
	store := newFakeMealStore()
	h := newHandlerWithAIKeyCfg(store, &fakeMealLogger{})

	body := map[string]string{"provider": "anthropic", "key": ""}
	rec := doRequest(h, "POST", "/api/v1/settings/ai-key", body, nil)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("missing key expected 400, got %d", rec.Code)
	}
}

func TestHandleSetAIKeyInvalidJSON(t *testing.T) {
	store := newFakeMealStore()
	h := newHandlerWithAIKeyCfg(store, &fakeMealLogger{})

	req := httptest.NewRequest("POST", "/api/v1/settings/ai-key", strings.NewReader("not json"))
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

func TestHandleSetAIKeyStoreError(t *testing.T) {
	store := newFakeMealStore()
	store.setAIKeyErr = errors.New("db down")
	h := newHandlerWithAIKeyCfg(store, &fakeMealLogger{})

	body := map[string]string{"provider": "anthropic", "key": "sk-test"}
	rec := doRequest(h, "POST", "/api/v1/settings/ai-key", body, nil)
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", rec.Code)
	}
}

func TestHandleSetAIKeyUnauthorized(t *testing.T) {
	store := newFakeMealStore()
	h := newHandlerWithAIKeyCfg(store, &fakeMealLogger{})

	body := map[string]string{"provider": "anthropic", "key": "sk-test"}
	rec := doRequest(h, "POST", "/api/v1/settings/ai-key", body, map[string]string{"Authorization": ""})
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

// --- handleDeleteAIKey ---

func TestHandleDeleteAIKey(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "DELETE", "/api/v1/settings/ai-key", nil, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	got := decodeJSON[map[string]string](t, rec)
	if got["status"] != "ok" {
		t.Errorf("status = %q, want ok", got["status"])
	}
}

func TestHandleDeleteAIKeyStoreError(t *testing.T) {
	store := newFakeMealStore()
	store.deleteAIKeyErr = errors.New("db down")
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "DELETE", "/api/v1/settings/ai-key", nil, nil)
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", rec.Code)
	}
}

func TestHandleDeleteAIKeyUnauthorized(t *testing.T) {
	store := newFakeMealStore()
	h := newHandler(store, &fakeMealLogger{})

	rec := doRequest(h, "DELETE", "/api/v1/settings/ai-key", nil, map[string]string{"Authorization": ""})
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}
