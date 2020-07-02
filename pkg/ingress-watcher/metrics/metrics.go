package metrics

import "github.com/prometheus/client_golang/prometheus"

const (
	namespace = "ingresswatcher"
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
