package updater

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/coreos/etcd/clientv3/concurrency"
	"github.com/cybozu-go/log"
	"github.com/cybozu-go/neco"
	"github.com/cybozu-go/neco/storage"
	"github.com/cybozu-go/well"
)

// Server represents neco-updater server
type Server struct {
	session *concurrency.Session
	storage storage.Storage
	timeout time.Duration
}

// NewServer returns a Server
func NewServer(session *concurrency.Session, storage storage.Storage, timeout time.Duration) Server {
	return Server{
		session: session,
		storage: storage,
		timeout: timeout,
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

	ctxWithTimeout, cancel := context.WithTimeout(context.Background(), s.timeout)
	defer cancel()
	err2 := e.Resign(ctxWithTimeout)
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
	pkg := DebPackageManager{}
	for {
		ss, err := s.storage.NewSnapshot(ctx)
		if err != nil {
			return err
		}
		action, err := s.NextAction(ctx, ss, pkg)
		if err != nil {
			return err
		}
		switch action {
		case ActionNone:
		case ActionReconfigure:
			err = s.storage.ClearStatus(ctx, true)
			if err != nil {
				return err
			}
			fallthrough
		case ActionNewVersion:
			rev, err := s.update(ctx, leaderKey, ss)
			if err == errStopped {
				err = s.storage.WaitRequestDeletion(ctx, rev)
				if err == nil {
					continue
				}
			}
			if err != nil {
				return err
			}
		case ActionWaitClear:
			err = s.storage.WaitRequestDeletion(ctx, ss.Revision)
			if err != nil {
				return err
			}
			continue
		}

		err = s.storage.WaitInfo(ctx, ss.Revision)
		if err != nil {
			return err
		}
	}
}

// startUpdate starts update with tag.  It returns created request and its
// modified revision.  It returns ErrNoMembers if no bootservers are registered
// in etcd.
func (s Server) update(ctx context.Context, leaderKey string, snapshot *storage.Snapshot) (int64, error) {
	servers := snapshot.Servers
	tag := snapshot.Latest
	if len(servers) == 0 {
		log.Info("No bootservers exists in etcd", map[string]interface{}{})
		return 0, ErrNoMembers
	}
	log.Info("Starting updating", map[string]interface{}{
		"version": tag,
		"servers": servers,
	})
	r := neco.UpdateRequest{
		Version:   tag,
		Servers:   servers,
		Stop:      false,
		StartedAt: time.Now(),
	}
	err := s.storage.PutRequest(ctx, r, leaderKey)
	if err != nil {
		return 0, err
	}

	_, rev, err := s.storage.GetRequestWithRev(ctx)
	if err != nil {
		return 0, err
	}

	notifier, err := s.notifier(ctx)
	if err != nil {
		return 0, err
	}
	watcher := newWorkerWatcher(&r, notifier)

	timeout, err := s.storage.GetWorkerTimeout(ctx)
	if err != nil {
		return 0, err
	}
	deadline := r.StartedAt.Add(timeout)
	ctxWithDeadline, cancel := context.WithDeadline(ctx, deadline)
	err = storage.NewWorkerWatcher(watcher.handleStatus).
		Watch(ctxWithDeadline, rev, s.storage)
	cancel()

	return rev, nil
}

// stopUpdate of the current request
func (s Server) stopUpdate(ctx context.Context, req *neco.UpdateRequest, leaderKey string) error {
	req.Stop = true
	return s.storage.PutRequest(ctx, *req, leaderKey)
}

func (s Server) notifier(ctx context.Context) (Notifier, error) {
	var notifier Notifier
	notifier, err := s.newSlackClient(ctx)
	if err == storage.ErrNotFound {
		notifier = nopNotifier{}
	} else if err != nil {
		return nil, err
	}
	return notifier, nil
}

func (s Server) newSlackClient(ctx context.Context) (*SlackClient, error) {
	webhookURL, err := s.storage.GetSlackNotification(ctx)
	if err == storage.ErrNotFound {
		return nil, storage.ErrNotFound
	} else if err != nil {
		return nil, err
	}

	var http *http.Client

	proxyURL, err := s.storage.GetProxyConfig(ctx)
	if err == storage.ErrNotFound {
	} else if err != nil {
		return nil, err
	} else {
		u, err := url.Parse(proxyURL)
		if err != nil {
			return nil, err
		}
		http = newHTTPClient(u)
	}

	return &SlackClient{URL: webhookURL, HTTP: http}, nil
}
