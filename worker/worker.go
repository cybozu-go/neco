package worker

import (
	"context"
	"errors"
	"strconv"

	"github.com/coreos/etcd/clientv3"
	"github.com/cybozu-go/neco"
	"github.com/cybozu-go/neco/storage"
)

// Worker implements Neco auto update worker process.

type Worker struct {
	mylrn   int
	ec      *clientv3.Client
	storage storage.Storage
	req     *neco.UpdateRequest
	status  *neco.UpdateStatus
	arrival map[int]bool
}

// NewWorker returns a *Worker
func NewWorker(ec *clientv3.Client) (*Worker, error) {
	mylrn, err := neco.MyLRN()
	if err != nil {
		return nil, err
	}

	return &Worker{
		mylrn:   mylrn,
		ec:      ec,
		storage: storage.NewStorage(ec),
	}, nil
}

func (w *Worker) initArrival(lrns []int) {
	m := make(map[int]bool)
	for _, lrn := range lrns {
		m[lrn] = false
	}
	w.arrival = m
}

func (w *Worker) allArrived() bool {
	for _, v := range w.arrival {
		if !v {
			return false
		}
	}
	return true
}

// Run executes neco-worker task indefinitely until ctx is cancelled.
func (w *Worker) Run(ctx context.Context) error {
	req, rev, err := w.storage.GetRequestWithRev(ctx)

	switch err {
	case storage.ErrNotFound:
		// nothing to do

	case nil:
		err = updateNeco(ctx, req.Version)
		if err != nil {
			return err
		}
		stMap, err := w.storage.GetStatuses(ctx, req.Servers)
		if err != nil {
			return err
		}
		if neco.UpdateAborted(req.Version, stMap) {
			return errors.New("update was aborted")
		}

		myStatus := stMap[w.mylrn]
		if myStatus == nil || req.Version != myStatus.Version {
			myStatus, err = w.RegisterStatus(ctx, req.Version)
			if err != nil {
				return err
			}
		}
		w.req = req
		w.status = myStatus
		w.initArrival(req.Servers)
	default:
		return err
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	ch := w.ec.Watch(ctx, storage.KeyStatusPrefix,
		clientv3.WithPrefix(),
		clientv3.WithRev(rev+1),
		clientv3.WithFilterDelete())
	for wr := range ch {
		err := wr.Err()
		if err != nil {
			return err
		}

		for _, ev := range wr.Events {
			err = w.dispatch(ctx, ev)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (w *Worker) dispatch(ctx context.Context, ev *clientv3.Event) error {
	key := string(ev.Kv.Key[len(storage.KeyStatusPrefix):])
	if key == "current" {
		return w.handleCurrent(ctx, ev)
	}

	lrn, err := strconv.Atoi(string(ev.Kv.Key[len(storage.KeyWorkerStatusPrefix):]))
	if err != nil {
		return err
	}

	if lrn == w.mylrn {
		return nil
	}
	return w.handleWorkerStatus(ctx, lrn, ev)
}

func updateNeco(ctx context.Context, version string) error {
	return nil
}

func (w *Worker) RegisterStatus(ctx context.Context, version string) (*neco.UpdateStatus, error) {
	return nil, nil
}

func (w *Worker) handleCurrent(ctx context.Context, ev *clientv3.Event) error {
	return nil
}

func (w *Worker) handleWorkerStatus(ctx context.Context, lrn int, ev *clientv3.Event) error {
	return nil
}
