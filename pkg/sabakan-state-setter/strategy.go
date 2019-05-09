package main

import (
	"strings"

	"github.com/cybozu-go/sabakan/v2"
	"github.com/prometheus/prom2json"
)

const (
	monitorHWStatusHealth       = "0"
	monitorHWStatusWarning      = "1"
	monitorHWStatusCritical     = "2"
	monitorHWStatusNull         = "-1"
	bossControllerPrefix        = "AHCI.Slot"
	nvmeControllerPrefix        = "PCIeSSD.Slot"
	storageControllerMetricName = "hw_storage_controller_status_health"
	stateMetricNotFound         = ""
)

var (
	partsMetricNames = []string{
		"hw_processor_status_health",
		"hw_system_memory_summary_status_health",
		"hw_chassis_temperature_status_health",
		"hw_chassis_voltage_status_health",
		"hw_storage_device_status_health",
	}
	allMetricNames = append(partsMetricNames, storageControllerMetricName)
)

func decideSabakanState(ms machineStateSource) string {
	if ms.serfStatus == nil || ms.serfStatus.Status != "alive" {
		return sabakan.StateUnreachable.GQLEnum()
	}

	suf, ok := ms.serfStatus.Tags["systemd-units-failed"]
	if !ok {
		return sabakan.StateUnhealthy.GQLEnum()
	}

	if len(suf) != 0 {
		return sabakan.StateUnhealthy.GQLEnum()
	}

	return decideByMonitorHW(ms)
}

func decideByMonitorHW(ms machineStateSource) string {
	if ms.metrics == nil {
		return sabakan.StateUnhealthy.GQLEnum()
	}

	for _, family := range ms.metrics {
		if !contains(family) {
			return stateMetricNotFound
		}

		for _, labelName := range partsMetricNames {
			if family.Name != labelName {
				continue
			}
			for _, m := range family.Metrics {
				metric, ok := m.(prom2json.Metric)
				if !ok {
					continue
				}
				if metric.Value != monitorHWStatusHealth {
					return sabakan.StateUnhealthy.GQLEnum()
				}
			}
		}

		if family.Name == storageControllerMetricName {
			for _, m := range family.Metrics {
				metric, ok := m.(prom2json.Metric)
				if !ok {
					continue
				}
				v, ok := metric.Labels["controller"]
				if !ok {
					continue
				}
				if strings.Contains(v, bossControllerPrefix) || strings.Contains(v, nvmeControllerPrefix) {
					if metric.Value != monitorHWStatusHealth {
						return sabakan.StateUnhealthy.GQLEnum()
					}
				}
			}
		}

	}
	return sabakan.StateHealthy.GQLEnum()
}

func contains(family *prom2json.Family) bool {
	for _, labelName := range allMetricNames {
		if family.Name == labelName {
			return true
		}
	}
	return false
}
