package sss

import (
	"net"

	"github.com/cybozu-go/log"
	serf "github.com/hashicorp/serf/client"
	dto "github.com/prometheus/client_model/go"
)

// machineStateSource is a struct of machine state collection
type machineStateSource struct {
	serial string
	ipv4   string

	serfStatus  *serf.Member
	metrics     map[string]*dto.MetricFamily
	machineType *machineType
}

func newMachineStateSource(m *machine, members []serf.Member, machineTypes []*machineType) *machineStateSource {
	machineType := findMachineType(m.Type, machineTypes)
	if machineType == nil {
		log.Warn(machineTypeLabelName+" is not defined", map[string]interface{}{
			"serial": m.Serial,
			"ipv4":   m.IPv4Addr,
			"name":   m.Type,
		})
	}
	return &machineStateSource{
		serial:      m.Serial,
		ipv4:        m.IPv4Addr,
		serfStatus:  findMember(members, m.IPv4Addr),
		machineType: machineType,
	}
}

func findMember(members []serf.Member, addr string) *serf.Member {
	for _, member := range members {
		if member.Addr.Equal(net.ParseIP(addr)) {
			return &member
		}
	}
	return nil
}

func findMachineType(name string, machineTypes []*machineType) *machineType {
	for _, mt := range machineTypes {
		if mt.Name == name {
			return mt
		}
	}
	return nil
}
