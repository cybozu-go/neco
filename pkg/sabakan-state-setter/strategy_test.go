package sss

import (
	"fmt"
	"testing"

	"github.com/cybozu-go/sabakan/v2"
	dto "github.com/prometheus/client_model/go"
)

type machineMetrics struct {
	Labels map[string]string
	Value  float64
}

func (m machineMetrics) toMetric() *dto.Metric {
	var labels []*dto.LabelPair
	for k, v := range m.Labels {
		k := k
		v := v
		labels = append(labels, &dto.LabelPair{
			Name:  &k,
			Value: &v,
		})
	}
	return &dto.Metric{
		Label: labels,
		Gauge: &dto.Gauge{
			Value: &m.Value,
		},
	}
}

func (m machineMetrics) toMetrics(name string) map[string]*dto.MetricFamily {
	return map[string]*dto.MetricFamily{
		name: {
			Name:   &name,
			Metric: []*dto.Metric{m.toMetric()},
		},
	}
}

func TestCheckSpecifyTarget(t *testing.T) {
	metrics := machineMetrics{
		Labels: map[string]string{"k1": "v1", "k2": "v2"},
		Value:  monitorHWStatusOK,
	}
	ms := machineStateSource{
		serial:  "1234",
		metrics: metrics.toMetrics("m1"),
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
				serfStatus: &serfStatus{
					Status:             "failed",
					SystemdUnitsFailed: strPtr(""),
				},
			},
		},
		{
			message:       "alive, but units failed",
			expected:      sabakan.StateUnhealthy,
			hasTransition: true,
			mss: machineStateSource{
				serfStatus: &serfStatus{
					Status:             "alive",
					SystemdUnitsFailed: strPtr("aaa"),
				},
			},
		},
		{
			message:       "alive, but `systemd-units-failed` is not set, and metrics are healthy",
			expected:      sabakan.MachineState(""),
			hasTransition: false,
			mss: machineStateSource{
				serfStatus: &serfStatus{
					Status: "alive",
				},
				metrics: machineMetrics{
					Labels: map[string]string{"aaa": "bbb"},
					Value:  monitorHWStatusOK,
				}.toMetrics("parts1"),
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
				serfStatus: &serfStatus{
					Status: "alive",
				},
				metrics: machineMetrics{
					Labels: map[string]string{"aaa": "bbb"},
					Value:  monitorHWStatusNull,
				}.toMetrics("parts1"),
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
				serfStatus: &serfStatus{
					Status:             "failed",
					SystemdUnitsFailed: strPtr(""),
				},
				metrics: machineMetrics{
					Labels: map[string]string{"aaa": "bbb"},
					Value:  monitorHWStatusWarning,
				}.toMetrics("parts1"),
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

	base := &serfStatus{
		Status:             "alive",
		SystemdUnitsFailed: strPtr(""),
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
				metrics: machineMetrics{
					Labels: map[string]string{"aaa": "bbb"},
					Value:  monitorHWStatusOK,
				}.toMetrics(parts1),
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
				metrics: machineMetrics{
					Labels: map[string]string{"aaa": "bbb"},
					Value:  monitorHWStatusWarning,
				}.toMetrics(parts1),
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
				metrics: map[string]*dto.MetricFamily{
					parts1: {
						Name: &parts1,
						Metric: []*dto.Metric{
							machineMetrics{
								Labels: map[string]string{"aaa": "bbb"},
								Value:  monitorHWStatusOK,
							}.toMetric(),
						},
					},
					parts2: {
						Name: &parts2,
						Metric: []*dto.Metric{
							machineMetrics{
								Labels: map[string]string{"ccc": "ddd"},
								Value:  monitorHWStatusOK,
							}.toMetric(),
						},
					},
					parts3: {
						Name: &parts3,
						Metric: []*dto.Metric{
							machineMetrics{
								Labels: map[string]string{"ccc": "ddd"},
								Value:  monitorHWStatusWarning,
							}.toMetric(),
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
				metrics: map[string]*dto.MetricFamily{
					parts1: {
						Name: &parts1,
						Metric: []*dto.Metric{
							machineMetrics{
								Labels: map[string]string{"aaa": "bbb", "ccc": "ddd", "eee": "fff"},
								Value:  monitorHWStatusOK,
							}.toMetric(),
							machineMetrics{
								Labels: map[string]string{"aaa": "bbb", "ccc": "ddd"},
								Value:  monitorHWStatusOK,
							}.toMetric(),
						},
					},
					parts2: {
						Name: &parts2,
						Metric: []*dto.Metric{
							machineMetrics{
								Labels: map[string]string{"ccc": "ddd"},
								Value:  monitorHWStatusOK,
							}.toMetric(),
						},
					},
					parts3: {
						Name: &parts3,
						Metric: []*dto.Metric{
							machineMetrics{
								Labels: map[string]string{"ccc": "ddd"},
								Value:  monitorHWStatusWarning,
							}.toMetric(),
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
				metrics: map[string]*dto.MetricFamily{
					parts1: {
						Name: &parts1,
						Metric: []*dto.Metric{
							machineMetrics{
								Labels: map[string]string{"aaa": "bbb", "ccc": "ddd", "eee": "fff"},
								Value:  monitorHWStatusOK,
							}.toMetric(),
							machineMetrics{
								Labels: map[string]string{"aaa": "bbb", "ccc": "ddd"},
								Value:  monitorHWStatusWarning,
							}.toMetric(),
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
				metrics: machineMetrics{
					Labels: map[string]string{"aaa": "bbb"},
					Value:  monitorHWStatusOK,
				}.toMetrics(parts1),
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
				metrics: map[string]*dto.MetricFamily{
					parts1: {
						Name: &parts1,
						Metric: []*dto.Metric{
							machineMetrics{
								Labels: map[string]string{"aaa": "bbb"},
								Value:  monitorHWStatusOK,
							}.toMetric(),
							machineMetrics{
								Labels: map[string]string{"ccc": "ddd"},
								Value:  monitorHWStatusOK,
							}.toMetric(),
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
				metrics: map[string]*dto.MetricFamily{
					parts1: {
						Name: &parts1,
						Metric: []*dto.Metric{
							machineMetrics{
								Labels: map[string]string{"aaa": "bbb"},
								Value:  monitorHWStatusOK,
							}.toMetric(),
							machineMetrics{
								Labels: map[string]string{"ccc": "ddd"},
								Value:  monitorHWStatusWarning,
							}.toMetric(),
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
				metrics: machineMetrics{
					Labels: map[string]string{"aaa": "bbb"},
					Value:  monitorHWStatusOK,
				}.toMetrics(parts1),
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
				metrics: machineMetrics{
					Labels: map[string]string{"aaa": "bbb"},
					Value:  monitorHWStatusOK,
				}.toMetrics(parts1),
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
				metrics: machineMetrics{
					Labels: map[string]string{"key1": "val1", "key2": "val2"},
					Value:  monitorHWStatusOK,
				}.toMetrics(parts1),
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
				metrics: machineMetrics{
					Labels: map[string]string{"key1": "val1", "key2": "val2"},
					Value:  monitorHWStatusOK,
				}.toMetrics(parts1),
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
				metrics: machineMetrics{
					Labels: map[string]string{"key1": "val1", "key2": "val2"},
					Value:  monitorHWStatusWarning,
				}.toMetrics(parts1),
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
				metrics: machineMetrics{
					Labels: map[string]string{"key1": "val1", "key2": "val2"},
					Value:  monitorHWStatusOK,
				}.toMetrics(parts1),
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
				metrics: machineMetrics{
					Labels: map[string]string{"key1": "val1", "key2": "val2"},
					Value:  monitorHWStatusOK,
				}.toMetrics(parts1),
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
				metrics: map[string]*dto.MetricFamily{
					parts1: {
						Name: &parts1,
						Metric: []*dto.Metric{
							machineMetrics{
								Labels: map[string]string{"device": "HDD.slot.1"},
								Value:  monitorHWStatusOK,
							}.toMetric(),
							machineMetrics{
								Labels: map[string]string{"device": "HDD.slot.2"},
								Value:  monitorHWStatusOK,
							}.toMetric(),
							machineMetrics{
								Labels: map[string]string{"device": "HDD.slot.3"},
								Value:  monitorHWStatusWarning,
							}.toMetric(),
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
				metrics: map[string]*dto.MetricFamily{
					parts1: {
						Name: &parts1,
						Metric: []*dto.Metric{
							machineMetrics{
								Labels: map[string]string{"device": "HDD.slot.1"},
								Value:  monitorHWStatusOK,
							}.toMetric(),
							machineMetrics{
								Labels: map[string]string{"device": "HDD.slot.2"},
								Value:  monitorHWStatusWarning,
							}.toMetric(),
							machineMetrics{
								Labels: map[string]string{"device": "HDD.slot.3"},
								Value:  monitorHWStatusWarning,
							}.toMetric(),
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
