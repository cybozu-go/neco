package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net"
	"os"

	"github.com/cybozu-go/log"
	gqlsabakan "github.com/cybozu-go/sabakan/v2/gql"
	"github.com/cybozu-go/well"
	serf "github.com/hashicorp/serf/client"
	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/prom2json"
	"github.com/vektah/gqlparser/gqlerror"
)

const machineTypeLabelName = "machine-type"

var (
	flagSabakanAddress = flag.String("sabakan-address", "http://localhost:10080", "sabakan address")
	flagConfigFile     = flag.String("config-file", "", "path of config file")
	flagParallelSize   = flag.Int("parallel", 30, "parallel size")
)

type machineStateSource struct {
	serial string
	ipv4   string

	serfStatus  *serf.Member
	metrics     map[string]machineMetrics
	machineType *machineType
	fetcher     fetcherFromMetricsServer
}

type machineMetrics []prom2json.Metric

type fetcherFromMetricsServer func(context.Context, string) (chan *dto.MetricFamily, error)

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
	well.Go(run)
	well.Stop()
	err := well.Wait()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	confFile, err := os.Open(*flagConfigFile)
	if err != nil {
		return err
	}
	defer confFile.Close()

	cfg, err := parseConfig(confFile)
	if err != nil {
		return err
	}

	sm := new(searchMachineResponse)
	gql, err := newGQLClient(*flagSabakanAddress)
	if err != nil {
		return err
	}

	sm, err = gql.getSabakanMachines(ctx)
	if err != nil {
		return err
	}
	if len(sm.SearchMachines) == 0 {
		return errors.New("no machines found")
	}

	// Get serf members
	serfc, err := serf.NewRPCClient("127.0.0.1:7373")
	if err != nil {
		return err
	}
	members, err := getSerfMembers(serfc)
	if err != nil {
		return err
	}

	// Construct a slice of MachineStateSource
	mss := make([]machineStateSource, 0, len(sm.SearchMachines))
	for _, m := range sm.SearchMachines {
		mss = append(mss, newMachineStateSource(m, members, cfg, connectMetricsServer))
	}

	// Get machine metrics
	smf := make(chan struct{}, *flagParallelSize)
	for i := 0; i < *flagParallelSize; i++ {
		smf <- struct{}{}
	}
	env := well.NewEnvironment(ctx)
	for _, m := range mss {
		if m.machineType == nil || len(m.machineType.MetricsCheckList) == 0 {
			continue
		}
		source := m
		env.Go(func(ctx context.Context) error {
			<-smf
			defer func() { smf <- struct{}{} }()
			addr := "http://" + source.ipv4 + ":9105/metrics"
			ch, err := m.fetcher(ctx, addr)
			if err != nil {
				return err
			}
			return source.readAndSetMetrics(ch)
		})
	}
	env.Stop()
	err = env.Wait()
	if err != nil {
		// do not exit
		log.Warn("error occurred when get metrics", map[string]interface{}{
			log.FnError: err.Error(),
		})
	}

	// For each machine sources, decide its next state, then update sabakan
	for _, ms := range mss {
		state := decideSabakanState(ms)
		err = gql.setSabakanState(ctx, ms, state)
		if err != nil {
			switch e := err.(type) {
			case *gqlerror.Error:
				// In the case of an invalid state transition, the log may continue to be output.
				// So the log is not output.
				if eType, ok := e.Extensions["type"]; ok && eType == gqlsabakan.ErrInvalidStateTransition {
					continue
				}
				log.Warn("gql error occurred when set state", map[string]interface{}{
					log.FnError: err.Error(),
					"serial":    ms.serial,
				})
			default:
				log.Warn("error occurred when set state", map[string]interface{}{
					log.FnError: err.Error(),
					"serial":    ms.serial,
				})
			}
		}
	}
	return nil
}

func newMachineStateSource(m machine, members []serf.Member, cfg *config, fetcher fetcherFromMetricsServer) machineStateSource {
	return machineStateSource{
		serial:      m.Spec.Serial,
		ipv4:        m.Spec.IPv4[0],
		serfStatus:  findMember(members, m.Spec.IPv4[0]),
		machineType: findMachineType(&m, cfg),
		metrics:     map[string]machineMetrics{},
		fetcher:     fetcher,
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

func findMachineType(m *machine, config *config) *machineType {
	mtLabel := findLabel(m.Spec.Labels, machineTypeLabelName)
	if mtLabel == nil {
		log.Warn(machineTypeLabelName+" is not set", map[string]interface{}{
			"serial": m.Spec.Serial,
		})
		return nil
	}
	for _, mt := range config.MachineTypes {
		if mt.Name == mtLabel.Value {
			return &mt
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
