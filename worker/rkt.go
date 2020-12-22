package worker

import (
	"context"

	"github.com/cybozu-go/neco"
	"github.com/cybozu-go/neco/storage"
	"github.com/cybozu-go/well"
)

func (o *operator) FetchImages(ctx context.Context, req *neco.UpdateRequest) error {
	p, err := o.storage.GetProxyConfig(ctx)
	if err != nil && err != storage.ErrNotFound {
		return err
	}
	envvars := neco.HTTPProxyEnv(p)

	env := well.NewEnvironment(ctx)
	for _, img := range neco.BootImages {
		img := img
		env.Go(func(ctx context.Context) error {
			fullname, err := neco.ContainerFullName(img)
			if err != nil {
				return err
			}
			return neco.FetchContainer(ctx, fullname, envvars)
		})
	}
	env.Stop()
	return env.Wait()
}
