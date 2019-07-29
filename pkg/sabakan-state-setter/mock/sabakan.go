package mock

import (
	"context"

	sss "github.com/cybozu-go/neco/pkg/sabakan-state-setter"
)

type gqlClient struct{}

// NewGQLClient is returns a mock sabakan GraphQL client
func NewGQLClient(address string) (sss.SabakanGQLClient, error) {
	return &gqlClient{}, nil
}

func (g *gqlClient) GetSabakanMachines(ctx context.Context) (*sss.SearchMachineResponse, error) {
	return nil, nil
}

func (g *gqlClient) UpdateSabakanState(ctx context.Context, ms *sss.MachineStateSource, state string) error {
	return nil
}
