package storage

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/coreos/etcd/clientv3"
	"github.com/cybozu-go/neco"
)

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

// WaitRequestDeletion waits for a UpdateRequest to be deleted.
func (s Storage) WaitRequestDeletion(ctx context.Context, rev int64) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	ch := s.etcd.Watch(ctx, KeyCurrent, clientv3.WithRev(rev+1), clientv3.WithFilterPut())
	resp := <-ch
	if err := resp.Err(); err != nil {
		return err
	}
	return nil
}

// WaitConfigChange waits config key change and return a non-nil error.
func (s Storage) WaitConfigChange(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	ch := s.etcd.Watch(ctx, KeyConfigPrefix, clientv3.WithPrefix())
	select {
	case <-ctx.Done():
		return nil
	case resp, ok := <-ch:
		if !ok {
			return nil
		}

		if resp.Err() != nil {
			return resp.Err()
		}
		return errors.New("detected config change")
	}
}
