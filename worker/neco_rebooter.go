package worker

import (
	"context"

	"github.com/cybozu-go/neco"
)

func (o *operator) UpdateNecoRebooter(ctx context.Context, req *neco.UpdateRequest) error {
	return neco.RestartService(ctx, neco.NecoRebooterService)
}
