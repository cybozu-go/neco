package server

import (
	"context"
	"time"

	"github.com/cybozu-go/cke"
	"github.com/cybozu-go/well"
)

// Integrator defines interface to integrate external addon into CKE server.
type Integrator interface {
	// StartWatch starts watching etcd until the context is canceled.
	//
	// It should send an empty struct to the channel when some event occurs.
	// To avoid blocking, use select when sending.
	//
	// If the integrator does not implement StartWatch, simply return nil.
	StartWatch(context.Context, chan<- struct{}) error

	// Do does something for CKE.  leaderKey is an etcd object key that
	// exists as long as the current process is the leader.
	Do(ctx context.Context, leaderKey string) error
}

// RunIntegrator simply executes Integrator until ctx is canceled.
// This is for debugging.
func RunIntegrator(ctx context.Context, it Integrator) error {
	ch := make(chan struct{}, 1)
	env := well.NewEnvironment(ctx)

	env.Go(func(ctx context.Context) error {
		return it.StartWatch(ctx, ch)
	})
	env.Go(func(ctx context.Context) error {
		for {
			select {
			case <-ctx.Done():
				return nil
			case <-ch:
			case <-time.After(5 * time.Second):
			}

			err := it.Do(ctx, cke.KeySabakanTemplate)
			if err != nil {
				return err
			}
		}
	})
	env.Stop()

	return env.Wait()
}
