package storage

import (
	"context"
	"encoding/json"

	"github.com/cybozu-go/neco"
)

// PutNotificationConfig stores NotificationConfig to storage
func (s Storage) PutNotificationConfig(ctx context.Context, n neco.NotificationConfig) error {
	data, err := json.Marshal(n)
	if err != nil {
		return err
	}

	_, err = s.etcd.Put(ctx, KeyNotification, string(data))
	return err
}

// GetNotificationConfig returns NotificationConfig from storage
// If not found, this returns ErrNotFound.
func (s Storage) GetNotificationConfig(ctx context.Context) (*neco.NotificationConfig, error) {
	resp, err := s.etcd.Get(ctx, KeyNotification)
	if err != nil {
		return nil, err
	}

	if resp.Count == 0 {
		return nil, ErrNotFound
	}

	n := new(neco.NotificationConfig)
	err = json.Unmarshal(resp.Kvs[0].Value, n)
	if err != nil {
		return nil, err
	}

	return n, nil
}
