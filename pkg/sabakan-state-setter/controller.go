package sss

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/neco/storage"
	"github.com/cybozu-go/sabakan/v3"
	gqlsabakan "github.com/cybozu-go/sabakan/v3/gql"
	"github.com/cybozu-go/well"
	"github.com/robfig/cron/v3"
	"github.com/vektah/gqlparser/v2/gqlerror"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/client/v3/concurrency"
)

// Controller is sabakan-state-setter controller
type Controller struct {
	// Etcd and leader election
	etcdClient    *clientv3.Client
	electionValue string
	sessionTTL    time.Duration

	// Clients
	necoExecutor       NecoCmdExecutor
	promClient         PrometheusClient
	sabakanClient      SabakanClientWrapper
	serfClient         SerfClient
	alertmanagerClient *alertmanagerClient

	// others
	interval          time.Duration
	parallelSize      int
	shutdownSchedule  string
	machineTypes      map[string]*machineType
	unhealthyMachines map[string]time.Time
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
func NewController(etcdClient *clientv3.Client, sabakanAddress, sabakanAddressHTTPS, serfAddress, configFile, electionValue string, interval time.Duration, parallelSize int, sessionTTL time.Duration) (*Controller, error) {
	shutdownSchedule, machineTypes, alertMonitor, err := readConfigFile(configFile)
	if err != nil {
		return nil, err
	}

	sabakanClient, err := newSabakanClientWrapper(sabakanAddress, sabakanAddressHTTPS)
	if err != nil {
		return nil, err
	}

	serfClient, err := newSerfClient(serfAddress)
	if err != nil {
		return nil, err
	}

	alertmanagerClient, err := newAlertmanagerClient(alertMonitor)
	if err != nil {
		return nil, err
	}

	promClient := newPromClient()
	necoExecutor := newNecoCmdExecutor()

	return &Controller{
		etcdClient:    etcdClient,
		electionValue: electionValue,
		sessionTTL:    sessionTTL,

		necoExecutor:       necoExecutor,
		promClient:         promClient,
		sabakanClient:      sabakanClient,
		serfClient:         serfClient,
		alertmanagerClient: alertmanagerClient,

		interval:          interval,
		parallelSize:      parallelSize,
		shutdownSchedule:  shutdownSchedule,
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

	if c.shutdownSchedule == "" {
		log.Info("skip to start shutdown cron job", nil)
	} else {
		sched, err := cron.ParseStandard(c.shutdownSchedule)
		if err != nil {
			return fmt.Errorf("failed to start shutdown cron job: %s", err.Error())
		}
		shutdownCron := cron.New()
		shutdownCron.Schedule(sched, cron.FuncJob(func() { c.machineShutdown(ctx) }))
		shutdownCron.Start()
		log.Info("start shutdown cron job", nil)
	}

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
	machines, err := c.sabakanClient.GetAllMachines(ctx)
	if err != nil {
		log.Warn("failed to get sabakan machines", map[string]interface{}{
			log.FnError: err.Error(),
		})
		return nil
	}
	if len(machines) == 0 {
		log.Warn("no machines found", nil)
		return nil
	}

	// Check machine types
	for _, m := range machines {
		if _, ok := c.machineTypes[m.Type]; !ok {
			log.Warn("unknown machine type", map[string]interface{}{
				"serial":       m.Serial,
				"machine_type": m.Type,
			})
		}
	}

	// Get serf status
	serfStatus, err := c.serfClient.GetSerfStatus()
	if err != nil {
		log.Warn("failed to get serf members", map[string]interface{}{
			log.FnError: err.Error(),
		})
		// If the serf service is restarted, the connection of the serf client is closed.
		// In this case, GetSerfStatus() will never succeed unless recreating the client.
		// So return an error to exit this process.
		return err
	}

	// Get alert statuses
	alertStatuses, err := c.alertmanagerClient.GetAlertStatuses(machines)
	if err != nil {
		log.Warn("failed to get alert statuses", map[string]interface{}{
			log.FnError: err.Error(),
		})
		// It is not a critical error that Alertmanager is down.
		alertStatuses = nil
	}

	newStateMap := map[string]sabakan.MachineState{}

	// Do machines health check
	healthcheckResult := c.machineHealthCheck(ctx, machines, serfStatus, alertStatuses)
	for serial, state := range healthcheckResult {
		newStateMap[serial] = state
	}

	// Do machines retirement
	retireResult := c.machineRetire(ctx, machines)
	for serial, state := range retireResult {
		newStateMap[serial] = state
	}

	now := time.Now()
	for _, m := range machines {
		newState, ok := newStateMap[m.Serial]
		switch {
		case !ok || newState == m.State || (newState == stateUnhealthyImmediate && m.State == sabakan.StateUnhealthy):
			c.ClearUnhealthy(m)
			continue
		case newState == stateUnhealthyImmediate:
			c.ClearUnhealthy(m)
			newState = sabakan.StateUnhealthy
		case newState == sabakan.StateUnhealthy:
			// Wait for the GracePeriod before changing the machine state to unhealthy.
			if ok := c.RegisterUnhealthy(m, now); !ok {
				continue
			}
		default:
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

func (c *Controller) machineHealthCheck(ctx context.Context, machines []*machine, serfStatus map[string]*serfStatus, alertStatuses map[string]*alertStatus) map[string]sabakan.MachineState {
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
		machineStateSources = append(machineStateSources, newMachineStateSource(m, serfStatus, alertStatuses, c.machineTypes))
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

			mfs, err := c.promClient.ConnectMetricsServer(ctx, source.ipv4)
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
		newState := mss.decideMachineStateCandidate()
		if newState == doNotChangeState {
			continue
		}
		newStateMap[mss.serial] = newState
	}
	return newStateMap
}

func (c *Controller) machineRetire(ctx context.Context, machines []*machine) map[string]sabakan.MachineState {
	newStateMap := map[string]sabakan.MachineState{}

	for _, m := range machines {
		switch m.State {
		case sabakan.StateRetiring:
		default:
			// Skip any state expect for StateRetiring.
			continue
		}

		err := c.sabakanClient.CryptsDelete(ctx, m.Serial)
		if err != nil {
			log.Warn("failed to delete crypts on sabakan", map[string]interface{}{
				log.FnError: err.Error(),
				"serial":    m.Serial,
				"ipv4":      m.IPv4Addr,
			})
			continue
		}

		cmdOutput, err := c.necoExecutor.TPMClear(ctx, m.Serial)
		if err != nil {
			log.Warn("failed to clear TPM", map[string]interface{}{
				log.FnError: err.Error(),
				"serial":    m.Serial,
				"ipv4":      m.IPv4Addr,
				"cmdlog":    string(cmdOutput),
			})
			continue
		}

		log.Info("retirement; deleting encryption keys has been executed successfully", map[string]interface{}{
			"serial": m.Serial,
			"ipv4":   m.IPv4Addr,
			"cmdlog": string(cmdOutput),
		})
		newStateMap[m.Serial] = sabakan.StateRetired
	}

	return newStateMap
}

func (c *Controller) machineShutdown(ctx context.Context) {
	machines, err := c.sabakanClient.GetRetiredMachines(ctx)
	if err != nil {
		log.Warn("shutdown; failed to get retired machines", map[string]interface{}{
			log.FnError: err.Error(),
		})
		return
	}
	if len(machines) == 0 {
		log.Info("shutdown; no retired machines found", nil)
		return
	}

	var errorMachines []string
	for _, m := range machines {
		cmdOutput, err := c.necoExecutor.PowerStatus(ctx, m.Serial)
		if err != nil {
			log.Warn("shutdown; failed to get power status", map[string]interface{}{
				log.FnError: err.Error(),
				"serial":    m.Serial,
				"ipv4":      m.IPv4Addr,
				"cmdlog":    string(cmdOutput),
			})
			errorMachines = append(errorMachines, m.Serial)
			continue
		}

		// When `neco power status` succeeds, only power status (e.g. "On", "Off") is output.
		powerStatus := strings.TrimSpace(string(cmdOutput))
		if powerStatus == "Off" {
			log.Info("shutdown; already powered OFF", map[string]interface{}{
				"serial": m.Serial,
				"ipv4":   m.IPv4Addr,
			})
			continue
		}

		cmdOutput, err = c.necoExecutor.PowerStop(ctx, m.Serial)
		if err != nil {
			log.Warn("shutdown; failed to shutdown", map[string]interface{}{
				log.FnError: err.Error(),
				"serial":    m.Serial,
				"ipv4":      m.IPv4Addr,
				"cmdlog":    string(cmdOutput),
			})
			errorMachines = append(errorMachines, m.Serial)
			continue
		}

		log.Info("shutdown has been executed successfully", map[string]interface{}{
			"serial": m.Serial,
			"ipv4":   m.IPv4Addr,
			"cmdlog": string(cmdOutput),
		})
	}
	if len(errorMachines) != 0 {
		log.Warn("shutdown; failed to shutdown some machines", map[string]interface{}{
			"serials": strings.Join(errorMachines, ","),
		})
	}
}
