package storage

import "context"

// GetVaultUnsealKey returns vault unseal key from storage
func (s Storage) GetVaultUnsealKey(ctx context.Context) (string, error) {
	return s.get(ctx, KeyVaultUnsealKey)
}

// PutVaultUnsealKey stores vault unseal key to storage
func (s Storage) PutVaultUnsealKey(ctx context.Context, key string) error {
	return s.put(ctx, KeyVaultUnsealKey, key)
}

// DeleteVaultUnsealKey deletes vault unseal key from storage
func (s Storage) DeleteVaultUnsealKey(ctx context.Context) error {
	return s.del(ctx, KeyVaultUnsealKey)
}

// GetVaultRootToken returns vault root token from storage
func (s Storage) GetVaultRootToken(ctx context.Context) (string, error) {
	return s.get(ctx, KeyVaultRootToken)
}

// PutVaultRootToken stores vault root token to storage
func (s Storage) PutVaultRootToken(ctx context.Context, token string) error {
	return s.put(ctx, KeyVaultRootToken, token)
}

// DeleteVaultRootToken deletes vault root token from storage
func (s Storage) DeleteVaultRootToken(ctx context.Context) error {
	return s.del(ctx, KeyVaultRootToken)
}
