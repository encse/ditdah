package trigger

import (
	"context"
	"testing"
	"time"
)

func TestActivateWakesWait(t *testing.T) {
	trigger := New()
	trigger.Activate()
	if err := trigger.Wait(t.Context()); err != nil {
		t.Fatalf("Wait() = %v", err)
	}
}

func TestActivateCoalescesPendingSignals(t *testing.T) {
	trigger := New()
	trigger.Activate()
	trigger.Activate()
	if err := trigger.Wait(t.Context()); err != nil {
		t.Fatalf("first Wait() = %v", err)
	}
	ctx, cancel := context.WithTimeout(t.Context(), 20*time.Millisecond)
	defer cancel()
	if err := trigger.Wait(ctx); err != context.DeadlineExceeded {
		t.Fatalf("second Wait() = %v, want deadline exceeded", err)
	}
}

func TestWaitReturnsContextError(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if err := New().Wait(ctx); err != context.Canceled {
		t.Fatalf("Wait() = %v, want context canceled", err)
	}
}
