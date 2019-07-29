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
		Name: "cs",
		MetricsCheckList: []targetMetric{
			{
				Name: "hw_systems_processors_status_health",
			},
			{
				Name: "hw_systems_storage_drives_status_health",
				Selector: &selector{
					LabelPrefix: map[string]string{"device": "PCIeSSD.Slot."},
				},
				MinimumHealthyCount: intPointer(2),
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
					"hw_systems_processors_status_health": {
						prom2json.Metric{
							Labels: map[string]string{"processor": "CPU.Socket.1"},
							Value:  monitorHWStatusHealth,
						},
						prom2json.Metric{
							Labels: map[string]string{"processor": "CPU.Socket.2"},
							Value:  monitorHWStatusWarning,
						},
					},
					"hw_systems_storage_drives_status_health": {
						prom2json.Metric{
							Labels: map[string]string{"device": "Disk.Direct.1-1:AHCI.Slot.1-1"},
							Value:  monitorHWStatusHealth,
						},
						prom2json.Metric{
							Labels: map[string]string{"device": "PCIeSSD.Slot.2-1"},
							Value:  monitorHWStatusHealth,
						},
						prom2json.Metric{
							Labels: map[string]string{"device": "PCIeSSD.Slot.3-1"},
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
	if gql.machines[0].Status.State != sabakan.MachineState(sabakan.StateUnhealthy.GQLEnum()) {
		t.Errorf("machine is not unhealthy: %s", gql.machines[0].Status.State)
	}
}
