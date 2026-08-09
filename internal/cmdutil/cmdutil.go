// Package cmdutil holds small helpers shared by command binaries.
package cmdutil

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/gsaraiva2109/dietdaemon/internal/store"
)

// OpenSQLiteStore opens the SQLite store used by one-shot command binaries.
func OpenSQLiteStore(path string) (*store.Store, error) {
	return store.New("sqlite", path, store.SQLiteDialect(), nil)
}

// SignalContext is cancelled when the process receives SIGINT or SIGTERM.
func SignalContext(parent context.Context) (context.Context, context.CancelFunc) {
	return signal.NotifyContext(parent, os.Interrupt, syscall.SIGTERM)
}
