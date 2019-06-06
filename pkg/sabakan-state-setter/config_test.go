package main

import (
	"errors"
	"strings"
	"testing"
)

func TestParseConfigFile(t *testing.T) {
	fileContent := `machine-types:
  - name: qemu
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

	_, err = parseConfig(strings.NewReader("machine-types:"))
	if err == nil {
		t.Error(errors.New("it should be raised an error"))
	}
}
