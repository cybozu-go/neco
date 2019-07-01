package main

import (
	"strings"

	"github.com/cybozu-go/log"
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

func (ms machineStateSource) decideSabakanState() string {
	if ms.serfStatus == nil {
		log.Info("unreachable; serf status is nil", map[string]interface{}{
			"serial": ms.serial,
		})
		return sabakan.StateUnreachable.GQLEnum()
	}
	if ms.serfStatus.Status != "alive" {
		log.Info("unreachable; serf status != alive", map[string]interface{}{
			"serial": ms.serial,
			"status": ms.serfStatus.Status,
		})
		return sabakan.StateUnreachable.GQLEnum()
	}

	suf, ok := ms.serfStatus.Tags[systemdUnitsFailedTag]
	if !ok {
		state := ms.decideByMonitorHW()
		if state == sabakan.StateHealthy.GQLEnum() {
			// Do nothing if there is no systemd-units-failed tag and no hardware failure.
			// In this case, the machine is starting up.
			return noStateTransition
		}
		return state
	}

	if len(suf) != 0 {
		log.Info("unhealthy; some systemd units failed", map[string]interface{}{
			"serial": ms.serial,
			"failed": suf,
		})
		return sabakan.StateUnhealthy.GQLEnum()
	}

	return ms.decideByMonitorHW()
}

func (ms machineStateSource) decideByMonitorHW() string {
	if ms.machineType == nil {
		log.Info("unhealthy; machine type is nil", map[string]interface{}{
			"serial": ms.serial,
		})
		return sabakan.StateUnhealthy.GQLEnum()
	}

	if len(ms.machineType.MetricsCheckList) == 0 {
		return sabakan.StateHealthy.GQLEnum()
	}

	if ms.metrics == nil {
		log.Info("unhealthy; metrics is nil", map[string]interface{}{
			"serial": ms.serial,
		})
		return sabakan.StateUnhealthy.GQLEnum()
	}

	for _, checkTarget := range ms.machineType.MetricsCheckList {
		_, ok := ms.metrics[checkTarget.Name]
		if !ok {
			log.Info("unhealthy; metrics do not contain check target", map[string]interface{}{
				"serial": ms.serial,
				"target": checkTarget.Name,
			})
			return sabakan.StateUnhealthy.GQLEnum()
		}

		var res string
		if checkTarget.Selector == nil {
			res = ms.checkAllTarget(checkTarget)
		} else {
			res = ms.checkSpecifiedTarget(checkTarget)
		}
		if res != sabakan.StateHealthy.GQLEnum() {
			return res
		}
	}

	return sabakan.StateHealthy.GQLEnum()
}

func (ms machineStateSource) checkAllTarget(checkTarget targetMetric) string {
	var healthyCount int
	metrics := ms.metrics[checkTarget.Name]

	var minCount int
	if checkTarget.MinimumHealthyCount == nil {
		minCount = len(metrics)
	} else {
		minCount = *checkTarget.MinimumHealthyCount
	}

	for _, m := range metrics {
		if m.Value != monitorHWStatusHealth {
			log.Info("unhealthy; metric is not healthy", map[string]interface{}{
				"serial": ms.serial,
				"name":   checkTarget.Name,
				"labels": m.Labels,
				"value":  m.Value,
			})
			continue
		}
		healthyCount++
	}

	if healthyCount < minCount {
		return sabakan.StateUnhealthy.GQLEnum()
	}

	return sabakan.StateHealthy.GQLEnum()
}

func (ms machineStateSource) checkSpecifiedTarget(checkTarget targetMetric) string {
	flagExists := false
	metrics := ms.metrics[checkTarget.Name]

	for _, m := range metrics {
		if checkTarget.Selector == nil ||
			(checkTarget.Selector.labels != nil && isMetricMatchedLabels(m, checkTarget.Selector.labels)) ||
			(checkTarget.Selector.labelPrefix != nil && isMetricHasPrefix(m, checkTarget.Selector.labelPrefix)) {

			if m.Value != monitorHWStatusHealth {
				log.Info("unhealthy; metric is not healthy", map[string]interface{}{
					"serial": ms.serial,
					"name":   checkTarget.Name,
					"labels": m.Labels,
					"value":  m.Value,
				})
				return sabakan.StateUnhealthy.GQLEnum()
			}
			flagExists = true
		}
	}

	if !flagExists {
		log.Info("unhealthy; metric with specified labels does not exist", map[string]interface{}{
			"serial":                ms.serial,
			"name":                  checkTarget.Name,
			"selector":              checkTarget.Selector,
			"minimum-healthy-count": checkTarget.MinimumHealthyCount,
		})
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

func isMetricHasPrefix(metric prom2json.Metric, labelPrefix map[string]string) bool {
	for k, prefix := range labelPrefix {
		labelVal, ok := metric.Labels[k]
		if !ok || !strings.HasPrefix(labelVal, prefix) {
			return false
		}
	}

	return true
}
