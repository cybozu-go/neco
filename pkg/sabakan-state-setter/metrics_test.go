package main

import (
	"context"
	"reflect"
	"testing"

	"github.com/cybozu-go/well"
	dto "github.com/prometheus/client_model/go"
)

func toPtrString(s string) *string {
	return &s
}

func toPtrMetricType(i dto.MetricType) *dto.MetricType {
	return &i
}

func mockFetcher(ctx context.Context, addr string) (chan *dto.MetricFamily, error) {
	ch := make(chan *dto.MetricFamily, 1024)
	return ch, nil
}

// TestReadAndSetMetrics
func TestReadAndSetMetrics(t *testing.T) {
	t.Fatal("exit")
	testCases := []struct {
		input  []*dto.MetricFamily
		expect map[string]machineMetrics
	}{
		{
			input:  []*dto.MetricFamily{},
			expect: make(map[string]machineMetrics),
		},
		{
			input: []*dto.MetricFamily{
				{
					Name: toPtrString("name1"),
					Help: toPtrString("help1"),
					Type: toPtrMetricType(dto.MetricType_COUNTER),
					Metric: []*dto.Metric{
						{
							Label: []*dto.LabelPair{
								{Name: toPtrString("label11"), Value: toPtrString("value11")},
								{Name: toPtrString("label12"), Value: toPtrString("value12")},
							},
						},
					},
				},
				{
					Name: toPtrString("name2"),
					Help: toPtrString("help2"),
					Type: toPtrMetricType(dto.MetricType_GAUGE),
					Metric: []*dto.Metric{
						{
							Label: []*dto.LabelPair{
								{Name: toPtrString("label21"), Value: toPtrString("value21")},
								{Name: toPtrString("label22"), Value: toPtrString("value22")},
							},
						},
					},
				},
			},
			expect: map[string]machineMetrics{
				"name1": machineMetrics{
					{
						Labels: map[string]string{
							"label11": "value11",
							"label12": "value12",
						},
						Value: "0.",
					},
				},
				"name2": machineMetrics{
					{
						Labels: map[string]string{
							"label21": "value21",
							"label22": "value22",
						},
						Value: "0.",
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
			fetcher:     mockFetcher,
		}
		env := well.NewEnvironment(context.Background())
		env.Go(func(ctx context.Context) error {
			ch, _ := s.fetcher(context.Background(), "xxx.xxx.xxx.xxx")
			for _, ii := range tt.input {
				ch <- ii
			}
			return s.readAndSetMetrics(ch)
		})
		env.Stop()
		err := env.Wait()
		if err != nil {
			t.Fatal("error occurred when get metrics")
		}

		if !reflect.DeepEqual(s.metrics, tt.expect) {
			t.Errorf("metrics map mismatch: acutual=%v, expect=%v", s.metrics, tt.expect)
		}
	}
}
