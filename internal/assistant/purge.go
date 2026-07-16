package assistant

import (
	"context"
	"log/slog"
	"time"
)

// PurgeStore is the subset of store methods the purge job needs.
type PurgeStore interface {
	PurgeDeletedChatSessions(ctx context.Context, olderThan time.Time) (int, error)
	PurgeLoginAttempts(ctx context.Context, olderThan time.Time) (int, error)
	PurgeAuthAuditEvents(ctx context.Context, olderThan time.Time) (int, error)
}

const (
	loginAttemptRetention = 24 * time.Hour
	authAuditRetention    = 90 * 24 * time.Hour
)

// PurgeRunner periodically hard-deletes chat sessions that have been
// soft-deleted for longer than the 30-day retention window.
type PurgeRunner struct {
	store    PurgeStore
	interval time.Duration
}

// NewPurgeRunner creates a PurgeRunner with the given store and tick interval.
func NewPurgeRunner(s PurgeStore, interval time.Duration) *PurgeRunner {
	return &PurgeRunner{store: s, interval: interval}
}

// Run ticks until ctx is cancelled, purging expired soft-deleted sessions.
func (r *PurgeRunner) Run(ctx context.Context) {
	t := time.NewTicker(r.interval)
	defer t.Stop()
	for {
		select {
		case <-t.C:
			now := time.Now()
			n, err := r.store.PurgeDeletedChatSessions(ctx, now.AddDate(0, 0, -30))
			if err != nil {
				slog.Error("purge deleted chat sessions", "err", err)
				continue
			}
			if n > 0 {
				slog.Info("purged deleted chat sessions", "count", n)
			}
			if n, err := r.store.PurgeLoginAttempts(ctx, now.Add(-loginAttemptRetention)); err != nil {
				slog.Error("purge login attempts", "err", err)
			} else if n > 0 {
				slog.Info("purged login attempts", "count", n)
			}
			if n, err := r.store.PurgeAuthAuditEvents(ctx, now.Add(-authAuditRetention)); err != nil {
				slog.Error("purge auth audit events", "err", err)
			} else if n > 0 {
				slog.Info("purged auth audit events", "count", n)
			}
		case <-ctx.Done():
			return
		}
	}
}
