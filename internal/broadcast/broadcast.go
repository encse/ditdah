// Package broadcast provides coalescing notifications to multiple subscribers.
package broadcast

import (
	"context"
	"sync"

	"morsemanual/internal/trigger"
)

// Broadcaster sends an activation to every current subscription.
type Broadcaster interface {
	Activate()
	Subscribe() Subscription
}

// Subscription receives coalesced broadcaster activations until closed.
type Subscription interface {
	Wait(ctx context.Context) error
	Close()
}

type broadcaster struct {
	mu            sync.Mutex
	subscriptions map[*subscription]struct{}
}

type subscription struct {
	broadcaster *broadcaster
	trigger     trigger.Trigger
	closeOnce   sync.Once
}

// New creates a broadcaster without subscriptions.
func New() Broadcaster {
	return &broadcaster{
		subscriptions: make(map[*subscription]struct{}),
	}
}

func (b *broadcaster) Activate() {
	b.mu.Lock()
	defer b.mu.Unlock()
	for subscription := range b.subscriptions {
		subscription.trigger.Activate()
	}
}

func (b *broadcaster) Subscribe() Subscription {
	subscription := &subscription{
		broadcaster: b,
		trigger:     trigger.New(),
	}
	b.mu.Lock()
	b.subscriptions[subscription] = struct{}{}
	b.mu.Unlock()
	return subscription
}

func (s *subscription) Wait(ctx context.Context) error {
	return s.trigger.Wait(ctx)
}

func (s *subscription) Close() {
	s.closeOnce.Do(func() {
		s.broadcaster.mu.Lock()
		delete(s.broadcaster.subscriptions, s)
		s.broadcaster.mu.Unlock()
	})
}
