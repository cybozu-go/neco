package main

import (
	"io"

	"gopkg.in/yaml.v2"
)

type Metric struct {
	Name   string            `yaml:"name"`
	Labels map[string]string `yaml:"labels,omitempty"`
}

type MachineType struct {
	Name    string   `yaml:"name"`
	Metrics []Metric `yaml:"metrics,omitempty"`
}

type Config struct {
	MachineTypes []MachineType `yaml:"machine-types"`
}

func parseConfig(reader io.Reader) (*Config, error) {
	cfg := new(Config)
	err := yaml.NewDecoder(reader).Decode(cfg)
	if err != nil {
		return nil, err
	}
	return cfg, nil
}
