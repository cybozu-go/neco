package watch

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/neco/pkg/ingress-watcher/metrics"
	"github.com/cybozu-go/well"
)

// Watcher watches target server health and creates metrics from it.
type Watcher struct {
	targetAddrs []string
	interval    time.Duration
	httpClient  *well.HTTPClient
}

// NewWatcher creates an Ingress watcher.
func NewWatcher(
	targetURLs []string,
	interval time.Duration,
	httpClient *well.HTTPClient,
) *Watcher {
	return &Watcher{
		targetAddrs: targetURLs,
		interval:    interval,
		httpClient:  httpClient,
	}
}

// Run repeats to get server health and send it via channel.
func (w *Watcher) Run(ctx context.Context) error {
	env := well.NewEnvironment(ctx)
	metrics.WatchInterval.Set(w.interval.Seconds())
	for _, t := range w.targetAddrs {
		// Initialize counter value as 0.
		// Not initialize HTTPGetSuccessfulTotal because it needs status code.
		metrics.HTTPGetTotal.WithLabelValues(t)
		metrics.HTTPGetFailTotal.WithLabelValues(t)

		t := t
		env.Go(func(ctx context.Context) error {
			tick := time.NewTicker(w.interval)
			defer tick.Stop()
			for {
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-tick.C:
					req, err := http.NewRequest("GET", t, nil)
					if err != nil {
						log.Error("failed to create new request.", map[string]interface{}{
							"url":       t,
							log.FnError: err,
						})
						return err
					}

					req = req.WithContext(ctx)
					res, err := w.httpClient.Do(req)
					metrics.HTTPGetTotal.WithLabelValues(t).Inc()
					if err != nil {
						log.Info("GET failed.", map[string]interface{}{
							"url":       t,
							log.FnError: err,
						})
						metrics.HTTPGetFailTotal.WithLabelValues(t).Inc()
					} else {
						log.Info("GET succeeded.", map[string]interface{}{
							"url":  t,
							"code": res.StatusCode,
						})
						metrics.HTTPGetSuccessfulTotal.WithLabelValues(strconv.Itoa(res.StatusCode), t).Inc()
						res.Body.Close()
					}
					metrics.UpdateTime.Set(float64(time.Now().Unix()))
				}
			}
		})
	}
	env.Stop()
	return env.Wait()
}
