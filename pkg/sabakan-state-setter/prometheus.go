package sss

import (
	"context"

	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/prom2json"
)

type promClient struct{}

// PrometheusClient is interface for prometheus client
type PrometheusClient interface {
	ConnectMetricsServer(ctx context.Context, addr string) (chan *dto.MetricFamily, error)
}

// newPromClient returns PrometheusClient
func newPromClient() PrometheusClient {
	return &promClient{}
}

// ConnectMetricsServer returns metrics from give exporter address
func (p *promClient) ConnectMetricsServer(ctx context.Context, addr string) (chan *dto.MetricFamily, error) {
	mfChan := make(chan *dto.MetricFamily, 1024)
	err := prom2json.FetchMetricFamilies(addr, mfChan, nil)
	if err != nil {
		return nil, err
	}
	return mfChan, nil
}
