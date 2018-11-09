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

// WatchWorkers watches worker changes until deadline is reached.
//
// `handler` will be called when worker status is changed.
// When watching is completed, `handler` should return (true, nil).
func (s Storage) WatchWorkers(ctx context.Context, deadline time.Time, rev int64,
	handler func(context.Context, int, *neco.UpdateStatus) (bool, error),
) error {
	deadlineCtx, cancel := context.WithDeadline(ctx, deadline)
	defer cancel()

	ch := s.etcd.Watch(
		deadlineCtx, KeyWorkerStatusPrefix,
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
			completed, err := handler(ctx, lrn, st)
			if err != nil {
				return err
			}
			if completed {
				return nil
			}
		}
	}
	return ErrTimedOut
}
