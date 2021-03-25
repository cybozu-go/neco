package sss

import (
	"github.com/cybozu-go/log"
	dto "github.com/prometheus/client_model/go"
)

// machineStateSource is a struct of machine state collection
type machineStateSource struct {
	serial string
	ipv4   string

	serfStatus  *serfStatus
	machineType *machineType
	metrics     map[string]*dto.MetricFamily
}

func newMachineStateSource(m *machine, serfStatuses map[string]*serfStatus, machineTypes map[string]*machineType) *machineStateSource {
	machineType, ok := machineTypes[m.Type]
	if !ok {
		log.Warn(machineTypeLabelName+" is not defined", map[string]interface{}{
			"serial": m.Serial,
			"ipv4":   m.IPv4Addr,
			"name":   m.Type,
		})
	}

	return &machineStateSource{
		serial:      m.Serial,
		ipv4:        m.IPv4Addr,
		serfStatus:  serfStatuses[m.IPv4Addr],
		machineType: machineType,
	}
}
