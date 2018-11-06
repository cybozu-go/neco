package worker

import (
	"context"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/coreos/etcd/clientv3"
	"github.com/cybozu-go/neco"
	"github.com/cybozu-go/neco/storage"
)

func proxyHTTPClient(proxyURL *url.URL) *http.Client {
	transport := &http.Transport{
		Proxy: http.ProxyURL(proxyURL),
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
			DualStack: true,
		}).DialContext,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	return &http.Client{
		Transport: transport,
		Timeout:   1 * time.Hour,
	}
}

func localHTTPClient() *http.Client {
	transport := &http.Transport{
		Proxy: nil,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
			DualStack: true,
		}).DialContext,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	return &http.Client{
		Transport: transport,
		Timeout:   10 * time.Minute,
	}
}

// Worker implements Neco auto update worker process.
// This is a state machine.
type Worker struct {
	mylrn       int
	version     string
	ec          *clientv3.Client
	storage     storage.Storage
	proxyClient *http.Client
	localClient *http.Client

	// internal states
	req     *neco.UpdateRequest
	status  *neco.UpdateStatus
	barrier Barrier
}

// NewWorker returns a *Worker.
func NewWorker(ctx context.Context, ec *clientv3.Client) (*Worker, error) {
	mylrn, err := neco.MyLRN()
	if err != nil {
		return nil, err
	}

	version, err := GetDebianVersion("neco")
	if err != nil {
		return nil, err
	}

	localClient := localHTTPClient()
	proxyClient := localClient

	st := storage.NewStorage(ec)
	proxy, err := st.GetProxyConfig(ctx)
	if err != nil {
		if err != storage.ErrNotFound {
			return nil, err
		}
	} else {
		if len(proxy) > 0 {
			proxyURL, err := url.Parse(proxy)
			if err != nil {
				return nil, err
			}
			proxyClient = proxyHTTPClient(proxyURL)
		}
	}

	return &Worker{
		mylrn:       mylrn,
		version:     version,
		ec:          ec,
		storage:     st,
		proxyClient: proxyClient,
		localClient: localClient,
	}, nil
}

// Run waits for update request from neco-updater, then executes
// update process with other workers.  To communicate with neco-updater
// and other workers, etcd objects are used.
//
// Run works as follows:
//
// 1. Check the current request.  If the request is not found, go to 5.
//
// 2. If locally installed neco package is older than the requested version,
//    neco-worker updates the package, then exits to be restarted by systemd.
//
// 3. Check the status of request and workers; if the update process was aborted,
//    or if the update process has completed successfully, also go to 5.
//
// 4. Update programs for the requested version.
//
// 5. Wait for the new request.    If there is a new one, neco-worker updates
//    the package and exists to be restarted by systemd.
func (w *Worker) Run(ctx context.Context) error {
	req, rev, err := w.storage.GetRequestWithRev(ctx)

	if err == storage.ErrNotFound {
		return w.waitRequest(ctx)
	}
	if err != nil {
		return err
	}

	if w.version != req.Version {
		return w.updateNeco(ctx, req.Version)
	}

	stMap, err := w.storage.GetStatuses(ctx)
	if err != nil {
		return err
	}
	if neco.UpdateAborted(req.Version, stMap) {
		goto WAIT
	}
	if neco.UpdateCompleted(req.Version, req.Servers, stMap) {
		goto WAIT
	}

	w.req = req
	w.status = stMap[w.mylrn]
	w.barrier = NewBarrier(req.Servers)
	err = w.update(ctx, rev)
	if err != nil {
		return err
	}

WAIT:
	return w.waitRequest(ctx)
}

func (w *Worker) update(ctx context.Context, rev int64) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	ch := w.ec.Watch(ctx, storage.KeyStatusPrefix,
		clientv3.WithPrefix(),
		clientv3.WithRev(rev+1),
		clientv3.WithFilterDelete())
	for wr := range ch {
		err := wr.Err()
		if err != nil {
			return err
		}

		for _, ev := range wr.Events {
			err = w.dispatch(ctx, ev)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (w *Worker) dispatch(ctx context.Context, ev *clientv3.Event) error {
	key := string(ev.Kv.Key[len(storage.KeyStatusPrefix):])
	if key == "current" {
		return w.handleCurrent(ctx, ev)
	}

	lrn, err := strconv.Atoi(string(ev.Kv.Key[len(storage.KeyWorkerStatusPrefix):]))
	if err != nil {
		return err
	}

	if lrn == w.mylrn {
		return nil
	}
	return w.handleWorkerStatus(ctx, lrn, ev)
}

func (w *Worker) updateNeco(ctx context.Context, version string) error {
	deb, err := neco.CurrentArtifacts.FindDebianPackage("neco")
	if err != nil {
		return err
	}
	deb.Release = version

	return InstallDebianPackage(ctx, w.proxyClient, &deb)
}

func (w *Worker) handleCurrent(ctx context.Context, ev *clientv3.Event) error {
	return nil
}

func (w *Worker) handleWorkerStatus(ctx context.Context, lrn int, ev *clientv3.Event) error {
	return nil
}

func (w *Worker) waitRequest(ctx context.Context) error {
	return nil
}
