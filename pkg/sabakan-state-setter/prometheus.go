package sss

import (
	"context"
	"net/http"
	"time"

	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
)

type promClient struct {
	client *http.Client
}

// PrometheusClient is interface for prometheus client
type PrometheusClient interface {
	ConnectMetricsServer(ctx context.Context, addr string) (map[string]*dto.MetricFamily, error)
}

// newPromClient returns PrometheusClient
func newPromClient() PrometheusClient {
	return &promClient{
		client: &http.Client{},
	}
}

// ConnectMetricsServer returns metrics from give exporter address
func (p *promClient) ConnectMetricsServer(ctx context.Context, addr string) (map[string]*dto.MetricFamily, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Second*10)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, addr, nil)
	if err != nil {
		return nil, err
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return (&expfmt.TextParser{}).TextToMetricFamilies(resp.Body)
}
