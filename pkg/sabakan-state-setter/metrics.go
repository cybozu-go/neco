package main

import (
	"github.com/cybozu-go/log"
	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/prom2json"
)

func (ms *machineStateSource) readAndSetMetrics(mfChan <-chan *dto.MetricFamily) error {
	var result []*prom2json.Family
	for mf := range mfChan {
		result = append(result, prom2json.NewFamily(mf))
	}

	for _, r := range result {
		var metrics machineMetrics
		for _, item := range r.Metrics {
			metric, ok := item.(prom2json.Metric)
			if !ok {
				log.Warn("failed to cast an item to prom2json.Metric", map[string]interface{}{
					"item": item,
				})
				continue
			}
			metrics = append(metrics, metric)
		}

		ms.metrics[r.Name] = metrics
	}

	return nil
}
