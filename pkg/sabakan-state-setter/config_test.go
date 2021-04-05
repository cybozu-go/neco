package sss

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestParseConfigFile(t *testing.T) {
	fileContent := `
shutdown-schedule: 00 16 * * *
machine-types:
  - name: qemu
    grace-period: 10s
  - name: boot
    metrics:
      - name: a
      - name: b
        labels:
          aaa: bbb
`
	shutdownSchedule, machineTypes, err := parseConfig(strings.NewReader(fileContent))
	if err != nil {
		t.Fatal(err)
	}
	if shutdownSchedule != "00 16 * * *" {
		t.Errorf("shutdownSchedule != \"00 16 * * *\", actual \"%s\"", shutdownSchedule)
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

	fileContent2 := `
machine-types:
  - name: qemu
`
	shutdownSchedule, machineTypes, err = parseConfig(strings.NewReader(fileContent2))
	if err != nil {
		t.Fatal(err)
	}
	if shutdownSchedule != "" {
		t.Errorf("shutdownSchedule != \"\", actual \"%s\"", shutdownSchedule)
	}
	if len(machineTypes) != 1 {
		t.Error("len(machineTypesMap) != 1, actual ", len(machineTypes))
	}

	_, _, err = parseConfig(strings.NewReader("machine-types:"))
	if err == nil {
		t.Error(errors.New("it should be raised an error"))
	}
}
