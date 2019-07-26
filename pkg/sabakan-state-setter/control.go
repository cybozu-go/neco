package main

import (
	"context"
	"errors"
	"os"
	"time"

	"github.com/cybozu-go/log"
	gqlsabakan "github.com/cybozu-go/sabakan/v2/gql"
	"github.com/cybozu-go/well"
	serf "github.com/hashicorp/serf/client"
	"github.com/vektah/gqlparser/gqlerror"
)

type controller struct {
	interval            time.Duration
	sabakanClient       *gqlClient
	machineTypes        []*machineType
	machineStateSources []*machineStateSource
}

func newController(intervalStr string, sabakanAddress string, configFile string) (*controller, error) {
	interval, err := time.ParseDuration(intervalStr)
	if err != nil {
		return nil, err
	}
	confFile, err := os.Open(configFile)
	if err != nil {
		return nil, err
	}
	defer confFile.Close()
	cfg, err := parseConfig(confFile)
	if err != nil {
		return nil, err
	}
	gql, err := newGQLClient(sabakanAddress)
	if err != nil {
		return nil, err
	}
	return &controller{
		interval:            interval,
		sabakanClient:       gql,
		machineTypes:        cfg.MachineTypes,
		machineStateSources: []*machineStateSource{}}, nil
}

func (c *controller) runPeriodically(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	ticker := time.NewTicker(c.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		}
		err := c.run(ctx)
		if err != nil {
			return err
		}
	}
}

func (c *controller) run(ctx context.Context) error {

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
		ms, err := newMachineStateSource(m, members, c.machineTypes, pms)
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
