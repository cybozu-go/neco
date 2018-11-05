package worker

import (
	"context"
	"fmt"

	"github.com/coreos/etcd/clientv3"
	"github.com/cybozu-go/neco/storage"
)

// Server represents neco-worker server
type Server struct {
	ec      *clientv3.Client
	storage storage.Storage
}

// NewServer returns a Server
func NewServer(ec *clientv3.Client, st storage.Storage) Server {
	return Server{ec: ec, storage: st}
}

func (s Server) Run(ctx context.Context) error {
	for {
		err := s.updateNeco(ctx)
		if err != nil {
			return err
		}

		err = s.syncWorkers(ctx)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s Server) updateNeco(ctx context.Context) error {
	req, err := s.storage.GetRequest(ctx)
	if err != nil {
		return err
	}
	if req.Stop {
		return nil
	}
	//TODO: check version

	//TODO: update

	return nil
}

func (s Server) syncWorkers(ctx context.Context) error {
	statuses, rev, err := s.storage.GetStatusesWithRev(ctx)
	if err != nil {
		return err
	}
	if len(statuses) == 0 {
		//TODO: PutStatus with revision
		fmt.Println(rev)
	}
	//TODO: wait other workers

	return nil
}
