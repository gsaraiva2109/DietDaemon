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

// suggestFallback is returned when the tool-calling loop exceeds maxToolRounds.
const suggestFallback = "Couldn't complete that request. Please try again."

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

	for round := 0; round < maxToolRounds; round++ {
		req := ports.ChatRequest{
			System:   systemPrompt,
			Messages: messages,
			Tools:    r.tools,
		}

		events, err := r.adapter.StreamChat(ctx, req)
		if err != nil {
			sendOut(ctx, out, ports.ChatEvent{Kind: "error", Err: fmt.Errorf("assistant: stream: %w", err)})
			return
		}

		var (
			textBuf   strings.Builder
			toolCalls []ports.ToolCallEvent
		)

		for evt := range events {
			switch evt.Kind {
			case "text-delta":
				textBuf.WriteString(evt.Text)
				if !sendOut(ctx, out, evt) { // forward in real-time
					return
				}

			case "tool-call":
				toolCalls = append(toolCalls, *evt.ToolCall)
				if !sendOut(ctx, out, evt) { // forward to client
					return
				}

			case "done":
				// Append accumulated assistant turn (text and/or tool_use
				// blocks) to history. ToolCalls must be preserved even when
				// there's no text — providers require the tool_use block
				// that a tool_result answers to still be present in history.
				if textBuf.Len() > 0 || len(toolCalls) > 0 {
					messages = append(messages, ports.ChatMessage{
						Role:      "assistant",
						Content:   textBuf.String(),
						ToolCalls: toolCalls,
					})
				}

				// Execute any tool calls from this round.
				if len(toolCalls) > 0 {
					for _, tc := range toolCalls {
						reply := r.executeCommand(ctx, userID, tc)
						if !sendOut(ctx, out, ports.ChatEvent{
							Kind: "tool-result",
							ToolCall: &ports.ToolCallEvent{
								ID:   tc.ID,
								Name: tc.Name,
								Args: reply,
							},
						}) {
							return
						}
						messages = append(messages, ports.ChatMessage{
							Role:       "tool",
							Content:    reply,
							ToolCallID: tc.ID,
							ToolName:   tc.Name,
						})
					}
					// Continue to next round (tool results may prompt more text).
					goto nextRound
				}

				// No tools — conversation complete.
				// Extract model-native suggestions before forwarding the done
				// event, so the frontend receives them before "done".
				if cleaned, suggestions := ExtractSuggestions(textBuf.String()); len(suggestions) > 0 {
					if !sendOut(ctx, out, ports.ChatEvent{Kind: "suggestions", Suggestions: suggestions}) {
						return
					}
					// Persist cleaned text (without the fenced block) so the
					// wire-protocol artifact doesn't reappear in history replays.
					// The text-delta events already streamed the raw block to the
					// client mid-turn; the frontend strips it visually, and the
					// handler persists the cleaned version below via the done event.
					_ = cleaned
				}
				sendOut(ctx, out, evt) // forward done event
				return

			case "error":
				sendOut(ctx, out, evt)
				return
			}

			select {
			case <-ctx.Done():
				return
			default:
			}
		}

		// If we reached here, the event channel closed without a done event.
		// Exit the loop.
		return

	nextRound:
	}

	// Exceeded max tool rounds.
	sendOut(ctx, out, ports.ChatEvent{
		Kind: "error",
		Err:  errors.New(suggestFallback),
	})
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
