// Package commands provides a command registry that dispatches inbound messages
// to registered Command handlers. Each handler implements the ports.Command
// interface and lives in its own file within this package.
package commands

import (
	"context"
	"fmt"
	"strings"

	"github.com/gsaraiva2109/dietdaemon/core/ports"
	"github.com/gsaraiva2109/dietdaemon/core/types"
)

// Registry holds all registered commands and dispatches messages to them.
type Registry struct {
	commands map[string]ports.Command // name -> command
	order    []string                 // registration order for /help
}

// NewRegistry creates an empty command registry.
func NewRegistry() *Registry {
	return &Registry{commands: make(map[string]ports.Command)}
}

// Register adds a command. Returns an error if the name or any alias is already
// taken.
func (r *Registry) Register(c ports.Command) error {
	if _, ok := r.commands[c.Name()]; ok {
		return fmt.Errorf("commands: duplicate name %q", c.Name())
	}
	for _, a := range c.Aliases() {
		if _, ok := r.commands[a]; ok {
			return fmt.Errorf("commands: duplicate alias %q", a)
		}
	}
	r.commands[c.Name()] = c
	r.order = append(r.order, c.Name())
	for _, a := range c.Aliases() {
		r.commands[a] = c
	}
	return nil
}

// Dispatch finds a command matching the text and calls its Handle method.
// The text should be the full message (e.g. "/target kcal=2000 protein=180").
// Returns false when no command matches. The returned error wraps
// infrastructure errors returned by the command handler.
func (r *Registry) Dispatch(ctx context.Context, msg types.InboundMessage, text string) (types.Reply, bool, error) {
	text = strings.TrimSpace(text)
	if !strings.HasPrefix(text, "/") {
		return types.Reply{}, false, nil
	}

	// Split into command name and args.
	parts := strings.SplitN(text, " ", 2)
	name := strings.ToLower(parts[0]) // normalize to lowercase
	args := ""
	if len(parts) > 1 {
		args = strings.TrimSpace(parts[1])
	}

	cmd, ok := r.commands[name]
	if !ok {
		return types.Reply{}, false, nil
	}

	reply, err := cmd.Handle(ctx, msg, args)
	if err != nil {
		return types.Reply{}, false, fmt.Errorf("commands: %s: %w", name, err)
	}
	return reply, true, nil
}

// List returns all registered commands (by primary name) in registration order.
func (r *Registry) List() []ports.Command {
	seen := make(map[string]bool)
	var out []ports.Command
	for _, name := range r.order {
		cmd := r.commands[name]
		if !seen[cmd.Name()] {
			seen[cmd.Name()] = true
			out = append(out, cmd)
		}
	}
	return out
}
