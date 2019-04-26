package main

import (
	"encoding/json"
	"strings"

	"github.com/cybozu-go/sabakan/v2"
)

const (
	monitorHWStatusHealth   = "0"
	monitorHWStatusWarning  = "1"
	monitorHWStatusCritical = "2"
	monitorHWStatusNull     = "-1"
	bossControllerPrefix    = "AHCI.Slot"
	nvmeControllerPrefix    = "PCIeSSD.Slot"
)

type familyMetrics struct {
	Labels []familyMetricsLabels `json:"labels"`
	Value  string                `json:"value"`
}

type familyMetricsLabels struct {
	Controller string `json:"controller,omitempty"`
	Device     string `json:"device,omitempty"`
	Sensor     string `json:"sensor,omitempty"`
}

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
		case "hw_processor_status_health", "hw_system_memory_summary_status_health", "hw_chassis_temperature_status_health", "hw_chassis_voltage_status_health":
			for _, m := range family.Metrics {
				var metrics familyMetrics
				data, ok := m.(*[]byte)
				if !ok {
					continue
				}
				err := json.Unmarshal(*data, &metrics)
				if err != nil {
					continue
				}

				if metrics.Value != monitorHWStatusHealth {
					return sabakan.StateUnhealthy.GQLEnum(), nil
				}
			}
		case "hw_storage_controller_status_health":
			for _, m := range family.Metrics {
				var metrics familyMetrics
				data, ok := m.(*[]byte)
				if !ok {
					continue
				}
				err := json.Unmarshal(*data, &metrics)
				if err != nil {
					continue
				}


				for _, l := range metrics.Labels {
				if strings.Contains(l.Controller, bossControllerPrefix) && strings.Contains(l.Controller, nvmeControllerPrefix) {
					if metrics.Value != monitorHWStatusHealth {
						return sabakan.StateUnhealthy.GQLEnum(), nil
					}
				}
				}
			}
		case "hw_storage_device_status_health":
			for _, m := range family.Metrics {
				var metrics familyMetrics
				data, ok := m.(*[]byte)
				if !ok {
					continue
				}
				err := json.Unmarshal(*data, &metrics)
				if err != nil {
					continue
				}

				if metrics.Value != monitorHWStatusHealth {
					return sabakan.StateUnhealthy.GQLEnum(), nil
				}
			}
		}
	}

	return sabakan.StateHealthy.GQLEnum(), nil
}
