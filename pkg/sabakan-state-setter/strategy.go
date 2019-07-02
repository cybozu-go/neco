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

		res := ms.checkTarget(checkTarget)
		if res != sabakan.StateHealthy.GQLEnum() {
			return res
		}
	}

	return sabakan.StateHealthy.GQLEnum()
}

func (ms machineStateSource) checkTarget(target targetMetric) string {
	var exists bool
	metrics := ms.metrics[target.Name]

	var healthyCount, minCount int

	slctr := target.Selector
	for _, m := range metrics {
		var matched bool
		switch {
		case slctr == nil:
			matched = true
		case slctr.Labels != nil && slctr.LabelPrefix != nil:
			matched = slctr.isMetricMatchedLabels(m) && slctr.isMetricHasPrefix(m)
		case slctr.Labels != nil:
			matched = slctr.isMetricMatchedLabels(m)
		case slctr.LabelPrefix != nil:
			matched = slctr.isMetricHasPrefix(m)
		}

		if matched {
			minCount++
			if m.Value != monitorHWStatusHealth {
				log.Info("unhealthy; metric is not healthy", map[string]interface{}{
					"serial": ms.serial,
					"name":   target.Name,
					"labels": m.Labels,
					"value":  m.Value,
				})
			} else {
				healthyCount++
			}
		}
		exists = exists || matched
	}

	if !exists {
		log.Info("unhealthy; metric with specified labels does not exist", map[string]interface{}{
			"serial":                ms.serial,
			"name":                  target.Name,
			"selector":              slctr,
			"minimum_healthy_count": target.MinimumHealthyCount,
		})
		return sabakan.StateUnhealthy.GQLEnum()
	}

	if target.MinimumHealthyCount != nil {
		minCount = *target.MinimumHealthyCount
	}

	if healthyCount < minCount {
		log.Info("unhealthy; minimum healthy count is not satisfied", map[string]interface{}{
			"serial":                ms.serial,
			"name":                  target.Name,
			"selector":              slctr,
			"minimum_healthy_count": minCount,
			"healthy_count":         healthyCount,
		})
		return sabakan.StateUnhealthy.GQLEnum()
	}
	return sabakan.StateHealthy.GQLEnum()
}

func (s *selector) isMetricMatchedLabels(metric prom2json.Metric) bool {
	for k, v := range s.Labels {
		labelVal, ok := metric.Labels[k]
		if !ok || v != labelVal {
			return false
		}
	}
	return true
}

func (s *selector) isMetricHasPrefix(metric prom2json.Metric) bool {
	for k, prefix := range s.LabelPrefix {
		labelVal, ok := metric.Labels[k]
		if !ok || !strings.HasPrefix(labelVal, prefix) {
			return false
		}
	}
	return true
}
