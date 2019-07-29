package sss

import (
	"context"
	"strings"

	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/prom2json"
)

type promMockClient struct{}

// NewMockPromClient is returns a mock sabakan GraphQL client
func newMockPromClient() PrometheusClient {
	return &promMockClient{}
}

func (g *promMockClient) ConnectMetricsServer(ctx context.Context, addr string) (chan *dto.MetricFamily, error) {
	input := `hw_systems_processors_status_health{processor="CPU.Socket.1"} 0
	hw_systems_processors_status_health{processor="CPU.Socket.2"} 1
	`
	ch := make(chan *dto.MetricFamily, 1024)
	err := prom2json.ParseReader(strings.NewReader(input), ch)
	if err != nil {
		return nil, err
	}

	return ch, nil
}
