// Package queue provides an in-memory hand-off between message ingest and the
// parse pipeline.
package queue

import (
	"context"
	"errors"
	"sync"
)

// ErrClosed is returned by Publish after the queue has been closed.
var ErrClosed = errors.New("queue closed")

// Memory is an in-process queue backed by a buffered channel.
//
// Shutdown ordering: cancel the producers' context before calling Close so no
// Publish is blocked on a full buffer when Close runs.
type Memory[T any] struct {
	ch     chan T
	mu     sync.RWMutex
	closed bool
}

// NewMemory creates an in-memory queue with the given buffer size.
func NewMemory[T any](buffer int) *Memory[T] {
	if buffer < 0 {
		buffer = 0
	}
	return &Memory[T]{ch: make(chan T, buffer)}
}

// Publish implements Queue. The read lock lets many producers enqueue
// concurrently while Close (write lock) waits for in-flight sends to drain.
func (m *Memory[T]) Publish(ctx context.Context, item T) error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.closed {
		return ErrClosed
	}
	select {
	case m.ch <- item:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Consume implements Queue.
func (m *Memory[T]) Consume() <-chan T { return m.ch }

// Close implements Queue. It is safe to call more than once.
func (m *Memory[T]) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return nil
	}
	m.closed = true
	close(m.ch)
	return nil
}
