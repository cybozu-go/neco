package sss

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/coreos/etcd/clientv3"
	"github.com/coreos/etcd/clientv3/concurrency"
	"github.com/cybozu-go/log"
	"github.com/cybozu-go/neco/storage"
	"github.com/cybozu-go/sabakan/v2"
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

	machineTypes map[string]*machineType

	unhealthyMachines map[string]time.Time

	etcdClient    *clientv3.Client
	sessionTTL    time.Duration
	electionValue string
}

// RegisterUnhealthy registers unhealthy machine and returns true
// if the machine has been unhealthy longer than the GracePeriod
// specified in its machine type.
func (c *Controller) RegisterUnhealthy(m *machine, now time.Time) bool {
	startTime, ok := c.unhealthyMachines[m.Serial]
	if !ok {
		c.unhealthyMachines[m.Serial] = now
		return false
	}
	machineType, ok := c.machineTypes[m.Type]
	if !ok {
		return true
	}
	return startTime.Add(machineType.GracePeriod.Duration).Before(now)
}

// ClearUnhealthy removes machine from unhealthy registry.
func (c *Controller) ClearUnhealthy(m *machine) {
	delete(c.unhealthyMachines, m.Serial)
}

// NewController returns controller for sabakan-state-setter
func NewController(etcdClient *clientv3.Client, sabakanAddress, serfAddress, configFile, electionValue string, interval time.Duration, parallelSize int, sessionTTL time.Duration) (*Controller, error) {
	machineTypes, err := readConfigFile(configFile)
	if err != nil {
		return nil, err
	}

	sabakanClient, err := newSabakanGQLClient(sabakanAddress)
	if err != nil {
		return nil, err
	}

	serfClient, err := newSerfClient(serfAddress)
	if err != nil {
		return nil, err
	}

	promClient := newPromClient()

	return &Controller{
		etcdClient:        etcdClient,
		electionValue:     electionValue,
		sessionTTL:        sessionTTL,
		interval:          interval,
		parallelSize:      parallelSize,
		sabakanClient:     sabakanClient,
		serfClient:        serfClient,
		promClient:        promClient,
		machineTypes:      machineTypes,
		unhealthyMachines: make(map[string]time.Time),
	}, nil
}

func (c *Controller) Run(ctx context.Context) error {
	session, err := concurrency.NewSession(c.etcdClient, concurrency.WithTTL(int(c.sessionTTL.Seconds())))
	if err != nil {
		return fmt.Errorf("failed to create new session: %s", err.Error())
	}
	defer func() {
		// Checking the session to avoid an error caused by duplicated closing.
		select {
		case <-session.Done():
			return
		default:
			session.Close()
		}
	}()

	election := concurrency.NewElection(session, storage.KeySabakanStateSetterLeader)

	// When the etcd is stopping, the Campaign will hang up.
	// So check the session and exit if the session is closed.
	doneCh := make(chan error)
	go func() {
		doneCh <- election.Campaign(ctx, c.electionValue)
	}()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-session.Done():
		return errors.New("failed to campaign: session is closed")
	case err := <-doneCh:
		if err != nil {
			return fmt.Errorf("failed to campaign: %s", err.Error())
		}
	}

	log.Info("I am the leader", map[string]interface{}{
		"session": session.Lease(),
	})
	leaderKey := election.Key()

	// Release the leader before terminating.
	defer func() {
		select {
		case <-session.Done():
			log.Warn("session is closed, skip resign", nil)
			return
		default:
			ctxWithTimeout, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			err := election.Resign(ctxWithTimeout)
			if err != nil {
				log.Error("failed to resign", map[string]interface{}{
					log.FnError: err,
				})
			}
		}
	}()

	env := well.NewEnvironment(ctx)
	env.Go(func(ctx context.Context) error {
		// runs state management periodically
		ticker := time.NewTicker(c.interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-ticker.C:
				if err := c.runOnce(ctx); err != nil {
					return err
				}
			}
		}
	})
	env.Go(func(ctx context.Context) error {
		err := watchLeaderKey(ctx, session, leaderKey)
		return fmt.Errorf("failed to watch leader key: %s", err.Error())
	})
	env.Stop()
	return env.Wait()
}

func watchLeaderKey(ctx context.Context, session *concurrency.Session, leaderKey string) error {
	ch := session.Client().Watch(ctx, leaderKey, clientv3.WithFilterPut())
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-session.Done():
			return errors.New("session is closed")
		case resp, ok := <-ch:
			if !ok {
				return errors.New("watch is closed")
			}
			if resp.Err() != nil {
				return resp.Err()
			}
			for _, ev := range resp.Events {
				if ev.Type == clientv3.EventTypeDelete {
					return errors.New("leader key is deleted")
				}
			}
		}
	}
}

func (c *Controller) runOnce(ctx context.Context) error {
	machines, err := c.sabakanClient.GetSabakanMachines(ctx)
	if err != nil {
		log.Warn("failed to get sabakan machines", map[string]interface{}{
			log.FnError: err.Error(),
		})
		// lint:ignore nilerr  Run tries this again.
		return nil
	}
	if len(machines) == 0 {
		log.Warn("no machines found", nil)
		return nil
	}

	// Check machine types
	undefinedMachineTypes := map[string][]string{}
	for _, m := range machines {
		if _, ok := c.machineTypes[m.Type]; !ok {
			undefinedMachineTypes[m.Type] = append(undefinedMachineTypes[m.Type], m.Serial)
		}
	}
	for undefinedType, invalidMachines := range undefinedMachineTypes {
		if undefinedType == "" {
			log.Warn("machine type is not specified", map[string]interface{}{
				"machines": strings.Join(invalidMachines, ","),
			})
		} else {
			log.Warn("specified machine type does not exist", map[string]interface{}{
				"machine-type": undefinedType,
				"machines":     strings.Join(invalidMachines, ","),
			})
		}
	}

	// Do machines health check
	newStateMap := c.machineHealthCheck(ctx, machines)

	// Do machines retire
	// T.B.D.

	now := time.Now()
	for _, m := range machines {
		newState, ok := newStateMap[m.Serial]
		if !ok || m.State == newState {
			continue
		}

		if newState == sabakan.StateUnhealthy {
			if ok := c.RegisterUnhealthy(m, now); !ok {
				continue
			}
		} else {
			c.ClearUnhealthy(m)
		}

		err := c.sabakanClient.UpdateSabakanState(ctx, m.Serial, newState)
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
					"serial":    m.Serial,
					"ipv4":      m.IPv4Addr,
				})
			default:
				log.Warn("error occurred when set state", map[string]interface{}{
					log.FnError: err.Error(),
					"serial":    m.Serial,
					"ipv4":      m.IPv4Addr,
				})
			}
		} else {
			log.Info("change state", map[string]interface{}{
				"serial": m.Serial,
				"ipv4":   m.IPv4Addr,
				"state":  newState,
			})
		}
	}

	return nil
}

func (c *Controller) machineHealthCheck(ctx context.Context, machines []*machine) map[string]sabakan.MachineState {
	// Get serf status
	serfStatus, err := c.serfClient.GetSerfStatus()
	if err != nil {
		log.Warn("failed to get serf members", map[string]interface{}{
			log.FnError: err.Error(),
		})
		return map[string]sabakan.MachineState{}
	}

	// Construct a slice of machineStateSource
	machineStateSources := make([]*machineStateSource, 0, len(machines))
	for _, m := range machines {
		switch m.State {
		case sabakan.StateUninitialized:
		case sabakan.StateHealthy:
		case sabakan.StateUnhealthy:
		case sabakan.StateUnreachable:
		default:
			// StateUpdating, StateRetiring or StateRetired machines will not do health check.
			log.Info("skip health check", map[string]interface{}{
				"serial": m.Serial,
				"ipv4":   m.IPv4Addr,
				"state":  m.State,
			})
			continue
		}
		machineStateSources = append(machineStateSources, newMachineStateSource(m, serfStatus, c.machineTypes))
	}

	// Get machine metrics
	sem := make(chan struct{}, c.parallelSize)
	for i := 0; i < c.parallelSize; i++ {
		sem <- struct{}{}
	}
	var wg sync.WaitGroup
	for _, mss := range machineStateSources {
		if mss.machineType == nil || len(mss.machineType.MetricsCheckList) == 0 {
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
					log.FnError: err.Error(),
					"serial":    source.serial,
					"ipv4":      source.ipv4,
				})
				return
			}
			source.metrics = mfs
		}(mss)
	}
	wg.Wait()

	// Decide next machine state
	newStateMap := map[string]sabakan.MachineState{}
	for _, mss := range machineStateSources {
		log.Info("do health check", map[string]interface{}{
			"serial": mss.serial,
			"ipv4":   mss.ipv4,
		})
		newState, hasTransition := mss.decideMachineStateCandidate()
		if !hasTransition {
			continue
		}
		newStateMap[mss.serial] = newState
	}
	return newStateMap
}
