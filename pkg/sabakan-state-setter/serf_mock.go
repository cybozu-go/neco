package sss

type serfMockClient struct {
}

func newMockSerfClient() (*serfMockClient, error) {
	return &serfMockClient{}, nil
}

// GetSerfMembers returns serf members
func (s *serfMockClient) GetSerfStatus() (map[string]*serfStatus, error) {
	return map[string]*serfStatus{
		"10.0.0.100": {
			Status:             "alive",
			SystemdUnitsFailed: strPtr(""),
		},
	}, nil
}
