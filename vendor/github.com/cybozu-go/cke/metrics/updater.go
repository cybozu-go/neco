package metrics

import (
	"context"
	"time"

	"github.com/cybozu-go/cke"
)

func alwaysAvailable(_ context.Context, _ storage) (bool, error) {
	return true, nil
}

var isLeader bool

// UpdateLeader updates "leader".
func UpdateLeader(flag bool) {
	if flag {
		leader.Set(1)
	} else {
		leader.Set(0)
	}
	isLeader = flag
}

// UpdateOperationPhase updates "operation_phase" and its timestamp.
func UpdateOperationPhase(phase cke.OperationPhase, ts time.Time) {
	for _, labelPhase := range cke.AllOperationPhases {
		if labelPhase == phase {
			operationPhase.WithLabelValues(string(labelPhase)).Set(1)
		} else {
			operationPhase.WithLabelValues(string(labelPhase)).Set(0)
		}
	}
	operationPhaseTimestampSeconds.Set(float64(ts.Unix()))
}

func isOperationPhaseAvailable(_ context.Context, _ storage) (bool, error) {
	return isLeader, nil
}

// UpdateSabakanIntegration updates Sabakan integration metrics.
func UpdateSabakanIntegration(isSuccessful bool, workersByRole map[string]int, unusedMachines int, ts time.Time) {
	sabakanIntegrationTimestampSeconds.Set(float64(ts.Unix()))
	if !isSuccessful {
		sabakanIntegrationSuccessful.Set(0)
		return
	}

	sabakanIntegrationSuccessful.Set(1)
	for role, num := range workersByRole {
		sabakanWorkers.WithLabelValues(role).Set(float64(num))
	}
	sabakanUnusedMachines.Set(float64(unusedMachines))
}

func isSabakanIntegrationAvailable(ctx context.Context, st storage) (bool, error) {
	if !isLeader {
		return false, nil
	}

	disabled, err := st.IsSabakanDisabled(ctx)
	if err != nil {
		return false, err
	}
	return !disabled, nil
}
