package sss

import (
	"testing"
	"time"

	serf "github.com/hashicorp/serf/client"
	"github.com/prometheus/prom2json"
)


func TestControllerRun(t *testing.T) {
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

	_ = &Controller{
		interval:     time.Minute,
		parallelSize: 2,
		//sabakanClient: ,
		machineTypes: []*machineType{machineType1},
		machineStateSources: []*machineStateSource{
			&machineStateSource{
				serial: "00000001",
				ipv4:   "10.0.0.100",
				serfStatus: &serf.Member{
					Status: "alive",
					Tags:   map[string]string{},
				},
				machineType: machineType1,
				metrics: map[string]machineMetrics{
					"parts1_status_health": {
						prom2json.Metric{
							Labels: map[string]string{"aaa": "bbb"},
							Value:  monitorHWStatusHealth,
						},
					},
				},
			},
		},
	}
}
