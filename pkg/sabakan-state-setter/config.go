package main

import (
	"errors"
	"io"

	"gopkg.in/yaml.v2"
)

type metric struct {
	Name   string            `yaml:"name"`
	Labels map[string]string `yaml:"labels,omitempty"`
}

type machineType struct {
	Name             string   `yaml:"name"`
	MetricsCheckList []metric `yaml:"metrics,omitempty"`
}

type connfig struct {
	MachineTypes []machineType `yaml:"machine-types"`
}

func parseConfig(reader io.Reader) (*connfig, error) {
	cfg := new(connfig)
	err := yaml.NewDecoder(reader).Decode(cfg)
	if err != nil {
		return nil, err
	}

	if len(cfg.MachineTypes) == 0 {
		return nil, errors.New("machine-types are not defined")
	}
	return cfg, nil
}
