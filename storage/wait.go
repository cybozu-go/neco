package storage

import (
	"context"
	"encoding/json"
	"errors"
	"sort"
	"strconv"

	"github.com/coreos/etcd/clientv3"
	"github.com/cybozu-go/log"
	"github.com/cybozu-go/neco"
)

// ErrTimedOut is returned when the request is timed out.
var ErrTimedOut = errors.New("timed out")

// WaitRequest waits for a UpdateRequest to be written to etcd and returns it.
func (s Storage) WaitRequest(ctx context.Context, rev int64) (*neco.UpdateRequest, int64, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	ch := s.etcd.Watch(ctx, KeyCurrent,
		clientv3.WithRev(rev+1),
		clientv3.WithFilterDelete())
	for wr := range ch {
		err := wr.Err()
		if err != nil {
			return nil, 0, err
		}

		if len(wr.Events) == 0 {
			continue
		}

		ev := wr.Events[0]
		req := new(neco.UpdateRequest)
		err = json.Unmarshal(ev.Kv.Value, req)
		if err != nil {
			return nil, 0, err
		}

		return req, ev.Kv.ModRevision, nil
	}

	return nil, 0, errors.New("waitRequest was interrupted")
}

// WaitForMemberUpdated waits for new member added or member removed until with
// check-update-interval. It returns (true, nil) when timed-out.  It returns
// (false, nil) if member list is updated.
func (s Storage) WaitForMemberUpdated(ctx context.Context, req *neco.UpdateRequest, rev int64) (timeout bool, err error) {
	interval, err := s.GetCheckUpdateInterval(ctx)
	if err != nil {
		return false, err
	}

	withTimeoutCtx, cancel := context.WithTimeout(ctx, interval)
	defer cancel()

	ch := s.etcd.Watch(
		withTimeoutCtx, KeyBootserversPrefix, clientv3.WithRev(rev+1), clientv3.WithPrefix(),
	)

	for resp := range ch {
		for _, ev := range resp.Events {
			lrn, err := strconv.Atoi(string(ev.Kv.Key[len(KeyBootserversPrefix):]))
			if err != nil {
				return false, err
			}
			if ev.Type == clientv3.EventTypePut {
				if i := sort.SearchInts(req.Servers, lrn); i < len(req.Servers) && req.Servers[i] == lrn {
					continue
				}
			}
			return false, nil
		}
	}
	if withTimeoutCtx.Err() == context.DeadlineExceeded {
		return true, nil
	}
	return false, withTimeoutCtx.Err()
}

// WaitRequestDeletion waits for a UpdateRequest to be deleted.
func (s Storage) WaitRequestDeletion(ctx context.Context, req *neco.UpdateRequest, rev int64) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	ch := s.etcd.Watch(ctx, KeyCurrent, clientv3.WithRev(rev+1), clientv3.WithFilterPut())
	resp := <-ch
	if err := resp.Err(); err != nil {
		return err
	}
	return nil
}

// WaitWorkers waits for worker finishes updates until timed-out
//
// It returns (false, nil) if all workers have been updated successfully.
// It returns (true, error) if a worker is aborted.
// Otherwise it returns (false, error).
func (s Storage) WaitWorkers(ctx context.Context, req *neco.UpdateRequest, rev int64) (bool, error) {
	timeout, err := s.GetWorkerTimeout(ctx)
	if err != nil {
		return false, err
	}

	statuses := make(map[int]*neco.UpdateStatus)

	deadline := req.StartedAt.Add(timeout)
	deadlineCtx, cancel := context.WithDeadline(ctx, deadline)
	defer cancel()

	ch := s.etcd.Watch(
		deadlineCtx, KeyWorkerStatusPrefix,
		clientv3.WithRev(rev+1), clientv3.WithFilterDelete(), clientv3.WithPrefix(),
	)
	for resp := range ch {
		for _, ev := range resp.Events {
			st := new(neco.UpdateStatus)
			err = json.Unmarshal(ev.Kv.Value, st)
			if err != nil {
				return false, err
			}
			lrn, err := strconv.Atoi(string(ev.Kv.Key[len(KeyWorkerStatusPrefix):]))
			if err != nil {
				return false, err
			}
			if st.Version != req.Version {
				continue
			}
			statuses[lrn] = st

			switch st.Cond {
			case neco.CondAbort:
				log.Warn("worker failed updating", map[string]interface{}{
					"version": req.Version,
					"lrn":     lrn,
					"message": st.Message,
				})
				return true, errors.New(st.Message)
			case neco.CondComplete:
				log.Info("worker finished updating", map[string]interface{}{
					"version": req.Version,
					"lrn":     lrn,
				})
			}
		}

		success := neco.UpdateCompleted(req.Version, req.Servers, statuses)
		if success {
			log.Info("all worker finished updating", map[string]interface{}{
				"version": req.Version,
				"servers": req.Servers,
			})
			return false, nil
		}
	}

	log.Warn("workers timed-out", map[string]interface{}{
		"version":    req.Version,
		"started_at": req.StartedAt,
		"timeout":    timeout,
	})
	return false, ErrTimedOut
}
