package index

import (
	"context"
	"database/sql"
	"math"
	"testing"

	_ "modernc.org/sqlite"
)

// openTestDB opens an in-memory SQLite database with the food_vectors schema.
func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	const q = `
		CREATE TABLE IF NOT EXISTS food_vectors (
			food_id TEXT PRIMARY KEY,
			dim     INTEGER NOT NULL,
			vec     BLOB NOT NULL
		);
	`
	if _, err := db.Exec(q); err != nil {
		t.Fatalf("create table: %v", err)
	}
	return db
}

// ---------------------------------------------------------------------------
// Cosine correctness
// ---------------------------------------------------------------------------

func TestCosineSimilarity(t *testing.T) {
	// Identical vectors → score 1.
	a := []float32{1, 2, 3}
	if got := cosineSimilarity(a, a); math.Abs(got-1.0) > 1e-6 {
		t.Errorf("cos(a,a) = %v, want 1.0", got)
	}

	// Orthogonal → score 0.
	b := []float32{1, 0}
	c := []float32{0, 1}
	if got := cosineSimilarity(b, c); math.Abs(got-0.0) > 1e-6 {
		t.Errorf("cos([1,0],[0,1]) = %v, want 0.0", got)
	}

	// Different lengths → 0.
	if got := cosineSimilarity([]float32{1}, []float32{1, 2}); got != 0 {
		t.Errorf("cos diff-len = %v, want 0", got)
	}

	// Empty → 0.
	if got := cosineSimilarity(nil, nil); got != 0 {
		t.Errorf("cos(nil,nil) = %v, want 0", got)
	}

	// Known value: cos([1,2], [3,4]) = (3+8)/sqrt(5*25) = 11/sqrt(125) ≈ 0.98387
	x := []float32{1, 2}
	y := []float32{3, 4}
	want := 11.0 / math.Sqrt(125.0)
	if got := cosineSimilarity(x, y); math.Abs(got-want) > 1e-6 {
		t.Errorf("cos = %v, want %v", got, want)
	}
}

// ---------------------------------------------------------------------------
// Float32 pack/unpack round-trip
// ---------------------------------------------------------------------------

func TestPackUnpackF32LE(t *testing.T) {
	orig := []float32{0.0, 1.0, -1.0, 0.5, -0.25, math.MaxFloat32, math.SmallestNonzeroFloat32}
	blob := packF32LE(orig)
	got, err := unpackF32LE(blob)
	if err != nil {
		t.Fatalf("unpack: %v", err)
	}
	if len(got) != len(orig) {
		t.Fatalf("len = %d, want %d", len(got), len(orig))
	}
	for i := range orig {
		if math.Abs(float64(got[i]-orig[i])) > 1e-12 {
			t.Errorf("idx %d: got %v, want %v", i, got[i], orig[i])
		}
	}
}

func TestUnpackBadBlob(t *testing.T) {
	_, err := unpackF32LE([]byte{1, 2, 3}) // not multiple of 4
	if err == nil {
		t.Error("expected error on misaligned blob")
	}
}

// ---------------------------------------------------------------------------
// Upsert + Nearest + Delete
// ---------------------------------------------------------------------------

func TestUpsertAndNearest(t *testing.T) {
	db := openTestDB(t)
	ix := New(db)
	ctx := context.Background()

	// Insert two vectors.
	requireNoErr(t, ix.Upsert(ctx, "food_a", []float32{1, 0, 0}))
	requireNoErr(t, ix.Upsert(ctx, "food_b", []float32{0, 1, 0}))

	// Query with [1, 0.1, 0] — closer to food_a.
	nn, err := ix.Nearest(ctx, []float32{1, 0.1, 0}, 2)
	requireNoErr(t, err)
	if len(nn) != 2 {
		t.Fatalf("got %d neighbors, want 2", len(nn))
	}
	if nn[0].FoodID != "food_a" {
		t.Errorf("top match = %q, want food_a", nn[0].FoodID)
	}
	if nn[0].Score <= nn[1].Score {
		t.Errorf("scores not descending: %v then %v", nn[0].Score, nn[1].Score)
	}
}

func TestNearestWithKLessThanN(t *testing.T) {
	db := openTestDB(t)
	ix := New(db)
	ctx := context.Background()

	requireNoErr(t, ix.Upsert(ctx, "a", []float32{1, 0}))
	requireNoErr(t, ix.Upsert(ctx, "b", []float32{0, 1}))
	requireNoErr(t, ix.Upsert(ctx, "c", []float32{0.5, 0.5}))

	nn, err := ix.Nearest(ctx, []float32{1, 0}, 1)
	requireNoErr(t, err)
	if len(nn) != 1 {
		t.Fatalf("got %d neighbors, want 1", len(nn))
	}
	if nn[0].FoodID != "a" {
		t.Errorf("top match = %q, want a", nn[0].FoodID)
	}
}

func TestNearestEmptyUser(t *testing.T) {
	db := openTestDB(t)
	ix := New(db)
	ctx := context.Background()

	nn, err := ix.Nearest(ctx, []float32{1, 0}, 5)
	requireNoErr(t, err)
	if len(nn) != 0 {
		t.Errorf("got %d neighbors, want 0", len(nn))
	}
}

func TestThresholdCutoff(t *testing.T) {
	db := openTestDB(t)
	ix := New(db)
	ctx := context.Background()

	// food_a is close, food_b is orthogonal.
	requireNoErr(t, ix.Upsert(ctx, "food_a", []float32{1, 0}))
	requireNoErr(t, ix.Upsert(ctx, "food_b", []float32{0, 1}))

	// Query near food_a: top score should be high, second much lower.
	nn, err := ix.Nearest(ctx, []float32{0.99, 0.01}, 2)
	requireNoErr(t, err)

	if nn[0].Score < 0.99 {
		t.Errorf("top score = %v, want > 0.99 for near-identical vector", nn[0].Score)
	}
	if nn[1].Score > 0.1 {
		t.Errorf("second score = %v, want < 0.1 for orthogonal vector", nn[1].Score)
	}
}

func TestDelete(t *testing.T) {
	db := openTestDB(t)
	ix := New(db)
	ctx := context.Background()

	requireNoErr(t, ix.Upsert(ctx, "food_a", []float32{1, 0}))
	requireNoErr(t, ix.Upsert(ctx, "food_b", []float32{0, 1}))

	// Delete food_a.
	requireNoErr(t, ix.Delete(ctx, "food_a"))

	nn, err := ix.Nearest(ctx, []float32{1, 0}, 2)
	requireNoErr(t, err)
	if len(nn) != 1 {
		t.Fatalf("got %d neighbors after delete, want 1", len(nn))
	}
	if nn[0].FoodID != "food_b" {
		t.Errorf("remaining = %q, want food_b", nn[0].FoodID)
	}
}

func TestUpsertReplaces(t *testing.T) {
	db := openTestDB(t)
	ix := New(db)
	ctx := context.Background()

	requireNoErr(t, ix.Upsert(ctx, "a", []float32{1, 0}))
	// Replace with different vector.
	requireNoErr(t, ix.Upsert(ctx, "a", []float32{0, 1}))

	nn, err := ix.Nearest(ctx, []float32{0, 1}, 1)
	requireNoErr(t, err)
	if nn[0].FoodID != "a" {
		t.Errorf("top = %q, want a", nn[0].FoodID)
	}
	// Should now be near-identical to [0, 1].
	if nn[0].Score < 0.999 {
		t.Errorf("score = %v, want ~1.0 after replace", nn[0].Score)
	}
}

func TestCacheInvalidation(t *testing.T) {
	db := openTestDB(t)
	ix := New(db)
	ctx := context.Background()

	// Load cache.
	requireNoErr(t, ix.Upsert(ctx, "a", []float32{1, 0}))
	_, err := ix.Nearest(ctx, []float32{1, 0}, 1)
	requireNoErr(t, err)

	// Verify cache populated.
	ix.mu.RLock()
	cached := ix.cache != nil
	ix.mu.RUnlock()
	if !cached {
		t.Fatal("expected cache to be populated")
	}

	// Upsert should invalidate.
	requireNoErr(t, ix.Upsert(ctx, "b", []float32{0, 1}))
	ix.mu.RLock()
	cached = ix.cache != nil
	ix.mu.RUnlock()
	if cached {
		t.Error("cache should be invalidated after Upsert")
	}

	// Delete should invalidate.
	_, _ = ix.Nearest(ctx, []float32{1, 0}, 1) // reload
	requireNoErr(t, ix.Delete(ctx, "b"))
	ix.mu.RLock()
	cached = ix.cache != nil
	ix.mu.RUnlock()
	if cached {
		t.Error("cache should be invalidated after Delete")
	}
}

func requireNoErr(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
