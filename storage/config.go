package storage

import (
	"context"
	"strconv"
	"time"

	"github.com/cybozu-go/neco"
)

// PutEnvConfig stores proxy config to storage.
func (s Storage) PutEnvConfig(ctx context.Context, env string) error {
	return s.put(ctx, KeyEnv, env)
}

// GetEnvConfig returns proxy config from storage.
func (s Storage) GetEnvConfig(ctx context.Context) (string, error) {
	return s.get(ctx, KeyEnv)
}

// PutSlackNotification stores SlackNotification to storage
func (s Storage) PutSlackNotification(ctx context.Context, url string) error {
	return s.put(ctx, KeyNotificationSlack, url)
}

// GetSlackNotification returns SlackNotification from storage
// If not found, this returns ErrNotFound.
func (s Storage) GetSlackNotification(ctx context.Context) (string, error) {
	return s.get(ctx, KeyNotificationSlack)
}

// PutProxyConfig stores proxy config to storage.
func (s Storage) PutProxyConfig(ctx context.Context, proxy string) error {
	return s.put(ctx, KeyProxy, proxy)
}

// GetProxyConfig returns proxy config from storage.
func (s Storage) GetProxyConfig(ctx context.Context) (string, error) {
	return s.get(ctx, KeyProxy)
}

// PutCheckUpdateInterval stores check-update-interval config to storage.
func (s Storage) PutCheckUpdateInterval(ctx context.Context, d time.Duration) error {
	data := strconv.FormatInt(int64(d), 10)
	return s.put(ctx, KeyCheckUpdateInterval, data)
}

// GetCheckUpdateInterval returns check-update-interval config from storage. It
// returns default value if the key does not exist.
func (s Storage) GetCheckUpdateInterval(ctx context.Context) (time.Duration, error) {
	data, err := s.get(ctx, KeyCheckUpdateInterval)
	if err == ErrNotFound {
		return neco.DefaultCheckUpdateInterval, nil
	}
	if err != nil {
		return 0, err
	}
	i, err := strconv.ParseInt(data, 10, 64)
	if err != nil {
		return 0, err
	}
	return time.Duration(i), nil
}

// PutWorkerTimeout stores worker-timeout config to storage.
func (s Storage) PutWorkerTimeout(ctx context.Context, d time.Duration) error {
	data := strconv.FormatInt(int64(d), 10)
	return s.put(ctx, KeyWorkerTimeout, data)
}

// GetWorkerTimeout returns worker-timeout config from storage. It returns
// default value if the key does not exist.
func (s Storage) GetWorkerTimeout(ctx context.Context) (time.Duration, error) {
	data, err := s.get(ctx, KeyWorkerTimeout)
	if err == ErrNotFound {
		return neco.DefaultWorkerTimeout, nil
	}
	if err != nil {
		return 0, err
	}
	i, err := strconv.ParseInt(data, 10, 64)
	if err != nil {
		return 0, err
	}
	return time.Duration(i), nil
}
