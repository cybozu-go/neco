package sss

import (
	"context"
	"strings"

	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
)

type promMockClient struct {
	metrics map[string]string
}

var _ PrometheusClient = &promMockClient{}

// NewMockPromClient is returns a mock sabakan GraphQL client
func newMockPromClient(metrics map[string]string) *promMockClient {
	return &promMockClient{metrics: metrics}
}

func (g *promMockClient) ConnectMetricsServer(ctx context.Context, addr string) (map[string]*dto.MetricFamily, error) {
	return (&expfmt.TextParser{}).TextToMetricFamilies(strings.NewReader(g.metrics[addr]))
}
