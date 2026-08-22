package syncutil

import (
	"context"
	"testing"
	"time"
)

func TestMailboxReceiveReturnsSentValue(t *testing.T) {
	mailbox := NewMailbox("initial")
	initial, err := mailbox.Receive(t.Context())
	if err != nil || initial != "initial" {
		t.Fatalf("initial Receive() = (%q, %v)", initial, err)
	}
	mailbox.Send("decoder")

	value, err := mailbox.Receive(t.Context())
	if err != nil {
		t.Fatalf("Receive() = %v", err)
	}
	if value != "decoder" {
		t.Fatalf("Receive() value = %q, want decoder", value)
	}
}

func TestMailboxSendCoalescesToLatestPendingValue(t *testing.T) {
	mailbox := NewMailbox("initial")
	mailbox.Send("logbook")
	mailbox.Send("decoder")

	value, err := mailbox.Receive(t.Context())
	if err != nil {
		t.Fatalf("Receive() = %v", err)
	}
	if value != "decoder" {
		t.Fatalf("Receive() value = %q, want decoder", value)
	}
	ctx, cancel := context.WithTimeout(t.Context(), 20*time.Millisecond)
	defer cancel()
	if _, err := mailbox.Receive(ctx); err != context.DeadlineExceeded {
		t.Fatalf("second Receive() = %v, want deadline exceeded", err)
	}
}

func TestMailboxReceiveReturnsContextError(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	mailbox := NewMailbox(42)
	if _, err := mailbox.Receive(ctx); err != context.Canceled {
		t.Fatalf("Receive() = %v, want context canceled", err)
	}
}
