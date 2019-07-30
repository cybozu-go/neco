package sss

import (
	"context"

	sabakan "github.com/cybozu-go/sabakan/v2"
)

type gqlMockClient struct {
	machine          *sabakan.Machine
	labelMachineType string
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
		labelMachineType: labelMachineType,
	}
}

func (g *gqlMockClient) GetSabakanMachines(ctx context.Context) (*SearchMachineResponse, error) {
	return &SearchMachineResponse{
		SearchMachines: []machine{
			{
				Spec: spec{
					Serial: "00000001",
					IPv4:   []string{"10.0.0.100"},
					Labels: []label{
						{
							Name:  "machine-type",
							Value: g.labelMachineType,
						},
					},
				},
			},
		},
	}, nil
}

func (g *gqlMockClient) UpdateSabakanState(ctx context.Context, ms *MachineStateSource, state string) error {
	g.machine.Status.State = sabakan.MachineState(state)
	return nil
}
