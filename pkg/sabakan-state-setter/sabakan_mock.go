package sss

import (
	"context"

	sabakan "github.com/cybozu-go/sabakan/v2"
)

type gqlMockClient struct {
	machine *sabakan.Machine
}

func newMockGQLClient(labelMachineType string) *gqlMockClient {
	return &gqlMockClient{
		machine: &sabakan.Machine{
			Spec: sabakan.MachineSpec{
				Serial: "00000001",
				Labels: map[string]string{
					"machine-type": labelMachineType,
				},
			},
			Status: sabakan.MachineStatus{
				State: sabakan.StateUninitialized,
			},
		},
	}
}

func (g *gqlMockClient) GetSabakanMachines(ctx context.Context) (*SearchMachineResponse, error) {
	return nil, nil
}

func (g *gqlMockClient) UpdateSabakanState(ctx context.Context, ms *MachineStateSource, state string) error {
	g.machine.Status.State = sabakan.MachineState(state)
	return nil
}
