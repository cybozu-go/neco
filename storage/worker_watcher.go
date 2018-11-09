package storage

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"time"

	"github.com/coreos/etcd/clientv3"
	"github.com/cybozu-go/neco"
)

// ErrTimedOut is returned when the request is timed out.
var ErrTimedOut = errors.New("timed out")

// WorkerWatcher has callback handlers to handle status changes
type WorkerWatcher struct {
	handleStatus func(context.Context, int, *neco.UpdateStatus) (bool, error)
	handleError  func(context.Context, error) error
}

// NewWorkerWatcher creates a new WorkerWatcher
func NewWorkerWatcher(
	handleStatus func(context.Context, int, *neco.UpdateStatus) (bool, error),
	handleError func(context.Context, error) error,
) WorkerWatcher {
	return WorkerWatcher{
		handleStatus: handleStatus,
		handleError:  handleError,
	}
}

// Watch watches worker changes until deadline is reached.
func (w WorkerWatcher) Watch(ctx context.Context, deadline time.Time, rev int64, storage Storage) error {
	deadlineCtx, cancel := context.WithDeadline(ctx, deadline)
	defer cancel()

	ch := storage.etcd.Watch(
		deadlineCtx, KeyWorkerStatusPrefix,
		clientv3.WithRev(rev+1), clientv3.WithFilterDelete(), clientv3.WithPrefix(),
	)
	for resp := range ch {
		for _, ev := range resp.Events {
			st := new(neco.UpdateStatus)
			err := json.Unmarshal(ev.Kv.Value, st)
			if err != nil {
				w.handleError(ctx, err)
				return err
			}
			lrn, err := strconv.Atoi(string(ev.Kv.Key[len(KeyWorkerStatusPrefix):]))
			if err != nil {
				w.handleError(ctx, err)
				return err
			}
			completed, err := w.handleStatus(ctx, lrn, st)
			if err != nil {
				w.handleError(ctx, err)
				return err
			}
			if completed {
				return nil
			}
		}
	}
	w.handleError(ctx, ErrTimedOut)
	return ErrTimedOut
}
