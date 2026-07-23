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
	text, ok := decodeChatMessage(w, r)
	if !ok {
		return
	}

	sessionID := r.PathValue("id")

	// Inject BYOK override (if AI_KEY_MODE=byok and the user has a key) before
	// falling back to the boot-configured adapter.
	ctx, err := h.injectChatAdapterOverride(r.Context(), userID)
	if err != nil {
		h.writeErr(w, err)
		return
	}
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
	history := h.chatHistory(r.Context(), userID, sessionID)
	if err := h.persistUserChatMessage(r.Context(), userID, sessionID, text); err != nil {
		h.writeErr(w, err)
		return
	}

	flusher, ok := startChatStream(w)
	if !ok {
		return
	}
	h.streamChatEvents(w, r.Context(), flusher, userID, sessionID, router.Run(ctx, userID, systemPrompt, history, text))
}

func decodeChatMessage(w http.ResponseWriter, r *http.Request) (string, bool) {
	r.Body = http.MaxBytesReader(w, r.Body, 32<<10)
	var req struct {
		Text string `json:"text"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid request body"})
		return "", false
	}
	if req.Text == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "text is required"})
		return "", false
	}
	return req.Text, true
}

// chatHistory loads replayable user and assistant turns before the new message.
func (h *Handler) chatHistory(ctx context.Context, userID, sessionID string) []ports.ChatMessage {
	if h.chatStore == nil {
		return nil
	}
	prior, err := h.chatStore.GetChatMessages(ctx, userID, sessionID)
	if err != nil {
		return nil
	}
	var history []ports.ChatMessage
	for _, m := range prior {
		if (m.Role == "user" || m.Role == "assistant") && m.Content != "" {
			history = append(history, ports.ChatMessage{Role: m.Role, Content: m.Content})
		}
	}
	return history
}

// persistUserChatMessage creates a client-generated session on its first message.
func (h *Handler) persistUserChatMessage(ctx context.Context, userID, sessionID, text string) error {
	if h.chatStore == nil {
		return nil
	}
	msgID := newHandlerID()
	err := h.chatStore.AppendChatMessage(ctx, msgID, userID, sessionID, "user", text, "")
	if !errors.Is(err, types.ErrNotFound) {
		return err
	}
	if err := h.chatStore.CreateChatSession(ctx, sessionID, userID, deriveSessionTitle(text)); err != nil {
		return err
	}
	return h.chatStore.AppendChatMessage(ctx, msgID, userID, sessionID, "user", text, "")
}

func startChatStream(w http.ResponseWriter) (http.Flusher, bool) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "streaming not supported"})
		return nil, false
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)
	flusher.Flush()
	return flusher, true
}

type chatStreamState struct {
	text  strings.Builder
	tools []toolInfo
}

func (h *Handler) streamChatEvents(w http.ResponseWriter, ctx context.Context, flusher http.Flusher, userID, sessionID string, events <-chan ports.ChatEvent) {
	state := chatStreamState{}
	for evt := range events {
		if h.handleChatStreamEvent(w, ctx, flusher, userID, sessionID, &state, evt) {
			return
		}
		select {
		case <-ctx.Done():
			return
		default:
		}
	}
}

func (h *Handler) handleChatStreamEvent(w http.ResponseWriter, ctx context.Context, flusher http.Flusher, userID, sessionID string, state *chatStreamState, evt ports.ChatEvent) bool {
	switch evt.Kind {
	case "text-delta":
		state.text.WriteString(evt.Text)
		writeSSE(w, "delta", map[string]string{"text": evt.Text})
	case "tool-call":
		writeSSE(w, "tool-call", map[string]string{"id": evt.ToolCall.ID, "name": evt.ToolCall.Name, "args": evt.ToolCall.Args})
	case "tool-result":
		state.tools = append(state.tools, toolInfo{ID: evt.ToolCall.ID, Name: evt.ToolCall.Name, Text: evt.ToolCall.Args})
		writeSSE(w, "tool-result", map[string]string{"id": evt.ToolCall.ID, "name": evt.ToolCall.Name, "text": evt.ToolCall.Args})
	case "suggestions":
		writeSSE(w, "suggestions", map[string]any{"options": evt.Suggestions})
	case "done":
		h.finishChatStream(w, ctx, flusher, userID, sessionID, state)
		return true
	case "error":
		h.failChatStream(w, ctx, flusher, userID, sessionID, state, evt.Err)
		return true
	}
	return false
}

func (h *Handler) finishChatStream(w http.ResponseWriter, ctx context.Context, flusher http.Flusher, userID, sessionID string, state *chatStreamState) {
	persistText := state.text.String()
	if cleaned, _ := assistant.ExtractSuggestions(persistText); cleaned != persistText {
		persistText = cleaned
	}
	h.persistAssistantMessages(ctx, userID, sessionID, persistText, state.tools)
	writeSSE(w, "done", map[string]string{})
	flusher.Flush()
}

func (h *Handler) failChatStream(w http.ResponseWriter, ctx context.Context, flusher http.Flusher, userID, sessionID string, state *chatStreamState, err error) {
	if err != nil {
		slog.Error("chat stream error", "err", err)
	}
	if state.text.Len() > 0 {
		h.persistAssistantMessages(ctx, userID, sessionID, state.text.String(), state.tools)
	}
	message := "Trouble reaching the AI provider. Please try again in a moment."
	if errors.Is(err, assistant.ErrMaxToolRounds) {
		message = "I tried a few different ways but couldn't finish that — try naming the food more simply, or give the exact grams."
	}
	writeSSE(w, "error", map[string]string{"message": message})
	flusher.Flush()
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
		if err := h.chatStore.AppendChatMessage(ctx, newHandlerID(), userID, sessionID, "tool", ti.Text, ti.Name); err != nil {
			slog.Error("persist chat message failed", "session_id", sessionID, "role", "tool", "err", err)
		}
	}
	// Save final assistant text.
	if strings.TrimSpace(text) != "" {
		if err := h.chatStore.AppendChatMessage(ctx, newHandlerID(), userID, sessionID, "assistant", text, ""); err != nil {
			slog.Error("persist chat message failed", "session_id", sessionID, "role", "assistant", "err", err)
		}
	}
}

// deriveSessionTitle builds a short session title from the first user message:
// collapses all whitespace (including newlines) to single spaces, trims, and
// truncates to ~60 runes with a trailing ellipsis if shortened.
func deriveSessionTitle(text string) string {
	collapsed := strings.Join(strings.Fields(text), " ")
	runes := []rune(collapsed)
	if len(runes) <= 60 {
		return collapsed
	}
	return string(runes[:60]) + "…"
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
	_, _ = fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, string(b))
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}
}
