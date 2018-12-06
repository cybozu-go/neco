package worker

import (
	"context"

	"github.com/cybozu-go/neco"
	"github.com/cybozu-go/neco/storage"
)

func (o *operator) fetchContainer(ctx context.Context, name string) error {
	p, err := o.storage.GetProxyConfig(ctx)
	if err != nil && err != storage.ErrNotFound {
		return err
	}

	env := neco.HTTPProxyEnv(p)

	fullname, err := neco.ContainerFullName(name)
	if err != nil {
		return err
	}
	return neco.FetchContainer(ctx, fullname, env)
}
