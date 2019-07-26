package sss

import (
	"context"

	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/prom2json"
)

// PrometheusClient is prometheus client
type PrometheusClient struct{}

// Prometheus is interface for prometheus client
type Prometheus interface {
	ConnectMetricsServer(ctx context.Context, addr string) (chan *dto.MetricFamily, error)
}

// NewPrometheusClient returns PrometheusClient
func NewPrometheusClient() *PrometheusClient {
	return &PrometheusClient{}
}

// ConnectMetricsServer returns metrics from give exporter address
func (p *PrometheusClient) ConnectMetricsServer(ctx context.Context, addr string) (chan *dto.MetricFamily, error) {
	mfChan := make(chan *dto.MetricFamily, 1024)
	err := prom2json.FetchMetricFamilies(addr, mfChan, "", "", true)
	if err != nil {
		return nil, err
	}
	return mfChan, nil
}
