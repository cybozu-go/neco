package sss

import (
	"testing"

	"github.com/cybozu-go/sabakan/v2"
	"github.com/google/go-cmp/cmp"
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

func (m machineMetrics) toMetricFamily(name string) *dto.MetricFamily {
	return &dto.MetricFamily{
		Name:   &name,
		Metric: []*dto.Metric{m.toMetric()},
	}
}

func (m machineMetrics) toMetricFamilyMap(name string) map[string]*dto.MetricFamily {
	return map[string]*dto.MetricFamily{
		name: m.toMetricFamily(name),
	}
}

func TestDecideMachineState(t *testing.T) {
	testCases := []struct {
		message  string
		expected sabakan.MachineState
		mss      machineStateSource
	}{
		{
			message:  "cannot get serf status",
			expected: sabakan.StateUnreachable,
			mss: machineStateSource{
				serial:     "123456789",
				ipv4:       "10.20.30.40",
				serfStatus: nil,
			},
		},
		{
			message:  "failed",
			expected: sabakan.StateUnreachable,
			mss: machineStateSource{
				serial: "123456789",
				ipv4:   "10.20.30.40",
				serfStatus: &serfStatus{
					Status:             "failed",
					SystemdUnitsFailed: strPtr(""),
				},
			},
		},
		{
			message:  "alive, but units failed",
			expected: sabakan.StateUnhealthy,
			mss: machineStateSource{
				serial: "123456789",
				ipv4:   "10.20.30.40",
				serfStatus: &serfStatus{
					Status:             "alive",
					SystemdUnitsFailed: strPtr("aaa"),
				},
			},
		},
		{
			message:  "alive, but `systemd-units-failed` is not set, and metrics are healthy",
			expected: sabakan.MachineState(""),
			mss: machineStateSource{
				serial: "123456789",
				ipv4:   "10.20.30.40",
				serfStatus: &serfStatus{
					Status: "alive",
				},
				metrics: machineMetrics{
					Labels: map[string]string{"aaa": "bbb"},
					Value:  monitorHWStatusOK,
				}.toMetricFamilyMap("parts1"),
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
			message:  "alive, but `systemd-units-failed` is not set, and metrics are null",
			expected: sabakan.StateUnhealthy,
			mss: machineStateSource{
				serial: "123456789",
				ipv4:   "10.20.30.40",
				serfStatus: &serfStatus{
					Status: "alive",
				},
				metrics: machineMetrics{
					Labels: map[string]string{"aaa": "bbb"},
					Value:  monitorHWStatusNull,
				}.toMetricFamilyMap("parts1"),
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
			message:  "failed, and metrics are warning",
			expected: sabakan.StateUnreachable,
			mss: machineStateSource{
				serial: "123456789",
				ipv4:   "10.20.30.40",
				serfStatus: &serfStatus{
					Status:             "failed",
					SystemdUnitsFailed: strPtr(""),
				},
				metrics: machineMetrics{
					Labels: map[string]string{"aaa": "bbb"},
					Value:  monitorHWStatusWarning,
				}.toMetricFamilyMap("parts1"),
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
		actual := tc.mss.decideMachineState()
		if actual != tc.expected {
			t.Error(tc.message, "| expected:", tc.expected, "actual:", actual)
		}
	}
}

func TestDecideBySerf(t *testing.T) {
	testCases := []struct {
		message    string
		expected   sabakan.MachineState
		serfStatus *serfStatus
		reason     *machineStateReason
	}{
		{
			message:    "cannot get serf status",
			expected:   sabakan.StateUnreachable,
			serfStatus: nil,
			reason: &machineStateReason{
				Message: "serf status is nil",
			},
		},
		{
			message:  "failed",
			expected: sabakan.StateUnreachable,
			serfStatus: &serfStatus{
				Status:             "failed",
				SystemdUnitsFailed: strPtr(""),
			},
			reason: &machineStateReason{
				Message: "serf status is not alive",
				Fields: map[string]interface{}{
					"status": "failed",
				},
			},
		},
		{
			message:  "alive",
			expected: sabakan.StateHealthy,
			serfStatus: &serfStatus{
				Status:             "alive",
				SystemdUnitsFailed: strPtr(""),
			},
		},
		{
			message:  "alive, but units failed",
			expected: sabakan.StateUnhealthy,
			serfStatus: &serfStatus{
				Status:             "alive",
				SystemdUnitsFailed: strPtr("aaa"),
			},
			reason: &machineStateReason{
				Message: "some systemd units failed",
				Fields: map[string]interface{}{
					"units": "aaa",
				},
			},
		},
		{
			message:  "alive, but `systemd-units-failed` is not set",
			expected: sabakan.MachineState(""),
			serfStatus: &serfStatus{
				Status: "alive",
			},
		},
	}

	for _, tc := range testCases {
		actualState, actualReason := decideBySerf(tc.serfStatus)
		if actualState != tc.expected || !cmp.Equal(actualReason, tc.reason) {
			t.Error(tc.message, "| expectedState:", tc.expected, "actualState:", actualState, cmp.Diff(actualReason, tc.reason))
		}
	}
}

func TestCheckTarget(t *testing.T) {
	metrics := machineMetrics{
		Labels: map[string]string{"k1": "v1", "k2": "v2"},
		Value:  monitorHWStatusOK,
	}.toMetricFamily("m1")
	target := targetMetric{
		Name: "m1",
		Selector: &selector{
			Labels: map[string]string{"k1": "v1"},
		},
	}

	// TODO: check reason
	res, _ := checkTarget(metrics, target)
	if res != sabakan.StateHealthy {
		t.Error("ms.checkTarget(checkTarget) != sabakan.StateHealthy", res)
	}
}

func TestDecideByMonitorHW(t *testing.T) {
	parts1 := "parts1_status_health"
	parts2 := "parts2_status_health"
	parts3 := "parts3_status_health"

	testCases := []struct {
		message  string
		expected sabakan.MachineState
		reason   *machineStateReason

		metrics     map[string]*dto.MetricFamily
		machineType *machineType
	}{
		{
			message:  "If checklist is empty, returns healthy",
			expected: sabakan.StateHealthy,
			reason:   nil,
			metrics:  nil,
			machineType: &machineType{
				Name:             "boot",
				MetricsCheckList: []targetMetric{},
			},
		},
		{
			message:  "If metrics is nil, returns unhealthy",
			expected: sabakan.StateUnhealthy,
			reason: &machineStateReason{
				Message: "metrics is nil",
			},
			metrics: nil,
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
		{
			message:  "Target metric exists, and it is healthy",
			expected: sabakan.StateHealthy,
			metrics: machineMetrics{
				Labels: map[string]string{"aaa": "bbb"},
				Value:  monitorHWStatusOK,
			}.toMetricFamilyMap(parts1),
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
		{
			message:  "Target metrics exist, and it is warning",
			expected: sabakan.StateUnhealthy,
			reason: &machineStateReason{
				Message: "one or more metric is not healthy",
				Fields: map[string]interface{}{
					"healthy_count":     0,
					"name":              "parts1_status_health",
					"num_metrics":       1,
					"selector":          "{aaa:bbb}",
					"unhealthy_metrics": "parts1_status_health{aaa:bbb}=1.000000",
				},
			},
			metrics: machineMetrics{
				Labels: map[string]string{"aaa": "bbb"},
				Value:  monitorHWStatusWarning,
			}.toMetricFamilyMap(parts1),
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
		{
			message:  "Target metrics exist, and they are healthy",
			expected: sabakan.StateHealthy,
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
		{
			message:  "Target metrics exist, and they are healthy (multiple labels)",
			expected: sabakan.StateHealthy,
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
		{
			message:  "There are multiple matching metrics, and the one of them is broken",
			expected: sabakan.StateUnhealthy,
			reason: &machineStateReason{
				Message: "one or more metric is not healthy",
				Fields: map[string]interface{}{
					"healthy_count":     1,
					"name":              "parts1_status_health",
					"num_metrics":       2,
					"selector":          "{aaa:bbb,ccc:ddd}",
					"unhealthy_metrics": "parts1_status_health{aaa:bbb,ccc:ddd}=1.000000",
				},
			},
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
		{
			message:  "Target metric exists, but label is not matched",
			expected: sabakan.StateUnhealthy,
			reason: &machineStateReason{
				Message: "metric with specified labels does not exist",
				Fields: map[string]interface{}{
					"name":     "parts1_status_health",
					"selector": "{not:existed}",
				},
			},
			metrics: machineMetrics{
				Labels: map[string]string{"aaa": "bbb"},
				Value:  monitorHWStatusOK,
			}.toMetricFamilyMap(parts1),
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
		{
			message:  "Target label is not specified, and the all of metrics is healthy",
			expected: sabakan.StateHealthy,
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
		{
			message:  "Target's label is not specified, and there is an unhealthy metric",
			expected: sabakan.StateUnhealthy,
			reason: &machineStateReason{
				Message: "one or more metric is not healthy",
				Fields: map[string]interface{}{
					"healthy_count":     1,
					"name":              "parts1_status_health",
					"num_metrics":       2,
					"selector":          "{}",
					"unhealthy_metrics": "parts1_status_health{ccc:ddd}=1.000000",
				},
			},
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
		{
			message:  "parts2 is not found",
			expected: sabakan.StateUnhealthy,
			reason: &machineStateReason{
				Message: "metrics do not contain check target",
				Fields: map[string]interface{}{
					"target": "parts2_status_health",
				},
			},
			metrics: machineMetrics{
				Labels: map[string]string{"aaa": "bbb"},
				Value:  monitorHWStatusOK,
			}.toMetricFamilyMap(parts1),
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
		{
			message:  "Target metric exists, and prefix is matched",
			expected: sabakan.StateHealthy,
			metrics: machineMetrics{
				Labels: map[string]string{"aaa": "bbb"},
				Value:  monitorHWStatusOK,
			}.toMetricFamilyMap(parts1),
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
		{
			message:  "Target metrics exist, and all prefix labels are matched",
			expected: sabakan.StateHealthy,
			metrics: machineMetrics{
				Labels: map[string]string{"key1": "val1", "key2": "val2"},
				Value:  monitorHWStatusOK,
			}.toMetricFamilyMap(parts1),
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
		{
			message:  "One of label prefix is not matched",
			expected: sabakan.StateUnhealthy,
			reason: &machineStateReason{
				Message: "metric with specified labels does not exist",
				Fields: map[string]interface{}{
					"name":     "parts1_status_health",
					"selector": "{key1:val*,key2:foo*}",
				},
			},
			metrics: machineMetrics{
				Labels: map[string]string{"key1": "val1", "key2": "val2"},
				Value:  monitorHWStatusOK,
			}.toMetricFamilyMap(parts1),
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
		{
			message:  "label prefix is matched, but the value is not healthy",
			expected: sabakan.StateUnhealthy,
			reason: &machineStateReason{
				Message: "one or more metric is not healthy",
				Fields: map[string]interface{}{
					"healthy_count":     0,
					"name":              "parts1_status_health",
					"num_metrics":       1,
					"selector":          "{key1:val*,key2:va*}",
					"unhealthy_metrics": "parts1_status_health{key1:val1,key2:val2}=1.000000",
				},
			},
			metrics: machineMetrics{
				Labels: map[string]string{"key1": "val1", "key2": "val2"},
				Value:  monitorHWStatusWarning,
			}.toMetricFamilyMap(parts1),
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
		{
			message:  "both labels and label prefix are declared, all labels are matched but label prefix is not matched",
			expected: sabakan.StateUnhealthy,
			reason: &machineStateReason{
				Message: "metric with specified labels does not exist",
				Fields: map[string]interface{}{
					"name":     "parts1_status_health",
					"selector": "{foo:bar*,key1:val1}",
				},
			},
			metrics: machineMetrics{
				Labels: map[string]string{"key1": "val1", "key2": "val2"},
				Value:  monitorHWStatusOK,
			}.toMetricFamilyMap(parts1),
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
		{
			message:  "both labels and label prefix are declared, all label prefix are matched but labels are not matched",
			expected: sabakan.StateUnhealthy,
			reason: &machineStateReason{
				Message: "metric with specified labels does not exist",
				Fields: map[string]interface{}{
					"name":     "parts1_status_health",
					"selector": "{key1:hoge,key2:val*}",
				},
			},
			metrics: machineMetrics{
				Labels: map[string]string{"key1": "val1", "key2": "val2"},
				Value:  monitorHWStatusOK,
			}.toMetricFamilyMap(parts1),
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
		{
			message:  "minimum healthy count is satisfied",
			expected: sabakan.StateHealthy,
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
		{
			message:  "minimum healthy count is not satisfied",
			expected: sabakan.StateUnhealthy,
			reason: &machineStateReason{
				Message: "minimum healthy count is not satisfied",
				Fields: map[string]interface{}{
					"healthy_count":         1,
					"minimum_healthy_count": 2,
					"name":                  "parts1_status_health",
					"selector":              "{device:HDD.*}",
					"unhealthy_metrics":     "parts1_status_health{device:HDD.slot.2}=1.000000,parts1_status_health{device:HDD.slot.3}=1.000000",
				},
			},
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
	}

	for _, tc := range testCases {
		actualState, actualReason := decideByMonitorHW(tc.machineType, tc.metrics)
		if actualState != tc.expected || !cmp.Equal(actualReason, tc.reason) {
			t.Error(tc.message, "| expectedState:", tc.expected, "actualState:", actualState, cmp.Diff(actualReason, tc.reason))
		}
	}
}

func intPointer(i int) *int {
	p := new(int)
	*p = i
	return p
}
