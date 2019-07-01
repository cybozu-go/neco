package main

import (
	"errors"
	"io"

	"gopkg.in/yaml.v2"
)

type targetMetric struct {
	Name                string    `yaml:"name"`
	Selector            *selector `yaml:"selector,omitempty"`
	MinimumHealthyCount *int      `yaml:"minimum-healthy-count,omitempty"`
}

type selector struct {
	labels      map[string]string
	labelPrefix map[string]string
}

type machineType struct {
	Name             string         `yaml:"name"`
	MetricsCheckList []targetMetric `yaml:"metrics,omitempty"`
}

type config struct {
	MachineTypes []machineType `yaml:"machine-types"`
}

func parseConfig(reader io.Reader) (*config, error) {
	cfg := new(config)
	err := yaml.NewDecoder(reader).Decode(cfg)
	if err != nil {
		return nil, err
	}

	if len(cfg.MachineTypes) == 0 {
		return nil, errors.New("machine-types are not defined")
	}
	return cfg, nil
}
