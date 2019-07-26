package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/well"
	serf "github.com/hashicorp/serf/client"
	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/prom2json"
)

const machineTypeLabelName = "machine-type"

var (
	flagSabakanAddress = flag.String("sabakan-address", "http://localhost:10080", "sabakan address")
	flagConfigFile     = flag.String("config-file", "", "path of config file")
	flagInterval       = flag.String("interval", "1m", "interval of scraping metrics")
	flagParallelSize   = flag.Int("parallel", 30, "parallel size")
	problematicStates  = []string{"unreachable", "unhealthy"}
)

type machineStateSource struct {
	serial string
	ipv4   string

	serfStatus  *serf.Member
	metrics     map[string]machineMetrics
	machineType *machineType

	stateCandidate               string
	stateCandidateFirstDetection time.Time
}

type machineMetrics []prom2json.Metric

func connectMetricsServer(ctx context.Context, addr string) (chan *dto.MetricFamily, error) {
	mfChan := make(chan *dto.MetricFamily, 1024)
	err := prom2json.FetchMetricFamilies(addr, mfChan, "", "", true)
	if err != nil {
		return nil, err
	}
	return mfChan, nil
}

func main() {
	flag.Parse()
	err := well.LogConfig{}.Apply()
	if err != nil {
		log.ErrorExit(err)
	}

	ctr, err := newController(*flagInterval, *flagSabakanAddress, *flagConfigFile)
	if err != nil {
		log.ErrorExit(err)
	}
	well.Go(func(ctx context.Context) error {
		return ctr.runPeriodically(ctx)
	})
	well.Stop()
	err = well.Wait()
	if err != nil && !well.IsSignaled(err) {
		fmt.Println(err)
		os.Exit(1)
	}
}

func newMachineStateSource(m machine, members []serf.Member, machineTypes []*machineType, pms []problematicMachine) (*machineStateSource, error) {
	mss := machineStateSource{
		serial:      m.Spec.Serial,
		ipv4:        m.Spec.IPv4[0],
		serfStatus:  findMember(members, m.Spec.IPv4[0]),
		machineType: findMachineType(&m, machineTypes),
		metrics:     map[string]machineMetrics{},
	}
	for _, pm := range pms {
		if mss.serial == pm.Serial {
			mss.stateCandidate = pm.State
			t, err := time.Parse("2009-11-10 23:00:00 +0000 UTC", pm.FirstDetection)
			if err != nil {
				return nil, err
			}
			mss.stateCandidateFirstDetection = t
		}
	}
	return &mss, nil
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
