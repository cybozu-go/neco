package sss

import (
	"context"
	"os"
	"time"

	"github.com/cybozu-go/log"
	gqlsabakan "github.com/cybozu-go/sabakan/v2/gql"
	"github.com/cybozu-go/well"
	"github.com/vektah/gqlparser/gqlerror"
)

// Controller is sabakan-state-setter controller
type Controller struct {
	interval     time.Duration
	parallelSize int

	// Clients
	sabakanClient SabakanGQLClient
	promClient    PrometheusClient
	serfClient    SerfClient

	machineTypes []*machineType
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

	sabakanClient, err := newSabakanGQLClient(sabakanAddress)
	if err != nil {
		return nil, err
	}

	serfClient, err := newSerfClient("127.0.0.1:7373")
	if err != nil {
		return nil, err
	}

	promClient := newPromClient()

	return &Controller{
		interval:      i,
		parallelSize:  parallelSize,
		sabakanClient: sabakanClient,
		serfClient:    serfClient,
		promClient:    promClient,
		machineTypes:  cfg.MachineTypes,
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
	sm, err := c.sabakanClient.GetSabakanMachines(ctx)
	if err != nil {
		log.Warn("failed to get sabakan machines", map[string]interface{}{
			log.FnError: err.Error(),
		})
		return nil
	}

	if sm == nil || len(sm.SearchMachines) == 0 {
		log.Warn("no machines found", nil)
		return nil
	}

	members, err := c.serfClient.GetSerfMembers()
	if err != nil {
		log.Warn("failed to get serf members", map[string]interface{}{
			log.FnError: err.Error(),
		})
		return nil
	}

	// Construct a slice of machineStateSource
	machineStateSources := make([]*MachineStateSource, len(sm.SearchMachines))
	for i, m := range sm.SearchMachines {
		machineStateSources[i] = newMachineStateSource(m, members, c.machineTypes)
	}

	// Get machine metrics
	sem := make(chan struct{}, c.parallelSize)
	for i := 0; i < c.parallelSize; i++ {
		sem <- struct{}{}
	}
	env := well.NewEnvironment(ctx)
	for _, m := range machineStateSources {
		if m.machineType == nil || len(m.machineType.MetricsCheckList) == 0 {
			continue
		}
		source := m
		env.Go(func(ctx context.Context) error {
			<-sem
			defer func() { sem <- struct{}{} }()
			addr := "http://" + source.ipv4 + ":9105/metrics"
			ch, err := c.promClient.ConnectMetricsServer(ctx, addr)
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

	for _, mss := range machineStateSources {
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
