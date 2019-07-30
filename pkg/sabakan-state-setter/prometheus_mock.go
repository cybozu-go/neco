package sss

import (
	"context"
	"strings"

	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/prom2json"
)

type promMockClient struct {
	input string
}

// NewMockPromClient is returns a mock sabakan GraphQL client
func newMockPromClient(input string) PrometheusClient {
	return &promMockClient{input: input}
}

func (g *promMockClient) ConnectMetricsServer(ctx context.Context, addr string) (chan *dto.MetricFamily, error) {
	ch := make(chan *dto.MetricFamily, 1024)
	err := prom2json.ParseReader(strings.NewReader(g.input), ch)
	if err != nil {
		return nil, err
	}

	return ch, nil
}
