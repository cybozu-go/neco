package worker

import (
	"context"

	"github.com/cybozu-go/neco"
	"github.com/cybozu-go/neco/progs/vault"
)

func (o *operator) StopVault(ctx context.Context, req *neco.UpdateRequest) error {
	return nil
}

func (o *operator) UpdateVault(ctx context.Context, req *neco.UpdateRequest) error {
	need, err := o.needContainerImageUpdate(ctx, "vault")
	if err != nil {
		return err
	}
	if need {
		err = o.fetchContainer(ctx, "vault")
		if err != nil {
			return err
		}
		err = vault.InstallTools(ctx)
		if err != nil {
			return err
		}
	}

	return nil
}
