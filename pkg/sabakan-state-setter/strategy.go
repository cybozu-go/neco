package sss

import (
	"github.com/cybozu-go/log"
	"github.com/cybozu-go/sabakan/v2"
)

const (
	monitorHWStatusOK       = 0
	monitorHWStatusWarning  = 1
	monitorHWStatusCritical = 2
	monitorHWStatusNull     = -1
	systemdUnitsFailedTag   = "systemd-units-failed"
)

func (mss *machineStateSource) decideMachineStateCandidate() (sabakan.MachineState, bool) {
	if mss.serfStatus == nil {
		log.Info("unreachable; serf status is nil", map[string]interface{}{
			"serial": mss.serial,
			"ipv4":   mss.ipv4,
		})
		return sabakan.StateUnreachable, true
	}
	if mss.serfStatus.Status != "alive" {
		log.Info("unreachable; serf status != alive", map[string]interface{}{
			"serial": mss.serial,
			"ipv4":   mss.ipv4,
			"status": mss.serfStatus.Status,
		})
		return sabakan.StateUnreachable, true
	}

	suf, ok := mss.serfStatus.Tags[systemdUnitsFailedTag]
	if !ok {
		state := mss.decideByMonitorHW()
		if state == sabakan.StateHealthy {
			// Do nothing if there is no systemd-units-failed tag and no hardware failure.
			// In this case, the machine is starting up.
			return sabakan.MachineState(""), false
		}
		return state, true
	}

	if len(suf) != 0 {
		log.Info("unhealthy; some systemd units failed", map[string]interface{}{
			"serial": mss.serial,
			"ipv4":   mss.ipv4,
			"failed": suf,
		})
		return sabakan.StateUnhealthy, true
	}

	return mss.decideByMonitorHW(), true
}

func (mss *machineStateSource) decideByMonitorHW() sabakan.MachineState {
	if mss.machineType == nil {
		log.Info("unhealthy; machine type is nil", map[string]interface{}{
			"serial": mss.serial,
			"ipv4":   mss.ipv4,
		})
		return sabakan.StateUnhealthy
	}

	if len(mss.machineType.MetricsCheckList) == 0 {
		return sabakan.StateHealthy
	}

	if mss.metrics == nil {
		log.Info("unhealthy; metrics is nil", map[string]interface{}{
			"serial": mss.serial,
			"ipv4":   mss.ipv4,
		})
		return sabakan.StateUnhealthy
	}

	for _, checkTarget := range mss.machineType.MetricsCheckList {
		_, ok := mss.metrics[checkTarget.Name]
		if !ok {
			log.Info("unhealthy; metrics do not contain check target", map[string]interface{}{
				"serial": mss.serial,
				"ipv4":   mss.ipv4,
				"target": checkTarget.Name,
			})
			return sabakan.StateUnhealthy
		}

		res := mss.checkTarget(checkTarget)
		if res != sabakan.StateHealthy {
			return res
		}
	}

	return sabakan.StateHealthy
}

func (mss *machineStateSource) checkTarget(target targetMetric) sabakan.MachineState {
	mf := mss.metrics[target.Name]
	matched := target.Selector.Match(mf)
	if len(matched) == 0 {
		log.Info("unhealthy; metric with specified labels does not exist", map[string]interface{}{
			"serial":   mss.serial,
			"ipv4":     mss.ipv4,
			"name":     target.Name,
			"selector": target.Selector,
		})
		return sabakan.StateUnhealthy
	}

	var healthyCount int
	for _, m := range matched {
		gauge := m.GetGauge()
		if gauge == nil {
			continue
		}
		if gauge.GetValue() == monitorHWStatusOK {
			healthyCount++
			continue
		}
		log.Info("unhealthy; metric is not healthy", map[string]interface{}{
			"serial": mss.serial,
			"ipv4":   mss.ipv4,
			"name":   target.Name,
			"labels": m.Label,
			"value":  gauge.GetValue(),
		})
	}

	if target.MinimumHealthyCount == nil {
		if healthyCount != len(matched) {
			log.Info("unhealthy; one or more metric is not healthy", map[string]interface{}{
				"serial":        mss.serial,
				"ipv4":          mss.ipv4,
				"name":          target.Name,
				"selector":      target.Selector,
				"num_metrics":   len(matched),
				"healthy_count": healthyCount,
			})
			return sabakan.StateUnhealthy
		}

		log.Info("healthy;", map[string]interface{}{
			"serial":   mss.serial,
			"ipv4":     mss.ipv4,
			"name":     target.Name,
			"selector": target.Selector,
		})
		return sabakan.StateHealthy
	}

	minCount := *target.MinimumHealthyCount
	if healthyCount < minCount {
		log.Info("unhealthy; minimum healthy count is not satisfied", map[string]interface{}{
			"serial":                mss.serial,
			"ipv4":                  mss.ipv4,
			"name":                  target.Name,
			"selector":              target.Selector,
			"minimum_healthy_count": minCount,
			"healthy_count":         healthyCount,
		})
		return sabakan.StateUnhealthy
	}

	log.Info("healthy;", map[string]interface{}{
		"serial":   mss.serial,
		"ipv4":     mss.ipv4,
		"name":     target.Name,
		"selector": target.Selector,
	})
	return sabakan.StateHealthy
}
