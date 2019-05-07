package main

import (
	"testing"

	"github.com/cybozu-go/sabakan/v2"
	serf "github.com/hashicorp/serf/client"
	"github.com/prometheus/prom2json"
)

func TestDecideSabakanState(t *testing.T) {
	testCases := []struct {
		mss      machineStateSource
		expected string
		message  string
	}{
		{
			mss: machineStateSource{
				serfStatus: &serf.Member{
					Status: "failed",
					Tags: map[string]string{
						"systemd-units-failed": "",
					},
				},
			},
			expected: sabakan.StateUnreachable.GQLEnum(),
			message:  "failed",
		}, {
			mss: machineStateSource{
				serfStatus: &serf.Member{
					Status: "alive",
					Tags: map[string]string{
						"systemd-units-failed": "aaa",
					},
				},
			},
			expected: sabakan.StateUnhealthy.GQLEnum(),
			message:  "alive, but units failed",
		}, {
			mss: machineStateSource{
				serfStatus: &serf.Member{
					Status: "alive",
				},
			},
			expected: sabakan.StateUnhealthy.GQLEnum(),
			message:  "alive, but `systemd-units-failed` is not set",
		},
	}

	for _, tc := range testCases {
		out := decideSabakanState(tc.mss)
		if out != tc.expected {
			t.Error(tc.message, "expected:", tc.expected, "actual:", out)
		}
	}
}

func TestDecideByMonitorHW(t *testing.T) {
	base := &serf.Member{
		Status: "alive",
		Tags:   map[string]string{"systemd-units-failed": ""},
	}
	testCases := []struct {
		mss      machineStateSource
		expected string
		message  string
	}{
		{
			mss: machineStateSource{
				serfStatus: base,
				metrics:    []*prom2json.Family{{}},
			},
			expected: stateMetricNotFound,
			message:  "empty metric returns empty string",
		},
		{
			mss: machineStateSource{
				serfStatus: base,
				metrics: []*prom2json.Family{
					{
						Name:    "hw_processor_status_health",
						Metrics: []interface{}{prom2json.Metric{Value: "1"}},
					},
				},
			},
			expected: sabakan.StateUnhealthy.GQLEnum(),
			message:  "CPU is unhealthy",
		},
		{
			mss: machineStateSource{
				serfStatus: base,
				metrics: []*prom2json.Family{
					{
						Name:    "hw_system_memory_summary_status_health",
						Metrics: []interface{}{prom2json.Metric{Value: "1"}},
					},
				},
			},
			expected: sabakan.StateUnhealthy.GQLEnum(),
			message:  "CPU is unhealthy",
		},
		{
			mss: machineStateSource{
				serfStatus: base,
				metrics: []*prom2json.Family{
					{
						Name:    "hw_chassis_temperature_status_health",
						Metrics: []interface{}{prom2json.Metric{Value: "1"}},
					},
				},
			},
			expected: sabakan.StateUnhealthy.GQLEnum(),
			message:  "CPU is unhealthy",
		},
		{
			mss: machineStateSource{
				serfStatus: base,
				metrics: []*prom2json.Family{
					{
						Name:    "hw_chassis_voltage_status_health",
						Metrics: []interface{}{prom2json.Metric{Value: "1"}},
					},
				},
			},
			expected: sabakan.StateUnhealthy.GQLEnum(),
			message:  "CPU is unhealthy",
		},
		{
			mss: machineStateSource{
				serfStatus: base,
				metrics: []*prom2json.Family{
					{
						Name:    "hw_chassis_temperature_status_health",
						Metrics: []interface{}{prom2json.Metric{Value: "0"}},
					},
					{
						Name:    "hw_chassis_voltage_status_health",
						Metrics: []interface{}{prom2json.Metric{Value: "0"}},
					},
				},
			},
			expected: sabakan.StateHealthy.GQLEnum(),
			message:  "healthy",
		},
		{
			mss: machineStateSource{
				serfStatus: base,
				metrics: []*prom2json.Family{
					{
						Name: "hw_storage_controller_status_health",
						Metrics: []interface{}{prom2json.Metric{
							Value: "1",
							Labels: map[string]string{
								"controller": "AHCI.Slot.1",
							},
						}},
					},
				},
			},
			expected: sabakan.StateUnhealthy.GQLEnum(),
			message:  "AHCI.Slot is unhealthy",
		},
		{
			mss: machineStateSource{
				serfStatus: base,
				metrics: []*prom2json.Family{
					{
						Name: "hw_storage_controller_status_health",
						Metrics: []interface{}{prom2json.Metric{
							Value: "1",
							Labels: map[string]string{
								"controller": "PCIeSSD.Slot.1",
							},
						}},
					},
				},
			},
			expected: sabakan.StateUnhealthy.GQLEnum(),
			message:  "PCIeSSD.Slot is unhealthy",
		},
		{
			mss: machineStateSource{
				serfStatus: base,
				metrics: []*prom2json.Family{
					{
						Name: "hw_storage_controller_status_health",
						Metrics: []interface{}{prom2json.Metric{
							Value: "0",
							Labels: map[string]string{
								"controller": "PCIeSSD.Slot.1",
							},
						}},
					},
				},
			},
			expected: sabakan.StateHealthy.GQLEnum(),
			message:  "PCIeSSD.Slot is healthy",
		},
		{
			mss: machineStateSource{
				serfStatus: base,
				metrics: []*prom2json.Family{
					{
						Name: "hw_storage_device_status_health",
						Metrics: []interface{}{prom2json.Metric{
							Value: "0",
						}},
					},
				},
			},
			expected: sabakan.StateHealthy.GQLEnum(),
			message:  "storage device is healthy",
		},
		{
			mss: machineStateSource{
				serfStatus: base,
				metrics: []*prom2json.Family{
					{
						Name: "hw_storage_device_status_health",
						Metrics: []interface{}{prom2json.Metric{
							Value: "1",
						}},
					},
				},
			},
			expected: sabakan.StateUnhealthy.GQLEnum(),
			message:  "storage device is unhealthy",
		},
	}

	for _, tc := range testCases {
		out := decideSabakanState(tc.mss)
		if out != tc.expected {
			t.Error(tc.message, "expected:", tc.expected, "actual:", out)
		}
	}
}
