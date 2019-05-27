package main

import (
	"reflect"
	"strings"
	"testing"

	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/prom2json"
)

func toPtrString(s string) *string {
	return &s
}

func toPtrMetricType(i dto.MetricType) *dto.MetricType {
	return &i
}

func TestReadAndSetMetrics(t *testing.T) {
	testCases := []struct {
		input  string
		expect map[string]machineMetrics
	}{
		{
			input:  "",
			expect: make(map[string]machineMetrics),
		},
		{
			input: `hw_processor_status_health{processor="CPU.Socket.1",system="System.Embedded.1"} 0
			hw_storage_controller_status_health{controller="AHCI.Embedded.1-1",system="System.Embedded.1"} -1
			hw_storage_controller_status_health{controller="AHCI.Embedded.2-1",system="System.Embedded.1"} -1
			hw_storage_controller_status_health{controller="AHCI.Slot.1-1",system="System.Embedded.1"} 0
			`,
			expect: map[string]machineMetrics{
				"hw_processor_status_health": {
					{
						Labels: map[string]string{
							"processor": "CPU.Socket.1",
							"system":    "System.Embedded.1",
						},
						Value: "0",
					},
				},
				"hw_storage_controller_status_health": {
					{
						Labels: map[string]string{
							"controller": "AHCI.Embedded.1-1",
							"system":     "System.Embedded.1",
						},
						Value: "-1",
					},
					{
						Labels: map[string]string{
							"controller": "AHCI.Embedded.2-1",
							"system":     "System.Embedded.1",
						},
						Value: "-1",
					},
					{
						Labels: map[string]string{
							"controller": "AHCI.Slot.1-1",
							"system":     "System.Embedded.1",
						},
						Value: "0",
					},
				},
			},
		},
	}

	for _, tt := range testCases {
		s := &machineStateSource{
			serial:      "serial",
			ipv4:        "ipv4",
			serfStatus:  nil,
			metrics:     map[string]machineMetrics{},
			machineType: nil,
		}
		ch := make(chan *dto.MetricFamily, 1024)
		err := prom2json.ParseReader(strings.NewReader(tt.input), ch)
		if err != nil {
			t.Fatal("cannot parse input", err)
		}
		s.readAndSetMetrics(ch)
		if !reflect.DeepEqual(s.metrics, tt.expect) {
			t.Errorf("metrics map mismatch: actual=%v, expect=%v", s.metrics, tt.expect)
		}
	}
}
