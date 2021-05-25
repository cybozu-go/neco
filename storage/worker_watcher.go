package storage

import (
	"context"
	"encoding/json"
	"strconv"

	"github.com/cybozu-go/neco"
	"go.etcd.io/etcd/clientv3"
)

// WorkerWatcher has callback handlers to handle status changes
type WorkerWatcher struct {
	handleStatus func(context.Context, int, *neco.UpdateStatus) bool
}

// NewWorkerWatcher creates a new WorkerWatcher
func NewWorkerWatcher(
	handleStatus func(context.Context, int, *neco.UpdateStatus) bool,
) WorkerWatcher {
	return WorkerWatcher{
		handleStatus: handleStatus,
	}
}

// Watch watches worker changes until deadline is reached.
// If the handleStatus returns true, this returns nil.
// Otherwise non-nil error is returned.
func (w WorkerWatcher) Watch(ctx context.Context, rev int64, storage Storage) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	ch := storage.etcd.Watch(
		ctx, KeyWorkerStatusPrefix,
		clientv3.WithRev(rev+1), clientv3.WithFilterDelete(), clientv3.WithPrefix(),
	)
	for resp := range ch {
		for _, ev := range resp.Events {
			st := new(neco.UpdateStatus)
			err := json.Unmarshal(ev.Kv.Value, st)
			if err != nil {
				return err
			}
			lrn, err := strconv.Atoi(string(ev.Kv.Key[len(KeyWorkerStatusPrefix):]))
			if err != nil {
				return err
			}
			completed := w.handleStatus(ctx, lrn, st)
			if completed {
				return nil
			}
		}
	}

	err := ctx.Err()
	if err == context.DeadlineExceeded {
		return ErrTimedOut
	}
	return err
}
