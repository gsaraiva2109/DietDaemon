package ports

import "context"

type modelOverrideKey struct{}

// WithModelOverride injects a per-request ModelAdapter override into ctx.
// Downstream callers (parser, suggest engine) check for an override before
// falling back to their boot-time adapter.
func WithModelOverride(ctx context.Context, m ModelAdapter) context.Context {
	return context.WithValue(ctx, modelOverrideKey{}, m)
}

// ModelOverrideFromContext returns the override adapter if one was injected
// via WithModelOverride.
func ModelOverrideFromContext(ctx context.Context) (ModelAdapter, bool) {
	m, ok := ctx.Value(modelOverrideKey{}).(ModelAdapter)
	return m, ok
}
