package main

import (
	"github.com/cybozu-go/sabakan/v2"
)

const (
	monitorHWStatusHealth   = "0"
	monitorHWStatusWarning  = "1"
	monitorHWStatusCritical = "2"
	monitorHWStatusNull     = "-1"
)

func decideSabakanState(ms machineStateSource) (string, error) {
	if ms.serfStatus.Status != "alive" {
		return sabakan.StateUnreachable.GQLEnum(), nil
	}

	suf, ok := ms.serfStatus.Tags["systemd-units-failed"]
	if !ok {
		return sabakan.StateUnhealthy.GQLEnum(), nil
	}

	if len(suf) != 0 {
		return sabakan.StateUnhealthy.GQLEnum(), nil
	}

	return decideByMonitorHW(ms)
}

func decideByMonitorHW(ms machineStateSource) (string, error) {
	if ms.metrics == nil {
		return sabakan.StateUnhealthy.GQLEnum(), nil
	}

	for _, family := range ms.metrics {
		switch family.Name {
		case "hw_processor_status_health", "hw_system_memory_summary_status_health":
			for _, m := range family.Metrics {
				l := m.(map[string]string)
				if l["value"] != monitorHWStatusHealth {
					return sabakan.StateUnhealthy.GQLEnum(), nil
				}
			}
		case "hw_storage_controller_status_health":
			for _, m := range family.Metrics {
				l := m.(map[string]string)
				if l["value"] != monitorHWStatusHealth || l["value"] != monitorHWStatusNull {
					return sabakan.StateUnhealthy.GQLEnum(), nil
				}
			}
			// TODO: Add storage device health(BOSS, NVMeSSD) and sensor temperature
		}
	}

	return sabakan.StateHealthy.GQLEnum(), nil
}
