package worker

import (
	"context"
	"os"

	"github.com/cybozu-go/neco"
	"github.com/cybozu-go/neco/storage"
)

func (o *operator) fetchContainer(ctx context.Context, name string) error {
	p, err := o.storage.GetProxyConfig(ctx)
	if err != nil && err != storage.ErrNotFound {
		return err
	}

	var env []string
	if p != "" {
		env = []string{"https_proxy=" + p, "http_proxy=" + p}
		env = append(env, os.Environ()...)
	}

	return neco.FetchContainer(ctx, name, env)
}
