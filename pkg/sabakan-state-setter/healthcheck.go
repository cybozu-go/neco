package sss

import (
	"fmt"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/sabakan/v3"
	dto "github.com/prometheus/client_model/go"
)

const (
	monitorHWStatusOK       = 0
	monitorHWStatusWarning  = 1
	monitorHWStatusCritical = 2
	monitorHWStatusNull     = -1
)

const doNotChangeState = sabakan.MachineState("")
const stateUnhealthyImmediate = sabakan.MachineState("unhealthy_immediate")

// machineStateSource is a struct of machine state collection
type machineStateSource struct {
	serial string
	ipv4   string

	serfStatus  *serfStatus
	alertStatus *alertStatus
	machineType *machineType
	metrics     map[string]*dto.MetricFamily
}

func newMachineStateSource(m *machine, serfStatuses map[string]*serfStatus, alertStatuses map[string]*alertStatus, machineTypes map[string]*machineType) *machineStateSource {
	return &machineStateSource{
		serial:      m.Serial,
		ipv4:        m.IPv4Addr,
		serfStatus:  serfStatuses[m.IPv4Addr],
		alertStatus: alertStatuses[m.IPv4Addr],
		machineType: machineTypes[m.Type],
	}
}

func (mss *machineStateSource) decideMachineStateCandidate() sabakan.MachineState {
	// Check the severest state first.

	if mss.serfStatus == nil {
		log.Info("unreachable; serf status is nil", map[string]interface{}{
			"serial": mss.serial,
			"ipv4":   mss.ipv4,
		})
		return sabakan.StateUnreachable
	}

	if mss.serfStatus.Status != "alive" {
		log.Info("unreachable; serf status != alive", map[string]interface{}{
			"serial": mss.serial,
			"ipv4":   mss.ipv4,
			"status": mss.serfStatus.Status,
		})
		return sabakan.StateUnreachable
	}

	// alertStatus can return StateUnreachable, so it should be checked before serfStatus StateUnhealthy.
	if mss.alertStatus != nil {
		msg := fmt.Sprintf("%s; alert is firing", mss.alertStatus.State)
		log.Info(msg, map[string]interface{}{
			"serial": mss.serial,
			"ipv4":   mss.ipv4,
			"alert":  mss.alertStatus.AlertName,
		})
		if mss.alertStatus.State == sabakan.StateUnhealthy {
			return stateUnhealthyImmediate
		}
		return mss.alertStatus.State
	}

	if mss.serfStatus.SystemdUnitsFailed != nil && *mss.serfStatus.SystemdUnitsFailed != "" {
		log.Info("unhealthy; some systemd units failed", map[string]interface{}{
			"serial": mss.serial,
			"ipv4":   mss.ipv4,
			"failed": *mss.serfStatus.SystemdUnitsFailed,
		})
		return sabakan.StateUnhealthy
	}

	state := mss.decideByMonitorHW()
	if state != sabakan.StateHealthy {
		return state
	}

	if mss.serfStatus.SystemdUnitsFailed == nil {
		// Do nothing if there is no systemd-units-failed tag and no hardware failure.
		// In this case, the machine is starting up.
		return doNotChangeState
	}

	return sabakan.StateHealthy
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
