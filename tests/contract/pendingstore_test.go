// Package contract contains interface-level tests that verify every
// implementation of a port behaves identically. Each test is parameterised by
// a factory so it can be run against every conforming adapter.
package contract

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/gsaraiva2109/dietdaemon/core/ports"
	"github.com/gsaraiva2109/dietdaemon/core/types"
	"github.com/gsaraiva2109/dietdaemon/internal/pending"
	"github.com/gsaraiva2109/dietdaemon/internal/store"
)

// pendingStoreFactory creates a ready PendingStore for testing.
type pendingStoreFactory func(t *testing.T) (ports.PendingStore, func())

func pendingInMemory(t *testing.T) (ports.PendingStore, func()) {
	t.Helper()
	s := pending.New(time.Hour)
	return s, func() {}
}

func pendingSQLite(t *testing.T) (ports.PendingStore, func()) {
	t.Helper()
	f, err := os.CreateTemp("", "dietdaemon-contract-*.db")
	if err != nil {
		t.Fatalf("create temp file: %v", err)
	}
	path := f.Name()
	f.Close()
	os.Remove(path)

	s, err := store.New(path)
	if err != nil {
		t.Fatalf("New(%q): %v", path, err)
	}
	return s, func() {
		s.Close()
		os.Remove(path)
	}
}

var pendingStores = map[string]pendingStoreFactory{
	"in-memory": pendingInMemory,
	"sqlite":    pendingSQLite,
}

// TestPendingStoreContract verifies that every PendingStore implementation
// obeys the same Save → Get → Delete → ErrNotFound lifecycle.
func TestPendingStoreContract(t *testing.T) {
	for name, factory := range pendingStores {
		t.Run(name, func(t *testing.T) {
			s, cleanup := factory(t)
			defer cleanup()

			ctx := context.Background()
			pm := types.PendingMeal{
				UserID:      "u1",
				At:          time.Date(2026, 6, 17, 12, 0, 0, 0, time.UTC),
				RawText:     "200g frango, 2 ovos",
				Confidence:  0.9,
				ParserTier:  types.TierDeterministic,
				ChannelMeta: map[string]string{"chat_id": "42"},
				Resolved: []types.ResolvedItem{
					{
						Parsed: types.ParsedItem{RawPhrase: "frango", Quantity: 200, Unit: "g", NormalizedGrams: 200},
						Match:  types.FoodMatch{FoodID: "f1", Name: "Frango", Source: "taco"},
					},
				},
				Pending: []types.ResolvedItem{
					{
						Parsed: types.ParsedItem{RawPhrase: "ovos", Quantity: 2, Unit: "un", NormalizedGrams: 0},
					},
				},
				CreatedAt: time.Now().UTC(),
			}

			// Get before save → ErrNotFound.
			if _, err := s.Get(ctx, "u1"); err != types.ErrNotFound {
				t.Errorf("Get before Save: expected ErrNotFound, got %v", err)
			}

			// Save.
			if err := s.Save(ctx, pm); err != nil {
				t.Fatalf("Save: %v", err)
			}

			// Get after save.
			got, err := s.Get(ctx, "u1")
			if err != nil {
				t.Fatalf("Get after Save: %v", err)
			}
			if got.RawText != pm.RawText {
				t.Errorf("RawText = %q, want %q", got.RawText, pm.RawText)
			}
			if got.Confidence != pm.Confidence {
				t.Errorf("Confidence = %f, want %f", got.Confidence, pm.Confidence)
			}
			if len(got.Resolved) != 1 {
				t.Errorf("len(Resolved) = %d, want 1", len(got.Resolved))
			}
			if len(got.Pending) != 1 {
				t.Errorf("len(Pending) = %d, want 1", len(got.Pending))
			}
			if got.ChannelMeta["chat_id"] != "42" {
				t.Errorf("ChannelMeta[chat_id] = %q, want %q", got.ChannelMeta["chat_id"], "42")
			}

			// Replace semantics.
			pm2 := pm
			pm2.RawText = "updated"
			if err := s.Save(ctx, pm2); err != nil {
				t.Fatalf("Save (replace): %v", err)
			}
			got, _ = s.Get(ctx, "u1")
			if got.RawText != "updated" {
				t.Errorf("after replace: RawText = %q, want %q", got.RawText, "updated")
			}

			// Delete.
			if err := s.Delete(ctx, "u1"); err != nil {
				t.Fatalf("Delete: %v", err)
			}

			// Get after delete → ErrNotFound.
			if _, err := s.Get(ctx, "u1"); err != types.ErrNotFound {
				t.Errorf("Get after Delete: expected ErrNotFound, got %v", err)
			}

			// Delete is idempotent.
			if err := s.Delete(ctx, "u1"); err != nil {
				t.Errorf("Delete (idempotent): %v", err)
			}
		})
	}
}
