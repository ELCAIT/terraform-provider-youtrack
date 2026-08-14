package helpers

import (
	"context"
	"time"
)

// WaitOrContextDone pauses for delay, returning ctx.Err() if the context is cancelled first.
//
// The timer is stopped on every path rather than left to expire on its own, so a retry loop
// abandoned early by a cancelled context releases it immediately instead of keeping it alive for
// the remainder of delay.
func WaitOrContextDone(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
