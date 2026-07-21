package api

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gsaraiva2109/dietdaemon/core/ports"
	"github.com/gsaraiva2109/dietdaemon/core/types"
	"github.com/gsaraiva2109/dietdaemon/internal/auth"
	"github.com/gsaraiva2109/dietdaemon/internal/config"
)

func TestDecryptAIKey(t *testing.T) {
	key := make([]byte, 32)
	ct, err := auth.Encrypt([]byte("secret-api-key"), key)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}

	got, err := decryptAIKey(base64.RawStdEncoding.EncodeToString(ct), key)
	if err != nil || string(got) != "secret-api-key" {
		t.Fatalf("decryptAIKey = %q, %v", got, err)
	}
	if _, err := decryptAIKey("not-base64!", key); err == nil {
		t.Error("decryptAIKey accepted invalid base64")
	}
}

func TestBuildBYOKAdaptersRejectUnsupportedProvider(t *testing.T) {
	if _, err := buildAdapterForProvider("ollama", "key", "", "", "", time.Second); err == nil {
		t.Error("buildAdapterForProvider accepted unsupported provider")
	}
	if _, err := buildChatAdapterForProvider("ollama", "key", "", "", "", time.Second); err == nil {
		t.Error("buildChatAdapterForProvider accepted unsupported provider")
	}
}

func TestBYOKFailuresDoNotFallBackToSharedAdapters(t *testing.T) {
	key := make([]byte, 32)
	byokConfig := &config.Config{AIKeyMode: "byok", AIKeyEncKey: key}

	t.Run("suggest lookup failure", func(t *testing.T) {
		store := newFakeMealStore()
		store.aiKeyErr = types.ErrNotFound
		h := newHandler(store, &fakeMealLogger{}, &fakeSuggester{})
		h.cfg = byokConfig

		rec := httptest.NewRecorder()
		h.handleSuggest(rec, httptest.NewRequest(http.MethodGet, "/api/v1/suggest", nil), "test-user")
		assertBYOKFailure(t, rec)
	})

	t.Run("meal parse decrypt failure", func(t *testing.T) {
		store := newFakeMealStore()
		store.aiKeyFound = true
		store.aiKeyEncrypted = "not-base64!"
		h := newHandler(store, &fakeMealLogger{})
		h.cfg = byokConfig

		rec := httptest.NewRecorder()
		h.handleLogMeal(rec, httptest.NewRequest(http.MethodPost, "/api/v1/meals", strings.NewReader(`{"text":"chicken"}`)), "test-user")
		assertBYOKFailure(t, rec)
	})

	t.Run("chat provider failure", func(t *testing.T) {
		store := newFakeMealStore()
		ct, err := auth.Encrypt([]byte("key"), key)
		if err != nil {
			t.Fatalf("Encrypt: %v", err)
		}
		store.aiKeyFound = true
		store.aiKeyProvider = "ollama"
		store.aiKeyEncrypted = base64.RawStdEncoding.EncodeToString(ct)
		h := newChatHandler(nil, nil)
		h.store = store
		h.cfg = byokConfig

		rec := httptest.NewRecorder()
		h.handleChatMessage(rec, httptest.NewRequest(http.MethodPost, "/api/v1/chat/sessions/test/messages", strings.NewReader(`{"text":"hello"}`)), "test-user")
		assertBYOKFailure(t, rec)
	})
}

func TestBYOKKeyAbsenceRetainsSharedAdapterFallback(t *testing.T) {
	h := newHandler(newFakeMealStore(), &fakeMealLogger{})
	h.cfg = &config.Config{AIKeyMode: "byok", AIKeyEncKey: make([]byte, 32)}

	ctx, err := h.injectModelOverride(t.Context(), "test-user")
	if err != nil {
		t.Fatalf("injectModelOverride: %v", err)
	}
	if _, ok := ports.ModelOverrideFromContext(ctx); ok {
		t.Error("missing BYOK key should retain the shared adapter fallback")
	}
}

func assertBYOKFailure(t *testing.T, rec *httptest.ResponseRecorder) {
	t.Helper()
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500: %s", rec.Code, rec.Body.String())
	}
	if rec.Body.String() != "{\"error\":\"internal server error\"}\n" {
		t.Fatalf("response = %q, want generic 500", rec.Body.String())
	}
}
