package pending

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/gsaraiva2109/dietdaemon/core/types"
)

func TestSaveGetDelete(t *testing.T) {
	s := New(time.Hour)
	ctx := context.Background()
	pm := types.PendingMeal{UserID: "u1", RawText: "2 eggs", CreatedAt: time.Now()}

	if err := s.Save(ctx, pm); err != nil {
		t.Fatalf("Save error = %v", err)
	}
	got, err := s.Get(ctx, "u1")
	if err != nil {
		t.Fatalf("Get error = %v", err)
	}
	if got.RawText != "2 eggs" {
		t.Errorf("RawText = %q, want \"2 eggs\"", got.RawText)
	}

	if err := s.Delete(ctx, "u1"); err != nil {
		t.Fatalf("Delete error = %v", err)
	}
	if _, err := s.Get(ctx, "u1"); !errors.Is(err, types.ErrNotFound) {
		t.Errorf("Get after delete = %v, want ErrNotFound", err)
	}
}

func TestGetMissing(t *testing.T) {
	s := New(time.Hour)
	if _, err := s.Get(context.Background(), "nobody"); !errors.Is(err, types.ErrNotFound) {
		t.Errorf("Get missing = %v, want ErrNotFound", err)
	}
}

func TestExpiry(t *testing.T) {
	now := time.Now()
	s := New(10 * time.Minute)
	s.now = func() time.Time { return now }
	ctx := context.Background()

	_ = s.Save(ctx, types.PendingMeal{UserID: "u1", CreatedAt: now})

	// 5 min later: still live.
	s.now = func() time.Time { return now.Add(5 * time.Minute) }
	if _, err := s.Get(ctx, "u1"); err != nil {
		t.Fatalf("Get within TTL = %v, want live", err)
	}

	// 11 min later: expired.
	s.now = func() time.Time { return now.Add(11 * time.Minute) }
	if _, err := s.Get(ctx, "u1"); !errors.Is(err, types.ErrNotFound) {
		t.Errorf("Get past TTL = %v, want ErrNotFound", err)
	}
}
