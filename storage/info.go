package storage

import (
	"context"
	"sort"
	"strconv"

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

// GetInfo returns the GitHub package version and the current list of boot servers.
func (s Storage) GetInfo(ctx context.Context) (string, []int, int64, error) {
	resp, err := s.etcd.Get(ctx, KeyNecoRelease)
	if err != nil {
		return "", nil, 0, err
	}

	rev := resp.Header.Revision
	version := ""
	if resp.Count != 0 {
		version = string(resp.Kvs[0].Value)
	}

	resp, err = s.etcd.Get(ctx, KeyBootserversPrefix,
		clientv3.WithPrefix(),
		clientv3.WithKeysOnly(),
		clientv3.WithRev(rev),
	)
	if err != nil {
		return "", nil, 0, err
	}

	var lrns []int
	if resp.Count != 0 {
		lrns = make([]int, resp.Count)
		for i, kv := range resp.Kvs {
			lrn, err := strconv.Atoi(string(kv.Key[len(KeyBootserversPrefix):]))
			if err != nil {
				return "", nil, 0, err
			}
			lrns[i] = lrn
		}
		sort.Ints(lrns)
	}

	return version, lrns, rev, err
}
