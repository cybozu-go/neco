package sss

import (
	"context"

	"github.com/cybozu-go/sabakan/v2"
)

type gqlMockClient struct {
	machines []*machine
}

func newMockGQLClient(machines []*machine) *gqlMockClient {
	return &gqlMockClient{machines: machines}
}

func (g *gqlMockClient) GetSabakanMachines(ctx context.Context) ([]*machine, error) {
	return g.machines, nil
}

func (g *gqlMockClient) UpdateSabakanState(ctx context.Context, serial string, state sabakan.MachineState) error {
	for _, m := range g.machines {
		if m.Serial == serial {
			m.State = state
		}
	}
	return nil
}

// test function
func (g *gqlMockClient) getState(serial string) sabakan.MachineState {
	for _, m := range g.machines {
		if m.Serial == serial {
			return m.State
		}
	}
	return ""
}
