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
		message  string
		expected string
		mss      machineStateSource
	}{
		{
			message:  "cannot get serf status",
			expected: sabakan.StateUnreachable.GQLEnum(),
			mss: machineStateSource{
				serfStatus: nil,
			},
		},
		{
			message:  "failed",
			expected: sabakan.StateUnreachable.GQLEnum(),
			mss: machineStateSource{
				serfStatus: &serf.Member{
					Status: "failed",
					Tags: map[string]string{
						"systemd-units-failed": "",
					},
				},
			},
		},
		{
			message:  "alive, but units failed",
			expected: sabakan.StateUnhealthy.GQLEnum(),
			mss: machineStateSource{
				serfStatus: &serf.Member{
					Status: "alive",
					Tags: map[string]string{
						"systemd-units-failed": "aaa",
					},
				},
			},
		},
		{
			message:  "alive, but `systemd-units-failed` is not set, and metrics are healthy",
			expected: noStateTransition,
			mss: machineStateSource{
				serfStatus: &serf.Member{
					Status: "alive",
				},
				metrics: map[string]machineMetrics{
					"parts1": {
						prom2json.Metric{
							Labels: map[string]string{"aaa": "bbb"},
							Value:  monitorHWStatusHealth,
						},
					},
				},
				machineType: &machineType{
					Name: "boot",
					MetricsCheckList: []metric{{
						Name:   "parts1",
						Labels: map[string]string{"aaa": "bbb"},
					}},
				},
			},
		},
		{
			message:  "alive, but `systemd-units-failed` is not set, and metrics are null",
			expected: sabakan.StateUnhealthy.GQLEnum(),
			mss: machineStateSource{
				serfStatus: &serf.Member{
					Status: "alive",
				},
				metrics: map[string]machineMetrics{
					"parts1": {
						prom2json.Metric{
							Labels: map[string]string{"aaa": "bbb"},
							Value:  monitorHWStatusNull,
						},
					},
				},
				machineType: &machineType{
					Name: "boot",
					MetricsCheckList: []metric{{
						Name:   "parts1",
						Labels: map[string]string{"aaa": "bbb"},
					}},
				},
			},
		},
		{
			message:  "failed, and metrics are warning",
			expected: sabakan.StateUnreachable.GQLEnum(),
			mss: machineStateSource{
				serfStatus: &serf.Member{
					Status: "failed",
					Tags: map[string]string{
						"systemd-units-failed": "",
					},
				},
				metrics: map[string]machineMetrics{
					"parts1": {
						prom2json.Metric{
							Labels: map[string]string{"aaa": "bbb"},
							Value:  monitorHWStatusWarning,
						},
					},
				},
				machineType: &machineType{
					Name: "boot",
					MetricsCheckList: []metric{{
						Name:   "parts1",
						Labels: map[string]string{"aaa": "bbb"},
					}},
				},
			},
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
		expected string
		mss      machineStateSource
	}{
		{
			message:  "If checklist is empty, returns healthy",
			expected: sabakan.StateHealthy.GQLEnum(),
			mss: machineStateSource{
				serfStatus: base,
				metrics:    nil,
				machineType: &machineType{
					Name:             "boot",
					MetricsCheckList: []metric{},
				},
			},
		},
		{
			message:  "If metrics is nil, returns unhealthy",
			expected: sabakan.StateUnhealthy.GQLEnum(),
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
		},
		{
			message:  "Target metric exists, and it is healthy",
			expected: sabakan.StateHealthy.GQLEnum(),
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
		},
		{
			message:  "Target metrics exist, and it is warning",
			expected: sabakan.StateUnhealthy.GQLEnum(),
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
		},
		{
			message:  "Target metrics exist, and they are healthy",
			expected: sabakan.StateHealthy.GQLEnum(),
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
		},
		{
			message:  "Target metrics exist, and they are healthy (multiple labels)",
			expected: sabakan.StateHealthy.GQLEnum(),
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
		},
		{
			message:  "There are multiple matching metrics, and the one of them is broken",
			expected: sabakan.StateUnhealthy.GQLEnum(),
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
		},
		{
			message:  "Target metric exists, but label is not matched",
			expected: sabakan.StateUnhealthy.GQLEnum(),
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
		},
		{
			message:  "Target label is not specified, and the all of metrics is healthy",
			expected: sabakan.StateHealthy.GQLEnum(),
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
		},
		{
			message:  "Target's label is not specified, and there is an unhealthy metric",
			expected: sabakan.StateUnhealthy.GQLEnum(),
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
		},
		{
			message:  "parts2 is not found",
			expected: sabakan.StateUnhealthy.GQLEnum(),
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
		},
	}

	for _, tc := range testCases {
		out := decideSabakanState(tc.mss)
		if out != tc.expected {
			t.Error(tc.message, "expected:", tc.expected, "actual:", out)
		}
	}
}
