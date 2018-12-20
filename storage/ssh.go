package storage

import (
	"context"
)

// PutSSHPubkey registers a SSH public key for nodes.
func (s Storage) PutSSHPubkey(ctx context.Context, key string) error {
	return s.put(ctx, KeySSHPubkey, key)
}

// GetSSHPubkey retrieves a SSH public key for nodes.
func (s Storage) GetSSHPubkey(ctx context.Context) (string, error) {
	return s.get(ctx, KeySSHPubkey)
}
