package updater

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/neco"
	"github.com/cybozu-go/neco/ext"
	"github.com/cybozu-go/neco/storage"
	"github.com/cybozu-go/well"
	"go.etcd.io/etcd/client/v3/concurrency"
)

// Server represents neco-updater server
type Server struct {
	session  *concurrency.Session
	storage  storage.Storage
	notifier ext.Notifier
}

// NewServer returns a Server
func NewServer(session *concurrency.Session, storage storage.Storage, notifier ext.Notifier) Server {
	return Server{
		session:  session,
		storage:  storage,
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

	ghc, err := ext.GitHubHTTPClient(ctx, s.storage)
	if err != nil {
		return err
	}
	checker := NewReleaseChecker(s.storage, leaderKey, ghc)
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

		action, err := NextAction(ss, timeout)
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
			err = s.notifier.NotifyInfo(req, "start boot servers reconfiguration.")
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
			err = s.notifier.NotifyInfo(req, "start updating the new release.")
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
	statuses := ss.Statuses
	if statuses == nil {
		statuses = make(map[int]*neco.UpdateStatus)
	}
	h := statusHandler{req: ss.Request, statuses: statuses, notifier: s.notifier}
	err := storage.NewWorkerWatcher(h.handleStatus).
		Watch(ctxWithDeadline, ss.Revision, s.storage)
	if err == storage.ErrTimedOut {
		log.Warn("workers take too long for update", map[string]interface{}{
			"version":    ss.Request.Version,
			"started_at": ss.Request.StartedAt,
			"timeout":    timeout.String(),
		})
		err = s.notifier.NotifyFailure(*ss.Request, "workers take too long for update: "+timeout.String())
		if err != nil {
			log.Warn("failed to notify", map[string]interface{}{log.FnError: err})
		}
		return nil
	}
	return err
}

type statusHandler struct {
	req      *neco.UpdateRequest
	statuses map[int]*neco.UpdateStatus
	notifier ext.Notifier
}

func (h statusHandler) handleStatus(ctx context.Context, lrn int, st *neco.UpdateStatus) bool {
	if st.Version != h.req.Version {
		return false
	}
	h.statuses[lrn] = st

	switch st.Cond {
	case neco.CondAbort:
		log.Warn("worker failed updating", map[string]interface{}{
			"version": h.req.Version,
			"lrn":     lrn,
			"message": st.Message,
		})
		err := h.notifier.NotifyFailure(*h.req, "update request was aborted: "+st.Message)
		if err != nil {
			log.Warn("failed to notify", map[string]interface{}{log.FnError: err})
		}
		return true
	case neco.CondComplete:
		log.Info("worker finished updating", map[string]interface{}{
			"version": h.req.Version,
			"lrn":     lrn,
		})
	}

	completed := neco.UpdateCompleted(h.req.Version, h.req.Servers, h.statuses)
	if completed {
		log.Info("all worker finished updating", map[string]interface{}{
			"version": h.req.Version,
			"servers": h.req.Servers,
		})
		err := h.notifier.NotifySucceeded(*h.req)
		if err != nil {
			log.Warn("failed to notify", map[string]interface{}{log.FnError: err})
		}
		return true
	}

	return false
}
