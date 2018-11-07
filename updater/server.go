package updater

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
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

		checker: NewReleaseChecker(storage),
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
	var current *neco.UpdateRequest
	var currentRev int64

	// Updater continues last update without create update reuqest with "skipRequest = true"
	var skipRequest bool

	current, currentRev, err := s.storage.GetRequestWithRev(ctx)
	if err == nil {
		target = current.Version
		if current.Stop {
			log.Info("Last updating is failed, wait for retrying", map[string]interface{}{
				"version": current.Version,
			})
			err := s.waitRetry(ctx, current, currentRev)
			if err != nil {
				return err
			}
		} else {
			log.Info("Last updating is still in progress, wait for workers", map[string]interface{}{
				"version": current.Version,
			})
			skipRequest = true
		}
	} else if err != nil && err != storage.ErrNotFound {
		return err
	}

	for {
		if len(target) == 0 {
			if s.checker.HasUpdate() {
				target = s.checker.GetLatest()
			}
		}
		if len(target) != 0 {
			// Found new update
			for {
				if !skipRequest {
					current, currentRev, err = s.startUpdate(ctx, target, leaderKey)
					if err == ErrNoMembers {
						break
					} else if err != nil {
						return err
					}
				}
				skipRequest = false

				err = s.waitWorkers(ctx, current, currentRev)
				if err == nil {
					break
				}
				if err != ErrUpdateFailed {
					return err
				}
				err = s.stopUpdate(ctx, current, leaderKey)
				if err != nil {
					return err
				}
				err = s.waitRetry(ctx, current, currentRev)
				if err != nil {
					return err
				}
			}
		}

		timeout, err := s.waitForMemberUpdated(ctx, current, currentRev)
		if timeout {
			// Clear target to check latest update
			target = ""
		} else if err != nil {
			return err
		}
	}
}

// startUpdate starts update with tag.  It returns created request and its
// modified revision.  It returns ErrNoMembers if no bootservers are registered
// in etcd.
func (s Server) startUpdate(ctx context.Context, tag, leaderKey string) (*neco.UpdateRequest, int64, error) {
	servers, err := s.storage.GetBootservers(ctx)
	if err != nil {
		return nil, 0, err
	}
	if len(servers) == 0 {
		log.Info("No bootservers exists in etcd", map[string]interface{}{})
		return nil, 0, ErrNoMembers
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
	err = s.storage.PutRequest(ctx, r, leaderKey)
	if err != nil {
		return nil, 0, err
	}
	return s.storage.GetRequestWithRev(ctx)
}

// stopUpdate of the current request
func (s Server) stopUpdate(ctx context.Context, req *neco.UpdateRequest, leaderKey string) error {
	req.Stop = true
	return s.storage.PutRequest(ctx, *req, leaderKey)
}

// waitWorkers waits for worker finishes updates until timed-out
func (s Server) waitWorkers(ctx context.Context, req *neco.UpdateRequest, rev int64) error {
	timeout, err := s.storage.GetWorkerTimeout(ctx)
	if err != nil {
		return err
	}

	statuses := make(map[int]*neco.UpdateStatus)

	deadline := req.StartedAt.Add(timeout)
	deadlineCtx, cancel := context.WithDeadline(ctx, deadline)
	defer cancel()

	ch := s.session.Client().Watch(
		deadlineCtx, storage.KeyStatusPrefix,
		clientv3.WithRev(rev+1), clientv3.WithFilterDelete(), clientv3.WithPrefix(),
	)
	for resp := range ch {
		for _, ev := range resp.Events {
			st := new(neco.UpdateStatus)
			err = json.Unmarshal(ev.Kv.Value, st)
			if err != nil {
				return err
			}
			lrn, err := strconv.Atoi(string(ev.Kv.Key[len(storage.KeyStatusPrefix):]))
			if err != nil {
				return err
			}
			if st.Version != req.Version {
				continue
			}
			statuses[lrn] = st

			switch st.Cond {
			case neco.CondAbort:
				log.Warn("worker failed updating", map[string]interface{}{
					"version": req.Version,
					"lrn":     lrn,
					"message": st.Message,
				})
				s.notifySlackServerFailure(ctx, *req, st)
				return ErrUpdateFailed
			case neco.CondComplete:
				log.Info("worker finished updating", map[string]interface{}{
					"version": req.Version,
					"lrn":     lrn,
				})
			}
		}

		success := neco.UpdateCompleted(req.Version, req.Servers, statuses)
		if success {
			log.Info("all worker finished updating", map[string]interface{}{
				"version": req.Version,
				"servers": req.Servers,
			})
			s.notifySlackSucceeded(ctx, *req)
			return nil
		}
	}

	log.Warn("workers timed-out", map[string]interface{}{
		"version":    req.Version,
		"started_at": req.StartedAt,
		"timeout":    timeout,
	})
	s.notifySlackTimeout(ctx, *req)
	return ErrUpdateFailed
}

func (s Server) waitRetry(ctx context.Context, req *neco.UpdateRequest, rev int64) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	ch := s.session.Client().Watch(ctx, storage.KeyCurrent, clientv3.WithRev(rev+1), clientv3.WithFilterPut())
	resp := <-ch
	if err := resp.Err(); err != nil {
		return err
	}
	return nil
}

// waitForMemberUpdated waits for new member added or member removed until with
// check-update-interval. It returns (true, nil) when timed-out.  It returns
// (false, nil) if member list is updated.
func (s Server) waitForMemberUpdated(ctx context.Context, req *neco.UpdateRequest, rev int64) (timeout bool, err error) {
	interval, err := s.storage.GetCheckUpdateInterval(ctx)
	if err != nil {
		return false, err
	}

	withTimeoutCtx, cancel := context.WithTimeout(ctx, interval)
	defer cancel()

	ch := s.session.Client().Watch(
		withTimeoutCtx, storage.KeyBootserversPrefix, clientv3.WithRev(rev+1), clientv3.WithPrefix(),
	)

	for resp := range ch {
		for _, ev := range resp.Events {
			lrn, err := strconv.Atoi(string(ev.Kv.Key[len(storage.KeyBootserversPrefix):]))
			if err != nil {
				return false, err
			}
			if ev.Type == clientv3.EventTypePut {
				if i := sort.SearchInts(req.Servers, lrn); i < len(req.Servers) && req.Servers[i] == lrn {
					continue
				}
			}
			return false, nil
		}
	}
	if withTimeoutCtx.Err() == context.DeadlineExceeded {
		return true, nil
	}
	return false, withTimeoutCtx.Err()
}

func (s Server) notifySlackSucceeded(ctx context.Context, req neco.UpdateRequest) error {
	slack, err := s.newSlackClient(ctx)
	if err == storage.ErrNotFound {
		return nil
	} else if err != nil {
		return err
	}

	att := Attachment{
		Color:      ColorGood,
		AuthorName: "Boot server updater",
		Title:      "Update completed successfully",
		Text:       "Updating on boot servers are completed successfully :tada: :tada: :tada:",
		Fields: []AttachmentField{
			{Title: "Version", Value: req.Version, Short: true},
			{Title: "Servers", Value: fmt.Sprintf("%v", req.Servers), Short: true},
			{Title: "Started at", Value: req.StartedAt.Format(time.RFC3339), Short: true},
		},
	}
	payload := Payload{Attachments: []Attachment{att}}
	return slack.PostWebHook(ctx, payload)
}

func (s Server) notifySlackServerFailure(ctx context.Context, req neco.UpdateRequest, st *neco.UpdateStatus) error {
	slack, err := s.newSlackClient(ctx)
	if err == storage.ErrNotFound {
		return nil
	} else if err != nil {
		return err
	}

	att := Attachment{
		Color:      ColorDanger,
		AuthorName: "Boot server updater",
		Title:      "Failed to update boot servers",
		Text:       "Failed to update boot servers due to some worker return(s) error :crying_cat_face:.  Please fix it manually.",
		Fields: []AttachmentField{
			{Title: "Version", Value: req.Version, Short: true},
			{Title: "Servers", Value: fmt.Sprintf("%v", req.Servers), Short: true},
			{Title: "Started at", Value: req.StartedAt.Format(time.RFC3339), Short: true},
			{Title: "Reason", Value: st.Message, Short: true},
		},
	}
	payload := Payload{Attachments: []Attachment{att}}
	return slack.PostWebHook(ctx, payload)
}

func (s Server) notifySlackTimeout(ctx context.Context, req neco.UpdateRequest) error {
	slack, err := s.newSlackClient(ctx)
	if err == storage.ErrNotFound {
		return nil
	} else if err != nil {
		return err
	}

	att := Attachment{
		Color:      ColorDanger,
		AuthorName: "Boot server updater",
		Title:      "Update failed on the boot servers",
		Text:       "Failed to update boot servers due to timed-out from worker updates :crying_cat_face:.  Please fix it manually.",
		Fields: []AttachmentField{
			{Title: "Version", Value: req.Version, Short: true},
			{Title: "Servers", Value: fmt.Sprintf("%v", req.Servers), Short: true},
			{Title: "Started at", Value: req.StartedAt.Format(time.RFC3339), Short: true},
		},
	}
	payload := Payload{Attachments: []Attachment{att}}
	return slack.PostWebHook(ctx, payload)
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
