package neco

import (
	"context"
	"time"
)

// SleepContext sleeps for d,  Returned err is not nil if ctx is canceled
func SleepContext(ctx context.Context, d time.Duration) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(d):
	}
	return nil
}
