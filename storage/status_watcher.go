package storage

import (
	"context"
	"encoding/json"
	"strconv"

	"github.com/cybozu-go/neco"
	"go.etcd.io/etcd/clientv3"
)

// StatusWatcher has callback handlers to handle status changes
type StatusWatcher struct {
	handleRequest func(context.Context, *neco.UpdateRequest) (bool, error)
	handleStatus  func(context.Context, int, *neco.UpdateStatus) (bool, error)
	handleError   func(context.Context, error) error
}

// NewStatusWatcher creates a new StatusWatcher
func NewStatusWatcher(
	handleRequest func(context.Context, *neco.UpdateRequest) (bool, error),
	handleStatus func(context.Context, int, *neco.UpdateStatus) (bool, error),
	handleError func(context.Context, error) error,
) StatusWatcher {
	return StatusWatcher{
		handleRequest: handleRequest,
		handleStatus:  handleStatus,
		handleError:   handleError,
	}
}

// Watch watches UpdateRequest or UpdateStatus is written to etcd
func (w StatusWatcher) Watch(ctx context.Context, storage Storage, rev int64) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	ch := storage.etcd.Watch(ctx, KeyStatusPrefix,
		clientv3.WithPrefix(),
		clientv3.WithRev(rev+1),
		clientv3.WithFilterDelete())
	for wr := range ch {
		err := wr.Err()
		if err != nil {
			w.handleError(ctx, err)
			return err
		}

		for _, ev := range wr.Events {
			completed, err := w.dispatch(ctx, ev)
			if err != nil {
				w.handleError(ctx, err)
				return err
			}
			if completed {
				return nil
			}
		}
	}

	return nil
}

func (w StatusWatcher) dispatch(ctx context.Context, ev *clientv3.Event) (bool, error) {
	if string(ev.Kv.Key) == KeyCurrent {
		req := new(neco.UpdateRequest)
		err := json.Unmarshal(ev.Kv.Value, req)
		if err != nil {
			return false, err
		}
		return w.handleRequest(ctx, req)
	}

	lrn, err := strconv.Atoi(string(ev.Kv.Key[len(KeyWorkerStatusPrefix):]))
	if err != nil {
		return false, err
	}

	st := new(neco.UpdateStatus)
	err = json.Unmarshal(ev.Kv.Value, st)
	if err != nil {
		return false, err
	}
	return w.handleStatus(ctx, lrn, st)
}
