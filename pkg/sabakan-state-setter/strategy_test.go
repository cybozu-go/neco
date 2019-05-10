package main

import (
	"testing"

	"github.com/cybozu-go/sabakan/v2"
	serf "github.com/hashicorp/serf/client"
	"github.com/prometheus/prom2json"
)

func TestCheckSpecifyTarget(t *testing.T) {
	metrics := machineMetrics{{
		Labels: map[string]string{"k1": "v1", "k2": "v2"},
		Value:  monitorHWStatusHealth,
	}}
	labels := map[string]string{"k1": "v1"}

	if res := checkSpecifiedTarget(metrics, labels); res != sabakan.StateHealthy.GQLEnum() {
		t.Error("checkSpecifiedTarget(metrics, labels) != sabakan.StateHealthy.GQLEnum()", res)
	}
}

func TestDecideSabakanState(t *testing.T) {
	testCases := []struct {
		mss      machineStateSource
		expected string
		message  string
	}{
		{
			mss: machineStateSource{
				serfStatus: nil,
			},
			expected: sabakan.StateUnreachable.GQLEnum(),
			message:  "cannot get serf status",
		}, {
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
	parts1 := "parts1_status_health"
	parts2 := "parts2_status_health"
	parts3 := "parts3_status_health"

	base := &serf.Member{
		Status: "alive",
		Tags:   map[string]string{"systemd-units-failed": ""},
	}
	testCases := []struct {
		message  string
		mss      machineStateSource
		expected string
	}{
		{
			message: "If checklist is empty, returns healthy",
			mss: machineStateSource{
				serfStatus: base,
				metrics:    nil,
				machineType: &machineType{
					Name:             "boot",
					MetricsCheckList: []metric{},
				},
			},
			expected: sabakan.StateHealthy.GQLEnum(),
		},
		{
			message: "If metrics is nil, returns unhealthy",
			mss: machineStateSource{
				serfStatus: base,
				metrics:    nil,
				machineType: &machineType{
					Name: "boot",
					MetricsCheckList: []metric{{
						Name:   "boot",
						Labels: map[string]string{"aaa": "bbb"},
					}},
				},
			},
			expected: sabakan.StateUnhealthy.GQLEnum(),
		},
		{
			message: "Target metric exists, and it is healthy",
			mss: machineStateSource{
				serfStatus: base,
				metrics: map[string]machineMetrics{
					parts1: {
						prom2json.Metric{
							Labels: map[string]string{"aaa": "bbb"},
							Value:  monitorHWStatusHealth,
						},
					},
				},
				machineType: &machineType{
					Name: "boot",
					MetricsCheckList: []metric{{
						Name:   parts1,
						Labels: map[string]string{"aaa": "bbb"}},
					},
				},
			},
			expected: sabakan.StateHealthy.GQLEnum(),
		},
		{
			message: "Target metrics exist, and it is healthy",
			mss: machineStateSource{
				serfStatus: base,
				metrics: map[string]machineMetrics{
					parts1: {
						prom2json.Metric{
							Labels: map[string]string{"aaa": "bbb"},
							Value:  monitorHWStatusWarning,
						},
					},
				},
				machineType: &machineType{
					Name: "boot",
					MetricsCheckList: []metric{{
						Name:   parts1,
						Labels: map[string]string{"aaa": "bbb"},
					}},
				},
			},
			expected: sabakan.StateUnhealthy.GQLEnum(),
		},
		{
			message: "Target metrics exist, and they are healthy",
			mss: machineStateSource{
				serfStatus: base,
				metrics: map[string]machineMetrics{
					parts1: {
						prom2json.Metric{
							Labels: map[string]string{"aaa": "bbb"},
							Value:  monitorHWStatusHealth,
						},
					},
					parts2: {
						prom2json.Metric{
							Labels: map[string]string{"ccc": "ddd"},
							Value:  monitorHWStatusHealth,
						},
					},
					parts3: {
						prom2json.Metric{
							Labels: map[string]string{"ccc": "ddd"},
							Value:  monitorHWStatusWarning,
						},
					},
				},
				machineType: &machineType{
					Name: "boot",
					MetricsCheckList: []metric{
						{
							Name:   parts1,
							Labels: map[string]string{"aaa": "bbb"},
						},
						{
							Name:   parts2,
							Labels: map[string]string{"ccc": "ddd"},
						},
					},
				},
			},
			expected: sabakan.StateHealthy.GQLEnum(),
		},
		{
			message: "Target metrics exist, and they are healthy (multiple labels)",
			mss: machineStateSource{
				serfStatus: base,
				metrics: map[string]machineMetrics{
					parts1: {
						prom2json.Metric{
							Labels: map[string]string{"aaa": "bbb", "ccc": "ddd", "eee": "fff"},
							Value:  monitorHWStatusHealth,
						},
						prom2json.Metric{
							Labels: map[string]string{"aaa": "bbb", "ccc": "ddd"},
							Value:  monitorHWStatusHealth,
						},
					},
					parts2: {
						prom2json.Metric{
							Labels: map[string]string{"ccc": "ddd"},
							Value:  monitorHWStatusHealth,
						},
					},
					parts3: {
						prom2json.Metric{
							Labels: map[string]string{"ccc": "ddd"},
							Value:  monitorHWStatusWarning,
						},
					},
				},
				machineType: &machineType{
					Name: "boot",
					MetricsCheckList: []metric{
						{
							Name:   parts1,
							Labels: map[string]string{"aaa": "bbb", "ccc": "ddd"},
						},
						{
							Name:   parts2,
							Labels: map[string]string{"ccc": "ddd"},
						},
					},
				},
			},
			expected: sabakan.StateHealthy.GQLEnum(),
		},
		{
			message: "There are multiple matching metrics, and the one of them is broken",
			mss: machineStateSource{
				serfStatus: base,
				metrics: map[string]machineMetrics{
					parts1: {
						prom2json.Metric{
							Labels: map[string]string{"aaa": "bbb", "ccc": "ddd", "eee": "fff"},
							Value:  monitorHWStatusHealth,
						},
						prom2json.Metric{
							Labels: map[string]string{"aaa": "bbb", "ccc": "ddd"},
							Value:  monitorHWStatusWarning,
						},
					},
				},
				machineType: &machineType{
					Name: "boot",
					MetricsCheckList: []metric{
						{
							Name:   parts1,
							Labels: map[string]string{"aaa": "bbb", "ccc": "ddd"},
						},
					},
				},
			},
			expected: sabakan.StateUnhealthy.GQLEnum(),
		},
		{
			message: "Target metric exists, but label is not matched",
			mss: machineStateSource{
				serfStatus: base,
				metrics: map[string]machineMetrics{
					parts1: {
						prom2json.Metric{
							Labels: map[string]string{"aaa": "bbb"},
							Value:  monitorHWStatusHealth,
						},
					},
				},
				machineType: &machineType{
					Name: "boot",
					MetricsCheckList: []metric{
						{
							Name:   parts1,
							Labels: map[string]string{"not": "existed"},
						},
					},
				},
			},
			expected: sabakan.StateUnhealthy.GQLEnum(),
		},
		{
			message: "Target label is not specified, and the all of metrics is healthy",
			mss: machineStateSource{
				serfStatus: base,
				metrics: map[string]machineMetrics{
					parts1: {
						prom2json.Metric{
							Labels: map[string]string{"aaa": "bbb"},
							Value:  monitorHWStatusHealth,
						},
						prom2json.Metric{
							Labels: map[string]string{"ccc": "ddd"},
							Value:  monitorHWStatusHealth,
						},
					},
				},
				machineType: &machineType{
					Name:             "boot",
					MetricsCheckList: []metric{{Name: parts1}},
				},
			},
			expected: sabakan.StateHealthy.GQLEnum(),
		},
		{
			message: "Target's label is not specified, and there is an unhealthy metric",
			mss: machineStateSource{
				serfStatus: base,
				metrics: map[string]machineMetrics{
					parts1: {
						prom2json.Metric{
							Labels: map[string]string{"aaa": "bbb"},
							Value:  monitorHWStatusHealth,
						},
						prom2json.Metric{
							Labels: map[string]string{"ccc": "ddd"},
							Value:  monitorHWStatusWarning,
						},
					},
				},
				machineType: &machineType{
					Name:             "boot",
					MetricsCheckList: []metric{{Name: parts1}},
				},
			},
			expected: sabakan.StateUnhealthy.GQLEnum(),
		},
		{
			message: "parts2 is not found",
			mss: machineStateSource{
				serfStatus: base,
				metrics: map[string]machineMetrics{
					parts1: {
						prom2json.Metric{
							Labels: map[string]string{"aaa": "bbb"},
							Value:  monitorHWStatusHealth,
						},
					},
				},
				machineType: &machineType{
					Name: "boot",
					MetricsCheckList: []metric{
						{
							Name:   parts1,
							Labels: map[string]string{"aaa": "bbb"},
						},
						{
							Name:   parts2,
							Labels: map[string]string{"aaa": "bbb"},
						},
					},
				},
			},
			expected: sabakan.StateUnhealthy.GQLEnum(),
		},
	}

	for _, tc := range testCases {
		out := decideSabakanState(tc.mss)
		if out != tc.expected {
			t.Error(tc.message, "expected:", tc.expected, "actual:", out)
		}
	}
}
