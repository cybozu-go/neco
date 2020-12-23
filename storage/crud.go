package storage

import (
	"context"

	"github.com/coreos/etcd/clientv3"
)

func (s Storage) get(ctx context.Context, key string) (string, error) {
	resp, err := s.etcd.Get(ctx, key)
	if err != nil {
		return "", err
	}
	if resp.Count == 0 {
		return "", ErrNotFound
	}
	return string(resp.Kvs[0].Value), nil
}

func (s Storage) list(ctx context.Context, prefix string) (map[string]string, error) {
	resp, err := s.etcd.Get(ctx, prefix, clientv3.WithPrefix())
	if err != nil {
		return nil, err
	}
	if resp.Count == 0 {
		return nil, ErrNotFound
	}
	result := make(map[string]string, resp.Count)
	for _, kv := range resp.Kvs {
		key := string(kv.Key)[len(prefix):]
		result[key] = string(kv.Value)
	}
	return result, nil
}

func (s Storage) put(ctx context.Context, key, value string) error {
	_, err := s.etcd.Put(ctx, key, value)
	return err
}

func (s Storage) del(ctx context.Context, key string) error {
	_, err := s.etcd.Delete(ctx, key)
	return err
}
