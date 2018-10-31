package storage

import (
	"context"

	"github.com/coreos/etcd/clientv3"
	"github.com/cybozu-go/neco"
)

// Storage is storage client
type Storage struct {
	etcd *clientv3.Client
}

// DumpArtifactSet stores ArtifactSet to storage
func (s Storage) DumpArtifactSet(ctx context.Context, lrn int) error {
	// TODO
	return nil
}

// GetArtifactSet returns ArtifactSet from storage
func (s Storage) GetArtifactSet(ctx context.Context, lrn int) (*neco.ArtifactSet, error) {
	// TODO
	return nil, nil
}

// GetBootservers returns LRNs of bootservers
func (s Storage) GetBootservers(ctx context.Context) ([]int, error) {
	// TODO
	return nil, nil
}

// PutRequest stores UpdateRequest to storage
func (s Storage) PutRequest(ctx context.Context, req neco.UpdateRequest) error {
	// TODO
	return nil
}

// GetRequest returns UpdateRequest from storage
func (s Storage) GetRequest(ctx context.Context) (*neco.UpdateRequest, error) {
	// TODO
	return nil, nil
}

// PutStatus stores UpdateStatus of bootserver to storage
func (s Storage) PutStatus(ctx context.Context, lrn int, st neco.UpdateStatus) error {
	// TODO
	return nil
}

// GetStatus returns UpdateStatus of bootserver from storage
func (s Storage) GetStatus(ctx context.Context, lrn int) (*neco.UpdateStatus, error) {
	// TODO
	return nil, nil
}

// PutNotificationConfig stores NotificationConfig to storage
func (s Storage) PutNotificationConfig(ctx context.Context, n neco.NotificationConfig) error {
	// TODO
	return nil
}

// GetNotificationConfig returns NotificationConfig from storage
func (s Storage) GetNotificationConfig(ctx context.Context) (*neco.NotificationConfig, error) {
	// TODO
	return nil, nil
}

// PutVaultUnsealKey stores vault unseal key to storage
func (s Storage) PutVaultUnsealKey(ctx context.Context, key string) error {
	// TODO
	return nil
}

// PutVaultRootToken stores vault root token to storage
func (s Storage) PutVaultRootToken(ctx context.Context, token string) error {
	// TODO
	return nil
}
