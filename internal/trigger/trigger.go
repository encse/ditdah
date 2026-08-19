// Package trigger provides a coalescing wake-up signal.
package trigger

import "context"

// Trigger coalesces pending activations into a single wake-up.
type Trigger interface {
	Activate()
	Wait(ctx context.Context) error
}

type trigger struct {
	activated chan struct{}
}

// New creates an inactive trigger.
func New() Trigger {
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
