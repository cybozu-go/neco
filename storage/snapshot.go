package storage

import (
	"context"
	"encoding/json"
	"strconv"

	"github.com/coreos/etcd/clientv3"
	"github.com/cybozu-go/neco"
)

// Snapshot is the up-to-date snapshot of etcd database.
type Snapshot struct {
	Revision int64
	Request  *neco.UpdateRequest
	Statuses map[int]*neco.UpdateStatus
	Latest   string
	Servers  []int
}

// NewSnapshot takes the up-to-date snapshot.
func (s Storage) NewSnapshot(ctx context.Context) (*Snapshot, error) {
	snap := &Snapshot{}

	resp, err := s.etcd.Get(ctx, KeyCurrent)
	if err != nil {
		return nil, err
	}

	rev := resp.Header.Revision
	snap.Revision = rev
	if resp.Count > 0 {
		req := new(neco.UpdateRequest)
		err = json.Unmarshal(resp.Kvs[0].Value, req)
		if err != nil {
			return nil, err
		}
		snap.Request = req
	}

	resp, err = s.etcd.Get(ctx, KeyBootserversPrefix,
		clientv3.WithPrefix(),
		clientv3.WithRev(rev),
		clientv3.WithKeysOnly())
	if err != nil {
		return nil, err
	}

	var lrns []int
	if resp.Count > 0 {
		lrns = make([]int, resp.Count)
		for i, kv := range resp.Kvs {
			lrn, err := strconv.Atoi(string(kv.Key[len(KeyBootserversPrefix):]))
			if err != nil {
				return nil, err
			}
			lrns[i] = lrn
		}
		snap.Servers = lrns
	}

	resp, err = s.etcd.Get(ctx, KeyNecoRelease, clientv3.WithRev(rev))
	if err != nil {
		return nil, err
	}
	if resp.Count > 0 {
		snap.Latest = string(resp.Kvs[0].Value)
	}

	statuses, err := s.getStatusesAt(ctx, rev)
	if err != nil {
		return nil, err
	}
	snap.Statuses = statuses

	return snap, nil
}
