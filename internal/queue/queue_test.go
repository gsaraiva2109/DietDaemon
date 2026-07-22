package queue

import (
	"context"
	"errors"
	"testing"
)

func TestPublishConsume(t *testing.T) {
	q := NewMemory[int](4)
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		if err := q.Publish(ctx, i); err != nil {
			t.Fatalf("Publish(%d) error = %v", i, err)
		}
	}
	if err := q.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	var got []int
	for v := range q.Consume() {
		got = append(got, v)
	}
	if len(got) != 3 || got[0] != 0 || got[2] != 2 {
		t.Fatalf("Consume drained %v, want [0 1 2]", got)
	}
}

func TestPublishAfterCloseFails(t *testing.T) {
	q := NewMemory[string](1)
	_ = q.Close()
	if err := q.Publish(context.Background(), "x"); !errors.Is(err, ErrClosed) {
		t.Fatalf("Publish after Close = %v, want ErrClosed", err)
	}
}

func TestPublishRespectsContext(t *testing.T) {
	q := NewMemory[int](1)
	ctx := context.Background()
	if err := q.Publish(ctx, 1); err != nil { // fills the buffer
		t.Fatalf("Publish error = %v", err)
	}

	cctx, cancel := context.WithCancel(ctx)
	cancel() // cancel before the blocked publish
	if err := q.Publish(cctx, 2); !errors.Is(err, context.Canceled) {
		t.Fatalf("Publish on full buffer with cancelled ctx = %v, want context.Canceled", err)
	}
}
