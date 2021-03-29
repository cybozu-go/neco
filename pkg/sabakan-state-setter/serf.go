package sss

import serf "github.com/hashicorp/serf/client"

const (
	systemdUnitsFailedTag = "systemd-units-failed"
)

type serfClient struct {
	*serf.RPCClient
}

// SerfClient is interface for serf client
type SerfClient interface {
	GetSerfStatus() (map[string]*serfStatus, error)
}

type serfStatus struct {
	Status             string
	SystemdUnitsFailed *string
}

func newSerfClient(address string) (SerfClient, error) {
	c, err := serf.NewRPCClient(address)
	if err != nil {
		return nil, err
	}
	return &serfClient{c}, nil
}

func strPtr(str string) *string {
	return &str
}

// GetSerfMembers returns serf members
func (s *serfClient) GetSerfStatus() (map[string]*serfStatus, error) {
	members, err := s.Members()
	if err != nil {
		return nil, err
	}

	ret := make(map[string]*serfStatus)
	for _, m := range members {
		ipv4 := m.Addr.String()

		var systemdStatus *string
		if tagValue, ok := m.Tags[systemdUnitsFailedTag]; ok {
			systemdStatus = strPtr(tagValue)
		}

		ret[ipv4] = &serfStatus{
			Status:             m.Status,
			SystemdUnitsFailed: systemdStatus,
		}
	}
	return ret, nil
}
