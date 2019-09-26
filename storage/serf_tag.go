package storage

import "context"

// PutSerfTagProxy stores serf tags proxy config to storage.
func (s Storage) PutSerfTagProxy(ctx context.Context, proxy string) error {
	return s.put(ctx, KeySerfTagProxy, proxy)
}

// GetSerfTagProxy returns serf tags proxy config from storage.
func (s Storage) GetSerfTagProxy(ctx context.Context) (string, error) {
	return s.get(ctx, KeySerfTagProxy)
}
