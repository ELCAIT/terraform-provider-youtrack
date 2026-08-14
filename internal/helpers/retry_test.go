package helpers

import (
	"context"
	"testing"
	"time"
)

func TestWaitOrContextDone(t *testing.T) {
	t.Parallel()

	t.Run("returns nil once the delay elapses", func(t *testing.T) {
		t.Parallel()

		if err := WaitOrContextDone(context.Background(), time.Millisecond); err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
	})

	t.Run("returns ctx error when cancelled first", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		if err := WaitOrContextDone(ctx, time.Second); err == nil {
			t.Fatal("expected an error when the context is already cancelled")
		}
	})
}
