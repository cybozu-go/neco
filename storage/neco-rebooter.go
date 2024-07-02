package storage

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"

	"github.com/cybozu-go/neco"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/client/v3/clientv3util"
	"go.etcd.io/etcd/client/v3/concurrency"
)

func (s Storage) IsNecoRebooterEnabled(ctx context.Context) (bool, error) {
	resp, err := s.etcd.Get(ctx, KeyNecoRebooterIsEnabled)
	if err != nil {
		return false, err
	}
	if resp.Count == 0 {
		return false, nil
	}

	return bytes.Equal([]byte("true"), resp.Kvs[0].Value), nil
}

func (s Storage) EnableNecoRebooter(ctx context.Context, flag bool) error {
	var val string
	if flag {
		val = "true"
	} else {
		val = "false"
	}
	_, err := s.etcd.Put(ctx, KeyNecoRebooterIsEnabled, val)
	return err
}

func (s Storage) GetRebootListEntry(ctx context.Context, index int64) (*neco.RebootListEntry, error) {
	resp, err := s.etcd.Get(ctx, rebootListEntryKey(index))
	if err != nil {
		return nil, err
	}
	if resp.Count == 0 {
		return nil, ErrNotFound
	}

	r := new(neco.RebootListEntry)
	err = json.Unmarshal(resp.Kvs[0].Value, r)
	if err != nil {
		return nil, err
	}

	return r, nil
}

func (s Storage) GetRebootListEntries(ctx context.Context) ([]*neco.RebootListEntry, error) {
	opts := []clientv3.OpOption{
		clientv3.WithPrefix(),
		clientv3.WithSort(clientv3.SortByKey, clientv3.SortAscend),
	}
	resp, err := s.etcd.Get(ctx, KeyNecoRebooterRebootList, opts...)
	if err != nil {
		return nil, err
	}

	if len(resp.Kvs) == 0 {
		return nil, nil
	}

	reboots := make([]*neco.RebootListEntry, len(resp.Kvs))
	for i, kv := range resp.Kvs {
		r := new(neco.RebootListEntry)
		err = json.Unmarshal(kv.Value, r)
		if err != nil {
			return nil, err
		}
		reboots[i] = r
	}

	return reboots, nil
}

func (s Storage) RemoveRebootListEntry(ctx context.Context, leaderKey string, entry *neco.RebootListEntry) error {
	key := rebootListEntryKey(entry.Index)
	resp, err := s.etcd.Txn(ctx).
		If(clientv3util.KeyExists(leaderKey)).
		Then(clientv3.OpDelete(key)).
		Commit()
	if err != nil {
		return err
	}
	if !resp.Succeeded {
		return ErrNoLeader
	}
	return nil
}

func (s Storage) UpdateRebootListEntry(ctx context.Context, entry *neco.RebootListEntry) error {
	key := rebootListEntryKey(entry.Index)
	data, err := json.Marshal(entry)
	if err != nil {
		return err
	}

RETRY:
	resp, err := s.etcd.Get(ctx, key)
	if err != nil {
		return err
	}
	if resp.Count == 0 {
		return ErrNotFound
	}

	rev := resp.Kvs[0].ModRevision
	txnResp, err := s.etcd.Txn(ctx).
		If(
			clientv3.Compare(clientv3.ModRevision(key), "=", rev),
		).
		Then(
			clientv3.OpPut(key, string(data)),
		).
		Commit()
	if err != nil {
		return err
	}
	if !txnResp.Succeeded {
		goto RETRY
	}

	return nil
}

func rebootListEntryKey(index int64) string {
	return fmt.Sprintf("%s%016x", KeyNecoRebooterRebootList, index)
}

func (s Storage) RegisterRebootListEntry(ctx context.Context, entry *neco.RebootListEntry) error {
RETRY:
	var writeIndex, writeIndexRev int64
	resp, err := s.etcd.Get(ctx, KeyNecoRebooterWriteIndex)
	if err != nil {
		return err
	}
	if resp.Count != 0 {
		value, err := strconv.ParseInt(string(resp.Kvs[0].Value), 10, 64)
		if err != nil {
			return err
		}
		writeIndex = value
		writeIndexRev = resp.Kvs[0].ModRevision
	}

	entry.Index = writeIndex
	data, err := json.Marshal(entry)
	if err != nil {
		return err
	}

	nextWriteIndex := strconv.FormatInt(writeIndex+1, 10)
	txnResp, err := s.etcd.Txn(ctx).
		If(
			clientv3.Compare(clientv3.ModRevision(KeyNecoRebooterWriteIndex), "=", writeIndexRev),
		).
		Then(
			clientv3.OpPut(rebootListEntryKey(writeIndex), string(data)),
			clientv3.OpPut(KeyNecoRebooterWriteIndex, nextWriteIndex),
		).
		Commit()
	if err != nil {
		return err
	}
	if !txnResp.Succeeded {
		goto RETRY
	}

	return nil
}

func (s Storage) GetProcessingGroup(ctx context.Context) (string, error) {
	resp, err := s.etcd.Get(ctx, KeyNecoRebooterProcessingGroup)
	if err != nil {
		return "", err
	}
	if resp.Count == 0 {
		return "", nil
	}
	return string(resp.Kvs[0].Value), nil
}

func (s Storage) UpdateProcessingGroup(ctx context.Context, group string) error {
RETRY:
	resp, err := s.etcd.Get(ctx, KeyNecoRebooterProcessingGroup)
	if err != nil {
		return err
	}

	var rev int64
	if resp.Count == 0 {
		rev = 0
	} else {
		rev = resp.Kvs[0].ModRevision
	}
	txnResp, err := s.etcd.Txn(ctx).
		If(
			clientv3.Compare(clientv3.ModRevision(KeyNecoRebooterProcessingGroup), "=", rev),
		).
		Then(
			clientv3.OpPut(KeyNecoRebooterProcessingGroup, group),
		).
		Commit()
	if err != nil {
		return err
	}
	if !txnResp.Succeeded {
		goto RETRY
	}

	return nil
}

func (s Storage) GetNecoRebooterLeader(ctx context.Context) (string, error) {
	session, err := concurrency.NewSession(s.etcd)
	if err != nil {
		return "", err
	}
	defer func() {
		// Checking the session to avoid an error caused by duplicated closing.
		select {
		case <-session.Done():
			return
		default:
			session.Close()
		}
	}()
	e := concurrency.NewElection(session, KeyNecoRebooterLeader)
	leader, err := e.Leader(ctx)
	if err != nil {
		return "", err
	}
	if len(leader.Kvs) == 0 {
		return "", errors.New("no leader")
	}
	return string(leader.Kvs[0].Value), nil
}
