package necorebooter

import (
	"context"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/cybozu-go/neco"
)

func TestMetrics(t *testing.T) {
	c, err := newTestController()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		err := cleanupEtcd()
		if err != nil {
			t.Fatal(err)
		}
	}()
	err = c.necoStorage.EnableNecoRebooter(context.Background(), true)
	if err != nil {
		t.Fatal(err)
	}
	rebootListEntries := []*neco.RebootListEntry{
		{
			Node:       "node1",
			Group:      "group1",
			RebootTime: "test1",
			Status:     neco.RebootListEntryStatusPending,
		},
		{
			Node:       "node2",
			Group:      "group2",
			RebootTime: "test2",
			Status:     neco.RebootListEntryStatusQueued,
		},
		{
			Node:       "node3",
			Group:      "group3",
			RebootTime: "test3",
			Status:     neco.RebootListEntryStatusCancelled,
		},
	}
	for _, r := range rebootListEntries {
		err = c.necoStorage.RegisterRebootListEntry(context.Background(), r)
		if err != nil {
			t.Fatal(err)
		}
	}
	err = c.necoStorage.UpdateProcessingGroup(context.Background(), "group2")
	if err != nil {
		t.Fatal(err)
	}
	hostname, err := os.Hostname()
	if err != nil {
		t.Fatal(err)
	}
	collector := NewCollector(c.necoStorage, hostname)
	metricsHandler := GetMetricsHandler(collector)
	ts := httptest.NewServer(metricsHandler)
	req := httptest.NewRequest("GET", "/metrics", nil)
	rec := httptest.NewRecorder()
	ts.Config.Handler.ServeHTTP(rec, req)

	cases := []struct {
		name   string
		expect string
	}{
		{
			name:   "leader",
			expect: `neco_rebooter_leader 1`,
		},
		{
			name:   "rebootListItems",
			expect: `neco_rebooter_reboot_list_items 3`,
		},
		{
			name:   "isEnabled",
			expect: `neco_rebooter_enabled 1`,
		},
		{
			name:   "processingGroup",
			expect: `neco_rebooter_processing_group{group="group2"} 1`,
		},
		{
			name:   "test1",
			expect: `neco_rebooter_reboot_list_status{group="group1",node="node1",rebootTime="test1",status="Pending"} 1`,
		},
		{
			name:   "test2",
			expect: `neco_rebooter_reboot_list_status{group="group2",node="node2",rebootTime="test2",status="Queued"} 1`,
		},
		{
			name:   "test3",
			expect: `neco_rebooter_reboot_list_status{group="group3",node="node3",rebootTime="test3",status="Cancelled"} 1`,
		},
	}
	metrics := rec.Body.String()
	for _, tc := range cases {
		if !strings.Contains(metrics, tc.expect) {
			t.Errorf("expected %s, but got %s", tc.expect, metrics)
		}
		if strings.Count(metrics, tc.expect) != 1 {
			t.Errorf("number of %s is expected to 1 %s, but got %d", metrics, tc.expect, strings.Count(metrics, tc.expect))
		}
	}
}
