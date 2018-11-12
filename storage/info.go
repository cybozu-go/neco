package storage

import (
	"context"

	"github.com/coreos/etcd/clientv3"
	"github.com/coreos/etcd/clientv3/clientv3util"
)

// RegisterBootserver registers a bootserver with etcd database.
func (s Storage) RegisterBootserver(ctx context.Context, lrn int) error {
	_, err := s.etcd.Put(ctx, keyBootServer(lrn), "")
	return err
}

// DeleteBootServer deletes boot server from etcd database.
func (s Storage) DeleteBootServer(ctx context.Context, lrn int) error {
	resp, err := s.etcd.Txn(ctx).
		Then(
			clientv3.OpDelete(keyBootServer(lrn)),
			clientv3.OpDelete(keyInstall(lrn), clientv3.WithPrefix()),
		).
		Commit()
	if err != nil {
		return err
	}
	if resp.Responses[0].GetResponseDeleteRange().Deleted == 0 {
		return ErrNotFound
	}
	return nil
}

// UpdateNecoRelease updates the neco package version with the latest GitHub release.
func (s Storage) UpdateNecoRelease(ctx context.Context, version, leaderKey string) error {
	resp, err := s.etcd.Txn(ctx).
		If(clientv3util.KeyExists(leaderKey)).
		Then(clientv3.OpPut(KeyNecoRelease, version)).
		Commit()
	if err != nil {
		return err
	}
	if !resp.Succeeded {
		return ErrNoLeader
	}
	return nil
}

// GetNecoRelease gets the neco package version recorded in etcd.
func (s Storage) GetNecoRelease(ctx context.Context) (string, error) {
	resp, err := s.etcd.Get(ctx, KeyNecoRelease)
	if err != nil {
		return "", err
	}

	if resp.Count == 0 {
		return "", nil
	}
	return string(resp.Kvs[0].Value), nil
}

// WaitInfo waits for update of keys under `info/`
func (s Storage) WaitInfo(ctx context.Context, rev int64) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	ch := s.etcd.Watch(ctx, KeyInfoPrefix,
		clientv3.WithPrefix(), clientv3.WithKeysOnly(), clientv3.WithRev(rev+1))

	resp := <-ch
	return resp.Err()
}
