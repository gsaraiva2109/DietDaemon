package api

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/gsaraiva2109/dietdaemon/core/ports"
)

// handleChatMessage streams an AI chat response over SSE.
// POST /api/v1/chat/sessions/{id}/messages
// Stage 1: Anthropic only, no tools, no persistence. Session {id} is accepted
// but not validated (stub until Stage 3).
func (h *Handler) handleChatMessage(w http.ResponseWriter, r *http.Request, userID string) {
	// Stage 1: return 503 if no chat adapter is configured.
	adapter := h.chatAdapter
	if adapter == nil {
		// Check for BYOK override.
		if override, ok := ports.ChatAdapterOverrideFromContext(r.Context()); ok {
			adapter = override
		}
	}
	if adapter == nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "chat not available with current COMPLETION_ADAPTER"})
		return
	}

	// Parse request body.
	var req struct {
		Text string `json:"text"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid request body"})
		return
	}
	if req.Text == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "text is required"})
		return
	}

	// Inject BYOK override if applicable.
	ctx := h.injectChatAdapterOverride(r.Context(), userID)
	if override, ok := ports.ChatAdapterOverrideFromContext(ctx); ok {
		adapter = override
	}

	// Flusher check (must be done before writing headers).
	flusher, ok := w.(http.Flusher)
	if !ok {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "streaming not supported"})
		return
	}

	// SSE headers.
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	// ponytail: hardcoded system prompt for Stage 1; Stage 3 adds i18n + per-user custom instructions.
	systemPrompt := "You are a helpful diet and nutrition assistant. You help users track meals, workouts, weight, water intake, sleep, and fasting. Be concise and supportive. Answer in the user's language."

	chatReq := ports.ChatRequest{
		System: systemPrompt,
		Messages: []ports.ChatMessage{
			{Role: "user", Content: req.Text},
		},
		// Stage 1: no tools; Stage 2 adds tool schema.
	}

	events, err := adapter.StreamChat(ctx, chatReq)
	if err != nil {
		slog.Error("chat stream error", "err", err)
		fmt.Fprintf(w, "event: error\ndata: {\"message\":\"%s\"}\n\n", "failed to start chat stream")
		flusher.Flush()
		return
	}

	for evt := range events {
		switch evt.Kind {
		case "text-delta":
			writeSSE(w, "delta", map[string]string{"text": evt.Text})
		case "tool-call":
			writeSSE(w, "tool-call", map[string]string{
				"id":   evt.ToolCall.ID,
				"name": evt.ToolCall.Name,
				"args": evt.ToolCall.Args,
			})
		case "done":
			writeSSE(w, "done", map[string]string{})
			flusher.Flush()
			return
		case "error":
			msg := "chat error"
			if evt.Err != nil {
				msg = evt.Err.Error()
			}
			writeSSE(w, "error", map[string]string{"message": msg})
			flusher.Flush()
			return
		}

		select {
		case <-r.Context().Done():
			return
		default:
		}
	}
}

// writeSSE writes a single SSE event to the response writer.
func writeSSE(w http.ResponseWriter, event string, data any) {
	b, err := json.Marshal(data)
	if err != nil {
		return
	}
	fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, string(b))
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}
}
