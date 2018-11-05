package worker

import (
	"context"
	"errors"
	"strconv"

	"github.com/coreos/etcd/clientv3"
	"github.com/cybozu-go/neco"
	"github.com/cybozu-go/neco/storage"
)

// Server represents neco-worker server
type Server struct {
	mylrn   int
	ec      *clientv3.Client
	storage storage.Storage
	req     *neco.UpdateRequest
	status  *neco.UpdateStatus
}

// NewServer returns a Server
func NewServer(ec *clientv3.Client) (Server, error) {
	mylrn, err := neco.MyLRN()
	if err != nil {
		return Server{}, err
	}
	return Server{mylrn: mylrn, ec: ec, storage: storage.NewStorage(ec)}, nil
}

// Run executes neco-worker task indefinitely until ctx is cancelled.
func (s Server) Run(ctx context.Context) error {
	req, rev, err := s.storage.GetRequestWithRev(ctx)

	switch err {
	case storage.ErrNotFound:
		// nothing to do

	case nil:
		err = updateNeco(ctx, req.Version)
		if err != nil {
			return err
		}
		stMap, err := s.storage.GetStatuses(ctx, req.Servers)
		if err != nil {
			return err
		}
		if neco.UpdateAborted(req.Version, stMap) {
			return errors.New("update was aborted")
		}

		myStatus := stMap[s.mylrn]
		if myStatus == nil || req.Version != myStatus.Version {
			myStatus, err = s.RegisterStatus(ctx, req.Version)
			if err != nil {
				return err
			}
		}
		s.req = req
		s.status = myStatus
	default:
		return err
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	ch := s.ec.Watch(ctx, storage.KeyStatusPrefix,
		clientv3.WithPrefix(),
		clientv3.WithRev(rev+1),
		clientv3.WithFilterDelete())
	for wr := range ch {
		err := wr.Err()
		if err != nil {
			return err
		}

		for _, ev := range wr.Events {
			err = s.dispatch(ctx, ev)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (s Server) dispatch(ctx context.Context, ev *clientv3.Event) error {
	key := string(ev.Kv.Key[len(storage.KeyStatusPrefix):])
	if key == "current" {
		return s.handleCurrent(ctx, ev)
	}

	lrn, err := strconv.Atoi(string(ev.Kv.Key[len(storage.KeyWorkerStatusPrefix):]))
	if err != nil {
		return err
	}

	if lrn == s.mylrn {
		return nil
	}
	return s.handleWorkerStatus(ctx, lrn, ev)
}

func updateNeco(ctx context.Context, version string) error {
	return nil
}

func (s Server) RegisterStatus(ctx context.Context, version string) (*neco.UpdateStatus, error) {
	return nil, nil
}

func (s Server) handleCurrent(ctx context.Context, ev *clientv3.Event) error {
	return nil
}

func (s Server) handleWorkerStatus(ctx context.Context, lrn int, ev *clientv3.Event) error {
	return nil
}
