package mock

import (
	"context"

	sss "github.com/cybozu-go/neco/pkg/sabakan-state-setter"

	dto "github.com/prometheus/client_model/go"
)

type promClient struct{}

// NewPromClient is returns a mock sabakan GraphQL client
func NewPromClient(address string) (sss.PrometheusClient, error) {
	return &promClient{}, nil
}

func (g *promClient) ConnectMetricsServer(ctx context.Context, addr string) (chan *dto.MetricFamily, error) {
	return nil, nil

}
