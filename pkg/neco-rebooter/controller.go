package necorebooter

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math/rand"
	"time"

	"github.com/cybozu-go/cke"
	"github.com/cybozu-go/neco"
	"github.com/cybozu-go/neco/storage"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/client/v3/concurrency"
)

type Controller struct {
	ckeStorage    cke.Storage
	etcdClient    clientv3.Client
	necoStorage   storage.Storage
	config        Config
	rebootTimes   map[string]RebootTime
	sessionTTL    time.Duration
	electionValue string
	interval      time.Duration
	leaderKey     string
	timeZone      *time.Location
}

type RebootArgs struct {
	rebootListEntries  []*neco.RebootListEntry
	rebootQueueEntries []*cke.RebootQueueEntry
	processingGroup    string
}

func NewController(config *Config, rt *map[string]RebootTime, ckeStorage *cke.Storage, etcdClient *clientv3.Client, necoStorage *storage.Storage, electionValue string) (*Controller, error) {
	tz, err := time.LoadLocation(config.TimeZone)
	if err != nil {
		return nil, err
	}
	return &Controller{
		config:        *config,
		rebootTimes:   *rt,
		etcdClient:    *etcdClient,
		ckeStorage:    *ckeStorage,
		necoStorage:   *necoStorage,
		sessionTTL:    1 * time.Minute,
		electionValue: electionValue,
		interval:      1 * time.Minute,
		timeZone:      tz,
	}, nil

}

func (c *Controller) removeCancelledEntry(ctx context.Context, rebootArgs RebootArgs) error {
	for _, rebootListEntry := range rebootArgs.rebootListEntries {
		if rebootListEntry.Status != neco.RebootListEntryStatusCancelled {
			continue
		}
		rebootQueueEntry := findRebootQueueEntryFromRebootListEntry(rebootArgs.rebootQueueEntries, *rebootListEntry)
		if rebootQueueEntry != nil && rebootQueueEntry.Status != cke.RebootStatusCancelled {
			rebootQueueEntry.Status = cke.RebootStatusCancelled
			err := c.ckeStorage.UpdateRebootsEntry(ctx, rebootQueueEntry)
			if err != nil {
				return err
			}
		}
		err := c.necoStorage.RemoveRebootListEntry(ctx, c.leaderKey, rebootListEntry)
		if err != nil {
			return err
		}
		slog.With(slog.String("operation", "removeCancelledEntry")).Info("removed cancelled entry", slog.String("node", rebootListEntry.Node))
	}
	return nil
}

func (c *Controller) removeCompletedEntry(ctx context.Context, rebootArgs RebootArgs) error {
	for _, rebootListEntry := range rebootArgs.rebootListEntries {
		if findRebootQueueEntryFromRebootListEntry(rebootArgs.rebootQueueEntries, *rebootListEntry) == nil && rebootListEntry.Status == neco.RebootListEntryStatusQueued {
			err := c.necoStorage.RemoveRebootListEntry(ctx, c.leaderKey, rebootListEntry)
			if err != nil {
				return err
			}
			slog.With(slog.String("operation", "removeCompletedEntry")).Info("removed completed entry", slog.String("node", rebootListEntry.Node))
		}
	}
	return nil
}

func (c *Controller) dequeueTimedOutEntry(ctx context.Context, rebootArgs RebootArgs) error {
	for _, rebootListEntry := range rebootArgs.rebootListEntries {
		if rebootListEntry.Status != neco.RebootListEntryStatusQueued {
			continue
		}
		rebootQueueEntry := findRebootQueueEntryFromRebootListEntry(rebootArgs.rebootQueueEntries, *rebootListEntry)
		// if the rebootListEntry is Queued and the rebootQueueEntry is not found, it means the entry is completed to reboot, so skipping the cancel process.
		if !c.isRebootable(rebootListEntry) && rebootQueueEntry != nil {
			rebootListEntry.Status = neco.RebootListEntryStatusPending
			err := c.necoStorage.UpdateRebootListEntry(ctx, rebootListEntry)
			if err != nil {
				return err
			}
			rebootQueueEntry.Status = cke.RebootStatusCancelled
			err = c.ckeStorage.UpdateRebootsEntry(ctx, rebootQueueEntry)
			if err != nil {
				return err
			}
			slog.With(slog.String("operation", "dequeueTimedOutEntry")).Info("node is out of rebootable time. rebootListEntry dequeued and rebootQueueEntry cancelled", slog.String("DequeuedNode", rebootListEntry.Node))
		}
	}
	return nil
}

func (c *Controller) dequeueAndCancelEntry(ctx context.Context, rebootArgs RebootArgs) error {
	for _, rebootListEntry := range rebootArgs.rebootListEntries {
		if rebootListEntry.Status != neco.RebootListEntryStatusQueued {
			continue
		}
		rebootQueueEntry := findRebootQueueEntryFromRebootListEntry(rebootArgs.rebootQueueEntries, *rebootListEntry)
		// if the rebootListEntry is Queued and the rebootQueueEntry is not found, it means the entry is completed to reboot, so skipping the cancel process.
		if rebootQueueEntry == nil {
			continue
		}
		rebootListEntry.Status = neco.RebootListEntryStatusPending
		err := c.necoStorage.UpdateRebootListEntry(ctx, rebootListEntry)
		if err != nil {
			return err
		}
		rebootQueueEntry.Status = cke.RebootStatusCancelled
		err = c.ckeStorage.UpdateRebootsEntry(ctx, rebootQueueEntry)
		if err != nil {
			return err
		}
		slog.With(slog.String("operation", "dequeueAndCancelEntry")).Info("rebootListEntry dequeued and rebootQueueEntry cancelled", slog.String("node", rebootListEntry.Node))
	}
	return nil
}

func (c *Controller) addRebootListEntry(ctx context.Context, rebootArgs RebootArgs) error {
	for _, rebootListEntry := range rebootArgs.rebootListEntries {
		if rebootListEntry.Status == neco.RebootListEntryStatusQueued || rebootListEntry.Group != rebootArgs.processingGroup {
			continue
		}
		if c.isRebootable(rebootListEntry) {
			rebootListEntry.Status = neco.RebootListEntryStatusQueued
			// when the node is already in the rebootQueueEntries, we should not add the node to the rebootQueueEntries and only update the status of the rebootListEntry.
			if findRebootQueueEntryFromRebootListEntry(rebootArgs.rebootQueueEntries, *rebootListEntry) != nil {
				err := c.necoStorage.UpdateRebootListEntry(ctx, rebootListEntry)
				if err != nil {
					return err
				}
				continue
			}
			err := c.ckeStorage.RegisterRebootsEntry(ctx, cke.NewRebootQueueEntry(rebootListEntry.Node))
			if err != nil {
				return err
			}
			err = c.necoStorage.UpdateRebootListEntry(ctx, rebootListEntry)
			if err != nil {
				return err
			}
			slog.With(slog.String("operation", "addRebootListEntry")).Info("AddRebootListEntry", slog.String("node", rebootListEntry.Node), slog.String("group", rebootListEntry.Group))
		}
	}
	return nil
}

func (c *Controller) moveNextGroup(ctx context.Context, rebootArgs RebootArgs) (string, error) {
	groups := getAllGroups(rebootArgs.rebootListEntries)
	if len(groups) < 1 {
		rand.Shuffle(len(groups), func(i, j int) { groups[i], groups[j] = groups[j], groups[i] })
	}
	if len(rebootArgs.rebootQueueEntries) == 0 || rebootArgs.processingGroup == "" {
		for _, group := range groups {
			if group == rebootArgs.processingGroup {
				continue
			}
			e := c.findRebootableNodeInGroup(rebootArgs.rebootListEntries, group)
			if len(e) > 0 {
				err := c.necoStorage.UpdateProcessingGroup(ctx, group)
				if err != nil {
					return "", err
				}
				slog.With(slog.String("operation", "moveNextGroup")).Info("moved to next group", slog.String("group", group))
				return group, nil
			}
		}
		slog.With(slog.String("operation", "moveNextGroup")).Info("no candidate group found", slog.String("currentGroup", rebootArgs.processingGroup))
		return rebootArgs.processingGroup, nil
	}
	slog.With(slog.String("operation", "moveNextGroup")).Info("current group is work in progress", slog.String("currentGroup", rebootArgs.processingGroup))
	return rebootArgs.processingGroup, nil
}

func (c *Controller) isRebootable(entry *neco.RebootListEntry) bool {
	rebootTime, ok := c.rebootTimes[entry.RebootTime]
	if !ok {
		slog.With(slog.String("operation", "isRebootable")).Error("reboot time not found", slog.String("rebootTime", entry.RebootTime), slog.String("node", entry.Node))
		return false
	}
	for _, deny := range rebootTime.Deny {
		if isWithinSchedule(deny, c.timeZone) {
			return false
		}
	}
	for _, allow := range rebootTime.Allow {
		if isWithinSchedule(allow, c.timeZone) {
			return true
		}
	}
	return false
}

func (c *Controller) findRebootableNodeInGroup(rebootListEntries []*neco.RebootListEntry, group string) []neco.RebootListEntry {
	nodes := make([]neco.RebootListEntry, 0)
	for _, entry := range rebootListEntries {
		if entry.Group == group && c.isRebootable(entry) {
			nodes = append(nodes, *entry)
		}
	}
	return nodes
}

func (c *Controller) runOnce(ctx context.Context) error {
	enabled, err := c.necoStorage.IsNecoRebooterEnabled(ctx)
	if err != nil {
		return err
	}
	rebootListEntries, err := c.necoStorage.GetRebootListEntries(ctx)
	if err != nil {
		return err
	}
	rebootQueueEntries, err := c.ckeStorage.GetRebootsEntries(ctx)
	if err != nil {
		return err
	}
	processingGroup, err := c.necoStorage.GetProcessingGroup(ctx)
	if err != nil {
		return err
	}

	rebootArgs := RebootArgs{
		rebootListEntries:  rebootListEntries,
		rebootQueueEntries: rebootQueueEntries,
		processingGroup:    processingGroup,
	}

	var cancelledRebootListEntries int
	for _, entry := range rebootListEntries {
		if entry.Status == neco.RebootListEntryStatusCancelled {
			cancelledRebootListEntries++
		}
	}
	if cancelledRebootListEntries > 0 {
		err := c.removeCancelledEntry(ctx, rebootArgs)
		if err != nil {
			return err
		}
		return nil
	}

	if enabled {
		nextGroup, err := c.moveNextGroup(ctx, rebootArgs)
		if err != nil {
			return err
		}
		if nextGroup != processingGroup {
			rebootArgs.processingGroup = nextGroup
		}
		err = c.removeCompletedEntry(ctx, rebootArgs)
		if err != nil {
			return err
		}
		err = c.dequeueTimedOutEntry(ctx, rebootArgs)
		if err != nil {
			return err
		}
		err = c.addRebootListEntry(ctx, rebootArgs)
		if err != nil {
			return err
		}
	} else {
		err = c.removeCompletedEntry(ctx, rebootArgs)
		if err != nil {
			return err
		}
		err = c.dequeueAndCancelEntry(ctx, rebootArgs)
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *Controller) Run(ctx context.Context) error {
	session, err := concurrency.NewSession(&c.etcdClient, concurrency.WithTTL(int(c.sessionTTL.Seconds())))
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

	election := concurrency.NewElection(session, storage.KeyNecoRebooterLeader)

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

	slog.Info("I am the leader", slog.Int64("session", int64(session.Lease())))
	leaderKey := election.Key()
	c.leaderKey = leaderKey

	// Release the leader before terminating.
	defer func() {
		select {
		case <-session.Done():
			slog.Warn("session is closed, skip resign")
			return
		default:
			ctxWithTimeout, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			err := election.Resign(ctxWithTimeout)
			if err != nil {
				slog.Error("failed to resign", "err", err)
			}
		}
	}()

	ctx, cancel := context.WithCancelCause(ctx)
	go func(ctx context.Context) {
		ticker := time.NewTicker(c.interval)
		defer func() {
			slog.Warn("ticker stopped")
			ticker.Stop()
		}()

		for {
			select {
			case <-ctx.Done():
				ctx.Err()
				return
			case <-ticker.C:
				if err := c.runOnce(ctx); err != nil {
					slog.Error("An error occurred in runOnce", "err", err)
					cancel(err)
					return
				}
			}
		}
	}(ctx)

	go func(ctx context.Context) {
		defer func() {
			slog.Warn("watcher stopped")
		}()
		err := watchLeaderKey(ctx, session, leaderKey)
		if err != nil {
			cancel(fmt.Errorf("failed to watch leader key: %s", err.Error()))
			return
		}
	}(ctx)

	<-ctx.Done()
	switch ctx.Err() {
	case context.DeadlineExceeded:
		return context.Cause(ctx)
	case context.Canceled:
		return context.Cause(ctx)
	}

	return nil
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
