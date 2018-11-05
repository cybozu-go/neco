package updater

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"
	"strconv"
	"time"

	"github.com/coreos/etcd/clientv3"
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

	checker ReleaseChecker
}

// NewServer returns a Server
func NewServer(session *concurrency.Session, storage storage.Storage, timeout time.Duration) Server {
	return Server{
		session: session,
		storage: storage,
		timeout: timeout,

		checker: NewReleaseChecker(
			storage,
			ReleaseClient{neco.GitHubRepoOwner, neco.GitHubRepoName},
			DebPackageManager{},
		),
	}
}

// Run runs neco-updater
func (s Server) Run(ctx context.Context) error {

	hostname, err := os.Hostname()
	if err != nil {
		return err
	}

	e := concurrency.NewElection(s.session, storage.KeyLeader)

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
	env.Go(func(ctx context.Context) error {
		return s.checker.Run(ctx)
	})
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
	var target string
	req, err := s.storage.GetRequest(ctx)
	if err == nil {
		target = req.Version
		if req.Stop {
			log.Info("Last updating is failed, wait for retrying", map[string]interface{}{
				"version": req.Version,
			})
			err := s.waitRetry(ctx)
			if err != nil {
				return err
			}
		}
	} else if err != nil && err != storage.ErrNotFound {
		return err
	}

	for {
		if len(target) == 0 {
			target = s.checker.GetLatest()
		}
		if len(target) != 0 {
			// Found new update
			for {
				err = s.startUpdate(ctx, target, leaderKey)
				if err == ErrNoMembers {
					break
				}
				err = s.waitWorkers(ctx)
				if err == nil {
					break
				}
				if err != ErrUpdateFailed {
					return err
				}
				err := s.stopUpdate(ctx, leaderKey)
				if err != nil {
					return err
				}
				err = s.waitRetry(ctx)
				if err != nil {
					return err
				}
			}
		}

		err := s.waitForMemberUpdated(ctx)
		if err == context.DeadlineExceeded {
			target = ""
		} else if err != nil {
			return err
		}
	}
}

// startUpdate starts update with tag.  It returns ErrNoMembers if no
// bootservers are registered in etcd.
func (s Server) startUpdate(ctx context.Context, tag, leaderKey string) error {
	servers, err := s.storage.GetBootservers(ctx)
	if err != nil {
		return err
	}
	if len(servers) == 0 {
		log.Info("No bootservers exists in etcd", map[string]interface{}{})
		return ErrNoMembers
	}
	log.Info("New neco release is found, starting updating", map[string]interface{}{
		"version": tag,
		"servers": servers,
	})
	r := neco.UpdateRequest{
		Version:   tag,
		Servers:   servers,
		Stop:      false,
		StartedAt: time.Now(),
	}
	return s.storage.PutRequest(ctx, r, leaderKey)
}

func (s Server) stopUpdate(ctx context.Context, leaderKey string) error {
	req, err := s.storage.GetRequest(ctx)
	if err != nil {
		return err
	}
	req.Stop = true
	return s.storage.PutRequest(ctx, *req, leaderKey)
}

// waitWorkers waits for worker finishes updates until timed-out
func (s Server) waitWorkers(ctx context.Context) error {
	timeout, err := s.storage.GetWorkerTimeout(ctx)
	if err != nil {
		return err
	}

	req, rev, err := s.storage.GetRequestWithRev(ctx)
	if err != nil {
		return err
	}
	statuses := make(map[int]neco.UpdateStatus)

	deadline := req.StartedAt.Add(timeout)
	deadlineCtx, cancel := context.WithDeadline(ctx, deadline)
	defer cancel()

	ch := s.session.Client().Watch(
		deadlineCtx, storage.KeyStatusPrefix,
		clientv3.WithRev(rev+1), clientv3.WithFilterDelete(), clientv3.WithPrefix(),
	)
	for resp := range ch {
		for _, ev := range resp.Events {
			var st neco.UpdateStatus
			err = json.Unmarshal(ev.Kv.Value, &st)
			if err != nil {
				return err
			}
			lrn, err := strconv.Atoi(string(ev.Kv.Key[len(storage.KeyStatusPrefix):]))
			if err != nil {
				return err
			}
			statuses[lrn] = st

			if st.Error {
				log.Info("worker failed updating", map[string]interface{}{
					"version": req.Version,
					"lrn":     lrn,
					"message": st.Message,
				})
				s.notifyFailure(ctx, *req, &st)
				return ErrUpdateFailed
			}
			if st.Finished {
				log.Info("worker finished updating", map[string]interface{}{
					"version": req.Version,
					"lrn":     lrn,
				})
			}
		}
		success := true
		for _, lrn := range req.Servers {
			if st, ok := statuses[lrn]; !ok || !st.Finished || !statuses[lrn].Error || st.Version != req.Version {
				success = false
				break
			}
		}
		if success {
			log.Info("all worker finished updating", map[string]interface{}{
				"version": req.Version,
				"servers": req.Servers,
			})
			s.notifySucceeded(ctx, *req)
			return nil
		}
	}

	log.Warn("workers timed-out", map[string]interface{}{
		"version":    req.Version,
		"started_at": req.StartedAt,
		"timeout":    timeout,
	})
	s.notifyFailure(ctx, *req, nil)
	return ErrUpdateFailed
}

func (s Server) waitRetry(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	_, rev, err := s.storage.GetRequestWithRev(ctx)
	if err == storage.ErrNotFound {
		return nil
	} else if err != nil {
		return err
	}

	ch := s.session.Client().Watch(ctx, storage.KeyCurrent, clientv3.WithRev(rev+1), clientv3.WithFilterPut())
	resp := <-ch
	if err := resp.Err(); err != nil {
		return err
	}
	return nil
}

// waitForMemberUpdated waits for new member added or member removed until with
// check-update-interval.  It returns nil error if member updated, or returns
// context.DeadlineExceeded if timed-out
func (s Server) waitForMemberUpdated(ctx context.Context) error {
	interval, err := s.storage.GetCheckUpdateInterval(ctx)
	if err != nil {
		return err
	}
	req, rev, err := s.storage.GetRequestWithRev(ctx)
	if err != nil {
		return err
	}
	lrns := req.Servers
	sort.Ints(lrns)

	withTimeoutCtx, cancel := context.WithTimeout(ctx, interval)
	defer cancel()

	ch := s.session.Client().Watch(
		withTimeoutCtx, storage.KeyBootserversPrefix, clientv3.WithRev(rev+1),
	)

	var updated bool
	var lastErr error
	for resp := range ch {
		for _, ev := range resp.Events {
			lrn, err := strconv.Atoi(string(ev.Kv.Key[len(storage.KeyBootserversPrefix):]))
			if err != nil {
				lastErr = err
				cancel()
			}
			if ev.Type == clientv3.EventTypePut {
				if i := sort.SearchInts(lrns, lrn); i < len(lrns) && lrns[i] == lrn {
					continue
				}
			}
			updated = true
			cancel()
		}
	}
	if lastErr != nil {
		return lastErr
	}
	if !updated {
		return context.DeadlineExceeded
	}
	return nil
}

func (s Server) notifySucceeded(ctx context.Context, req neco.UpdateRequest) error {
	url, err := s.storage.GetSlackNotification(ctx)
	if err == storage.ErrNotFound {
		return nil
	} else if err != nil {
		return err
	}

	text := fmt.Sprintf("Updates to version `%s' are completed successfully on boot servers at lrn=%v.", req.Version, req.Servers)
	payload := Payload{
		Username:  "[SUCCESS] Boot server updater",
		IconEmoji: ":imp:",
		Text:      text,
	}
	return NotifySlack(ctx, url, payload)
}

func (s Server) notifyFailure(ctx context.Context, req neco.UpdateRequest, st *neco.UpdateStatus) error {
	url, err := s.storage.GetSlackNotification(ctx)
	if err == storage.ErrNotFound {
		return nil
	} else if err != nil {
		return err
	}

	text := "Failed to finish update boot server to version `%s' due to timed-out"
	if st != nil {
		text = fmt.Sprintf("Failed to finish update boot server to version `%s': %s", st.Version, st.Message)
	}
	payload := Payload{
		Username:  "[FAILURE] Boot server updater",
		IconEmoji: ":imp:",
		Text:      text,
	}
	return NotifySlack(context.Background(), url, payload)
}
