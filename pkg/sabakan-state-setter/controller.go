package sss

import (
	"context"
	"os"
	"sync"
	"time"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/sabakan/v2"
	gqlsabakan "github.com/cybozu-go/sabakan/v2/gql"
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

	unhealthyMachines map[string]time.Time
}

// RegisterUnhealthy registers unhealthy machine and returns true
// if the machine has been unhealthy longer than the GracePeriod
// specified in its machine type.
func (c *Controller) RegisterUnhealthy(mss *machineStateSource, now time.Time) bool {
	startTime, ok := c.unhealthyMachines[mss.serial]
	if !ok {
		c.unhealthyMachines[mss.serial] = now
		return false
	}

	return startTime.Add(mss.machineType.GracePeriod.Duration).Before(now)
}

// ClearUnhealthy removes machine from unhealthy registry.
func (c *Controller) ClearUnhealthy(mss *machineStateSource) {
	delete(c.unhealthyMachines, mss.serial)
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
		interval:          i,
		parallelSize:      parallelSize,
		sabakanClient:     sabakanClient,
		serfClient:        serfClient,
		promClient:        promClient,
		machineTypes:      cfg.MachineTypes,
		unhealthyMachines: make(map[string]time.Time),
	}, nil
}

// RunPeriodically runs state management periodically
func (c *Controller) RunPeriodically(ctx context.Context) error {
	ticker := time.NewTicker(c.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		}
		if err := c.run(ctx); err != nil {
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
		// lint:ignore nilerr  RunPeriodically tries this again.
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
		// lint:ignore nilerr  RunPeriodically tries this again.
		return nil
	}

	// Construct a slice of machineStateSource
	machineStateSources := make([]*machineStateSource, len(sm.SearchMachines))
	for i, m := range sm.SearchMachines {
		machineStateSources[i] = newMachineStateSource(m, members, c.machineTypes)
	}

	// Get machine metrics
	sem := make(chan struct{}, c.parallelSize)
	for i := 0; i < c.parallelSize; i++ {
		sem <- struct{}{}
	}
	var wg sync.WaitGroup
	for _, m := range machineStateSources {
		if m.machineType == nil || len(m.machineType.MetricsCheckList) == 0 {
			continue
		}
		wg.Add(1)
		go func(source *machineStateSource) {
			<-sem
			defer func() {
				sem <- struct{}{}
				wg.Done()
			}()

			addr := "http://" + source.ipv4 + ":9105/metrics"
			mfs, err := c.promClient.ConnectMetricsServer(ctx, addr)
			if err != nil {
				log.Warn("failed to get metrics", map[string]interface{}{
					"addr":      addr,
					log.FnError: err,
				})
				return
			}
			source.metrics = mfs
		}(m)
	}
	wg.Wait()

	now := time.Now()

	for _, mss := range machineStateSources {
		newState, hasTransition := mss.decideMachineStateCandidate()
		if !hasTransition {
			continue
		}

		if newState == sabakan.StateUnhealthy {
			if ok := c.RegisterUnhealthy(mss, now); !ok {
				continue
			}
		} else {
			c.ClearUnhealthy(mss)
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
					"ipv4":      mss.ipv4,
				})
			default:
				log.Warn("error occurred when set state", map[string]interface{}{
					log.FnError: err.Error(),
					"serial":    mss.serial,
					"ipv4":      mss.ipv4,
				})
			}
		} else {
			log.Info("change state", map[string]interface{}{
				"serial": mss.serial,
				"ipv4":   mss.ipv4,
				"state":  newState,
			})
		}
	}

	return nil
}
