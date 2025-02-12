package sss

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/cybozu-go/sabakan/v3"
	dto "github.com/prometheus/client_model/go"
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

func (s *selector) Match(mf *dto.MetricFamily) []*dto.Metric {
	if s == nil {
		return mf.Metric
	}

	var result []*dto.Metric
	for _, m := range mf.Metric {
		if s.isMetricMatchedLabels(m) && s.isMetricHasPrefix(m) {
			result = append(result, m)
		}
	}
	return result
}

// isMetricMatchedLabels returns true if metric has labels and values specified in selector.Labels.
// If selector.Labels == nil, this returns true.
func (s *selector) isMetricMatchedLabels(metric *dto.Metric) bool {
OUTER:
	for k, v := range s.Labels {
		for _, label := range metric.GetLabel() {
			if label.GetName() != k {
				continue
			}
			if label.GetValue() == v {
				continue OUTER
			}
			return false
		}
		return false
	}
	return true
}

// isMetricHasPrefix returns true if metric has labels and values specified in selector.LabelPrefix.
// If selector.LabelPrefix == nil, this returns true.
func (s *selector) isMetricHasPrefix(metric *dto.Metric) bool {
OUTER:
	for k, prefix := range s.LabelPrefix {
		for _, label := range metric.GetLabel() {
			if label.GetName() != k {
				continue
			}
			if strings.HasPrefix(label.GetValue(), prefix) {
				continue OUTER
			}
			return false
		}
		return false
	}
	return true
}

type machineType struct {
	Name             string         `json:"name"`
	MetricsCheckList []targetMetric `json:"metrics,omitempty"`
	GracePeriod      duration       `json:"grace-period"`
}

type triggerAlert struct {
	Name         string               `json:"name"`
	Labels       map[string]string    `json:"labels"`
	AddressLabel string               `json:"address-label"`
	SerialLabel  string               `json:"serial-label"`
	State        sabakan.MachineState `json:"state"`
}

type alertMonitor struct {
	AlertmanagerEndpoint string         `json:"alertmanager-endpoint"`
	TriggerAlerts        []triggerAlert `json:"trigger-alerts"`
}

type config struct {
	ShutdownSchedule string         `json:"shutdown-schedule,omitempty"`
	MachineTypes     []*machineType `json:"machine-types"`
	AlertMonitor     *alertMonitor  `json:"alert-monitor"`
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

func readConfigFile(name string) (string, map[string]*machineType, *alertMonitor, error) {
	cf, err := os.Open(name)
	if err != nil {
		return "", nil, nil, err
	}
	defer cf.Close()

	shutdownSchedule, machineTypes, alertConfig, err := parseConfig(cf)
	if err != nil {
		return "", nil, nil, err
	}
	return shutdownSchedule, machineTypes, alertConfig, nil
}

func parseConfig(reader io.Reader) (string, map[string]*machineType, *alertMonitor, error) {
	data, err := io.ReadAll(reader)
	if err != nil {
		return "", nil, nil, err
	}

	cfg := &config{}
	err = yaml.Unmarshal(data, cfg)
	if err != nil {
		return "", nil, nil, err
	}

	if len(cfg.MachineTypes) == 0 {
		return "", nil, nil, errors.New("machine-types are not defined")
	}
	machineTypes := make(map[string]*machineType)
	for _, t := range cfg.MachineTypes {
		if t.GracePeriod.Duration == 0 {
			t.GracePeriod.Duration = time.Hour
		}
		machineTypes[t.Name] = t
	}

	if cfg.AlertMonitor != nil {
		for _, triggerAlert := range cfg.AlertMonitor.TriggerAlerts {
			if (len(triggerAlert.AddressLabel) == 0) == (len(triggerAlert.SerialLabel) == 0) {
				return "", nil, nil, fmt.Errorf("exactly one of `address-label` and `serial-label` is required for %q", triggerAlert.Name)
			}
			switch triggerAlert.State {
			case sabakan.StateUnreachable:
			case sabakan.StateUnhealthy:
			default:
				return "", nil, nil, fmt.Errorf("invalid state %q in %q", triggerAlert.State, triggerAlert.Name)
			}
		}
	}

	return cfg.ShutdownSchedule, machineTypes, cfg.AlertMonitor, nil
}
