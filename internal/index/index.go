// Package index provides a brute-force cosine-similarity nearest-neighbour
// index over per-user food embedding vectors stored in SQLite. The personal
// food library is small (tens to low hundreds of entries), so O(N) per query
// is fine and avoids an external vector DB dependency.
//
// Vectors are stored as little-endian float32 BLOBs in the food_vectors table.
// Each user's vectors are loaded into memory lazily on first query and cached
// until an Upsert or Delete invalidates that user's cache.
package index

import (
	"context"
	"database/sql"
	"encoding/binary"
	"fmt"
	"math"
	"sync"

	"github.com/gsaraiva2109/dietdaemon/core/types"
)

// Neighbor is a single nearest-neighbor result.
type Neighbor struct {
	FoodID string
	Score  float64 // cosine similarity, 0..1
}

// Index is the embedding nearest-neighbour store.
type Index interface {
	Upsert(ctx context.Context, userID, foodID string, vec []float32) error
	Nearest(ctx context.Context, userID string, vec []float32, k int) ([]Neighbor, error)
	Delete(ctx context.Context, userID, foodID string) error
}

// Compile-time interface guard.
var _ Index = (*SQLIndex)(nil)

// SQLIndex implements Index backed by a SQLite food_vectors table.
type SQLIndex struct {
	db *sql.DB

	mu     sync.RWMutex
	caches map[string][]entry // userID -> vectors, invalidated on write
}

type entry struct {
	foodID string
	vec    []float32
}

// New returns a ready SQLIndex backed by db. The food_vectors table must
// already exist (applied via migration 003_food_vectors.sql).
func New(db *sql.DB) *SQLIndex {
	return &SQLIndex{
		db:     db,
		caches: make(map[string][]entry),
	}
}

// Upsert inserts or replaces the vector for (userID, foodID).
func (ix *SQLIndex) Upsert(ctx context.Context, userID, foodID string, vec []float32) error {
	blob := packF32LE(vec)
	const q = `
		INSERT OR REPLACE INTO food_vectors (user_id, food_id, dim, vec)
		VALUES (?, ?, ?, ?)
	`
	_, err := ix.db.ExecContext(ctx, q, userID, foodID, len(vec), blob)
	if err != nil {
		return fmt.Errorf("index: upsert: %w", err)
	}

	ix.invalidate(userID)
	return nil
}

// Nearest returns the k nearest neighbours by cosine similarity. When fewer
// than k vectors exist for the user, all are returned (sorted by score desc).
// Returns an empty slice (not an error) when the user has no vectors.
func (ix *SQLIndex) Nearest(ctx context.Context, userID string, vec []float32, k int) ([]Neighbor, error) {
	entries, err := ix.loadUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	if len(entries) == 0 {
		return nil, nil
	}

	neighbors := make([]Neighbor, 0, len(entries))
	for _, e := range entries {
		score := cosineSimilarity(vec, e.vec)
		neighbors = append(neighbors, Neighbor{FoodID: e.foodID, Score: score})
	}

	// Sort descending by score, keep top k.
	sortByScore(neighbors)
	if k > 0 && k < len(neighbors) {
		neighbors = neighbors[:k]
	}
	return neighbors, nil
}

// Delete removes the vector for (userID, foodID). Idempotent.
func (ix *SQLIndex) Delete(ctx context.Context, userID, foodID string) error {
	const q = `DELETE FROM food_vectors WHERE user_id = ? AND food_id = ?`
	_, err := ix.db.ExecContext(ctx, q, userID, foodID)
	if err != nil {
		return fmt.Errorf("index: delete: %w", err)
	}
	ix.invalidate(userID)
	return nil
}

// ---------------------------------------------------------------------------
// Cache
// ---------------------------------------------------------------------------

func (ix *SQLIndex) invalidate(userID string) {
	ix.mu.Lock()
	delete(ix.caches, userID)
	ix.mu.Unlock()
}

func (ix *SQLIndex) loadUser(ctx context.Context, userID string) ([]entry, error) {
	ix.mu.RLock()
	cached, ok := ix.caches[userID]
	ix.mu.RUnlock()
	if ok {
		return cached, nil
	}

	ix.mu.Lock()
	defer ix.mu.Unlock()

	// Double-check after acquiring write lock.
	if cached, ok = ix.caches[userID]; ok {
		return cached, nil
	}

	const q = `SELECT food_id, vec FROM food_vectors WHERE user_id = ?`
	rows, err := ix.db.QueryContext(ctx, q, userID)
	if err != nil {
		return nil, fmt.Errorf("index: load: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var entries []entry
	for rows.Next() {
		var foodID string
		var blob []byte
		if err := rows.Scan(&foodID, &blob); err != nil {
			return nil, fmt.Errorf("index: scan: %w", err)
		}
		vec, err := unpackF32LE(blob)
		if err != nil {
			return nil, fmt.Errorf("index: unpack: %w", err)
		}
		entries = append(entries, entry{foodID: foodID, vec: vec})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("index: rows: %w", err)
	}

	if entries == nil {
		entries = []entry{}
	}
	ix.caches[userID] = entries
	return entries, nil
}

// ---------------------------------------------------------------------------
// Cosine similarity
// ---------------------------------------------------------------------------

// cosineSimilarity returns the cosine similarity between two vectors. Returns
// 0 when either vector is zero-length.
func cosineSimilarity(a, b []float32) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}
	var dot, normA, normB float64
	for i := range a {
		da := float64(a[i])
		db := float64(b[i])
		dot += da * db
		normA += da * da
		normB += db * db
	}
	if normA == 0 || normB == 0 {
		return 0
	}
	return dot / math.Sqrt(normA*normB)
}

// ---------------------------------------------------------------------------
// Float32 <-> little-endian BLOB
// ---------------------------------------------------------------------------

func packF32LE(vec []float32) []byte {
	blob := make([]byte, len(vec)*4)
	for i, v := range vec {
		binary.LittleEndian.PutUint32(blob[i*4:], math.Float32bits(v))
	}
	return blob
}

func unpackF32LE(blob []byte) ([]float32, error) {
	if len(blob)%4 != 0 {
		return nil, types.ErrNoMatch // unexpected: reuse sentinel for "bad data"
	}
	vec := make([]float32, len(blob)/4)
	for i := range vec {
		bits := binary.LittleEndian.Uint32(blob[i*4:])
		vec[i] = math.Float32frombits(bits)
	}
	return vec, nil
}

// ---------------------------------------------------------------------------
// Top-k sort (simple insertion sort, N is small)
// ---------------------------------------------------------------------------

func sortByScore(nn []Neighbor) {
	for i := 1; i < len(nn); i++ {
		for j := i; j > 0 && nn[j].Score > nn[j-1].Score; j-- {
			nn[j], nn[j-1] = nn[j-1], nn[j]
		}
	}
}
