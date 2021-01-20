package sss

import (
	"context"
	"strings"

	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
)

type promMockClient struct {
	input string
}

// NewMockPromClient is returns a mock sabakan GraphQL client
func newMockPromClient(input string) PrometheusClient {
	return &promMockClient{input: input}
}

func (g *promMockClient) ConnectMetricsServer(ctx context.Context, addr string) (map[string]*dto.MetricFamily, error) {
	return (&expfmt.TextParser{}).TextToMetricFamilies(strings.NewReader(g.input))
}
