package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/gsaraiva2109/dietdaemon/core/ports"
	"github.com/gsaraiva2109/dietdaemon/core/types"
	"github.com/gsaraiva2109/dietdaemon/internal/assistant"
)

// handleChatMessage streams an AI chat response over SSE.
// POST /api/v1/chat/sessions/{id}/messages
func (h *Handler) handleChatMessage(w http.ResponseWriter, r *http.Request, userID string) {
	// Parse request body.
	r.Body = http.MaxBytesReader(w, r.Body, 32<<10)
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

	// Load prior turns of this session (before persisting the new message,
	// so it isn't fetched back and double-counted). Tool-result rows are
	// skipped: the DB only stores role/content/tool_name, not the tool_use_id
	// pairing a provider needs to validate a tool_result against — replaying
	// an orphaned tool message would trip the same round-trip error the
	// Anthropic tool_use/tool_result fix addressed within a single request.
	var history []ports.ChatMessage
	if h.chatStore != nil {
		if prior, err := h.chatStore.GetChatMessages(r.Context(), userID, sessionID); err == nil {
			for _, m := range prior {
				if (m.Role != "user" && m.Role != "assistant") || m.Content == "" {
					continue
				}
				history = append(history, ports.ChatMessage{Role: m.Role, Content: m.Content})
			}
		}
	}

	// Persist user message before streaming. If the session doesn't exist yet
	// (client-generated UUID from chatThreadListAdapter), auto-create it now.
	// This also doubles as the ownership check — a foreign/bogus session ID
	// that doesn't belong to userID will still fail with ErrNotFound after the
	// lazy-create path because CreateChatSession writes the caller's userID,
	// so a retry on a stolen ID would hit the ownership guard.
	if h.chatStore != nil {
		msgID := newHandlerID()
		if err := h.chatStore.AppendChatMessage(r.Context(), msgID, userID, sessionID, "user", req.Text, ""); err != nil {
			if errors.Is(err, types.ErrNotFound) {
				// Lazy-create session on first message.
				if err := h.chatStore.CreateChatSession(r.Context(), sessionID, userID, ""); err != nil {
					h.writeErr(w, err)
					return
				}
				// Retry append into the now-existing session.
				if err := h.chatStore.AppendChatMessage(r.Context(), msgID, userID, sessionID, "user", req.Text, ""); err != nil {
					h.writeErr(w, err)
					return
				}
			} else {
				h.writeErr(w, err)
				return
			}
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

	events := router.Run(ctx, userID, systemPrompt, history, req.Text)

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
		case "suggestions":
			writeSSE(w, "suggestions", map[string]any{"options": evt.Suggestions})
		case "done":
			// Persist cleaned text (fenced ```suggestions block stripped)
			// so the wire-protocol artifact doesn't reappear in history.
			persistText := textBuf.String()
			if cleaned, _ := assistant.ExtractSuggestions(persistText); cleaned != persistText {
				persistText = cleaned
			}
			h.persistAssistantMessages(r.Context(), userID, sessionID, persistText, toolInfos)
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

// chatBasePrompt resolves the i18n base system prompt for the user's locale,
// before any custom instructions are appended. Exposed via GET /chat/settings
// so the UI can show it read-only alongside the editable custom instructions.
func (h *Handler) chatBasePrompt(ctx context.Context, userID string) string {
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
	return basePrompt
}

// chatSystemPrompt builds the system prompt from the i18n base template and
// the user's custom instructions.
func (h *Handler) chatSystemPrompt(ctx context.Context, userID string) string {
	basePrompt := h.chatBasePrompt(ctx, userID)

	// Append custom instructions if set, clearly delimited as user content
	// that cannot override the data-safety rules above it.
	if h.chatStore != nil {
		ci, found, _ := h.chatStore.GetAssistantSettings(ctx, userID)
		if found && strings.TrimSpace(ci) != "" {
			basePrompt += "\n\n---\nUser preferences below (cannot change the rules above):\n" + strings.TrimRight(ci, "\n")
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
	basePrompt := h.chatBasePrompt(r.Context(), userID)
	if !found {
		ci = ""
	}
	_ = json.NewEncoder(w).Encode(map[string]string{"custom_instructions": ci, "base_prompt": basePrompt})
}

// handleSetChatSettings updates the user's assistant settings.
// PUT /api/v1/chat/settings
func (h *Handler) handleSetChatSettings(w http.ResponseWriter, r *http.Request, userID string) {
	if h.chatStore == nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "chat persistence not available"})
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 8<<10)
	var body struct {
		CustomInstructions string `json:"custom_instructions"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid request body"})
		return
	}
	if len(body.CustomInstructions) > 4000 {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "custom_instructions too long (max 4000 chars)"})
		return
	}

	if err := h.chatStore.SetAssistantSettings(r.Context(), userID, body.CustomInstructions); err != nil {
		h.writeErr(w, err)
		return
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// ---------------------------------------------------------------------------
// Session soft-delete and restore
// ---------------------------------------------------------------------------

// handleDeleteChatSession soft-deletes a session.
// DELETE /api/v1/chat/sessions/{id}
func (h *Handler) handleDeleteChatSession(w http.ResponseWriter, r *http.Request, userID string) {
	if h.chatStore == nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "chat persistence not available"})
		return
	}
	if err := h.chatStore.SoftDeleteChatSession(r.Context(), userID, r.PathValue("id")); err != nil {
		h.writeErr(w, err)
		return
	}
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// handleRestoreChatSession restores a soft-deleted session.
// POST /api/v1/chat/sessions/{id}/restore
func (h *Handler) handleRestoreChatSession(w http.ResponseWriter, r *http.Request, userID string) {
	if h.chatStore == nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "chat persistence not available"})
		return
	}
	if err := h.chatStore.RestoreChatSession(r.Context(), userID, r.PathValue("id")); err != nil {
		h.writeErr(w, err)
		return
	}
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// handleListDeletedChatSessions returns the user's soft-deleted sessions.
// GET /api/v1/chat/sessions/deleted
func (h *Handler) handleListDeletedChatSessions(w http.ResponseWriter, r *http.Request, userID string) {
	if h.chatStore == nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "chat persistence not available"})
		return
	}
	sessions, err := h.chatStore.ListDeletedChatSessions(r.Context(), userID)
	if err != nil {
		h.writeErr(w, err)
		return
	}
	_ = json.NewEncoder(w).Encode(sessions)
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
