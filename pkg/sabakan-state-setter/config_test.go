package sss

import (
	"strings"
	"testing"
	"time"
)

func TestParseConfigFile(t *testing.T) {
	fileContent := `
shutdown-schedule: 0 11 * * *
machine-types:
  - name: qemu
    grace-period: 10s
  - name: boot
    metrics:
      - name: a
      - name: b
        labels:
          aaa: bbb
alert-monitor:
  alertmanager-endpoint: http://alertmanager:9093/api/v2/
  trigger-alerts:
    - name: DiskNotRecognized
      labels:
        severity: error
      serial-label: serial
      state: unhealthy
    - name: LLDPDown
      address-label: address
      state: unreachable
`
	shutdownSchedule, machineTypes, alertMonitor, err := parseConfig(strings.NewReader(fileContent))
	if err != nil {
		t.Fatal(err)
	}
	if shutdownSchedule != "0 11 * * *" {
		t.Errorf("shutdownSchedule != \"0 11 * * *\", actual \"%s\"", shutdownSchedule)
	}
	if len(machineTypes) != 2 {
		t.Error("len(machineTypesMap) != 2, actual ", len(machineTypes))
	}
	if machineTypes["qemu"].GracePeriod.Duration != 10*time.Second {
		t.Error("GracePeriod is not set")
	}
	if machineTypes["boot"].GracePeriod.Duration != time.Hour {
		t.Error("default value of GracePeriod is not set")
	}
	if alertMonitor == nil {
		t.Fatal("alertMonitor == nil")
	}
	if alertMonitor.AlertmanagerEndpoint != `http://alertmanager:9093/api/v2/` {
		t.Error("alertMonitor.AlertmanagerEndpoint != `http://alertmanager:9093/api/v2/`")
	}
	if len(alertMonitor.TriggerAlerts) != 2 {
		t.Fatal("len(alertMonitor.TriggerAlerts) != 2")
	}
	if alertMonitor.TriggerAlerts[0].Name != `DiskNotRecognized` {
		t.Error("alertMonitor.TriggerAlerts[0].Name != `DiskNotRecognized`")
	}
	if len(alertMonitor.TriggerAlerts[0].Labels) != 1 {
		t.Fatal("len(alertMonitor.TriggerAlerts[0].Labels) != 1")
	}
	if alertMonitor.TriggerAlerts[0].Labels[`severity`] != `error` {
		t.Error("alertMonitor.TriggerAlerts[0].Labels[`severity`] != `error`")
	}
	if alertMonitor.TriggerAlerts[0].AddressLabel != `` {
		t.Error("alertMonitor.TriggerAlerts[0].AddressLabel != ``")
	}
	if alertMonitor.TriggerAlerts[0].SerialLabel != `serial` {
		t.Error("alertMonitor.TriggerAlerts[0].SerialLabel != `serial`")
	}
	if alertMonitor.TriggerAlerts[0].State != `unhealthy` {
		t.Error("alertMonitor.TriggerAlerts[0].State != `unhealthy`")
	}
	// The contents of alertMonitor.TriggerAlerts[1] are not checked in detail. It is enough that there were no parse errors.

	fileContent2 := `
machine-types:
  - name: qemu
`
	shutdownSchedule, machineTypes, alertMonitor, err = parseConfig(strings.NewReader(fileContent2))
	if err != nil {
		t.Fatal(err)
	}
	if shutdownSchedule != "" {
		t.Errorf("shutdownSchedule != \"\", actual \"%s\"", shutdownSchedule)
	}
	if len(machineTypes) != 1 {
		t.Error("len(machineTypesMap) != 1, actual ", len(machineTypes))
	}
	if alertMonitor != nil {
		t.Error("alertMonitor != nil")
	}

	_, _, _, err = parseConfig(strings.NewReader("machine-types:"))
	if err == nil {
		t.Error("empty machine-types was not rejected")
	}

	fileContent3 := `
machine-types:
  - name: qemu
alert-monitor:
  alertmanager-endpoint: http://alertmanager:9093/api/v2/
  trigger-alerts:
    - name: Foo
      state: unhealthy
`
	_, _, _, err = parseConfig(strings.NewReader(fileContent3))
	if err == nil {
		t.Error("exactly one of address-label and serial-label is required, but it was not checked")
	}

	fileContent4 := `
machine-types:
  - name: qemu
alert-monitor:
  alertmanager-endpoint: http://alertmanager:9093/api/v2/
  trigger-alerts:
    - name: Foo
      address-label: address
      serial-label: serial
      state: unhealthy
`
	_, _, _, err = parseConfig(strings.NewReader(fileContent4))
	if err == nil {
		t.Error("exactly one of address-label and serial-label is required, but it was not checked")
	}

	fileContent5 := `
machine-types:
  - name: qemu
alert-monitor:
  alertmanager-endpoint: http://alertmanager:9093/api/v2/
  trigger-alerts:
    - name: Foo
      address-label: address
      state: foobar
`
	_, _, _, err = parseConfig(strings.NewReader(fileContent5))
	if err == nil {
		t.Error("invalid state was not rejected")
	}
}
