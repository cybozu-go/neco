package sss

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
	return s.status, nil
}
