package sss

import (
	"net"

	serf "github.com/hashicorp/serf/client"
)

type serfMockClient struct {
}

// newPromClient returns PrometheusClient
func newMockSerfClient() (*serfMockClient, error) {
	return &serfMockClient{}, nil
}

// GetSerfMembers returns serf members
func (s *serfMockClient) GetSerfMembers() ([]serf.Member, error) {
	return []serf.Member{
		{
			Status: "alive",
			Tags: map[string]string{
				systemdUnitsFailedTag: "",
			},
			Addr: net.ParseIP("10.0.0.100"),
		},
	}, nil
}
