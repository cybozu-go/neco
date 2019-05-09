package main

import (
	"strings"
	"testing"
)

func TestParseConfigFile(t *testing.T) {
	fileContent := `machine-types:
  - name: qemu
  - name: dell-14g-boot
    metrics:
      - name: hw_chassis_voltage_status_health
      - name: hw_storage_controller_status_health
        labels:
          controller: AHCI.Slot.1-1
`
	cfg, err := parseConfig(strings.NewReader(fileContent))
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.MachineTypes) != 2 {
		t.Error("len(cfg.MachineTypes) != 2. actual ", len(cfg.MachineTypes))
	}
}
