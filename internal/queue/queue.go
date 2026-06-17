// Package queue provides the decoupling boundary between producers (message
// ingest) and consumers (the parse pipeline). The in-memory implementation here
// is the default for the modular monolith; a durable bus (Redis Streams, NATS)
// can later replace it behind the same Queue interface without touching callers.
package queue

import (
	"context"
	"errors"
	"sync"
)

// ErrClosed is returned by Publish after the queue has been closed.
var ErrClosed = errors.New("queue closed")

// Queue is a typed, ordered hand-off from producers to consumers.
type Queue[T any] interface {
	// Publish enqueues an item, blocking while the buffer is full until space
	// frees up or ctx is cancelled. Returns ErrClosed if the queue is closed.
	Publish(ctx context.Context, item T) error
	// Consume returns the channel consumers range over. The channel is closed
	// when the queue is closed, ending the range.
	Consume() <-chan T
	// Close stops the queue; subsequent Publish calls return ErrClosed.
	Close() error
}

// Memory is an in-process Queue backed by a buffered channel.
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
