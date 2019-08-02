package sss

import (
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"time"

	"sigs.k8s.io/yaml"
)

type targetMetric struct {
	Name                string    `json:"name"`
	Selector            *selector `json:"selector,omitempty"`
	MinimumHealthyCount *int      `json:"minimum-healthy-count,omitempty"`
}

type selector struct {
	Labels      map[string]string `json:"labels,omitempty"`
	LabelPrefix map[string]string `json:"label-prefix,omitempty"`
}

type machineType struct {
	Name             string         `json:"name"`
	MetricsCheckList []targetMetric `json:"metrics,omitempty"`
	GracePeriod      duration       `json:"grace-period"`
}

type config struct {
	MachineTypes []*machineType `json:"machine-types"`
}

type duration struct {
	time.Duration
}

func (d duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.String())
}

func (d *duration) UnmarshalJSON(b []byte) error {
	var v interface{}
	err := json.Unmarshal(b, &v)
	if err != nil {
		return err
	}

	switch v := v.(type) {
	case float64:
		d.Duration = time.Duration(v)
		return nil
	case string:
		duration, err := time.ParseDuration(v)
		if err != nil {
			return err
		}
		d.Duration = duration
		return nil
	default:
		return errors.New("invalid duration")
	}
}

func parseConfig(reader io.Reader) (*config, error) {
	data, err := ioutil.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	cfg := &config{}
	err = yaml.Unmarshal(data, cfg)
	if err != nil {
		return nil, err
	}

	if len(cfg.MachineTypes) == 0 {
		return nil, errors.New("machine-types are not defined")
	}
	for _, t := range cfg.MachineTypes {
		if t.GracePeriod.Duration == 0 {
			t.GracePeriod.Duration = time.Hour
		}
	}
	return cfg, nil
}
