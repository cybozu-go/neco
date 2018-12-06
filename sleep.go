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

// RetryWithSleep invoke f until it succeeds or reach to max.
func RetryWithSleep(ctx context.Context, max int, d time.Duration, f func(ctx context.Context) error, logger func(err error)) error {
	var err error
	for i := 0; i < max; i++ {
		err = f(ctx)
		if err == nil {
			return nil
		}
		logger(err)
		err2 := SleepContext(ctx, d)
		if err2 != nil {
			return err2
		}
	}
	return err
}
