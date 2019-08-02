package sss

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestParseConfigFile(t *testing.T) {
	fileContent := `machine-types:
  - name: qemu
    grace-period: 10s
  - name: boot
    metrics:
      - name: a
      - name: b
        labels:
          aaa: bbb
`
	cfg, err := parseConfig(strings.NewReader(fileContent))
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.MachineTypes) != 2 {
		t.Error("len(cfg.MachineTypes) != 2, actual ", len(cfg.MachineTypes))
	}
	if cfg.MachineTypes[0].GracePeriod.Duration != 10*time.Second {
		t.Error("GracePeriod is not set")
	}
	if cfg.MachineTypes[1].GracePeriod.Duration != time.Hour {
		t.Error("default value of GracePeriod is not set")
	}

	_, err = parseConfig(strings.NewReader("machine-types:"))
	if err == nil {
		t.Error(errors.New("it should be raised an error"))
	}
}
