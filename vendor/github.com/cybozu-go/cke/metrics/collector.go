package metrics

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	v3 "github.com/coreos/etcd/clientv3"
	"github.com/cybozu-go/cke"
	"github.com/cybozu-go/log"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type logger struct{}

func (l logger) Println(v ...interface{}) {
	log.Error(fmt.Sprint(v...), nil)
}

const (
	scrapeTimeout = time.Second * 8
)

// collector is a metrics collector for CKE.
type collector struct {
	metrics map[string]metricGroup
	storage storage
}

// metricGroup represents collectors and availability of metric.
type metricGroup struct {
	collectors  []prometheus.Collector
	isAvailable func(context.Context, storage) (bool, error)
}

// storage is abstraction of cke.Storage.
// This abstraction is for mock test.
type storage interface {
	IsSabakanDisabled(context.Context) (bool, error)
}

// NewCollector returns a new prometheus.Collector.
func NewCollector(client *v3.Client) prometheus.Collector {
	return &collector{
		metrics: map[string]metricGroup{
			"leader": {
				collectors:  []prometheus.Collector{leader},
				isAvailable: alwaysAvailable,
			},
			"operation_phase": {
				collectors:  []prometheus.Collector{operationPhase, operationPhaseTimestampSeconds},
				isAvailable: isOperationPhaseAvailable,
			},
			"sabakan_integration": {
				collectors:  []prometheus.Collector{sabakanIntegrationSuccessful, sabakanIntegrationTimestampSeconds, sabakanWorkers, sabakanUnusedMachines},
				isAvailable: isSabakanIntegrationAvailable,
			},
		},
		storage: &cke.Storage{Client: client},
	}
}

// GetHandler returns http.Handler for prometheus metrics.
func GetHandler(collector prometheus.Collector) http.Handler {
	registry := prometheus.NewRegistry()
	registry.MustRegister(collector)

	handler := promhttp.HandlerFor(registry,
		promhttp.HandlerOpts{
			ErrorLog:      logger{},
			ErrorHandling: promhttp.ContinueOnError,
		})

	return handler
}

// Describe implements Collector.Describe().
func (c collector) Describe(ch chan<- *prometheus.Desc) {
	for _, metric := range c.metrics {
		for _, col := range metric.collectors {
			col.Describe(ch)
		}
	}
}

// Collect implements Collector.Collect().
func (c collector) Collect(ch chan<- prometheus.Metric) {
	ctx, cancel := context.WithTimeout(context.Background(), scrapeTimeout)
	defer cancel()

	var wg sync.WaitGroup
	for key, metric := range c.metrics {
		wg.Add(1)
		go func(key string, metric metricGroup) {
			defer wg.Done()
			available, err := metric.isAvailable(ctx, c.storage)
			if err != nil {
				log.Warn("unable to decide whether metrics are available", map[string]interface{}{
					"name":      key,
					log.FnError: err,
				})
				return
			}
			if !available {
				return
			}

			for _, col := range metric.collectors {
				col.Collect(ch)
			}
		}(key, metric)
	}
	wg.Wait()
}
