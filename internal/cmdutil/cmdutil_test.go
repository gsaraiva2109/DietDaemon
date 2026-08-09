package cmdutil

import (
	"context"
	"path/filepath"
	"testing"
)

func TestOpenSQLiteStore(t *testing.T) {
	st, err := OpenSQLiteStore(filepath.Join(t.TempDir(), "dietdaemon.db"))
	if err != nil {
		t.Fatalf("OpenSQLiteStore: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
}

func TestSignalContextHonorsCanceledParent(t *testing.T) {
	parent, cancelParent := context.WithCancel(context.Background())
	cancelParent()
	ctx, stop := SignalContext(parent)
	t.Cleanup(stop)

	select {
	case <-ctx.Done():
	case <-t.Context().Done():
		t.Fatal("SignalContext did not inherit parent cancellation")
	}
}
