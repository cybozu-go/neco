package storage

import (
	"context"
	"strconv"
	"time"

	"github.com/cybozu-go/neco"
)

// PutSlackNotification stores SlackNotification to storage
func (s Storage) PutSlackNotification(ctx context.Context, url string) error {
	_, err := s.etcd.Put(ctx, KeyNotificationSlack, url)
	return err
}

// GetSlackNotification returns SlackNotification from storage
// If not found, this returns ErrNotFound.
func (s Storage) GetSlackNotification(ctx context.Context) (string, error) {
	resp, err := s.etcd.Get(ctx, KeyNotificationSlack)
	if err != nil {
		return "", err
	}
	if resp.Count == 0 {
		return "", ErrNotFound
	}
	return string(resp.Kvs[0].Value), nil
}

// PutProxyConfig stores proxy config to storage.
func (s Storage) PutProxyConfig(ctx context.Context, proxy string) error {
	_, err := s.etcd.Put(ctx, KeyProxy, proxy)
	return err
}

// GetProxyConfig returns proxy config from storage.
func (s Storage) GetProxyConfig(ctx context.Context) (string, error) {
	resp, err := s.etcd.Get(ctx, KeyProxy)
	if err != nil {
		return "", err
	}

	if resp.Count == 0 {
		return "", ErrNotFound
	}
	return string(resp.Kvs[0].Value), nil
}

// PutCheckUpdateInterval stores check-update-interval config to storage.
func (s Storage) PutCheckUpdateInterval(ctx context.Context, d time.Duration) error {
	data := strconv.FormatInt(int64(d), 10)
	_, err := s.etcd.Put(ctx, KeyCheckUpdateInterval, data)
	return err
}

// GetCheckUpdateInterval returns check-update-interval config from storage. It
// returns default value if the key does not exist.
func (s Storage) GetCheckUpdateInterval(ctx context.Context) (time.Duration, error) {
	resp, err := s.etcd.Get(ctx, KeyCheckUpdateInterval)
	if err != nil {
		return 0, err
	}
	if resp.Count == 0 {
		return neco.DefaultCheckUpdateInterval, nil
	}
	i, err := strconv.ParseInt(string(resp.Kvs[0].Value), 10, 64)
	if err != nil {
		return 0, err
	}
	return time.Duration(i), nil
}

// PutWorkerTimeout stores worker-timeout config to storage.
func (s Storage) PutWorkerTimeout(ctx context.Context, d time.Duration) error {
	data := strconv.FormatInt(int64(d), 10)
	_, err := s.etcd.Put(ctx, KeyWorkerTimeout, data)
	return err
}

// GetWorkerTimeout returns worker-timeout config from storage. It returns
// default value if the key does not exist.
func (s Storage) GetWorkerTimeout(ctx context.Context) (time.Duration, error) {
	resp, err := s.etcd.Get(ctx, KeyWorkerTimeout)
	if err != nil {
		return 0, err
	}
	if resp.Count == 0 {
		return neco.DefaultWorkerTimeout, nil
	}
	i, err := strconv.ParseInt(string(resp.Kvs[0].Value), 10, 64)
	if err != nil {
		return 0, err
	}
	return time.Duration(i), nil
}
