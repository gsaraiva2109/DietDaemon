package api

import (
	"encoding/base64"
	"encoding/json"
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

// TestBYOKChatOverrideUsedInsteadOfSharedAdapter covers the success path of
// injectChatAdapterOverride: when the user has a confirmed BYOK key,
// handleChatMessage must route through the per-user adapter (pointed at a
// local test server standing in for the real provider) instead of the
// boot-configured shared adapter.
func TestBYOKChatOverrideUsedInsteadOfSharedAdapter(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: {\"choices\":[{\"index\":0,\"delta\":{\"content\":\"byok-response\"}}]}\n\ndata: [DONE]\n\n"))
	}))
	defer srv.Close()

	key := make([]byte, 32)
	ct, err := auth.Encrypt([]byte("sk-test"), key)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}

	store := newFakeMealStore()
	store.aiKeyFound = true
	store.aiKeyProvider = "openai"
	store.aiKeyEncrypted = base64.RawStdEncoding.EncodeToString(ct)

	// The shared adapter answers "shared-fallback" — if the BYOK override
	// didn't take effect, that's what the client would see instead.
	h := newChatHandler([]ports.ChatEvent{{Kind: "text-delta", Text: "shared-fallback"}, {Kind: "done"}}, nil)
	h.store = store
	h.cfg = &config.Config{
		AIKeyMode:     "byok",
		AIKeyEncKey:   key,
		OpenAIBaseURL: srv.URL,
		ModelTimeout:  5 * time.Second,
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/chat/sessions/test/messages", strings.NewReader(`{"text":"hello"}`))
	rec := httptest.NewRecorder()
	h.handleChatMessage(rec, req, "test-user")

	var gotText string
	for _, e := range parseSSE(rec.Body.String()) {
		if e.Event != "delta" {
			continue
		}
		var data map[string]string
		if err := json.Unmarshal([]byte(e.Data), &data); err == nil {
			gotText += data["text"]
		}
	}
	if strings.Contains(gotText, "shared-fallback") {
		t.Fatalf("BYOK override was not used; got the shared adapter's response: %q", gotText)
	}
	if !strings.Contains(gotText, "byok-response") {
		t.Fatalf("expected the BYOK-overridden adapter's response, got %q", gotText)
	}
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
