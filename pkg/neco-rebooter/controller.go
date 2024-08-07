package necorebooter

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
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

type EntrySet struct {
	rebootListEntry  *neco.RebootListEntry
	rebootQueueEntry *cke.RebootQueueEntry
}

type EntriesCollection struct {
	CompletedEntry []*neco.RebootListEntry
	TimedOutEntry  []EntrySet
	NewEntry       []*neco.RebootListEntry
	CancelledEntry []EntrySet
	QueuedEntry    []EntrySet
}

var (
	allGroups []string
)

func NewController(config *Config, rt map[string]RebootTime, ckeStorage *cke.Storage, etcdClient *clientv3.Client, necoStorage *storage.Storage, electionValue string) (*Controller, error) {
	tz, err := time.LoadLocation(config.TimeZone)
	if err != nil {
		return nil, err
	}
	return &Controller{
		config:        *config,
		rebootTimes:   rt,
		etcdClient:    *etcdClient,
		ckeStorage:    *ckeStorage,
		necoStorage:   *necoStorage,
		sessionTTL:    1 * time.Minute,
		electionValue: electionValue,
		interval:      1 * time.Minute,
		timeZone:      tz,
	}, nil

}

func (c *Controller) removeCancelledEntry(ctx context.Context, entries []EntrySet) error {
	for _, entry := range entries {
		if entry.rebootQueueEntry != nil {
			entry.rebootQueueEntry.Status = cke.RebootStatusCancelled
			err := c.ckeStorage.UpdateRebootsEntry(ctx, entry.rebootQueueEntry)
			if err != nil {
				return err
			}
		}
		err := c.necoStorage.RemoveRebootListEntry(ctx, c.leaderKey, entry.rebootListEntry)
		if err != nil {
			return err
		}
		slog.With(slog.String("operation", "removeCancelledEntry")).Info("removed cancelled entry", slog.String("node", entry.rebootListEntry.Node))
	}
	return nil
}

func (c *Controller) removeCompletedEntry(ctx context.Context, entries []*neco.RebootListEntry) error {
	for _, entry := range entries {
		err := c.necoStorage.RemoveRebootListEntry(ctx, c.leaderKey, entry)
		if err != nil {
			return err
		}
		slog.With(slog.String("operation", "removeCompletedEntry")).Info("removed completed entry", slog.String("node", entry.Node))
	}
	return nil
}

func (c *Controller) dequeueAndCancelEntry(ctx context.Context, entries []EntrySet) error {
	for _, entry := range entries {
		entry.rebootListEntry.Status = neco.RebootListEntryStatusPending
		err := c.necoStorage.UpdateRebootListEntry(ctx, entry.rebootListEntry)
		if err != nil {
			return err
		}
		if entry.rebootQueueEntry != nil {
			entry.rebootQueueEntry.Status = cke.RebootStatusCancelled
			err = c.ckeStorage.UpdateRebootsEntry(ctx, entry.rebootQueueEntry)
			if err != nil {
				return err
			}
			slog.With(slog.String("operation", "dequeueAndCancelEntry")).Info("rebootListEntry dequeued and rebootQueueEntry cancelled", slog.String("node", entry.rebootListEntry.Node))
		}
	}
	return nil
}

func (c *Controller) addRebootListEntry(ctx context.Context, entries []*neco.RebootListEntry) error {
	for _, entry := range entries {
		entry.Status = neco.RebootListEntryStatusQueued
		err := c.ckeStorage.RegisterRebootsEntry(ctx, cke.NewRebootQueueEntry(entry.Node))
		if err != nil {
			return err
		}
		err = c.necoStorage.UpdateRebootListEntry(ctx, entry)
		if err != nil {
			return err
		}
		slog.With(slog.String("operation", "addRebootListEntry")).Info("AddRebootListEntry", slog.String("node", entry.Node), slog.String("group", entry.Group))
	}
	return nil
}

func (c *Controller) moveToNextGroup(ctx context.Context, candidate []string, processingGroup string) (string, error) {
	for _, group := range candidate {
		if group != processingGroup {
			err := c.necoStorage.UpdateProcessingGroup(ctx, group)
			if err != nil {
				return "", err
			}
			slog.With(slog.String("operation", "moveNextGroup")).Info("moved to next group", slog.String("group", group))
			return group, nil
		}
	}
	return processingGroup, nil
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

func (c *Controller) collectEntries(rebootListEntries []*neco.RebootListEntry, rebootQueueEntries []*cke.RebootQueueEntry, processingGroup string) EntriesCollection {
	completedEntry := []*neco.RebootListEntry{}
	timedOutEntry := []EntrySet{}
	newEntry := []*neco.RebootListEntry{}
	cancelledEntry := []EntrySet{}
	queuedEntry := []EntrySet{}
	for _, entry := range rebootListEntries {
		rqEntry := findRebootQueueEntryFromRebootListEntry(rebootQueueEntries, *entry)
		switch entry.Status {
		case neco.RebootListEntryStatusCancelled:
			cancelledEntry = append(cancelledEntry, EntrySet{entry, rqEntry})
		case neco.RebootListEntryStatusPending:
			if entry.Group == processingGroup && c.isRebootable(entry) {
				newEntry = append(newEntry, entry)
			}
		case neco.RebootListEntryStatusQueued:
			if rqEntry == nil {
				completedEntry = append(completedEntry, entry)
			} else {
				queuedEntry = append(queuedEntry, EntrySet{entry, rqEntry})
				if !c.isRebootable(entry) {
					timedOutEntry = append(timedOutEntry, EntrySet{entry, rqEntry})
				}
			}
		}
	}
	return EntriesCollection{
		CompletedEntry: completedEntry,
		TimedOutEntry:  timedOutEntry,
		NewEntry:       newEntry,
		CancelledEntry: cancelledEntry,
		QueuedEntry:    queuedEntry,
	}
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
	// this logic is implemented to perist shuffuled groups between runOnce calls
	currnetAllGroups := getAllGroups(rebootListEntries)
	if !isEqualContents(allGroups, currnetAllGroups) {
		slog.Info("groups changed, updating allGroups", slog.String("new", fmt.Sprintf("%v", currnetAllGroups)))
		allGroups = currnetAllGroups
	}

	candidate := []string{}
	for _, group := range allGroups {
		rebootableEntries := c.findRebootableNodeInGroup(rebootListEntries, group)
		if len(rebootableEntries) > 0 {
			candidate = append(candidate, group)
		}
	}
	if len(candidate) != 0 && enabled {
		if len(rebootQueueEntries) != 0 {
			isStuck := true
			for _, rq := range rebootQueueEntries {
				if rq.Status == cke.RebootStatusRebooting || rq.DrainBackOffCount == 0 {
					isStuck = false
				}
			}
			if isStuck {
				slog.Info("rebootQueue is stuck, moving to next group")
				collection := c.collectEntries(rebootListEntries, rebootQueueEntries, processingGroup)
				err = c.dequeueAndCancelEntry(ctx, collection.QueuedEntry)
				if err != nil {
					return err
				}
				processingGroup, err = c.moveToNextGroup(ctx, candidate, processingGroup)
				if err != nil {
					return err
				}
			}
		} else {
			processingGroup, err = c.moveToNextGroup(ctx, candidate, processingGroup)
			if err != nil {
				return err
			}
		}
	} else if !enabled {
		slog.Info("skipping moveToNextGroup, neco-rebooter is disabled")
	} else {
		slog.Info("no rebootable nodes found")
	}

	collection := c.collectEntries(rebootListEntries, rebootQueueEntries, processingGroup)

	if len(collection.CancelledEntry) > 0 {
		err = c.removeCancelledEntry(ctx, collection.CancelledEntry)
		if err != nil {
			return err
		}
		return nil
	}
	err = c.removeCompletedEntry(ctx, collection.CompletedEntry)
	if err != nil {
		return err
	}
	if enabled {
		err = c.addRebootListEntry(ctx, collection.NewEntry)
		if err != nil {
			return err
		}
		err = c.dequeueAndCancelEntry(ctx, collection.TimedOutEntry)
		if err != nil {
			return err
		}
	} else {
		err = c.dequeueAndCancelEntry(ctx, collection.QueuedEntry)
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
