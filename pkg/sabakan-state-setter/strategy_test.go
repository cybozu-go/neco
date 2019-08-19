package sss

import (
	"fmt"
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
	ms := machineStateSource{
		serial: "1234",
		metrics: map[string]machineMetrics{
			"m1": metrics,
		},
	}
	checkTarget := targetMetric{
		Name: "m1",
		Selector: &selector{
			Labels: map[string]string{"k1": "v1"},
		},
	}

	if res := ms.checkTarget(checkTarget); res != sabakan.StateHealthy {
		t.Error("ms.checkTarget(checkTarget) != sabakan.StateHealthy", res)
	}
}

func TestDecideSabakanState(t *testing.T) {
	testCases := []struct {
		message       string
		hasTransition bool
		expected      sabakan.MachineState
		mss           machineStateSource
	}{
		{
			message:       "cannot get serf status",
			expected:      sabakan.StateUnreachable,
			hasTransition: true,
			mss: machineStateSource{
				serfStatus: nil,
			},
		},
		{
			message:       "failed",
			expected:      sabakan.StateUnreachable,
			hasTransition: true,
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
			message:       "alive, but units failed",
			expected:      sabakan.StateUnhealthy,
			hasTransition: true,
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
			message:       "alive, but `systemd-units-failed` is not set, and metrics are healthy",
			expected:      sabakan.MachineState(""),
			hasTransition: false,
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
					MetricsCheckList: []targetMetric{{
						Name: "parts1",
						Selector: &selector{
							Labels: map[string]string{"aaa": "bbb"},
						},
					}},
				},
			},
		},
		{
			message:       "alive, but `systemd-units-failed` is not set, and metrics are null",
			expected:      sabakan.StateUnhealthy,
			hasTransition: true,
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
					MetricsCheckList: []targetMetric{{
						Name: "parts1",
						Selector: &selector{
							Labels: map[string]string{"aaa": "bbb"},
						},
					}},
				},
			},
		},
		{
			message:       "failed, and metrics are warning",
			expected:      sabakan.StateUnreachable,
			hasTransition: true,
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
					MetricsCheckList: []targetMetric{{
						Name: "parts1",
						Selector: &selector{
							Labels: map[string]string{"aaa": "bbb"},
						},
					}},
				},
			},
		},
	}

	for _, tc := range testCases {
		out, hasTransition := tc.mss.decideMachineStateCandidate()
		if hasTransition != tc.hasTransition || out != tc.expected {
			t.Error(tc.message, "expected:", tc.expected, "actual:", out, "hasTransition:", hasTransition)
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
		expected sabakan.MachineState
		mss      machineStateSource
	}{
		{
			message:  "If checklist is empty, returns healthy",
			expected: sabakan.StateHealthy,
			mss: machineStateSource{
				serfStatus: base,
				metrics:    nil,
				machineType: &machineType{
					Name:             "boot",
					MetricsCheckList: []targetMetric{},
				},
			},
		},
		{
			message:  "If metrics is nil, returns unhealthy",
			expected: sabakan.StateUnhealthy,
			mss: machineStateSource{
				serfStatus: base,
				metrics:    nil,
				machineType: &machineType{
					Name: "boot",
					MetricsCheckList: []targetMetric{{
						Name: "boot",
						Selector: &selector{
							Labels: map[string]string{"aaa": "bbb"},
						},
					}},
				},
			},
		},
		{
			message:  "Target metric exists, and it is healthy",
			expected: sabakan.StateHealthy,
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
					MetricsCheckList: []targetMetric{{
						Name: parts1,
						Selector: &selector{
							Labels: map[string]string{"aaa": "bbb"},
						},
					}},
				},
			},
		},
		{
			message:  "Target metrics exist, and it is warning",
			expected: sabakan.StateUnhealthy,
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
					MetricsCheckList: []targetMetric{{
						Name: parts1,
						Selector: &selector{
							Labels: map[string]string{"aaa": "bbb"},
						},
					}},
				},
			},
		},
		{
			message:  "Target metrics exist, and they are healthy",
			expected: sabakan.StateHealthy,
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
					MetricsCheckList: []targetMetric{
						{
							Name: parts1,
							Selector: &selector{
								Labels: map[string]string{"aaa": "bbb"},
							},
						},
						{
							Name: parts2,
							Selector: &selector{
								Labels: map[string]string{"ccc": "ddd"},
							},
						},
					},
				},
			},
		},
		{
			message:  "Target metrics exist, and they are healthy (multiple labels)",
			expected: sabakan.StateHealthy,
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
					MetricsCheckList: []targetMetric{
						{
							Name: parts1,
							Selector: &selector{
								Labels: map[string]string{"aaa": "bbb", "ccc": "ddd"},
							},
						},
						{
							Name: parts2,
							Selector: &selector{
								Labels: map[string]string{"ccc": "ddd"},
							},
						},
					},
				},
			},
		},
		{
			message:  "There are multiple matching metrics, and the one of them is broken",
			expected: sabakan.StateUnhealthy,
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
					MetricsCheckList: []targetMetric{
						{
							Name: parts1,
							Selector: &selector{
								Labels: map[string]string{"aaa": "bbb", "ccc": "ddd"},
							},
						},
					},
				},
			},
		},
		{
			message:  "Target metric exists, but label is not matched",
			expected: sabakan.StateUnhealthy,
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
					MetricsCheckList: []targetMetric{
						{
							Name: parts1,
							Selector: &selector{
								Labels: map[string]string{"not": "existed"},
							},
						},
					},
				},
			},
		},
		{
			message:  "Target label is not specified, and the all of metrics is healthy",
			expected: sabakan.StateHealthy,
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
					MetricsCheckList: []targetMetric{{Name: parts1}},
				},
			},
		},
		{
			message:  "Target's label is not specified, and there is an unhealthy metric",
			expected: sabakan.StateUnhealthy,
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
					MetricsCheckList: []targetMetric{{Name: parts1}},
				},
			},
		},
		{
			message:  "parts2 is not found",
			expected: sabakan.StateUnhealthy,
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
					MetricsCheckList: []targetMetric{
						{
							Name: parts1,
							Selector: &selector{
								Labels: map[string]string{"aaa": "bbb"},
							},
						},
						{
							Name: parts2,
							Selector: &selector{
								Labels: map[string]string{"aaa": "bbb"},
							},
						},
					},
				},
			},
		},
		{
			message:  "Target metric exists, and prefix is matched",
			expected: sabakan.StateHealthy,
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
					MetricsCheckList: []targetMetric{
						{
							Name: parts1,
							Selector: &selector{
								LabelPrefix: map[string]string{"aaa": "bb"},
							},
						},
					},
				},
			},
		},
		{
			message:  "Target metrics exist, and all prefix labels are matched",
			expected: sabakan.StateHealthy,
			mss: machineStateSource{
				serfStatus: base,
				metrics: map[string]machineMetrics{
					parts1: {
						prom2json.Metric{
							Labels: map[string]string{"key1": "val1", "key2": "val2"},
							Value:  monitorHWStatusHealth,
						},
					},
				},
				machineType: &machineType{
					Name: "boot",
					MetricsCheckList: []targetMetric{
						{
							Name: parts1,
							Selector: &selector{
								LabelPrefix: map[string]string{
									"key1": "val",
									"key2": "va",
								},
							},
						},
					},
				},
			},
		},
		{
			message:  "One of label prefix is not matched",
			expected: sabakan.StateUnhealthy,
			mss: machineStateSource{
				serfStatus: base,
				metrics: map[string]machineMetrics{
					parts1: {
						prom2json.Metric{
							Labels: map[string]string{"key1": "val1", "key2": "val2"},
							Value:  monitorHWStatusHealth,
						},
					},
				},
				machineType: &machineType{
					Name: "boot",
					MetricsCheckList: []targetMetric{
						{
							Name: parts1,
							Selector: &selector{
								LabelPrefix: map[string]string{
									"key1": "val",
									"key2": "foo",
								},
							},
						},
					},
				},
			},
		},
		{
			message:  "label prefix is matched, but the value is not healthy",
			expected: sabakan.StateUnhealthy,
			mss: machineStateSource{
				serfStatus: base,
				metrics: map[string]machineMetrics{
					parts1: {
						prom2json.Metric{
							Labels: map[string]string{"key1": "val1", "key2": "val2"},
							Value:  monitorHWStatusWarning,
						},
					},
				},
				machineType: &machineType{
					Name: "boot",
					MetricsCheckList: []targetMetric{
						{
							Name: parts1,
							Selector: &selector{
								LabelPrefix: map[string]string{
									"key1": "val",
									"key2": "va",
								},
							},
						},
					},
				},
			},
		},
		{
			message:  "both labels and label prefix are declared, all labels are matched but label prefix is not matched",
			expected: sabakan.StateUnhealthy,
			mss: machineStateSource{
				serfStatus: base,
				metrics: map[string]machineMetrics{
					parts1: {
						prom2json.Metric{
							Labels: map[string]string{
								"key1": "val1",
								"key2": "val2",
							},
							Value: monitorHWStatusHealth,
						},
					},
				},
				machineType: &machineType{
					Name: "boot",
					MetricsCheckList: []targetMetric{
						{
							Name: parts1,
							Selector: &selector{
								Labels: map[string]string{
									"key1": "val1",
								},
								LabelPrefix: map[string]string{
									"foo": "bar",
								},
							},
						},
					},
				},
			},
		},
		{
			message:  "both labels and label prefix are declared, all label prefix are matched but labels are not matched",
			expected: sabakan.StateUnhealthy,
			mss: machineStateSource{
				serfStatus: base,
				metrics: map[string]machineMetrics{
					parts1: {
						prom2json.Metric{
							Labels: map[string]string{
								"key1": "val1",
								"key2": "val2",
							},
							Value: monitorHWStatusHealth,
						},
					},
				},
				machineType: &machineType{
					Name: "boot",
					MetricsCheckList: []targetMetric{
						{
							Name: parts1,
							Selector: &selector{
								Labels: map[string]string{
									"key1": "hoge",
								},
								LabelPrefix: map[string]string{
									"key2": "val",
								},
							},
						},
					},
				},
			},
		},
		{
			message:  "minimum healthy count is satisfied",
			expected: sabakan.StateHealthy,
			mss: machineStateSource{
				serfStatus: base,
				metrics: map[string]machineMetrics{
					parts1: {
						prom2json.Metric{
							Labels: map[string]string{"device": "HDD.slot.1"},
							Value:  monitorHWStatusHealth,
						},
						prom2json.Metric{
							Labels: map[string]string{"device": "HDD.slot.2"},
							Value:  monitorHWStatusHealth,
						},
						prom2json.Metric{
							Labels: map[string]string{"device": "HDD.slot.3"},
							Value:  monitorHWStatusWarning,
						},
					},
				},
				machineType: &machineType{
					Name: "boot",
					MetricsCheckList: []targetMetric{
						{
							Name: parts1,
							Selector: &selector{
								LabelPrefix: map[string]string{
									"device": "HDD.",
								},
							},
							MinimumHealthyCount: intPointer(2),
						},
					},
				},
			},
		},
		{
			message:  "minimum healthy count is not satisfied",
			expected: sabakan.StateUnhealthy,
			mss: machineStateSource{
				serfStatus: base,
				metrics: map[string]machineMetrics{
					parts1: {
						prom2json.Metric{
							Labels: map[string]string{"device": "HDD.slot.1"},
							Value:  monitorHWStatusHealth,
						},
						prom2json.Metric{
							Labels: map[string]string{"device": "HDD.slot.2"},
							Value:  monitorHWStatusWarning,
						},
						prom2json.Metric{
							Labels: map[string]string{"device": "HDD.slot.3"},
							Value:  monitorHWStatusWarning,
						},
					},
				},
				machineType: &machineType{
					Name: "boot",
					MetricsCheckList: []targetMetric{
						{
							Name: parts1,
							Selector: &selector{
								LabelPrefix: map[string]string{
									"device": "HDD.",
								},
							},
							MinimumHealthyCount: intPointer(2),
						},
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		fmt.Println("TEST:", tc.message)
		out, _ := tc.mss.decideMachineStateCandidate()
		if out != tc.expected {
			t.Error(tc.message, "| expected:", tc.expected, "actual:", out)
		}
	}
}

func intPointer(i int) *int {
	p := new(int)
	*p = i
	return p
}
