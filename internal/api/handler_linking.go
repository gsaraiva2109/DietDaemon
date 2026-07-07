package api

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log/slog"
	"math/big"
	"net/http"
	"time"
)

// ---------------------------------------------------------------------------
// Bot account linking handlers -- create link code, complete link, SSE stream.
// ---------------------------------------------------------------------------

// handleCreateLinkCode generates a one-time linking code for bot account linking.
// POST /api/v1/bot/link-code
func (h *Handler) handleCreateLinkCode(w http.ResponseWriter, r *http.Request, userID string) {
	var req struct {
		Platform string `json:"platform"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid request body"})
		return
	}
	if req.Platform == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "platform is required"})
		return
	}

	code := generateLinkCode()
	if err := h.store.CreateLinkingCode(r.Context(), userID, req.Platform, code); err != nil {
		slog.Error("create linking code", "err", err)
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "failed to create linking code"})
		return
	}

	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]string{"code": code})
}

// handleCompleteLink is the dashboard-side endpoint to complete a bot linking
// flow. The user enters the code on the dashboard; this endpoint validates the
// code and marks it as consumed.
// POST /api/v1/bot/link
func (h *Handler) handleCompleteLink(w http.ResponseWriter, r *http.Request, userID string) {
	var req struct {
		Code string `json:"code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid request body"})
		return
	}
	if req.Code == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "code is required"})
		return
	}

	lc, err := h.store.LookupLinkingCode(r.Context(), req.Code)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid or expired code"})
		return
	}

	// Verify the code belongs to the authenticated user.
	if lc.UserID != userID {
		w.WriteHeader(http.StatusForbidden)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "code does not belong to this account"})
		return
	}

	if err := h.store.ConsumeLinkingCode(r.Context(), req.Code); err != nil {
		slog.Error("consume linking code", "err", err)
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "failed to complete linking"})
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]string{"status": "linked"})
}

// handleStreamLinkCode is an SSE endpoint that streams the status of a linking
// code. The client connects after generating a code and receives a "linked"
// event when the bot consumes the code (via /link). If the code expires the
// stream closes without a "linked" event.
// GET /api/v1/bot/link-code/{code}/stream
func (h *Handler) handleStreamLinkCode(w http.ResponseWriter, r *http.Request, userID string) {
	code := r.PathValue("code")
	if code == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "code is required"})
		return
	}

	// Verify the code exists and belongs to this user before streaming.
	lc, err := h.store.LookupLinkingCode(r.Context(), code)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid or expired code"})
		return
	}
	if lc.UserID != userID {
		w.WriteHeader(http.StatusForbidden)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "code does not belong to this account"})
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "streaming not supported"})
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	// Parse expiry to compute deadline.
	expiresAt, err := time.Parse("2006-01-02 15:04:05", lc.ExpiresAt)
	if err != nil {
		expiresAt = time.Now().UTC().Add(10 * time.Minute)
	}
	deadline := time.NewTimer(time.Until(expiresAt))
	defer deadline.Stop()

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case <-deadline.C:
			// Code expired without being consumed.
			fmt.Fprintf(w, "event: expired\ndata: {}\n\n")
			flusher.Flush()
			return
		case <-ticker.C:
			current, err := h.store.LookupLinkingCodeAny(r.Context(), code)
			if err != nil {
				// Code no longer exists — treat as expired.
				fmt.Fprintf(w, "event: expired\ndata: {}\n\n")
				flusher.Flush()
				return
			}
			if current.UsedAt != "" {
				fmt.Fprintf(w, "event: linked\ndata: {}\n\n")
				flusher.Flush()
				return
			}
		}
	}
}

// generateLinkCode returns a 6-character alphanumeric code suitable for
// one-time linking. It excludes ambiguous characters (0/O, 1/I/l).
func generateLinkCode() string {
	const alphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	b := make([]byte, 6)
	for i := range b {
		n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(alphabet))))
		b[i] = alphabet[n.Int64()]
	}
	return string(b)
}
