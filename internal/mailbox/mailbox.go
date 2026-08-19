// Package mailbox provides latest-value, coalescing delivery.
package mailbox

import (
	"context"
	"sync"
)

// Mailbox delivers the latest value pending for one receiver.
type Mailbox[T any] interface {
	Send(value T)
	Receive(ctx context.Context) (T, error)
}

type mailbox[T any] struct {
	mu      sync.Mutex
	ready   chan struct{}
	value   T
	pending bool
}

// New creates a mailbox containing initialValue.
func New[T any](initialValue T) Mailbox[T] {
	mailbox := &mailbox[T]{
		ready:   make(chan struct{}, 1),
		value:   initialValue,
		pending: true,
	}
	mailbox.ready <- struct{}{}
	return mailbox
}

func (m *mailbox[T]) Send(value T) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.value = value
	if m.pending {
		return
	}
	m.pending = true
	m.ready <- struct{}{}
}

func (m *mailbox[T]) Receive(ctx context.Context) (T, error) {
	if err := ctx.Err(); err != nil {
		var zero T
		return zero, err
	}
	select {
	case <-ctx.Done():
		var zero T
		return zero, ctx.Err()
	case <-m.ready:
		return m.take(), nil
	}
}

func (m *mailbox[T]) take() T {
	m.mu.Lock()
	defer m.mu.Unlock()
	value := m.value
	m.pending = false
	return value
}
