package syncutil

import (
	"context"
	"testing"
	"time"
)

func TestBroadcasterActivateWakesEverySubscription(t *testing.T) {
	broadcaster := NewBroadcaster()
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

func TestBroadcasterActivateCoalescesForEachSubscription(t *testing.T) {
	broadcaster := NewBroadcaster()
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

func TestBroadcasterCloseUnsubscribes(t *testing.T) {
	broadcaster := NewBroadcaster()
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

func TestBroadcasterSlowSubscriptionDoesNotBlockActivate(t *testing.T) {
	broadcaster := NewBroadcaster()
	subscription := broadcaster.Subscribe()
	defer subscription.Close()

	for range 1000 {
		broadcaster.Activate()
	}

	if err := subscription.Wait(t.Context()); err != nil {
		t.Fatalf("Wait() = %v", err)
	}
}
