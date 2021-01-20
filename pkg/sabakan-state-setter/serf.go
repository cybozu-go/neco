package sss

import serf "github.com/hashicorp/serf/client"

type serfClient struct {
	*serf.RPCClient
}

// SerfClient is interface for serf client
type SerfClient interface {
	GetSerfMembers() ([]serf.Member, error)
}

func newSerfClient(address string) (SerfClient, error) {
	c, err := serf.NewRPCClient(address)
	if err != nil {
		return nil, err
	}
	return &serfClient{c}, nil
}

// GetSerfMembers returns serf members
func (s *serfClient) GetSerfMembers() ([]serf.Member, error) {
	return s.Members()
}
