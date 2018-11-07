package worker

import (
	"context"
	"fmt"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/neco"
)

// FinalStep is the largest step of update process.
const FinalStep = 2

func (w *Worker) runStep(ctx context.Context) (bool, error) {
	var err error
	switch w.step {
	case 1:
		err = w.operator.UpdateEtcd(ctx, w.req)
	case 2:
		err = w.operator.UpdateVault(ctx, w.req)
	default:
		return false, fmt.Errorf("unexpected step: %d", w.step)
	}

	if err != nil {
		st := neco.UpdateStatus{
			Version: w.req.Version,
			Step:    w.step,
			Cond:    neco.CondAbort,
			Message: err.Error(),
		}

		err2 := w.storage.PutStatus(ctx, w.mylrn, st)
		if err2 != nil {
			log.Warn("failed to put status", map[string]interface{}{
				log.FnError: err2.Error(),
			})
		}

		return false, err
	}

	if w.step != FinalStep {
		w.step++
		w.barrier = NewBarrier(w.req.Servers)
		st := neco.UpdateStatus{
			Version: w.req.Version,
			Step:    w.step,
			Cond:    neco.CondRunning,
		}
		err = w.storage.PutStatus(ctx, w.mylrn, st)
		if err != nil {
			return false, err
		}
		return false, nil
	}

	st := neco.UpdateStatus{
		Version: w.req.Version,
		Step:    w.step,
		Cond:    neco.CondComplete,
	}
	err = w.storage.PutStatus(ctx, w.mylrn, st)
	if err != nil {
		return false, err
	}
	return true, nil
}
