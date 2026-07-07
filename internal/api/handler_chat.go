package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/gsaraiva2109/dietdaemon/core/ports"
	"github.com/gsaraiva2109/dietdaemon/internal/assistant"
)

// handleChatMessage streams an AI chat response over SSE.
// POST /api/v1/chat/sessions/{id}/messages
func (h *Handler) handleChatMessage(w http.ResponseWriter, r *http.Request, userID string) {
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

	sessionID := r.PathValue("id")

	// Inject BYOK override (if AI_KEY_MODE=byok and the user has a key) before
	// falling back to the boot-configured adapter.
	ctx := h.injectChatAdapterOverride(r.Context(), userID)
	router := h.assistantRouter
	if override, ok := ports.ChatAdapterOverrideFromContext(ctx); ok {
		router = assistant.New(override, h.chatCommands, h.toolDescs)
	}
	if router == nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "chat not available with current COMPLETION_ADAPTER"})
		return
	}

	// Resolve localized system prompt.
	systemPrompt := h.chatSystemPrompt(r.Context(), userID)

	// Persist user message before streaming — also doubles as the session
	// ownership check (AppendChatMessage no-ops and returns ErrNotFound if
	// sessionID doesn't belong to userID), so a foreign/bogus session ID is
	// rejected before any SSE headers go out or the LLM gets called.
	if h.chatStore != nil {
		if err := h.chatStore.AppendChatMessage(r.Context(), newHandlerID(), userID, sessionID, "user", req.Text, ""); err != nil {
			h.writeErr(w, err)
			return
		}
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

	// Accumulators for persistence on done.
	var (
		textBuf   strings.Builder
		toolInfos []toolInfo // saved on done alongside assistant message
	)

	events := router.Run(ctx, userID, systemPrompt, req.Text)

	for evt := range events {
		switch evt.Kind {
		case "text-delta":
			textBuf.WriteString(evt.Text)
			writeSSE(w, "delta", map[string]string{"text": evt.Text})
		case "tool-call":
			writeSSE(w, "tool-call", map[string]string{
				"id":   evt.ToolCall.ID,
				"name": evt.ToolCall.Name,
				"args": evt.ToolCall.Args,
			})
		case "tool-result":
			toolInfos = append(toolInfos, toolInfo{
				ID:   evt.ToolCall.ID,
				Name: evt.ToolCall.Name,
				Text: evt.ToolCall.Args,
			})
			writeSSE(w, "tool-result", map[string]string{
				"id":   evt.ToolCall.ID,
				"name": evt.ToolCall.Name,
				"text": evt.ToolCall.Args,
			})
		case "done":
			h.persistAssistantMessages(r.Context(), userID, sessionID, textBuf.String(), toolInfos)
			writeSSE(w, "done", map[string]string{})
			flusher.Flush()
			return
		case "error":
			if evt.Err != nil {
				slog.Error("chat stream error", "err", evt.Err)
			}
			// Save what we have so far before sending error.
			if textBuf.Len() > 0 {
				h.persistAssistantMessages(r.Context(), userID, sessionID, textBuf.String(), toolInfos)
			}
			writeSSE(w, "error", map[string]string{"message": "chat error, please try again"})
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

// chatSystemPrompt builds the system prompt from the i18n base template and
// the user's custom instructions.
func (h *Handler) chatSystemPrompt(ctx context.Context, userID string) string {
	// Resolve locale from the user record.
	locale := "en"
	if h.store != nil {
		u, err := h.store.GetUser(ctx, userID)
		if err == nil && u.Locale != "" {
			locale = u.Locale
		}
	}

	basePrompt := "You are a helpful diet and nutrition assistant. You help users track meals, workouts, weight, water intake, sleep, and fasting. Be concise and supportive. Answer in the user's language."
	if h.i18nBundle != nil {
		if resolved := h.i18nBundle.T(locale, "assistant.system_prompt", nil); resolved != "" {
			basePrompt = resolved
		}
	}

	// Append custom instructions if set.
	if h.chatStore != nil {
		ci, found, _ := h.chatStore.GetAssistantSettings(ctx, userID)
		if found && strings.TrimSpace(ci) != "" {
			basePrompt += "\n\n" + strings.TrimRight(ci, "\n")
		}
	}

	return basePrompt
}

// toolInfo records a tool execution result for persistence.
type toolInfo struct {
	ID   string
	Name string
	Text string
}

// persistAssistantMessages saves the assistant's text and any tool results to the DB.
func (h *Handler) persistAssistantMessages(ctx context.Context, userID, sessionID, text string, tools []toolInfo) {
	if h.chatStore == nil {
		return
	}
	// Save tool results first (they happened before the final text).
	for _, ti := range tools {
		_ = h.chatStore.AppendChatMessage(ctx, newHandlerID(), userID, sessionID, "tool", ti.Text, ti.Name)
	}
	// Save final assistant text.
	if strings.TrimSpace(text) != "" {
		_ = h.chatStore.AppendChatMessage(ctx, newHandlerID(), userID, sessionID, "assistant", text, "")
	}
}

// ---------------------------------------------------------------------------
// Session CRUD
// ---------------------------------------------------------------------------

// handleCreateChatSession creates a new chat session.
// POST /api/v1/chat/sessions
func (h *Handler) handleCreateChatSession(w http.ResponseWriter, r *http.Request, userID string) {
	if h.chatStore == nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "chat persistence not available"})
		return
	}

	var body struct {
		Title string `json:"title"`
	}
	// Body is optional — empty body is fine, creates with default empty title.
	if r.ContentLength > 0 {
		_ = json.NewDecoder(r.Body).Decode(&body)
	}

	id := newHandlerID()
	if err := h.chatStore.CreateChatSession(r.Context(), id, userID, body.Title); err != nil {
		h.writeErr(w, err)
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]string{"id": id})
}

// handleListChatSessions returns all sessions for the authenticated user.
// GET /api/v1/chat/sessions
func (h *Handler) handleListChatSessions(w http.ResponseWriter, r *http.Request, userID string) {
	if h.chatStore == nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "chat persistence not available"})
		return
	}

	sessions, err := h.chatStore.ListChatSessions(r.Context(), userID)
	if err != nil {
		h.writeErr(w, err)
		return
	}

	_ = json.NewEncoder(w).Encode(sessions)
}

// handleGetChatMessages returns the message history for a session.
// GET /api/v1/chat/sessions/{id}/messages
func (h *Handler) handleGetChatMessages(w http.ResponseWriter, r *http.Request, userID string) {
	if h.chatStore == nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "chat persistence not available"})
		return
	}

	messages, err := h.chatStore.GetChatMessages(r.Context(), userID, r.PathValue("id"))
	if err != nil {
		h.writeErr(w, err)
		return
	}

	_ = json.NewEncoder(w).Encode(messages)
}

// ---------------------------------------------------------------------------
// Assistant settings (custom instructions)
// ---------------------------------------------------------------------------

// handleGetChatSettings returns the user's assistant settings.
// GET /api/v1/chat/settings
func (h *Handler) handleGetChatSettings(w http.ResponseWriter, r *http.Request, userID string) {
	if h.chatStore == nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "chat persistence not available"})
		return
	}

	ci, found, err := h.chatStore.GetAssistantSettings(r.Context(), userID)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	if !found {
		_ = json.NewEncoder(w).Encode(map[string]string{"custom_instructions": ""})
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]string{"custom_instructions": ci})
}

// handleSetChatSettings updates the user's assistant settings.
// PUT /api/v1/chat/settings
func (h *Handler) handleSetChatSettings(w http.ResponseWriter, r *http.Request, userID string) {
	if h.chatStore == nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "chat persistence not available"})
		return
	}

	var body struct {
		CustomInstructions string `json:"custom_instructions"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid request body"})
		return
	}

	if err := h.chatStore.SetAssistantSettings(r.Context(), userID, body.CustomInstructions); err != nil {
		h.writeErr(w, err)
		return
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
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
