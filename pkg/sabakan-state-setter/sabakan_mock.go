package sss

import (
	"context"

	"github.com/cybozu-go/sabakan/v2"
)

type sabakanMockClient struct {
	machines []*machine
	count    map[string]int
}

func newMockSabakanClient(machines []*machine) *sabakanMockClient {
	return &sabakanMockClient{
		machines: machines,
		count:    map[string]int{},
	}
}

func (c *sabakanMockClient) GetSabakanMachines(ctx context.Context) ([]*machine, error) {
	return c.machines, nil
}

func (c *sabakanMockClient) UpdateSabakanState(ctx context.Context, serial string, state sabakan.MachineState) error {
	for _, m := range c.machines {
		if m.Serial == serial {
			m.State = state
		}
	}
	return nil
}

func (c *sabakanMockClient) CryptsDelete(ctx context.Context, serial string) error {
	c.count[serial]++
	return nil
}

// test function
func (c *sabakanMockClient) getState(serial string) sabakan.MachineState {
	for _, m := range c.machines {
		if m.Serial == serial {
			return m.State
		}
	}
	return ""
}

// test function
func (c *sabakanMockClient) getCryptsDeleteCount(serial string) int {
	return c.count[serial]
}
