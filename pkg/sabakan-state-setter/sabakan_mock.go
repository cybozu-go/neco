package sss

import (
	"context"

	sabakan "github.com/cybozu-go/sabakan/v2"
)

type gqlMockClient struct {
	machines []*sabakan.Machine
}

func newMockGQLClient() *gqlMockClient {
	return &gqlMockClient{
		machines: []*sabakan.Machine{
			{
				Spec: sabakan.MachineSpec{
					Serial: "00000001",
				},
				Status: sabakan.MachineStatus{
					State: sabakan.StateUninitialized,
				},
			},
		},
	}
}

func (g *gqlMockClient) GetSabakanMachines(ctx context.Context) (*SearchMachineResponse, error) {
	return nil, nil
}

func (g *gqlMockClient) UpdateSabakanState(ctx context.Context, ms *MachineStateSource, state string) error {
	g.machines[0].Status.State = sabakan.MachineState(state)
	return nil
}
