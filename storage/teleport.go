package storage

import (
	"context"
)

// GetTeleportAuthToken returns auth token
func (s Storage) GetTeleportAuthToken(ctx context.Context) (string, error) {
	return s.get(ctx, KeyTeleportAuthToken)
}

// PutTeleportAuthToken stores auth token
func (s Storage) PutTeleportAuthToken(ctx context.Context, token string) error {
	return s.put(ctx, KeyTeleportAuthToken, token)
}
