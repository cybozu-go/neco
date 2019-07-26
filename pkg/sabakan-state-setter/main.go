package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net"
	"os"
	"time"

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
	interval, err := time.ParseDuration(*flagInterval)
	if err != nil {
		log.ErrorExit(err)
	}

	well.Go(func(ctx context.Context) error {
		return runPeriodically(ctx, interval)
	})
	well.Stop()
	err = well.Wait()
	if err != nil && !well.IsSignaled(err) {
		fmt.Println(err)
		os.Exit(1)
	}
}

func runPeriodically(ctx context.Context, interval time.Duration) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		}
		err := run(ctx)
		if err != nil {
			return err
		}
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

	var pms []problematicMachine

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
		ms, err := newMachineStateSource(m, members, cfg, pms)
		if err != nil {
			return err
		}
		mss = append(mss, *ms)
	}

	// Get machine metrics
	sem := make(chan struct{}, *flagParallelSize)
	for i := 0; i < *flagParallelSize; i++ {
		sem <- struct{}{}
	}
	env := well.NewEnvironment(ctx)
	for _, m := range mss {
		if m.machineType == nil || len(m.machineType.MetricsCheckList) == 0 {
			continue
		}
		source := m
		env.Go(func(ctx context.Context) error {
			<-sem
			defer func() { sem <- struct{}{} }()
			addr := "http://" + source.ipv4 + ":9105/metrics"
			ch, err := connectMetricsServer(ctx, addr)
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

	var newProblematicMachineStates []problematicMachine
	for _, ms := range mss {
		newState := ms.decideMachineStateCandidate()

		if isProblematicState(newState) {
			newPM := problematicMachine{Serial: ms.serial, State: newState}
			if ms.stateCandidate == newState {
				newPM.FirstDetection = ms.stateCandidateFirstDetection.UTC().String()
			} else {
				newPM.FirstDetection = time.Now().String()
			}
			newProblematicMachineStates = append(newProblematicMachineStates, newPM)
		}

		if !ms.needUpdateState(newState, time.Now()) {
			continue
		}

		err = gql.updateSabakanState(ctx, ms, newState)
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

func newMachineStateSource(m machine, members []serf.Member, cfg *config, pms []problematicMachine) (*machineStateSource, error) {
	mss := machineStateSource{
		serial:      m.Spec.Serial,
		ipv4:        m.Spec.IPv4[0],
		serfStatus:  findMember(members, m.Spec.IPv4[0]),
		machineType: findMachineType(&m, cfg),
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
