package sss

import (
	"context"
	"testing"
	"time"

	sabakan "github.com/cybozu-go/sabakan/v2"

	serf "github.com/hashicorp/serf/client"
	"github.com/prometheus/prom2json"
)

func TestControllerRun(t *testing.T) {
	t.Parallel()

	machineType1 := &machineType{
		Name: "boot",
		MetricsCheckList: []targetMetric{
			{
				Name: "parts1",
				Selector: &selector{
					Labels: map[string]string{"aaa": "bbb"},
				},
			},
		},
	}

	gql := newMockGQLClient()
	ctr := &Controller{
		interval:      time.Minute,
		parallelSize:  2,
		sabakanClient: gql,
		prom:          newMockPromClient(),
		machineTypes:  []*machineType{machineType1},
		machineStateSources: []*MachineStateSource{
			{
				serial: "00000001",
				ipv4:   "10.0.0.100",
				serfStatus: &serf.Member{
					Status: "alive",
					Tags: map[string]string{
						systemdUnitsFailedTag: "",
					},
				},
				machineType: machineType1,
				metrics: map[string]machineMetrics{
					"parts1": {
						prom2json.Metric{
							Labels: map[string]string{"aaa": "bbb"},
							Value:  monitorHWStatusHealth,
						},
					},
				},
			},
		},
	}

	err := ctr.run(context.Background())
	if err != nil {
		t.Error(err)
	}
	if gql.machines[0].Status.State != sabakan.MachineState(sabakan.StateHealthy.GQLEnum()) {
		t.Errorf("machine is not healthy: %s", gql.machines[0].Status.State)
	}
}
