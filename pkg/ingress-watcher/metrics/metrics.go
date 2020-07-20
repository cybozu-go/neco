package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

const (
	namespace = "ingresswatcher"
)

// WatchInterval returns the interval of watch.
var WatchInterval = prometheus.NewGauge(
	prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "watch_interval",
		Help:      "interval of watch.",
	},
)

// UpdateTime returns the last update time of watched result.
var UpdateTime = prometheus.NewGauge(
	prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "update_time",
		Help:      "last update time of watched result.",
	},
)

// HTTPGetTotal returns the total successful count of http get.
var HTTPGetTotal = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "http_get_total",
		Help:      "The total count of http get.",
	},
	[]string{"path"},
)

// HTTPGetSuccessfulTotal returns the total successful count of http get.
var HTTPGetSuccessfulTotal = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "http_get_successful_total",
		Help:      "The total successful count of http get.",
	},
	[]string{"code", "path"},
)

// HTTPGetFailTotal returns the total fail count of http get.
var HTTPGetFailTotal = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "http_get_fail_total",
		Help:      "The total fail count of http get.",
	},
	[]string{"path"},
)
