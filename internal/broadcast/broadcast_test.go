package broadcast

import (
	"context"
	"testing"
	"time"
)

func TestActivateWakesEverySubscription(t *testing.T) {
	broadcaster := New()
	first := broadcaster.Subscribe()
	defer first.Close()
	second := broadcaster.Subscribe()
	defer second.Close()

	broadcaster.Activate()

	if err := first.Wait(t.Context()); err != nil {
		t.Fatalf("first Wait() = %v", err)
	}
	if err := second.Wait(t.Context()); err != nil {
		t.Fatalf("second Wait() = %v", err)
	}
}

func TestActivateCoalescesForEachSubscription(t *testing.T) {
	broadcaster := New()
	subscription := broadcaster.Subscribe()
	defer subscription.Close()
	broadcaster.Activate()
	broadcaster.Activate()

	if err := subscription.Wait(t.Context()); err != nil {
		t.Fatalf("first Wait() = %v", err)
	}
	ctx, cancel := context.WithTimeout(t.Context(), 20*time.Millisecond)
	defer cancel()
	if err := subscription.Wait(ctx); err != context.DeadlineExceeded {
		t.Fatalf("second Wait() = %v, want deadline exceeded", err)
	}
}

func TestCloseUnsubscribes(t *testing.T) {
	broadcaster := New()
	subscription := broadcaster.Subscribe()
	subscription.Close()
	subscription.Close()

	broadcaster.Activate()

	ctx, cancel := context.WithTimeout(t.Context(), 20*time.Millisecond)
	defer cancel()
	if err := subscription.Wait(ctx); err != context.DeadlineExceeded {
		t.Fatalf("Wait() after Close() = %v, want deadline exceeded", err)
	}
}

func TestSlowSubscriptionDoesNotBlockActivate(t *testing.T) {
	broadcaster := New()
	subscription := broadcaster.Subscribe()
	defer subscription.Close()

	for range 1000 {
		broadcaster.Activate()
	}

	if err := subscription.Wait(t.Context()); err != nil {
		t.Fatalf("Wait() = %v", err)
	}
}
