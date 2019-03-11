package storage

import (
	"context"
	"encoding/json"

	"github.com/coreos/etcd/clientv3"
	"github.com/coreos/etcd/clientv3/clientv3util"
	"github.com/cybozu-go/neco"
)

func (s Storage) getContentsUpdateStatus(ctx context.Context, key string) (*neco.ContentsUpdateStatus, error) {
	resp, err := s.etcd.Get(ctx, key)
	if err != nil {
		return nil, err
	}

	if resp.Count == 0 {
		return nil, ErrNotFound
	}

	st := new(neco.ContentsUpdateStatus)
	err = json.Unmarshal(resp.Kvs[0].Value, st)
	if err != nil {
		return nil, err
	}

	return st, nil
}

func (s Storage) putContentsUpdateStatus(ctx context.Context, key string, st *neco.ContentsUpdateStatus, leaderKey string) error {
	data, err := json.Marshal(st)
	if err != nil {
		return err
	}

	resp, err := s.etcd.Txn(ctx).
		If(clientv3util.KeyExists(leaderKey)).
		Then(clientv3.OpPut(key, string(data))).
		Commit()
	if err != nil {
		return err
	}

	if !resp.Succeeded {
		return ErrNoLeader
	}

	return nil
}

// GetSabakanContentsStatus returns update status of Sabakan contents.
func (s Storage) GetSabakanContentsStatus(ctx context.Context) (*neco.ContentsUpdateStatus, error) {
	return s.getContentsUpdateStatus(ctx, KeySabakanContents)
}

// PutSabakanContentsStatus puts update status of Sabakan contents, only if caller is the leader.
func (s Storage) PutSabakanContentsStatus(ctx context.Context, st *neco.ContentsUpdateStatus, leaderKey string) error {
	return s.putContentsUpdateStatus(ctx, KeySabakanContents, st, leaderKey)
}

// GetCKEContentsStatus returns update status of CKE contents.
func (s Storage) GetCKEContentsStatus(ctx context.Context) (*neco.ContentsUpdateStatus, error) {
	return s.getContentsUpdateStatus(ctx, KeyCKEContents)
}

// PutCKEContentsStatus puts update status of CKE contents, only if caller is the leader.
func (s Storage) PutCKEContentsStatus(ctx context.Context, st *neco.ContentsUpdateStatus, leaderKey string) error {
	return s.putContentsUpdateStatus(ctx, KeyCKEContents, st, leaderKey)
}

// GetDHCPJSONContentsStatus returns update status of dhcp.json for sabakan.
func (s Storage) GetDHCPJSONContentsStatus(ctx context.Context) (*neco.ContentsUpdateStatus, error) {
	return s.getContentsUpdateStatus(ctx, KeyDHCPJSONContents)
}

// PutDHCPJSONContentsStatus puts update status of dhcp.json for sabakan, only if caller is the leader.
func (s Storage) PutDHCPJSONContentsStatus(ctx context.Context, st *neco.ContentsUpdateStatus, leaderKey string) error {
	return s.putContentsUpdateStatus(ctx, KeyDHCPJSONContents, st, leaderKey)
}

// GetCKETemplateContentsStatus returns update status of cke-template.yml.
func (s Storage) GetCKETemplateContentsStatus(ctx context.Context) (*neco.ContentsUpdateStatus, error) {
	return s.getContentsUpdateStatus(ctx, KeyCKETemplateContents)
}

// PutCKETemplateContentsStatus puts update status of cke-template.yml, only if caller is the leader.
func (s Storage) PutCKETemplateContentsStatus(ctx context.Context, st *neco.ContentsUpdateStatus, leaderKey string) error {
	return s.putContentsUpdateStatus(ctx, KeyCKETemplateContents, st, leaderKey)
}
