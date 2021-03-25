package sss

import (
	"fmt"
	"sort"
	"strings"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/sabakan/v2"
	dto "github.com/prometheus/client_model/go"
)

const (
	monitorHWStatusOK       = 0
	monitorHWStatusWarning  = 1
	monitorHWStatusCritical = 2
	monitorHWStatusNull     = -1
)

const noTransition = sabakan.MachineState("")

// machineStateSource is a struct of machine state collection
type machineStateSource struct {
	serial string
	ipv4   string

	serfStatus  *serfStatus
	machineType *machineType
	metrics     map[string]*dto.MetricFamily
}

type machineStateReason struct {
	Message string
	Fields  map[string]interface{}
}

func logging(serial, ipv4 string, state sabakan.MachineState, reason *machineStateReason) {
	var builder strings.Builder
	builder.WriteString(string(state))
	if reason != nil && reason.Message != "" {
		builder.WriteString("; ")
		builder.WriteString(reason.Message)
	}

	fields := map[string]interface{}{
		"serial": serial,
		"ipv4":   ipv4,
	}
	if reason != nil {
		for k, v := range reason.Fields {
			fields[k] = v
		}
	}

	log.Info(builder.String(), fields)
}

func newMachineStateSource(m *machine, serfStatuses map[string]*serfStatus, machineTypes map[string]*machineType) *machineStateSource {
	return &machineStateSource{
		serial:      m.Serial,
		ipv4:        m.IPv4Addr,
		serfStatus:  serfStatuses[m.IPv4Addr],
		machineType: machineTypes[m.Type],
	}
}

func (mss *machineStateSource) decideMachineState() sabakan.MachineState {
	state, reason := decideBySerf(mss.serfStatus)
	if state != noTransition && state != sabakan.StateHealthy {
		logging(mss.serial, mss.ipv4, state, reason)
		return state
	}

	stateByMonitorHW, reasonByMonitorHW := decideByMonitorHW(mss.machineType, mss.metrics)
	if stateByMonitorHW != sabakan.StateHealthy {
		logging(mss.serial, mss.ipv4, stateByMonitorHW, reasonByMonitorHW)
		return stateByMonitorHW
	}

	if state == noTransition {
		return state
	}

	logging(mss.serial, mss.ipv4, state, reason)
	return state
}

func decideBySerf(serfStatus *serfStatus) (sabakan.MachineState, *machineStateReason) {
	if serfStatus == nil {
		return sabakan.StateUnreachable, &machineStateReason{
			Message: "serf status is nil",
		}
	}

	if serfStatus.Status != "alive" {
		return sabakan.StateUnreachable, &machineStateReason{
			Message: "serf status is not alive",
			Fields: map[string]interface{}{
				"status": serfStatus.Status,
			},
		}
	}

	if serfStatus.SystemdUnitsFailed == nil {
		// Do nothing if there is no systemd-units-failed tag and no hardware failure.
		// In this case, the machine is starting up.
		return noTransition, nil
	}

	if *serfStatus.SystemdUnitsFailed != "" {
		return sabakan.StateUnhealthy, &machineStateReason{
			Message: "some systemd units failed",
			Fields: map[string]interface{}{
				"units": *serfStatus.SystemdUnitsFailed,
			},
		}
	}

	return sabakan.StateHealthy, nil
}

func decideByMonitorHW(machineType *machineType, metrics map[string]*dto.MetricFamily) (sabakan.MachineState, *machineStateReason) {
	if machineType == nil {
		return sabakan.StateUnhealthy, &machineStateReason{
			Message: "machine type is not defined",
		}
	}

	if len(machineType.MetricsCheckList) == 0 {
		return sabakan.StateHealthy, nil
	}

	if metrics == nil {
		return sabakan.StateUnhealthy, &machineStateReason{
			Message: "metrics is nil",
		}
	}

	for _, target := range machineType.MetricsCheckList {
		metricsFamily, ok := metrics[target.Name]
		if !ok {
			return sabakan.StateUnhealthy, &machineStateReason{
				Message: "metrics do not contain check target",
				Fields: map[string]interface{}{
					"target": target.Name,
				},
			}
		}

		res, reason := checkTarget(metricsFamily, target)
		if res != sabakan.StateHealthy {
			return res, reason
		}
	}

	return sabakan.StateHealthy, nil
}

func labelToString(label []*dto.LabelPair) string {
	var ss []string
	for _, l := range label {
		ss = append(ss, l.GetName()+":"+l.GetValue())
	}
	sort.Strings(ss)
	return strings.Join(ss, ",")
}

func checkTarget(metricsFamily *dto.MetricFamily, target targetMetric) (sabakan.MachineState, *machineStateReason) {
	matched := target.Selector.Match(metricsFamily)
	if len(matched) == 0 {
		return sabakan.StateUnhealthy, &machineStateReason{
			Message: "metric with specified labels does not exist",
			Fields: map[string]interface{}{
				"name":     target.Name,
				"selector": target.Selector.String(),
			},
		}
	}

	var unhealthyMetrics []string
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
		unhealthyMetrics = append(unhealthyMetrics, fmt.Sprintf("%s{%s}=%f", target.Name, labelToString(m.Label), gauge.GetValue()))
	}

	if target.MinimumHealthyCount != nil {
		minCount := *target.MinimumHealthyCount
		if healthyCount < minCount {
			return sabakan.StateUnhealthy, &machineStateReason{
				Message: "minimum healthy count is not satisfied",
				Fields: map[string]interface{}{
					"name":                  target.Name,
					"selector":              target.Selector.String(),
					"minimum_healthy_count": minCount,
					"healthy_count":         healthyCount,
					"unhealthy_metrics":     strings.Join(unhealthyMetrics, ","),
				},
			}
		}
		return sabakan.StateHealthy, nil
	}

	if healthyCount != len(matched) {
		return sabakan.StateUnhealthy, &machineStateReason{
			Message: "one or more metric is not healthy",
			Fields: map[string]interface{}{
				"name":              target.Name,
				"selector":          target.Selector.String(),
				"num_metrics":       len(matched),
				"healthy_count":     healthyCount,
				"unhealthy_metrics": strings.Join(unhealthyMetrics, ","),
			},
		}
	}
	return sabakan.StateHealthy, nil

}
