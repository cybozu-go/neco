package worker

import (
	"context"

	"github.com/cybozu-go/neco/progs/sabakan"
	"github.com/cybozu-go/neco/storage"
)

func (o *operator) getDockerAuth(ctx context.Context, st storage.Storage) (*sabakan.DockerAuth, error) {
	passwd, err := o.storage.GetQuayPassword(ctx)
	if err != nil {
		return nil, err
	}
	username, err := o.storage.GetQuayUsername(ctx)
	if err != nil {
		return nil, err
	}
	return &sabakan.DockerAuth{
		Username: username,
		Password: passwd,
	}, nil
}
