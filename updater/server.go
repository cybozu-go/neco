package updater

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/coreos/etcd/clientv3/concurrency"
	"github.com/cybozu-go/log"
	"github.com/cybozu-go/neco"
	"github.com/cybozu-go/neco/ext"
	"github.com/cybozu-go/neco/storage"
	"github.com/cybozu-go/well"
)

// Server represents neco-updater server
type Server struct {
	session  *concurrency.Session
	storage  storage.Storage
	pkg      PackageManager
	notifier ext.Notifier

	currentSnapshot storage.Snapshot
}

// NewServer returns a Server
func NewServer(session *concurrency.Session, storage storage.Storage, pkg PackageManager, notifier ext.Notifier) Server {
	return Server{
		session:  session,
		storage:  storage,
		pkg:      pkg,
		notifier: notifier,
	}
}

// Run runs neco-updater
func (s Server) Run(ctx context.Context) error {

	hostname, err := os.Hostname()
	if err != nil {
		return err
	}

	e := concurrency.NewElection(s.session, storage.KeyUpdaterLeader)

RETRY:
	select {
	case <-s.session.Done():
		return errors.New("session has been orphaned")
	default:
	}

	err = e.Campaign(ctx, hostname)
	if err != nil {
		return err
	}
	leaderKey := e.Key()

	log.Info("I am the leader", map[string]interface{}{
		"session": s.session.Lease(),
	})

	env := well.NewEnvironment(ctx)
	env.Go(func(ctx context.Context) error {
		return s.runLoop(ctx, leaderKey)
	})
	checker := NewReleaseChecker(s.storage, leaderKey)
	env.Go(checker.Run)
	env.Stop()
	err = env.Wait()

	// workaround for etcd clientv3 bug that hangs up when the first
	// endpoint is stopping.
	ctxWithTimeout, cancel := context.WithTimeout(ctx, 5*time.Second)
	err2 := e.Resign(ctxWithTimeout)
	cancel()
	if err2 != nil {
		return err2
	}
	if err == storage.ErrNoLeader {
		log.Warn("lost the leadership", nil)
		goto RETRY
	}
	return err
}

func (s Server) runLoop(ctx context.Context, leaderKey string) error {
	timeout, err := s.storage.GetWorkerTimeout(ctx)
	if err != nil {
		return err
	}
	for {
		ss, err := s.storage.NewSnapshot(ctx)
		if err != nil {
			return err
		}

		action, err := NextAction(ctx, ss, s.pkg, timeout)
		if err != nil {
			return err
		}
		log.Info("next action", map[string]interface{}{
			"action": action.String(),
		})

		switch action {
		case ActionWaitInfo:
			err = s.storage.WaitInfo(ctx, ss.Revision)
			if err != nil {
				return err
			}
		case ActionReconfigure:
			req := neco.UpdateRequest{
				Version:   ss.Request.Version,
				Servers:   ss.Servers,
				StartedAt: time.Now().UTC(),
			}
			err = s.storage.PutReconfigureRequest(ctx, req, leaderKey)
			if err != nil {
				return err
			}
			err = s.notifier.NotifyInfo(ctx, *s.currentSnapshot.Request, "start boot servers reconfiguration.")
			if err != nil {
				log.Warn("failed to notify", map[string]interface{}{log.FnError: err})
			}
		case ActionNewVersion:
			req := neco.UpdateRequest{
				Version:   ss.Latest,
				Servers:   ss.Servers,
				StartedAt: time.Now().UTC(),
			}
			err = s.storage.PutRequest(ctx, req, leaderKey)
			if err != nil {
				return err
			}
			err = s.notifier.NotifyInfo(ctx, *s.currentSnapshot.Request, "start updating the new release.")
			if err != nil {
				log.Warn("failed to notify", map[string]interface{}{log.FnError: err})
			}
		case ActionWaitWorkers:
			err = s.waitComplete(ctx, leaderKey, ss, timeout)
			if err != nil {
				return err
			}
		case ActionStop:
			req := *ss.Request
			req.Stop = true
			err = s.storage.PutRequest(ctx, req, leaderKey)
			if err != nil {
				return err
			}
		case ActionWaitClear:
			err = s.storage.WaitRequestDeletion(ctx, ss.Revision)
			if err != nil {
				return err
			}
		default:
			return fmt.Errorf("invalid action %s: %d", action.String(), int(action))
		}
	}
}

func (s Server) waitComplete(ctx context.Context, leaderKey string, ss *storage.Snapshot, timeout time.Duration) error {
	deadline := ss.Request.StartedAt.Add(timeout)
	ctxWithDeadline, cancel := context.WithDeadline(ctx, deadline)
	defer cancel()
	s.currentSnapshot = *ss
	err := storage.NewWorkerWatcher(s.handleStatus).
		Watch(ctxWithDeadline, ss.Revision, s.storage)
	if err == storage.ErrTimedOut {
		log.Warn("workers timed-out", map[string]interface{}{
			"version":    s.currentSnapshot.Request.Version,
			"started_at": s.currentSnapshot.Request.StartedAt,
			"timeout":    timeout.String(),
		})
		err = s.notifier.NotifyFailure(ctx, *s.currentSnapshot.Request, fmt.Sprintf("timeout occurred: %s", timeout.String()))
		if err != nil {
			log.Warn("failed to notify", map[string]interface{}{log.FnError: err})
		}
		return nil
	}
	return err
}

func (s Server) handleStatus(ctx context.Context, lrn int, st *neco.UpdateStatus) bool {
	if st.Version != s.currentSnapshot.Request.Version {
		return false
	}
	s.currentSnapshot.Statuses[lrn] = st

	switch st.Cond {
	case neco.CondAbort:
		log.Warn("worker failed updating", map[string]interface{}{
			"version": s.currentSnapshot.Request.Version,
			"lrn":     lrn,
			"message": st.Message,
		})
		err := s.notifier.NotifyFailure(ctx, *s.currentSnapshot.Request, "update request was aborted: "+st.Message)
		if err != nil {
			log.Warn("failed to notify", map[string]interface{}{log.FnError: err})
		}
		return true
	case neco.CondComplete:
		log.Info("worker finished updating", map[string]interface{}{
			"version": s.currentSnapshot.Request.Version,
			"lrn":     lrn,
		})
	}

	completed := neco.UpdateCompleted(s.currentSnapshot.Request.Version, s.currentSnapshot.Servers, s.currentSnapshot.Statuses)
	if completed {
		log.Info("all worker finished updating", map[string]interface{}{
			"version": s.currentSnapshot.Request.Version,
			"servers": s.currentSnapshot.Servers,
		})
		err := s.notifier.NotifySucceeded(ctx, *s.currentSnapshot.Request)
		if err != nil {
			log.Warn("failed to notify", map[string]interface{}{log.FnError: err})
		}
		return true
	}

	return false
}
