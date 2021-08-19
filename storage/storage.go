package storage

import (
	"context"
	"encoding/json"
	"strconv"

	"github.com/cybozu-go/neco"
	"go.etcd.io/etcd/clientv3"
	"go.etcd.io/etcd/clientv3/clientv3util"
)

// Storage is storage client
type Storage struct {
	etcd *clientv3.Client
}

// NewStorage returns Storage that stores data in etcd.
func NewStorage(etcd *clientv3.Client) Storage {
	return Storage{etcd}
}

// RecordContainerTag records installed container image tag
func (s Storage) RecordContainerTag(ctx context.Context, lrn int, name string) error {
	img, err := neco.CurrentArtifacts.FindContainerImage(name)
	if err != nil {
		return err
	}
	key := keyContainer(lrn, name)
	_, err = s.etcd.Put(ctx, key, img.Tag)
	return err
}

// GetContainerTag returns installed container image tag
func (s Storage) GetContainerTag(ctx context.Context, lrn int, name string) (string, error) {
	key := keyContainer(lrn, name)
	resp, err := s.etcd.Get(ctx, key)
	if err != nil {
		return "", err
	}
	if resp.Count == 0 {
		return "", ErrNotFound
	}
	return string(resp.Kvs[0].Value), nil
}

// RecordDebVersion records installed debian package version
func (s Storage) RecordDebVersion(ctx context.Context, lrn int, name string) error {
	deb, err := neco.CurrentArtifacts.FindDebianPackage(name)
	if err != nil {
		return err
	}
	key := keyDeb(lrn, name)
	_, err = s.etcd.Put(ctx, key, deb.Release)
	return err
}

// GetDebVersion returns installed debian package version
func (s Storage) GetDebVersion(ctx context.Context, lrn int, name string) (string, error) {
	key := keyDeb(lrn, name)
	resp, err := s.etcd.Get(ctx, key)
	if err != nil {
		return "", err
	}
	if resp.Count == 0 {
		return "", ErrNotFound
	}
	return string(resp.Kvs[0].Value), nil
}

// PutRequest stores UpdateRequest to storage
// leaderKey is the current leader key.
// If the caller has lost the leadership, this returns ErrNoLeader.
func (s Storage) PutRequest(ctx context.Context, req neco.UpdateRequest, leaderKey string) error {
	data, err := json.Marshal(req)
	if err != nil {
		return err
	}

	resp, err := s.etcd.Txn(ctx).
		If(clientv3util.KeyExists(leaderKey)).
		Then(clientv3.OpPut(KeyCurrent, string(data))).
		Commit()
	if err != nil {
		return err
	}

	if !resp.Succeeded {
		return ErrNoLeader
	}

	return nil
}

// PutReconfigureRequest stores UpdateRequest to storage and
// delete worker statuses in a single transaction.
// leaderKey is the current leader key.
// If the caller has lost the leadership, this returns ErrNoLeader.
func (s Storage) PutReconfigureRequest(ctx context.Context, req neco.UpdateRequest, leaderKey string) error {
	data, err := json.Marshal(req)
	if err != nil {
		return err
	}

	resp, err := s.etcd.Txn(ctx).
		If(clientv3util.KeyExists(leaderKey)).
		Then(
			clientv3.OpPut(KeyCurrent, string(data)),
			clientv3.OpDelete(KeyWorkerStatusPrefix, clientv3.WithPrefix()),
		).
		Commit()
	if err != nil {
		return err
	}

	if !resp.Succeeded {
		return ErrNoLeader
	}

	return nil
}

// GetRequestWithRev returns UpdateRequest from storage with ModRevision and Revision.
// If there is no request, this returns ErrNotFound
func (s Storage) GetRequestWithRev(ctx context.Context) (req *neco.UpdateRequest, modRev int64, rev int64, err error) {
	resp, err := s.etcd.Get(ctx, KeyCurrent)
	if err != nil {
		return nil, 0, 0, err
	}

	if resp.Count == 0 {
		return nil, 0, resp.Header.Revision, ErrNotFound
	}

	req = new(neco.UpdateRequest)
	err = json.Unmarshal(resp.Kvs[0].Value, req)
	if err != nil {
		return nil, 0, 0, err
	}

	return req, resp.Kvs[0].ModRevision, resp.Header.Revision, nil
}

// GetRequest returns UpdateRequest from storage
// If there is no request, this returns ErrNotFound
func (s Storage) GetRequest(ctx context.Context) (*neco.UpdateRequest, error) {
	req, _, _, err := s.GetRequestWithRev(ctx)
	return req, err
}

// PutStatus stores UpdateStatus of a bootserver to storage
func (s Storage) PutStatus(ctx context.Context, lrn int, st neco.UpdateStatus) error {
	data, err := json.Marshal(st)
	if err != nil {
		return err
	}

	_, err = s.etcd.Put(ctx, keyStatus(lrn), string(data))
	return err
}

// GetStatus returns UpdateStatus of a bootserver from storage
// If not found, this returns ErrNotFound.
func (s Storage) GetStatus(ctx context.Context, lrn int) (*neco.UpdateStatus, error) {
	resp, err := s.etcd.Get(ctx, keyStatus(lrn))
	if err != nil {
		return nil, err
	}

	if resp.Count == 0 {
		return nil, ErrNotFound
	}

	st := new(neco.UpdateStatus)
	err = json.Unmarshal(resp.Kvs[0].Value, st)
	if err != nil {
		return nil, err
	}

	return st, nil
}

func (s Storage) getStatusesAt(ctx context.Context, rev int64) (map[int]*neco.UpdateStatus, error) {
	resp, err := s.etcd.Get(ctx, KeyWorkerStatusPrefix,
		clientv3.WithPrefix(),
		clientv3.WithRev(rev))
	if err != nil {
		return nil, err
	}

	if resp.Count == 0 {
		return nil, nil
	}

	stMap := make(map[int]*neco.UpdateStatus)
	for _, kv := range resp.Kvs {
		lrn, err := strconv.Atoi(string(kv.Key[len(KeyWorkerStatusPrefix):]))
		if err != nil {
			return nil, err
		}

		u := new(neco.UpdateStatus)
		err = json.Unmarshal(kv.Value, u)
		if err != nil {
			return nil, err
		}

		stMap[lrn] = u
	}

	return stMap, nil
}

// GetStatuses returns UpdateStatus of existing boot servers.
func (s Storage) GetStatuses(ctx context.Context) (map[int]*neco.UpdateStatus, error) {
	return s.getStatusesAt(ctx, 0)
}

// ClearStatusAndContents removes KeyStatusPrefix/* and KeyContentsPrefix/* from storage.
//
// It first checks that "stop" field in KeyCurrent is true.  If not,
// ErrNotStopped will be returned.
//
// Then it removes status keys in a single transaction.
func (s Storage) ClearStatusAndContents(ctx context.Context) error {
RETRY:
	req, modRev, _, err := s.GetRequestWithRev(ctx)
	if err != nil {
		return err
	}

	if !req.Stop {
		return ErrNotStopped
	}

	resp, err := s.etcd.Txn(ctx).
		If(clientv3.Compare(clientv3.ModRevision(KeyCurrent), "=", modRev)).
		Then(
			clientv3.OpDelete(KeyStatusPrefix, clientv3.WithPrefix()),
			clientv3.OpDelete(KeyContentsPrefix, clientv3.WithPrefix()),
		).
		Commit()
	if err != nil {
		return err
	}

	if !resp.Succeeded {
		goto RETRY
	}

	return nil
}
