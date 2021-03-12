package watch

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/neco/pkg/ingress-watcher/metrics"
	"github.com/cybozu-go/well"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

const timeoutDuration = 550 * time.Millisecond

const (
	watchIntervalName          = "ingresswatcher_watch_interval"
	httpGetTotalName           = "ingresswatcher_http_get_total"
	httpGetSuccessfulTotalName = "ingresswatcher_http_get_successful_total"
	httpGetFailTotalName       = "ingresswatcher_http_get_fail_total"
)

var namesWithPath = []string{httpGetTotalName, httpGetSuccessfulTotalName, httpGetFailTotalName}
var namesWithoutPath = []string{watchIntervalName}

func containsSliceString(s []string, str string) bool {
	for _, x := range s {
		if x == str {
			return true
		}
	}
	return false
}

func TestWatcherRun(t *testing.T) {
	type fields struct {
		targetURLs []string
		interval   time.Duration
		httpClient *http.Client
	}

	tests := []struct {
		name              string
		fields            fields
		resultWithPath    map[string]float64
		resultWithoutPath map[string]float64
	}{
		{
			name: "GET success every 100ms in 550ms",
			fields: fields{
				targetURLs: []string{"foo", "bar"},
				interval:   100 * time.Millisecond,
				httpClient: newTestClient(func(req *http.Request) (*http.Response, error) {
					return &http.Response{
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(bytes.NewBuffer([]byte(""))),
						Header:     make(http.Header),
					}, nil
				}),
			},
			resultWithPath: map[string]float64{
				httpGetTotalName:           5,
				httpGetSuccessfulTotalName: 5,
				httpGetFailTotalName:       0,
			},
			resultWithoutPath: map[string]float64{
				watchIntervalName: 0.1,
			},
		},

		{
			name: "GET fail every 100ms in 550ms",
			fields: fields{
				targetURLs: []string{"foo"},
				interval:   100 * time.Millisecond,
				httpClient: newTestClient(func(req *http.Request) (*http.Response, error) {
					return nil, errors.New("error")
				}),
			},
			resultWithPath: map[string]float64{
				httpGetTotalName:           5,
				httpGetSuccessfulTotalName: 0,
				httpGetFailTotalName:       5,
			},
			resultWithoutPath: map[string]float64{
				watchIntervalName: 0.1,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry := prometheus.NewRegistry()
			metrics.HTTPGetTotal.Reset()
			metrics.HTTPGetSuccessfulTotal.Reset()
			metrics.HTTPGetFailTotal.Reset()
			registry.MustRegister(
				metrics.WatchInterval,
				metrics.HTTPGetTotal,
				metrics.HTTPGetSuccessfulTotal,
				metrics.HTTPGetFailTotal,
			)

			// create watcher and run
			w := NewWatcher(
				tt.fields.targetURLs,
				tt.fields.interval,
				&well.HTTPClient{
					Client:   tt.fields.httpClient,
					Severity: log.LvDebug,
				},
			)
			env := well.NewEnvironment(context.Background())
			env.Go(func(ctx context.Context) error {
				ctx, cancel := context.WithTimeout(ctx, timeoutDuration)
				defer cancel()
				return w.Run(ctx)
			})
			env.Stop()
			env.Wait()

			// parse mertics family
			metricsFamily, err := registry.Gather()
			if err != nil {
				t.Fatal(err)
			}

			type metricKey struct {
				name string
				path string
			}
			mfMapWithPath := make(map[metricKey]*dto.Metric)
			mfMapWithoutPath := make(map[string]*dto.Metric)
			for _, mf := range metricsFamily {
				if containsSliceString(namesWithPath, *mf.Name) {
					if len(mf.Metric) != len(tt.fields.targetURLs) {
						t.Fatalf("%s: metric %s should contain only one element.", tt.name, *mf.Name)
					}
					for _, met := range mf.Metric {
						p := labelToMap(met.Label)["path"]
						mfMapWithPath[metricKey{*mf.Name, p}] = met
					}
				} else {
					mfMapWithoutPath[*mf.Name] = mf.Metric[0]
				}
			}

			// assert results
			for _, n := range namesWithPath {
				for _, ta := range w.targetAddrs {
					m, ok := mfMapWithPath[metricKey{n, ta}]
					if !ok && tt.resultWithPath[n] != 0 {
						t.Errorf(
							"%s: value for %s{path=%s} should be %f but not found.",
							tt.name,
							n,
							ta,
							tt.resultWithPath[n],
						)
						continue
					}
					if !ok && tt.resultWithPath[n] == 0 {
						continue
					}

					v, ok := tt.resultWithPath[n]
					if !ok {
						t.Fatalf("%s: value for %s{path=%s} not found", tt.name, n, ta)
					}
					if v != *m.Counter.Value {
						t.Errorf(
							"%s: value for %s{path=%s} is wrong.  expected: %f, actual: %f",
							tt.name,
							n,
							ta,
							v,
							*m.Counter.Value,
						)
					}
				}
			}

			for _, n := range namesWithoutPath {
				m, ok := mfMapWithoutPath[n]
				if !ok && tt.resultWithoutPath[n] != 0 {
					t.Errorf(
						"%s: value for %s should be %f but not found.",
						tt.name,
						n,
						tt.resultWithPath[n],
					)
					continue
				}
				if !ok && tt.resultWithoutPath[n] == 0 {
					continue
				}

				v, ok := tt.resultWithoutPath[n]
				if !ok {
					t.Fatalf("%s: value for %s not found", tt.name, n)
				}
				if v != *m.Gauge.Value {
					t.Errorf(
						"%s: value for %s is wrong.  expected: %f, actual: %f",
						tt.name,
						n,
						v,
						*m.Gauge.Value,
					)
				}
			}
		})
	}
}

type RoundTripFunc func(req *http.Request) (*http.Response, error)

func (f RoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func newTestClient(fn RoundTripFunc) *http.Client {
	return &http.Client{Transport: fn}
}

func labelToMap(labelPair []*dto.LabelPair) map[string]string {
	res := make(map[string]string)
	for _, l := range labelPair {
		res[*l.Name] = *l.Value
	}
	return res
}
