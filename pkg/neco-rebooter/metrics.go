package necorebooter

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/cybozu-go/neco/storage"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type logger struct{}

func (l logger) Println(v ...interface{}) {
	slog.Error(fmt.Sprint(v...))
}

const (
	scrapeTimeout = time.Second * 8
)

type collector struct {
	leader           *prometheus.Desc
	rebootListItems  *prometheus.Desc
	isEnabled        *prometheus.Desc
	processingGroup  *prometheus.Desc
	rebootListStatus *prometheus.Desc
	storage          storage.Storage
	hostname         string
}

func NewCollector(storage storage.Storage, hostname string) *collector {
	return &collector{
		leader: prometheus.NewDesc(
			"neco_rebooter_leader",
			"",
			nil,
			nil,
		),
		rebootListItems: prometheus.NewDesc(
			"neco_rebooter_reboot_list_items",
			"",
			nil,
			nil,
		),
		isEnabled: prometheus.NewDesc(
			"neco_rebooter_enabled",
			"",
			nil,
			nil,
		),
		processingGroup: prometheus.NewDesc(
			"neco_rebooter_processing_group",
			"",
			[]string{"group"},
			nil,
		),
		rebootListStatus: prometheus.NewDesc(
			"neco_rebooter_reboot_list_status",
			"",
			[]string{"node", "rebootTime", "group", "status"},
			nil,
		),
		storage:  storage,
		hostname: hostname,
	}
}

func GetMetricsHandler(collector prometheus.Collector) http.Handler {
	registry := prometheus.NewRegistry()
	registry.MustRegister(collector)
	handler := promhttp.HandlerFor(registry,
		promhttp.HandlerOpts{
			ErrorLog:      logger{},
			ErrorHandling: promhttp.ContinueOnError,
		})
	return handler
}

func (c *collector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.leader
	ch <- c.rebootListItems
	ch <- c.isEnabled
	ch <- c.processingGroup
	ch <- c.rebootListStatus
}

func (c *collector) Collect(ch chan<- prometheus.Metric) {
	ctx, cancel := context.WithTimeout(context.Background(), scrapeTimeout)
	defer cancel()
	c.updateLeader(ch, ctx)
	c.updateRebootListItems(ch, ctx)
	c.updateIsEnabled(ch, ctx)
	c.updateProcessingGroup(ch, ctx)
	c.updateRebootListStatus(ch, ctx)
}

func (c *collector) updateLeader(ch chan<- prometheus.Metric, ctx context.Context) {
	leaderHostName, err := c.storage.GetNecoRebooterLeader(ctx)
	if err != nil {
		slog.Error("failed to get leader hostname", "err", err)
		return
	}
	if leaderHostName == c.hostname {
		ch <- prometheus.MustNewConstMetric(c.leader, prometheus.GaugeValue, 1)
	} else {
		ch <- prometheus.MustNewConstMetric(c.leader, prometheus.GaugeValue, 0)
	}
}

func (c *collector) updateRebootListItems(ch chan<- prometheus.Metric, ctx context.Context) {
	entries, err := c.storage.GetRebootListEntries(ctx)
	if err != nil {
		slog.Error("failed to get reboot list", "err", err)
		return
	}
	ch <- prometheus.MustNewConstMetric(c.rebootListItems, prometheus.GaugeValue, float64(len(entries)))
}

func (c *collector) updateIsEnabled(ch chan<- prometheus.Metric, ctx context.Context) {
	enabled, err := c.storage.IsNecoRebooterEnabled(ctx)
	if err != nil {
		slog.Error("failed to get reboot list enabled", "err", err)
		return
	}
	if enabled {
		ch <- prometheus.MustNewConstMetric(c.isEnabled, prometheus.GaugeValue, 1)
	} else {
		ch <- prometheus.MustNewConstMetric(c.isEnabled, prometheus.GaugeValue, 0)
	}
}

func (c *collector) updateProcessingGroup(ch chan<- prometheus.Metric, ctx context.Context) {
	group, err := c.storage.GetProcessingGroup(ctx)
	if err != nil {
		slog.Error("failed to get processing group", "err", err)
		return
	}
	ch <- prometheus.MustNewConstMetric(c.processingGroup, prometheus.GaugeValue, 1, group)
}

func (c *collector) updateRebootListStatus(ch chan<- prometheus.Metric, ctx context.Context) {
	entries, err := c.storage.GetRebootListEntries(ctx)
	if err != nil {
		slog.Error("failed to get reboot list", "err", err)
		return
	}
	for _, entry := range entries {
		ch <- prometheus.MustNewConstMetric(c.rebootListStatus, prometheus.GaugeValue, 1, entry.Node, entry.RebootTime, entry.Group, entry.Status)
	}
}
