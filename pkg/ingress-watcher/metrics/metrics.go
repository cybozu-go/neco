package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/model"
)

const (
	namespace = "ingresswatcher"
)

// WatchInterval returns the interval of watch.
var WatchInterval = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "watch_interval",
		Help:      "interval of watch.",
	},
	[]string{model.InstanceLabel},
)

// HTTPGetTotal returns the total successful count of http get.
var HTTPGetTotal = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "http_get_total",
		Help:      "The total count of http get.",
	},
	[]string{"path", model.InstanceLabel},
)

// HTTPGetSuccessfulTotal returns the total successful count of http get.
var HTTPGetSuccessfulTotal = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "http_get_successful_total",
		Help:      "The total successful count of http get.",
	},
	[]string{"code", "path", model.InstanceLabel},
)

// HTTPGetFailTotal returns the total fail count of http get.
var HTTPGetFailTotal = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "http_get_fail_total",
		Help:      "The total fail count of http get.",
	},
	[]string{"path", model.InstanceLabel},
)
