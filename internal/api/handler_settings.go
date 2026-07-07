package api

import (
	"encoding/base64"
	"encoding/json"
	"net/http"

	"github.com/gsaraiva2109/dietdaemon/internal/auth"
)

// ---------------------------------------------------------------------------
// BYOK: per-user AI API key settings
// ---------------------------------------------------------------------------

// aiKeyStatus is the response body for GET /api/v1/settings/ai-key.
type aiKeyStatus struct {
	HasKey   bool   `json:"has_key"`
	Provider string `json:"provider,omitempty"`
}

// handleGetAIKey returns whether the user has a stored AI key and its provider.
// The key itself is never returned — not even the encrypted form.
func (h *Handler) handleGetAIKey(w http.ResponseWriter, r *http.Request, userID string) {
	provider, _, found, err := h.store.GetUserAIKey(r.Context(), userID)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	_ = json.NewEncoder(w).Encode(aiKeyStatus{HasKey: found, Provider: provider})
}

// handleSetAIKey encrypts and stores a per-user AI API key.
func (h *Handler) handleSetAIKey(w http.ResponseWriter, r *http.Request, userID string) {
	if h.cfg == nil || len(h.cfg.AIKeyEncKey) == 0 {
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "AI_KEY_ENC_KEY is not configured"})
		return
	}

	var body struct {
		Provider string `json:"provider"`
		Key      string `json:"key"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid JSON body: " + err.Error()})
		return
	}
	if body.Provider != "anthropic" && body.Provider != "openai" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "provider must be \"anthropic\" or \"openai\""})
		return
	}
	if body.Key == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "key is required"})
		return
	}

	// Encrypt + base64-encode (identical pattern to TOTP).
	ct, err := auth.Encrypt([]byte(body.Key), h.cfg.AIKeyEncKey)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	encKey := base64.RawStdEncoding.EncodeToString(ct)

	if err := h.store.SetUserAIKey(r.Context(), userID, body.Provider, encKey); err != nil {
		h.writeErr(w, err)
		return
	}
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// handleDeleteAIKey removes the user's stored AI key.
func (h *Handler) handleDeleteAIKey(w http.ResponseWriter, r *http.Request, userID string) {
	if err := h.store.DeleteUserAIKey(r.Context(), userID); err != nil {
		h.writeErr(w, err)
		return
	}
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
