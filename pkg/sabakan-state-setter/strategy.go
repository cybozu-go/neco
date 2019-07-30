package sss

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

func (mss *machineStateSource) decideMachineStateCandidate() string {
	if mss.serfStatus == nil {
		log.Info("unreachable; serf status is nil", map[string]interface{}{
			"serial": mss.serial,
		})
		return sabakan.StateUnreachable.GQLEnum()
	}
	if mss.serfStatus.Status != "alive" {
		log.Info("unreachable; serf status != alive", map[string]interface{}{
			"serial": mss.serial,
			"status": mss.serfStatus.Status,
		})
		return sabakan.StateUnreachable.GQLEnum()
	}

	suf, ok := mss.serfStatus.Tags[systemdUnitsFailedTag]
	if !ok {
		state := mss.decideByMonitorHW()
		if state == sabakan.StateHealthy.GQLEnum() {
			// Do nothing if there is no systemd-units-failed tag and no hardware failure.
			// In this case, the machine is starting up.
			return noStateTransition
		}
		return state
	}

	if len(suf) != 0 {
		log.Info("unhealthy; some systemd units failed", map[string]interface{}{
			"serial": mss.serial,
			"failed": suf,
		})
		return sabakan.StateUnhealthy.GQLEnum()
	}

	return mss.decideByMonitorHW()
}

func (mss *machineStateSource) decideByMonitorHW() string {
	if mss.machineType == nil {
		log.Info("unhealthy; machine type is nil", map[string]interface{}{
			"serial": mss.serial,
		})
		return sabakan.StateUnhealthy.GQLEnum()
	}

	if len(mss.machineType.MetricsCheckList) == 0 {
		return sabakan.StateHealthy.GQLEnum()
	}

	if mss.metrics == nil {
		log.Info("unhealthy; metrics is nil", map[string]interface{}{
			"serial": mss.serial,
		})
		return sabakan.StateUnhealthy.GQLEnum()
	}

	for _, checkTarget := range mss.machineType.MetricsCheckList {
		_, ok := mss.metrics[checkTarget.Name]
		if !ok {
			log.Info("unhealthy; metrics do not contain check target", map[string]interface{}{
				"serial": mss.serial,
				"target": checkTarget.Name,
			})
			return sabakan.StateUnhealthy.GQLEnum()
		}

		res := mss.checkTarget(checkTarget)
		if res != sabakan.StateHealthy.GQLEnum() {
			return res
		}
	}

	return sabakan.StateHealthy.GQLEnum()
}

func (mss *machineStateSource) checkTarget(target targetMetric) string {
	var exists bool
	metrics := mss.metrics[target.Name]

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
					"serial": mss.serial,
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

	if target.MinimumHealthyCount != nil {
		minCount = *target.MinimumHealthyCount
	}

	if !exists {
		log.Info("unhealthy; metric with specified labels does not exist", map[string]interface{}{
			"serial":                mss.serial,
			"name":                  target.Name,
			"selector":              slctr,
			"minimum_healthy_count": minCount,
		})
		return sabakan.StateUnhealthy.GQLEnum()
	}

	if healthyCount < minCount {
		log.Info("unhealthy; minimum healthy count is not satisfied", map[string]interface{}{
			"serial":                mss.serial,
			"name":                  target.Name,
			"selector":              slctr,
			"minimum_healthy_count": minCount,
			"healthy_count":         healthyCount,
		})
		return sabakan.StateUnhealthy.GQLEnum()
	}

	log.Info("healthy;", map[string]interface{}{
		"serial":                mss.serial,
		"name":                  target.Name,
		"selector":              slctr,
		"minimum_healthy_count": minCount,
		"healthy_count":         healthyCount,
	})
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
