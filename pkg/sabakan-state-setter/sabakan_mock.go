package sss

import (
	"context"

	"github.com/cybozu-go/sabakan/v2"
)

type gqlMockClient struct {
	machines []*machine
}

func newMockGQLClient() *gqlMockClient {
	return &gqlMockClient{
		machines: []*machine{
			{
				Serial:   "00000001",
				Type:     "qemu",
				IPv4Addr: "10.0.0.100",
				BMCAddr:  "20.0.0.100",
				State:    sabakan.StateUninitialized,
			},
		},
	}
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
