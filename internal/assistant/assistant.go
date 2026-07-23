// Package assistant implements the tool-calling router for the AI chat assistant.
// It streams LLM responses token-by-token and executes DietDaemon slash-commands
// as tools when the model requests them, feeding results back into the conversation.
//
// The router is independently testable without HTTP — it works purely with the
// ports.ChatAdapter interface and a command registry.
package assistant

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/gsaraiva2109/dietdaemon/core/ports"
	"github.com/gsaraiva2109/dietdaemon/core/types"
)

const maxToolRounds = 6

// ErrMaxToolRounds is returned when the tool-calling loop exceeds maxToolRounds
// without the model ever finishing with plain text. Callers can check for it
// with errors.Is to show a more specific message than a generic stream failure.
var ErrMaxToolRounds = errors.New("assistant: exceeded max tool rounds")

// Router runs the tool-calling loop for the conversational assistant.
type Router struct {
	adapter  ports.ChatAdapter
	commands map[string]ports.Command // name -> command
	tools    []ports.ToolSpec
}

// New creates a Router. toolDescs maps command names to resolved description
// text (i18n-resolved by the caller with an appropriate locale).
func New(adapter ports.ChatAdapter, cmds []ports.Command, toolDescs map[string]string) *Router {
	tools := make([]ports.ToolSpec, 0, len(cmds))
	cmdMap := make(map[string]ports.Command, len(cmds))
	for _, c := range cmds {
		name := c.Name()
		desc := toolDescs[name]
		if desc == "" {
			desc = name
		}
		// LLM tool-call names must match ^[a-zA-Z0-9_-]+$ (OpenAI/Anthropic
		// both reject "/"), but command names are Telegram-style "/foo" —
		// strip the slash for the wire spec and key the map the same way so
		// executeCommand can look tool calls up directly.
		wireName := strings.TrimPrefix(name, "/")
		tools = append(tools, ports.ToolSpec{Name: wireName, Description: desc})
		cmdMap[wireName] = c
	}
	return &Router{
		adapter:  adapter,
		commands: cmdMap,
		tools:    tools,
	}
}

// Run executes the tool-calling loop. history is the prior turns of this
// session (may be nil for a new session); it is seeded ahead of userMessage
// so the model has memory of the conversation so far. The caller reads
// events from the returned channel and writes SSE to the HTTP client.
func (r *Router) Run(ctx context.Context, userID, systemPrompt string, history []ports.ChatMessage, userMessage string) <-chan ports.ChatEvent {
	ch := make(chan ports.ChatEvent, 32)
	go r.loop(ctx, ch, userID, systemPrompt, history, userMessage)
	return ch
}

// sendOut delivers evt to out, or bails if ctx is cancelled first — without
// this, a client that disconnects mid-stream while out's buffer is full
// leaks this goroutine (and the underlying adapter stream) forever. Tries a
// non-blocking send first: a bare `select { case out<-evt: case <-ctx.Done(): }`
// picks randomly between simultaneously-ready cases, so it can drop an event
// even when out has buffer room, if ctx happens to already be cancelled.
func sendOut(ctx context.Context, out chan<- ports.ChatEvent, evt ports.ChatEvent) bool {
	select {
	case out <- evt:
		return true
	default:
	}
	select {
	case out <- evt:
		return true
	case <-ctx.Done():
		return false
	}
}

func (r *Router) loop(ctx context.Context, out chan<- ports.ChatEvent, userID, systemPrompt string, history []ports.ChatMessage, userMessage string) {
	defer close(out)

	messages := make([]ports.ChatMessage, 0, len(history)+1)
	messages = append(messages, history...)
	messages = append(messages, ports.ChatMessage{Role: "user", Content: userMessage})

	for range maxToolRounds {
		if !r.runRound(ctx, out, userID, systemPrompt, &messages) {
			return
		}
	}

	// Exceeded max tool rounds.
	sendOut(ctx, out, ports.ChatEvent{Kind: "error", Err: ErrMaxToolRounds})
}

// runRound streams one model response. It returns true only when tool results
// were produced and another model round is needed.
func (r *Router) runRound(ctx context.Context, out chan<- ports.ChatEvent, userID, systemPrompt string, messages *[]ports.ChatMessage) bool {
	req := ports.ChatRequest{System: systemPrompt, Messages: *messages, Tools: r.tools}
	events, err := r.adapter.StreamChat(ctx, req)
	if err != nil {
		sendOut(ctx, out, ports.ChatEvent{Kind: "error", Err: fmt.Errorf("assistant: stream: %w", err)})
		return false
	}

	var textBuf strings.Builder
	var toolCalls []ports.ToolCallEvent
	for evt := range events {
		switch evt.Kind {
		case "text-delta":
			textBuf.WriteString(evt.Text)
			if !sendOut(ctx, out, evt) {
				return false
			}
		case "tool-call":
			toolCalls = append(toolCalls, *evt.ToolCall)
			if !sendOut(ctx, out, evt) {
				return false
			}
		case "done":
			return r.finishRound(ctx, out, userID, messages, textBuf.String(), toolCalls, evt)
		case "error":
			sendOut(ctx, out, evt)
			return false
		}
		if ctx.Err() != nil {
			return false
		}
	}
	return false
}

func (r *Router) finishRound(ctx context.Context, out chan<- ports.ChatEvent, userID string, messages *[]ports.ChatMessage, text string, toolCalls []ports.ToolCallEvent, done ports.ChatEvent) bool {
	if text != "" || len(toolCalls) > 0 {
		*messages = append(*messages, ports.ChatMessage{Role: "assistant", Content: text, ToolCalls: toolCalls})
	}
	if len(toolCalls) > 0 {
		return r.sendToolResults(ctx, out, userID, messages, toolCalls)
	}
	if !sendSuggestions(ctx, out, text) {
		return false
	}
	sendOut(ctx, out, done)
	return false
}

func (r *Router) sendToolResults(ctx context.Context, out chan<- ports.ChatEvent, userID string, messages *[]ports.ChatMessage, toolCalls []ports.ToolCallEvent) bool {
	for _, tc := range toolCalls {
		reply := r.executeCommand(ctx, userID, tc)
		if !sendOut(ctx, out, ports.ChatEvent{Kind: "tool-result", ToolCall: &ports.ToolCallEvent{ID: tc.ID, Name: tc.Name, Args: reply}}) {
			return false
		}
		*messages = append(*messages, ports.ChatMessage{Role: "tool", Content: reply, ToolCallID: tc.ID, ToolName: tc.Name})
	}
	return true
}

// sendSuggestions forwards model-native suggestions before the terminal done
// event so the frontend can render them with the completed assistant turn.
func sendSuggestions(ctx context.Context, out chan<- ports.ChatEvent, text string) bool {
	_, suggestions := ExtractSuggestions(text)
	return len(suggestions) == 0 || sendOut(ctx, out, ports.ChatEvent{Kind: "suggestions", Suggestions: suggestions})
}

// executeCommand looks up a command by name and calls its Handle method.
func (r *Router) executeCommand(ctx context.Context, userID string, tc ports.ToolCallEvent) string {
	cmd, ok := r.commands[tc.Name]
	if !ok {
		return fmt.Sprintf("unknown command: %s", tc.Name)
	}

	msg := types.InboundMessage{
		UserID:      userID,
		Text:        "/" + tc.Name + " " + tc.Args,
		ChannelMeta: map[string]string{"channel": "web"},
	}

	reply, err := cmd.Handle(ctx, msg, tc.Args)
	if err != nil {
		return fmt.Sprintf("error executing %s: %v", tc.Name, err)
	}
	return reply.Text
}
