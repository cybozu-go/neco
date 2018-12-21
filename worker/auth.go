package worker

import (
	"context"

	"github.com/cybozu-go/neco"
	"github.com/cybozu-go/neco/storage"
)

func getDockerAuth(ctx context.Context, st storage.Storage) (*neco.DockerAuth, error) {
	passwd, err := o.storage.GetQuayPassword(ctx)
	if err != nil {
		return nil, err
	}
	username, err := o.storage.GetQuayUsername(ctx)
	if err != nil {
		return nil, err
	}
	return &neco.DockerAuth{
		Username: username,
		Password: passwd,
	}, nil
}
