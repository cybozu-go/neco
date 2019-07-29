package sss

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

// Controller is sabakan-state-setter controller
type Controller struct {
	interval            time.Duration
	parallelSize        int
	sabakanClient       SabakanGQLClient
	prom                PrometheusClient
	machineTypes        []*machineType
	machineStateSources []*MachineStateSource
}

// NewController returns controller for sabakan-state-setter
func NewController(ctx context.Context, sabakanAddress, configFile, interval string, parallelSize int) (*Controller, error) {
	i, err := time.ParseDuration(interval)
	if err != nil {
		return nil, err
	}
	cf, err := os.Open(configFile)
	if err != nil {
		return nil, err
	}
	defer cf.Close()
	cfg, err := parseConfig(cf)
	if err != nil {
		return nil, err
	}
	gqlc, err := newGQLClient(sabakanAddress)
	if err != nil {
		return nil, err
	}
	sm, err := gqlc.GetSabakanMachines(ctx)
	if err != nil {
		return nil, err
	}
	if len(sm.SearchMachines) == 0 {
		return nil, errors.New("no machines found")
	}

	// Get serf members
	serfc, err := serf.NewRPCClient("127.0.0.1:7373")
	if err != nil {
		return nil, err
	}
	members, err := getSerfMembers(serfc)
	if err != nil {
		return nil, err
	}

	// Construct a slice of machineStateSource
	mssSlice := make([]*MachineStateSource, len(sm.SearchMachines))
	for i, m := range sm.SearchMachines {
		mssSlice[i] = newMachineStateSource(m, members, cfg.MachineTypes)
	}

	return &Controller{
		interval:            i,
		parallelSize:        parallelSize,
		sabakanClient:       gqlc,
		prom:                newPromClient(),
		machineTypes:        cfg.MachineTypes,
		machineStateSources: mssSlice,
	}, nil
}

// RunPeriodically runs state management periodically
func (c *Controller) RunPeriodically(ctx context.Context) error {
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

func (c *Controller) run(ctx context.Context) error {
	// Get machine metrics
	sem := make(chan struct{}, c.parallelSize)
	for i := 0; i < c.parallelSize; i++ {
		sem <- struct{}{}
	}
	env := well.NewEnvironment(ctx)
	for _, m := range c.machineStateSources {
		if m.machineType == nil || len(m.machineType.MetricsCheckList) == 0 {
			continue
		}
		source := m
		env.Go(func(ctx context.Context) error {
			<-sem
			defer func() { sem <- struct{}{} }()
			addr := "http://" + source.ipv4 + ":9105/metrics"
			ch, err := c.prom.ConnectMetricsServer(ctx, addr)
			if err != nil {
				return err
			}
			return source.readAndSetMetrics(ch)
		})
	}
	env.Stop()
	err := env.Wait()
	if err != nil {
		// do not exit
		log.Warn("error occurred when get metrics", map[string]interface{}{
			log.FnError: err.Error(),
		})
	}

	for _, mss := range c.machineStateSources {
		newState := mss.decideMachineStateCandidate()

		if !mss.needUpdateState(newState, time.Now()) {
			continue
		}

		if mss.stateCandidate != newState {
			mss.stateCandidate = newState
			mss.stateCandidateFirstDetection = time.Now()
		}

		err := c.sabakanClient.UpdateSabakanState(ctx, mss, newState)
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
					"serial":    mss.serial,
				})
			default:
				log.Warn("error occurred when set state", map[string]interface{}{
					log.FnError: err.Error(),
					"serial":    mss.serial,
				})
			}
		}
	}

	return nil
}
