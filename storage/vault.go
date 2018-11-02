package storage

import "context"

// GetVaultUnsealKey returns vault unseal key from storage
func (s Storage) GetVaultUnsealKey(ctx context.Context) (string, error) {
	resp, err := s.etcd.Get(ctx, KeyVaultUnsealKey)
	if err != nil {
		return "", err
	}
	if resp.Count == 0 {
		return "", ErrNotFound
	}
	return string(resp.Kvs[0].Value), nil
}

// PutVaultUnsealKey stores vault unseal key to storage
func (s Storage) PutVaultUnsealKey(ctx context.Context, key string) error {
	_, err := s.etcd.Put(ctx, KeyVaultUnsealKey, key)
	return err
}

// GetVaultRootToken returns vault root token from storage
func (s Storage) GetVaultRootToken(ctx context.Context) (string, error) {
	resp, err := s.etcd.Get(ctx, KeyVaultRootToken)
	if err != nil {
		return "", err
	}
	if resp.Count == 0 {
		return "", ErrNotFound
	}
	return string(resp.Kvs[0].Value), nil
}

// PutVaultRootToken stores vault root token to storage
func (s Storage) PutVaultRootToken(ctx context.Context, token string) error {
	_, err := s.etcd.Put(ctx, KeyVaultRootToken, token)
	return err
}

// DeleteVaultRootToken delete vault root token from storage
func (s Storage) DeleteVaultRootToken(ctx context.Context) error {
	_, err := s.etcd.Delete(ctx, KeyVaultRootToken)
	return err
}
