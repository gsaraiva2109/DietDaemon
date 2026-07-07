// Package assistant implements the tool-calling router for the AI chat assistant.
// It streams LLM responses token-by-token and executes DietDaemon slash-commands
// as tools when the model requests them, feeding results back into the conversation.
//
// The router is independently testable without HTTP — it works purely with the
// ports.ChatAdapter interface and a command registry.
package assistant

import (
	"context"
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
		tools = append(tools, ports.ToolSpec{Name: name, Description: desc})
		cmdMap[name] = c
	}
	return &Router{
		adapter:  adapter,
		commands: cmdMap,
		tools:    tools,
	}
}

// Run executes the tool-calling loop. It streams events to the returned channel.
// The caller reads events and writes SSE to the HTTP client.
func (r *Router) Run(ctx context.Context, userID, systemPrompt, userMessage string) <-chan ports.ChatEvent {
	ch := make(chan ports.ChatEvent, 32)
	go r.loop(ctx, ch, userID, systemPrompt, userMessage)
	return ch
}

func (r *Router) loop(ctx context.Context, out chan<- ports.ChatEvent, userID, systemPrompt, userMessage string) {
	defer close(out)

	messages := []ports.ChatMessage{
		{Role: "user", Content: userMessage},
	}

	for round := 0; round < maxToolRounds; round++ {
		req := ports.ChatRequest{
			System:   systemPrompt,
			Messages: messages,
			Tools:    r.tools,
		}

		events, err := r.adapter.StreamChat(ctx, req)
		if err != nil {
			out <- ports.ChatEvent{Kind: "error", Err: fmt.Errorf("assistant: stream: %w", err)}
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
				out <- evt // forward in real-time

			case "tool-call":
				toolCalls = append(toolCalls, *evt.ToolCall)
				out <- evt // forward to client

			case "done":
				// Append accumulated assistant text (if any) to history.
				if textBuf.Len() > 0 {
					messages = append(messages, ports.ChatMessage{
						Role:    "assistant",
						Content: textBuf.String(),
					})
				}

				// Execute any tool calls from this round.
				if len(toolCalls) > 0 {
					for _, tc := range toolCalls {
						reply := r.executeCommand(ctx, userID, tc)
						out <- ports.ChatEvent{
							Kind: "tool-result",
							ToolCall: &ports.ToolCallEvent{
								ID:   tc.ID,
								Name: tc.Name,
								Args: reply,
							},
						}
						messages = append(messages, ports.ChatMessage{
							Role:    "tool",
							Content: reply,
						})
					}
					// Continue to next round (tool results may prompt more text).
					goto nextRound
				}

				// No tools — conversation complete.
				out <- evt // forward done event
				return

			case "error":
				out <- evt
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
	out <- ports.ChatEvent{
		Kind: "error",
		Err:  fmt.Errorf(suggestFallback),
	}
}

// executeCommand looks up a command by name and calls its Handle method.
func (r *Router) executeCommand(ctx context.Context, userID string, tc ports.ToolCallEvent) string {
	cmd, ok := r.commands[tc.Name]
	if !ok {
		return fmt.Sprintf("unknown command: %s", tc.Name)
	}

	msg := types.InboundMessage{
		UserID:      userID,
		Text:        tc.Name + " " + tc.Args,
		ChannelMeta: map[string]string{"channel": "web"},
	}

	reply, err := cmd.Handle(ctx, msg, tc.Args)
	if err != nil {
		return fmt.Sprintf("error executing %s: %v", tc.Name, err)
	}
	return reply.Text
}
