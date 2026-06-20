package pendingstore

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	_ "modernc.org/sqlite"

	"github.com/gsaraiva2109/dietdaemon/core/types"
)

// setupDB opens an in-memory SQLite database and creates the pending_state
// table that matches migration 004.
func setupDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open in-memory db: %v", err)
	}
	const q = `
		CREATE TABLE pending_state (
			user_id    TEXT PRIMARY KEY,
			created_at INTEGER NOT NULL,
			payload    BLOB NOT NULL
		)
	`
	if _, err := db.Exec(q); err != nil {
		_ = db.Close()
		t.Fatalf("create table: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func TestSaveGetDelete(t *testing.T) {
	db := setupDB(t)
	s := New(db, time.Hour)
	ctx := context.Background()

	pm := types.PendingMeal{
		UserID:     "u1",
		RawText:    "2 eggs",
		CreatedAt:  time.Now(),
		Confidence: 0.85,
		ParserTier: types.TierDeterministic,
		ChannelMeta: map[string]string{
			"chat_id": "42",
		},
		Resolved: []types.ResolvedItem{
			{
				Parsed: types.ParsedItem{RawPhrase: "egg", Quantity: 2, Unit: "unit", NormalizedGrams: 100},
				Match:  types.FoodMatch{FoodID: "egg", Name: "Egg", Source: "food_library", Per100g: types.Macros{Calories: 155, Protein: 13}},
				Macros: types.Macros{Calories: 155, Protein: 13},
			},
		},
		Pending: []types.ResolvedItem{
			{
				Parsed: types.ParsedItem{RawPhrase: "unknown", Quantity: 1, Unit: "slice"},
			},
		},
	}

	if err := s.Save(ctx, pm); err != nil {
		t.Fatalf("Save error = %v", err)
	}

	got, err := s.Get(ctx, "u1")
	if err != nil {
		t.Fatalf("Get error = %v", err)
	}
	if got.RawText != "2 eggs" {
		t.Errorf("RawText = %q, want %q", got.RawText, "2 eggs")
	}
	if got.Confidence != 0.85 {
		t.Errorf("Confidence = %v, want 0.85", got.Confidence)
	}
	if got.ParserTier != types.TierDeterministic {
		t.Errorf("ParserTier = %v, want TierDeterministic", got.ParserTier)
	}
	if got.ChannelMeta["chat_id"] != "42" {
		t.Errorf("ChannelMeta[chat_id] = %q, want %q", got.ChannelMeta["chat_id"], "42")
	}
	if len(got.Resolved) != 1 || got.Resolved[0].Match.FoodID != "egg" {
		t.Errorf("Resolved = %+v, want 1 item with FoodID=egg", got.Resolved)
	}
	if len(got.Pending) != 1 || got.Pending[0].Parsed.RawPhrase != "unknown" {
		t.Errorf("Pending = %+v, want 1 item with RawPhrase=unknown", got.Pending)
	}

	if err := s.Delete(ctx, "u1"); err != nil {
		t.Fatalf("Delete error = %v", err)
	}
	if _, err := s.Get(ctx, "u1"); !errors.Is(err, types.ErrNotFound) {
		t.Errorf("Get after delete = %v, want ErrNotFound", err)
	}
}

func TestGetMissing(t *testing.T) {
	db := setupDB(t)
	s := New(db, time.Hour)

	if _, err := s.Get(context.Background(), "nobody"); !errors.Is(err, types.ErrNotFound) {
		t.Errorf("Get missing = %v, want ErrNotFound", err)
	}
}

func TestExpiry(t *testing.T) {
	db := setupDB(t)
	now := time.Now()

	s := New(db, 10*time.Minute)
	s.now = func() time.Time { return now }

	ctx := context.Background()
	_ = s.Save(ctx, types.PendingMeal{UserID: "u1", CreatedAt: now})

	// Within TTL: still live.
	s.now = func() time.Time { return now.Add(5 * time.Minute) }
	if _, err := s.Get(ctx, "u1"); err != nil {
		t.Fatalf("Get within TTL = %v, want live", err)
	}

	// Past TTL: expired, lazy-deleted.
	s.now = func() time.Time { return now.Add(11 * time.Minute) }
	if _, err := s.Get(ctx, "u1"); !errors.Is(err, types.ErrNotFound) {
		t.Errorf("Get past TTL = %v, want ErrNotFound", err)
	}

	// Row should be gone.
	var count int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM pending_state WHERE user_id = ?`, "u1").Scan(&count); err != nil {
		t.Fatalf("count query: %v", err)
	}
	if count != 0 {
		t.Errorf("row still exists after lazy delete, got count=%d", count)
	}
}

func TestOverwrite(t *testing.T) {
	db := setupDB(t)
	s := New(db, time.Hour)
	ctx := context.Background()

	first := types.PendingMeal{UserID: "u1", RawText: "first", CreatedAt: time.Now()}
	second := types.PendingMeal{UserID: "u1", RawText: "second", CreatedAt: time.Now()}

	_ = s.Save(ctx, first)
	_ = s.Save(ctx, second)

	got, err := s.Get(ctx, "u1")
	if err != nil {
		t.Fatalf("Get error = %v", err)
	}
	if got.RawText != "second" {
		t.Errorf("RawText = %q, want %q (second overwrites first)", got.RawText, "second")
	}

	// Only one row.
	var count int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM pending_state WHERE user_id = ?`, "u1").Scan(&count); err != nil {
		t.Fatalf("count query: %v", err)
	}
	if count != 1 {
		t.Errorf("row count = %d, want 1", count)
	}
}

func TestDeleteMissing(t *testing.T) {
	db := setupDB(t)
	s := New(db, time.Hour)

	// Deleting a missing user should not error.
	if err := s.Delete(context.Background(), "nobody"); err != nil {
		t.Errorf("Delete missing = %v, want nil", err)
	}
}
