package sss

import (
	"net"

	"github.com/cybozu-go/log"
	serf "github.com/hashicorp/serf/client"
	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/prom2json"
)

const machineTypeLabelName = "machine-type"

// machineStateSource is a struct of machine state collection
type machineStateSource struct {
	serial string
	ipv4   string

	serfStatus  *serf.Member
	metrics     map[string]machineMetrics
	machineType *machineType
}

type machineMetrics []prom2json.Metric

func newMachineStateSource(m machine, members []serf.Member, machineTypes []*machineType) *machineStateSource {
	return &machineStateSource{
		serial:      m.Spec.Serial,
		ipv4:        m.Spec.IPv4[0],
		serfStatus:  findMember(members, m.Spec.IPv4[0]),
		machineType: findMachineType(&m, machineTypes),
		metrics:     map[string]machineMetrics{},
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

func findMachineType(m *machine, machineTypes []*machineType) *machineType {
	mtLabel := findLabel(m.Spec.Labels, machineTypeLabelName)
	if mtLabel == nil {
		log.Warn(machineTypeLabelName+" is not set", map[string]interface{}{
			"serial": m.Spec.Serial,
		})
		return nil
	}
	for _, mt := range machineTypes {
		if mt.Name == mtLabel.Value {
			return mt
		}
	}

	log.Warn(machineTypeLabelName+"["+mtLabel.Value+"] is not defined", map[string]interface{}{
		"serial": m.Spec.Serial,
	})
	return nil
}

func findLabel(labels []label, key string) *label {
	for _, l := range labels {
		if l.Name == key {
			return &l
		}
	}
	return nil
}

func (mss *machineStateSource) readAndSetMetrics(mfChan <-chan *dto.MetricFamily) error {
	var result []*prom2json.Family
	for mf := range mfChan {
		result = append(result, prom2json.NewFamily(mf))
	}

	for _, r := range result {
		var metrics machineMetrics
		for _, item := range r.Metrics {
			metric, ok := item.(prom2json.Metric)
			if !ok {
				log.Warn("failed to cast an item to prom2json.Metric", map[string]interface{}{
					"item": item,
				})
				continue
			}
			metrics = append(metrics, metric)
		}

		mss.metrics[r.Name] = metrics
	}

	return nil
}
