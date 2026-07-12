package foodimport

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"testing"

	"github.com/gsaraiva2109/dietdaemon/core/ports"
	"github.com/gsaraiva2109/dietdaemon/core/types"
)

// fakeSource emits a fixed set of synthetic foods, or fails if err is set.
type fakeSource struct {
	name       string
	count      int
	fetchErr   error
	fetchCalls int
}

func (f *fakeSource) Name() string { return f.name }

func (f *fakeSource) FetchBulk(ctx context.Context, filter ports.BulkFilter, emit func(types.FoodMatch) error) error {
	f.fetchCalls++
	if f.fetchErr != nil {
		return f.fetchErr
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
	calls [][]types.FoodMatch
}

func (s *fakeStore) BulkUpsertFoods(ctx context.Context, foods []types.FoodMatch) error {
	cp := make([]types.FoodMatch, len(foods))
	copy(cp, foods)
	s.calls = append(s.calls, cp)
	return nil
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
