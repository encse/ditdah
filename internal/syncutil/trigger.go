// Package syncutil provides lifecycle-aware synchronization helpers.
package syncutil

import "context"

// Trigger coalesces pending activations into a single wake-up.
type Trigger interface {
	Activate()
	Wait(ctx context.Context) error
}

type trigger struct {
	activated chan struct{}
}

// NewTrigger creates an inactive trigger.
func NewTrigger() Trigger {
	return &trigger{activated: make(chan struct{}, 1)}
}

func (t *trigger) Activate() {
	select {
	case t.activated <- struct{}{}:
	default:
	}
}

func (t *trigger) Wait(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.activated:
		return nil
	}
}
