package sss

import "errors"

type serfMockClient struct {
	status map[string]*serfStatus
}

func newMockSerfClient(status map[string]*serfStatus) (*serfMockClient, error) {
	return &serfMockClient{
		status: status,
	}, nil
}

// GetSerfMembers returns serf members
func (s *serfMockClient) GetSerfStatus() (map[string]*serfStatus, error) {
	if s.status == nil {
		return nil, errors.New("failed to get serf status")
	}
	return s.status, nil
}
