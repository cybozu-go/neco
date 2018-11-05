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
	version "github.com/hashicorp/go-version"
)

// Server represents neco-updater server
type Server struct {
	session *concurrency.Session
	github  releaseInterface
	storage storage.Storage
}

// NewServer returns a Server
func NewServer(session *concurrency.Session, storage storage.Storage) Server {
	return Server{
		session: session,
		github:  releaseClient{"cybozu-go", "neco"},
		storage: storage,
	}
}

// Run runs neco-updater
func (s Server) Run(ctx context.Context) error {

	hostname, err := os.Hostname()
	if err != nil {
		return err
	}

	e := concurrency.NewElection(s.session, storage.KeyLeader)

	select {
	case <-s.session.Done():
		return errors.New("session has been orphaned")
	default:
	}

	err = e.Campaign(ctx, hostname)
	if err != nil {
		return err
	}
	defer e.Resign(ctx)

	leaderKey := e.Key()

	log.Info("I am the leader", map[string]interface{}{
		"session": s.session.Lease(),
	})

	var target string
	req, err := s.storage.GetRequest(ctx)
	if err != nil && err != storage.ErrNotFound {
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
	} else if err != nil {
		return err
	}

	for {
		if len(target) == 0 {
			target, err = s.checkUpdateNewVersion(ctx)
			if err != nil {
				return err
			}
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

		timeout, err := s.waitForNext(ctx)
		if err != nil {
			return err
		}
		if timeout {
			target = ""
		}
	}

}

// checkUpdateNewVersion checks if newer version exits compare with installed version
// If no newer version exists, return empty tag with nil error
func (s Server) checkUpdateNewVersion(ctx context.Context) (string, error) {
	env, err := s.storage.GetEnvConfig(ctx)
	if err == storage.ErrNotFound {
		return "", nil
	}
	if err != nil {
		return "", err
	}

	var latest string
	if env == neco.StagingEnv {
		latest, err = s.github.GetLatestReleaseTag(ctx)
	} else if env == neco.ProdEnv {
		latest, err = s.github.GetLatestPreReleaseTag(ctx)
	} else {
		return "", errors.New("unknown env: " + env)
	}
	if err == ErrNoReleases {
		return "", nil
	}
	if err != nil {
		return "", err
	}

	current, err := neco.InstalledNecoVersion(ctx)
	if err != nil {
		return "", err
	}

	latestVer, err := version.NewVersion(latest)
	if err != nil {
		return "", err
	}

	currentVer, err := version.NewVersion(current)
	if err != nil {
		return "", err
	}

	if !latestVer.GreaterThan(currentVer) {
		return "", nil
	}
	return latest, nil
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
	for _, lrn := range req.Servers {
		st, err := s.storage.GetStatusAt(ctx, lrn, rev)
		if err == storage.ErrNotFound {
			continue
		}
		if err != nil {
			return err
		}
		statuses[lrn] = *st
	}

	deadline := req.StartedAt.Add(timeout)
	deadlineCtx, cancel := context.WithDeadline(ctx, deadline)
	defer cancel()

	for _, lrn := range req.Servers {
		st, ok := statuses[lrn]
		if !ok || !st.Finished || !statuses[lrn].Error || st.Version != req.Version {
			continue
		}
		log.Info("worker failed updating", map[string]interface{}{
			"version": req.Version,
			"lrn":     lrn,
			"message": st.Message,
		})
		s.notifyFailure(ctx, *req, &st)
		return ErrUpdateFailed
	}

	allSucceeded := func() bool {
		for _, lrn := range req.Servers {
			if st, ok := statuses[lrn]; !ok || !st.Finished || !statuses[lrn].Error || st.Version != req.Version {
				return false
			}
		}
		return true
	}
	if allSucceeded() {
		log.Info("all worker finished updating", map[string]interface{}{
			"version": req.Version,
			"servers": req.Servers,
		})
		s.notifySucceeded(ctx, *req)
		return nil
	}

	ch := s.session.Client().Watch(
		deadlineCtx, storage.KeyStatusPrefix,
		clientv3.WithRev(rev+1), clientv3.WithFilterPut(), clientv3.WithPrefix(),
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
		if allSucceeded() {
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

	ch := s.session.Client().Watch(ctx, storage.KeyCurrent, clientv3.WithRev(rev+1), clientv3.WithFilterDelete())
	<-ch
	return nil
}

// waitForNext sleeps for check-update-interval or boot servers changed
// It returns true if timed-out for check-update-interval, and returns false if member changed
func (s Server) waitForNext(ctx context.Context) (timeout bool, err error) {
	interval, err := s.storage.GetCheckUpdateInterval(ctx)
	if err != nil {
		return false, err
	}
	req, rev, err := s.storage.GetRequestWithRev(ctx)
	if err != nil {
		return false, err
	}
	lrns := req.Servers
	sort.Ints(lrns)

	updateCh := make(chan struct{})

	env := well.NewEnvironment(ctx)
	env.Go(func(ctx context.Context) error {
		ch := s.session.Client().Watch(ctx, storage.KeyBootserversPrefix, clientv3.WithRev(rev+1))
		for resp := range ch {
			for _, ev := range resp.Events {
				lrn, err := strconv.Atoi(string(ev.Kv.Key[len(storage.KeyBootserversPrefix):]))
				if err != nil {
					return err
				}
				if ev.Type == clientv3.EventTypePut {
					if i := sort.SearchInts(lrns, lrn); i < len(lrns) && lrns[i] == lrn {
						continue
					}
				}
				updateCh <- struct{}{}
				return nil
			}
		}
		return nil
	})
	env.Stop()
	defer func() {
		env.Cancel(nil)
		env.Wait()
	}()

	select {
	case <-ctx.Done():
		return false, ctx.Err()
	case <-time.After(interval):
		return true, nil
	case <-updateCh:
	}
	return false, nil
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
	return NotifySlack(context.Background(), url, payload)
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
