package foodimport

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/gsaraiva2109/dietdaemon/core/ports"
	"github.com/gsaraiva2109/dietdaemon/core/types"
)

// fakeSource emits a fixed set of synthetic foods, or fails if err is set.
type fakeSource struct {
	name        string
	count       int
	fetchErr    error
	fetchCalls  int
	duringFetch func()
}

func (f *fakeSource) Name() string { return f.name }

func (f *fakeSource) FetchBulk(ctx context.Context, filter ports.BulkFilter, emit func(types.FoodMatch) error) error {
	f.fetchCalls++
	if f.fetchErr != nil {
		return f.fetchErr
	}
	if f.duringFetch != nil {
		duringFetch := f.duringFetch
		f.duringFetch = nil
		duringFetch()
	}
	for i := 0; i < f.count; i++ {
		if err := emit(types.FoodMatch{FoodID: fmt.Sprintf("%s-%d", f.name, i), Name: fmt.Sprintf("food %d", i)}); err != nil {
			return err
		}
	}
	return nil
}

// fakeStore records every batch passed to BulkUpsertFoods.
type fakeStore struct {
	calls        [][]types.FoodMatch
	fingerprints map[string]string
	setCalls     int
}

func (s *fakeStore) BulkUpsertFoods(ctx context.Context, foods []types.FoodMatch) error {
	cp := make([]types.FoodMatch, len(foods))
	copy(cp, foods)
	s.calls = append(s.calls, cp)
	return nil
}

func (s *fakeStore) GetFoodImportFingerprint(ctx context.Context, source string) (string, error) {
	fingerprint, ok := s.fingerprints[source]
	if !ok {
		return "", types.ErrNotFound
	}
	return fingerprint, nil
}

func (s *fakeStore) SetFoodImportFingerprint(ctx context.Context, source, fingerprint string) error {
	if s.fingerprints == nil {
		s.fingerprints = make(map[string]string)
	}
	s.fingerprints[source] = fingerprint
	s.setCalls++
	return nil
}

// fakeEmbedder records BackfillEmbeddings calls and returns canned counts.
type fakeEmbedder struct {
	calls    int
	embedded int
	failed   int
	err      error
	lastCtx  context.Context
}

func (e *fakeEmbedder) BackfillEmbeddings(ctx context.Context, progress func(done, total int)) (int, int, error) {
	e.calls++
	e.lastCtx = ctx
	if progress != nil {
		progress(e.embedded+e.failed, e.embedded+e.failed)
	}
	return e.embedded, e.failed, e.err
}

func TestRunOnce_EmbedderBackfillsAfterImport(t *testing.T) {
	src := &fakeSource{name: "taco", count: 2}
	store := &fakeStore{}
	emb := &fakeEmbedder{embedded: 2}
	r := New(store, []ports.BulkSource{src}, map[string]ports.BulkFilter{"taco": {}}, 0, slog.Default()).WithEmbedder(emb)

	r.RunOnce(context.Background())

	if emb.calls != 1 {
		t.Fatalf("BackfillEmbeddings calls = %d, want 1", emb.calls)
	}
}

func TestRunOnce_EmbedderRunsEvenWhenAllSourcesFail(t *testing.T) {
	failing := &fakeSource{name: "usda", fetchErr: errors.New("boom")}
	store := &fakeStore{}
	emb := &fakeEmbedder{}
	r := New(store, []ports.BulkSource{failing}, map[string]ports.BulkFilter{}, 0, slog.Default()).WithEmbedder(emb)

	r.RunOnce(context.Background())

	if emb.calls != 1 {
		t.Fatalf("BackfillEmbeddings calls = %d, want 1 (backfill should still run to pick up any previously-missed foods)", emb.calls)
	}
}

func TestRunOnce_EmbedderErrorDoesNotPanic(t *testing.T) {
	src := &fakeSource{name: "taco", count: 1}
	store := &fakeStore{}
	emb := &fakeEmbedder{err: errors.New("ollama unreachable")}
	r := New(store, []ports.BulkSource{src}, map[string]ports.BulkFilter{"taco": {}}, 0, slog.Default()).WithEmbedder(emb)

	r.RunOnce(context.Background()) // must not panic

	if emb.calls != 1 {
		t.Fatalf("BackfillEmbeddings calls = %d, want 1", emb.calls)
	}
}

func TestRunOnce_NoEmbedderIsNoOp(t *testing.T) {
	src := &fakeSource{name: "taco", count: 1}
	store := &fakeStore{}
	r := New(store, []ports.BulkSource{src}, map[string]ports.BulkFilter{"taco": {}}, 0, slog.Default())

	r.RunOnce(context.Background()) // must not panic with r.embedder == nil

	if len(store.calls) != 1 {
		t.Fatalf("BulkUpsertFoods calls = %d, want 1", len(store.calls))
	}
}

func TestRunOnce_BatchingAndFinalPartialBatch(t *testing.T) {
	src := &fakeSource{name: "usda", count: 1200} // 2 full batches of 500 + 1 partial of 200
	store := &fakeStore{}
	r := New(store, []ports.BulkSource{src}, map[string]ports.BulkFilter{"usda": {}}, 0, slog.Default())

	r.RunOnce(context.Background())

	if src.fetchCalls != 1 {
		t.Fatalf("FetchBulk calls = %d, want 1", src.fetchCalls)
	}
	if len(store.calls) != 3 {
		t.Fatalf("BulkUpsertFoods calls = %d, want 3", len(store.calls))
	}
	if len(store.calls[0]) != 500 || len(store.calls[1]) != 500 || len(store.calls[2]) != 200 {
		t.Fatalf("batch sizes = %d, %d, %d; want 500, 500, 200",
			len(store.calls[0]), len(store.calls[1]), len(store.calls[2]))
	}

	total := 0
	for _, batch := range store.calls {
		total += len(batch)
	}
	if total != 1200 {
		t.Fatalf("total rows written = %d, want 1200", total)
	}
}

func TestRunOnce_AllSourcesRunEvenIfOneFails(t *testing.T) {
	failing := &fakeSource{name: "usda", fetchErr: errors.New("boom")}
	ok := &fakeSource{name: "taco", count: 3}
	store := &fakeStore{}
	r := New(store, []ports.BulkSource{failing, ok}, map[string]ports.BulkFilter{}, 0, slog.Default())

	r.RunOnce(context.Background())

	if failing.fetchCalls != 1 {
		t.Fatalf("failing source FetchBulk calls = %d, want 1", failing.fetchCalls)
	}
	if ok.fetchCalls != 1 {
		t.Fatalf("ok source FetchBulk calls = %d, want 1", ok.fetchCalls)
	}
	if len(store.calls) != 1 || len(store.calls[0]) != 3 {
		t.Fatalf("store.calls = %+v, want one batch of 3", store.calls)
	}
}

func TestRunOnce_LocalFileSkipsUnchangedDataset(t *testing.T) {
	for _, source := range []string{"usda", "openfoodfacts", "taco"} {
		t.Run(source, func(t *testing.T) {
			path := writeDataset(t, "foods.json", "one")
			src := &fakeSource{name: source, count: 2}
			store := &fakeStore{}
			r := NewWithLocalPaths(store, []ports.BulkSource{src}, map[string]ports.BulkFilter{source: {}}, 0, slog.Default(), map[string]string{source: path}, nil)

			r.RunOnce(t.Context())
			r.RunOnce(t.Context())

			if src.fetchCalls != 1 || len(store.calls) != 1 || store.setCalls != 1 {
				t.Fatalf("fetches=%d upserts=%d fingerprint writes=%d; want 1, 1, 1", src.fetchCalls, len(store.calls), store.setCalls)
			}
		})
	}
}

func TestRunOnce_LocalFileOrFilterChangeImportsAgain(t *testing.T) {
	path := writeDataset(t, "foods.json", "one")
	src := &fakeSource{name: "usda", count: 1}
	store := &fakeStore{}
	r := NewWithLocalPaths(store, []ports.BulkSource{src}, map[string]ports.BulkFilter{"usda": {}}, 0, slog.Default(), map[string]string{"usda": path}, nil)
	r.RunOnce(t.Context())
	replaceDataset(t, path, "two")
	r.RunOnce(t.Context())
	r.filters["usda"] = ports.BulkFilter{MaxRows: 1}
	r.RunOnce(t.Context())
	if src.fetchCalls != 3 {
		t.Fatalf("FetchBulk calls = %d, want 3", src.fetchCalls)
	}
}

func TestRunOnce_LocalFileFailureAndMidImportChangeDoNotSaveFingerprint(t *testing.T) {
	path := writeDataset(t, "foods.json", "one")
	store := &fakeStore{}
	failing := &fakeSource{name: "usda", fetchErr: errors.New("boom")}
	r := NewWithLocalPaths(store, []ports.BulkSource{failing}, map[string]ports.BulkFilter{"usda": {}}, 0, slog.Default(), map[string]string{"usda": path}, nil)
	r.RunOnce(t.Context())
	if store.setCalls != 0 {
		t.Fatalf("fingerprint writes after failure = %d, want 0", store.setCalls)
	}

	changed := &fakeSource{name: "usda", count: 1, duringFetch: func() { replaceDataset(t, path, "two") }}
	r.sources[0] = changed
	r.RunOnce(t.Context())
	if store.setCalls != 0 {
		t.Fatalf("fingerprint writes after change during import = %d, want 0", store.setCalls)
	}
	r.RunOnce(t.Context())
	if changed.fetchCalls != 2 || store.setCalls != 1 {
		t.Fatalf("retry fetches=%d fingerprint writes=%d; want 2, 1", changed.fetchCalls, store.setCalls)
	}
}

func TestRunOnce_APISourceImportsEveryTime(t *testing.T) {
	src := &fakeSource{name: "openfoodfacts", count: 1}
	store := &fakeStore{}
	r := New(store, []ports.BulkSource{src}, map[string]ports.BulkFilter{"openfoodfacts": {}}, 0, slog.Default())
	r.RunOnce(t.Context())
	r.RunOnce(t.Context())
	if src.fetchCalls != 2 {
		t.Fatalf("FetchBulk calls = %d, want 2", src.fetchCalls)
	}
}

func TestRunOnce_LocalSourceRefreshesBeforeFirstAndChangedImport(t *testing.T) {
	path := writeDataset(t, "taco.csv", "one")
	store := &fakeStore{}
	initial := &fakeSource{name: "taco"}
	refreshes := 0
	r := NewWithLocalPaths(store, []ports.BulkSource{initial}, map[string]ports.BulkFilter{"taco": {}}, 0, slog.Default(), map[string]string{"taco": path}, map[string]SourceFactory{
		"taco": func() (ports.BulkSource, error) {
			refreshes++
			return &fakeSource{name: "taco", count: 1}, nil
		},
	})
	r.RunOnce(t.Context())
	r.RunOnce(t.Context())
	replaceDataset(t, path, "two")
	r.RunOnce(t.Context())
	if refreshes != 2 {
		t.Fatalf("source refreshes = %d, want 2", refreshes)
	}
}

func writeDataset(t *testing.T, name, contents string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func replaceDataset(t *testing.T, path, contents string) {
	t.Helper()
	replacement := path + ".next"
	if err := os.WriteFile(replacement, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(replacement, path); err != nil {
		t.Fatal(err)
	}
}
