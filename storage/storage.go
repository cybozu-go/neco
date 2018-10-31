package storage

import (
	"context"

	"github.com/coreos/etcd/clientv3"
	"github.com/cybozu-go/neco"
)

type Storage struct {
	etcd *clientv3.Client
}

func (s Storage) DumpArtifacts(ctx context.Context, lrn int) error {
	// TODO
	return nil
}

func (s Storage) GetArtifacts(ctx context.Context, lrn int) (*neco.ArtifactSet, error) {
	// TODO
	return nil, nil
}

func (s Storage) GetBootservers(ctx context.Context) ([]int, error) {
	// TODO
	return nil, nil
}

func (s Storage) PutRequest(ctx context.Context, req neco.UpdateRequest) error {
	// TODO
	return nil
}

func (s Storage) GetRequest(ctx context.Context) (*neco.UpdateRequest, error) {
	// TODO
	return nil, nil
}

func (s Storage) PutStatus(ctx context.Context, lrn int, st neco.UpdateStatus) error {
	// TODO
	return nil
}

func (s Storage) GetStatus(ctx context.Context, lrn int) (*neco.UpdateStatus, error) {
	// TODO
	return nil, nil
}

func (s Storage) PutNotification(ctx context.Context, n neco.Notification) error {
	// TODO
	return nil
}

func (s Storage) GetNotification(ctx context.Context) (*neco.Notification, error) {
	// TODO
	return nil, nil
}

func (s Storage) PutVaultUnsealKey(ctx context.Context, key string) error {
	// TODO
	return nil
}

func (s Storage) PutVaultRootToken(ctx context.Context, token string) error {
	// TODO
	return nil
}
