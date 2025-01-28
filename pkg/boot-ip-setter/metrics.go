package main

import (
	"log/slog"
	"sync/atomic"

	"github.com/prometheus/client_golang/prometheus"
)

type metricsCollector struct {
	logger       *slog.Logger
	hostname     string
	netif        NetworkInterface
	errorCounter *atomic.Int32
	metrics      *metrics
}

type metrics struct {
	hostnameGaugeVec *prometheus.GaugeVec
	addressGaugeVec  *prometheus.GaugeVec
	errorCounter     prometheus.Counter
}

func newCollector(logger *slog.Logger, hostname string, netif NetworkInterface, errorCounter *atomic.Int32) prometheus.Collector {
	c := &metricsCollector{
		logger:       logger,
		hostname:     hostname,
		netif:        netif,
		errorCounter: errorCounter,
	}
	c.metrics = &metrics{
		hostnameGaugeVec: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: "boot_ip_setter",
				Name:      "hostname",
				Help:      "The hostname this program runs on.",
			},
			[]string{"hostname"},
		),
		addressGaugeVec: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: "boot_ip_setter",
				Name:      "interface_address",
				Help:      "The IP address set to the target interface.",
			},
			[]string{"interface", "ipv4"},
		),
		errorCounter: prometheus.NewCounter(
			prometheus.CounterOpts{
				Namespace: "boot_ip_setter",
				Name:      "interface_operation_errors_total",
				Help:      "The number of times the interface operation failed.",
			},
		),
	}
	return c
}

func (c *metricsCollector) Describe(ch chan<- *prometheus.Desc) {
	c.metrics.hostnameGaugeVec.Describe(ch)
	c.metrics.addressGaugeVec.Describe(ch)
	c.metrics.errorCounter.Describe(ch)
}

func (c *metricsCollector) Collect(ch chan<- prometheus.Metric) {
	c.metrics.hostnameGaugeVec.Reset()
	c.metrics.hostnameGaugeVec.WithLabelValues(c.hostname).Set(1)
	c.metrics.hostnameGaugeVec.Collect(ch)

	addrs, err := c.netif.ListAddrs()
	if err != nil {
		c.errorCounter.Add(1)
		c.logger.Error("failed to list addresses", "error", err)
	}
	c.metrics.addressGaugeVec.Reset()
	for _, a := range addrs {
		c.metrics.addressGaugeVec.WithLabelValues(c.netif.Name(), a).Set(1)
	}
	c.metrics.addressGaugeVec.Collect(ch)

	delta := c.errorCounter.Swap(0)
	c.metrics.errorCounter.Add(float64(delta))
	c.metrics.errorCounter.Collect(ch)
}
