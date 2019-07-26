package worker

import (
	"context"
	"os"

	"github.com/cybozu-go/neco"
)

func (o *operator) UpdateSabakanStateSetter(ctx context.Context, req *neco.UpdateRequest) error {
	// Old sabakan-state-setter uses systemd's timer. It's not necessary now.
	filename := neco.TimerFile("sabakan-state-setter")
	_, err := os.Stat(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	err = neco.StopTimer(ctx, "sabakan-state-setter")
	if err != nil {
		return err
	}
	err = neco.DisableTimer(ctx, "sabakan-state-setter")
	if err != nil {
		return err
	}
	err = os.Remove(filename)
	if err != nil {
		return err
	}

	return neco.RestartService(ctx, neco.SabakanStateSetterService)
}
