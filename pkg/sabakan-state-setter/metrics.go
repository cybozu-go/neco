package main

import (
	"context"

	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/prom2json"
)

func (source *machineStateSource) getMetrics(ctx context.Context) error {
	mfChan := make(chan *dto.MetricFamily, 1024)
	addr := "http://" + source.ipv4 + ":9105/metrics"
	err := prom2json.FetchMetricFamilies(addr, mfChan, "", "", true)
	if err != nil {
		return err
	}
	var result []*prom2json.Family
	for mf := range mfChan {
		result = append(result, prom2json.NewFamily(mf))
	}

	var metrics machineMetrics
	for _, r := range result {
		for _, item := range r.Metrics {
			metric, ok := item.(prom2json.Metric)
			if !ok {
				continue
			}
			metrics = append(metrics, metric)
		}
		source.metrics[r.Name] = metrics
	}

	return nil
}
