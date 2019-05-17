package main

import (
	"github.com/cybozu-go/sabakan/v2"
	"github.com/prometheus/prom2json"
)

const (
	monitorHWStatusHealth   = "0"
	monitorHWStatusWarning  = "1"
	monitorHWStatusCritical = "2"
	monitorHWStatusNull     = "-1"
	systemdUnitsFailedTag   = "systemd-units-failed"
	noStateTransition       = "no-transition"
)

func decideSabakanState(ms machineStateSource) string {
	if ms.serfStatus == nil || ms.serfStatus.Status != "alive" {
		return sabakan.StateUnreachable.GQLEnum()
	}

	suf, ok := ms.serfStatus.Tags[systemdUnitsFailedTag]
	if !ok {
		state := decideByMonitorHW(ms)
		if state == sabakan.StateHealthy.GQLEnum() {
			// Do nothing if there is no systemd-units-failed tag and no hardware failure.
			// In this case, the machine is starting up.
			return noStateTransition
		}
		return state
	}

	if len(suf) != 0 {
		return sabakan.StateUnhealthy.GQLEnum()
	}

	return decideByMonitorHW(ms)
}

func decideByMonitorHW(ms machineStateSource) string {
	if ms.machineType == nil {
		return sabakan.StateUnhealthy.GQLEnum()
	}

	if len(ms.machineType.MetricsCheckList) == 0 {
		return sabakan.StateHealthy.GQLEnum()
	}

	if ms.metrics == nil {
		return sabakan.StateUnhealthy.GQLEnum()
	}

	for _, checkTarget := range ms.machineType.MetricsCheckList {
		metrics, ok := ms.metrics[checkTarget.Name]
		if !ok {
			return sabakan.StateUnhealthy.GQLEnum()
		}

		var res string
		if checkTarget.Labels == nil {
			res = checkAllTarget(metrics)
		} else {
			res = checkSpecifiedTarget(metrics, checkTarget.Labels)
		}
		if res != sabakan.StateHealthy.GQLEnum() {
			return res
		}
	}

	return sabakan.StateHealthy.GQLEnum()
}

func checkAllTarget(metrics machineMetrics) string {
	for _, m := range metrics {
		if m.Value != monitorHWStatusHealth {
			return sabakan.StateUnhealthy.GQLEnum()
		}
	}
	return sabakan.StateHealthy.GQLEnum()
}

func checkSpecifiedTarget(metrics machineMetrics, labels map[string]string) string {
	flagExists := false
	for _, m := range metrics {
		if !isMetricMatchedLabels(m, labels) {
			continue
		}
		if m.Value != monitorHWStatusHealth {
			return sabakan.StateUnhealthy.GQLEnum()
		}
		flagExists = true
	}
	if !flagExists {
		return sabakan.StateUnhealthy.GQLEnum()
	}
	return sabakan.StateHealthy.GQLEnum()
}

func isMetricMatchedLabels(metric prom2json.Metric, labels map[string]string) bool {
	for k, v := range labels {
		labelVal, ok := metric.Labels[k]
		if !ok || v != labelVal {
			return false
		}
	}
	return true
}
