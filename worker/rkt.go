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
	rt, err := neco.GetContainerRuntime(p)
	if err != nil {
		return err
	}

	env := well.NewEnvironment(ctx)
	for _, name := range neco.BootImages {
		img, err := neco.CurrentArtifacts.FindContainerImage(name)
		if err != nil {
			return err
		}
		env.Go(func(ctx context.Context) error {
			return rt.Pull(ctx, img)
		})
	}
	env.Stop()
	return env.Wait()
}
