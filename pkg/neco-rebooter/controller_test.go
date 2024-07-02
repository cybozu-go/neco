package necorebooter

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/cybozu-go/cke"
	"github.com/cybozu-go/etcdutil"
	"github.com/cybozu-go/neco"
	"github.com/cybozu-go/neco/storage"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/client/v3/concurrency"
)

func newTestController() (*Controller, error) {
	etcdAddr := "http://localhost:2379"

	ckeConfig := cke.NewEtcdConfig()
	ckeConfig.Endpoints = []string{etcdAddr}
	etcd, err := etcdutil.NewClient(ckeConfig)
	if err != nil {
		return nil, err
	}
	cs := cke.Storage{Client: etcd}

	necoConfig := etcdutil.NewConfig(neco.NecoPrefix)
	necoConfig.Endpoints = []string{etcdAddr}
	etcd2, err := etcdutil.NewClient(necoConfig)
	if err != nil {
		return nil, err
	}
	ns := storage.NewStorage(etcd2)
	hostName, err := os.Hostname()
	if err != nil {
		return nil, err
	}
	c := Controller{
		ckeStorage:    cs,
		necoStorage:   ns,
		etcdClient:    *etcd2,
		electionValue: hostName,
		timeZone:      time.UTC,
	}

	s, err := concurrency.NewSession(&c.etcdClient, concurrency.WithTTL(600))
	if err != nil {
		return nil, err
	}
	e := concurrency.NewElection(s, storage.KeyNecoRebooterLeader)

	err = e.Campaign(context.TODO(), c.electionValue)
	if err != nil {
		return nil, err
	}
	c.leaderKey = e.Key()
	return &c, nil
}

func cleanupEtcd() error {
	cfg := etcdutil.NewConfig("")
	cfg.Endpoints = []string{"http://localhost:2379"}
	client, err := etcdutil.NewClient(cfg)
	if err != nil {
		return err
	}
	_, err = client.Delete(context.Background(), neco.CKEPrefix, clientv3.WithPrefix())
	if err != nil {
		return err
	}
	_, err = client.Delete(context.Background(), neco.NecoPrefix, clientv3.WithPrefix())
	if err != nil {
		return err
	}
	return nil
}

func TestRemoveCancelledEntry(t *testing.T) {
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
	rebootListEntries := []*neco.RebootListEntry{
		{
			Node:       "node1",
			Group:      "group1",
			RebootTime: "test1",
			Status:     neco.RebootListEntryStatusCancelled,
		},
		{
			Node:       "node2",
			Group:      "group1",
			RebootTime: "test1",
			Status:     neco.RebootListEntryStatusCancelled,
		},
		{
			Node:       "node3",
			Group:      "group1",
			RebootTime: "test1",
			Status:     neco.RebootListEntryStatusQueued,
		},
	}
	rebootQueueEntries := []*cke.RebootQueueEntry{
		{
			Node:   "node1",
			Status: cke.RebootStatusQueued,
		},
	}
	for _, r := range rebootListEntries {
		err = c.necoStorage.RegisterRebootListEntry(context.Background(), r)
		if err != nil {
			t.Fatal(err)
		}
	}
	for _, r := range rebootQueueEntries {
		err = c.ckeStorage.RegisterRebootsEntry(context.Background(), r)
		if err != nil {
			t.Fatal(err)
		}
	}
	rebootArgs := RebootArgs{
		rebootListEntries:  rebootListEntries,
		rebootQueueEntries: rebootQueueEntries,
		processingGroup:    "",
	}
	err = c.removeCancelledEntry(context.Background(), rebootArgs)
	if err != nil {
		t.Fatal(err)
	}
	rlEntries, err := c.necoStorage.GetRebootListEntries(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(rlEntries) != 1 {
		t.Error("rebootListEntry not removed")
	}
	if rlEntries[0].Node != "node3" || rlEntries[0].Status != neco.RebootListEntryStatusQueued {
		t.Error("removed entry is not expected")
	}
	rqEntries, err := c.ckeStorage.GetRebootsEntries(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if rqEntries[0].Status != cke.RebootStatusCancelled {
		t.Error("rebootQueueEntry not updated to cancelled")
	}
}

func TestRemoveCompletedEntry(t *testing.T) {
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

	rebootListEntries := []*neco.RebootListEntry{
		{
			Node:       "node1",
			Group:      "group1",
			RebootTime: "test1",
			Status:     neco.RebootListEntryStatusQueued,
		},
		{
			Node:       "node2",
			Group:      "group1",
			RebootTime: "test1",
			Status:     neco.RebootListEntryStatusQueued,
		},
		{
			Node:       "node3",
			Group:      "group2",
			RebootTime: "test1",
			Status:     neco.RebootListEntryStatusPending,
		},
	}
	rebootQueueEntries := []*cke.RebootQueueEntry{
		{
			Node:   "node1",
			Status: cke.RebootStatusQueued,
		},
	}
	expectedRebootListEntries := map[string]string{
		"node1": neco.RebootListEntryStatusQueued,
		"node3": neco.RebootListEntryStatusPending,
	}

	for _, r := range rebootListEntries {
		err = c.necoStorage.RegisterRebootListEntry(context.Background(), r)
		if err != nil {
			t.Fatal(err)
		}
	}
	for _, r := range rebootQueueEntries {
		err = c.ckeStorage.RegisterRebootsEntry(context.Background(), r)
		if err != nil {
			t.Fatal(err)
		}
	}
	rebootArgs := RebootArgs{
		rebootListEntries:  rebootListEntries,
		rebootQueueEntries: rebootQueueEntries,
		processingGroup:    "",
	}
	err = c.removeCompletedEntry(context.Background(), rebootArgs)
	if err != nil {
		t.Fatal(err)
	}
	rlEntries, err := c.necoStorage.GetRebootListEntries(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(rlEntries) != 2 {
		t.Error("removeCompletedEntry failed")
	}
	for _, r := range rlEntries {
		if r.Status != expectedRebootListEntries[r.Node] {
			t.Errorf("rebootListEntry %s is not expected, actual %s", r.Node, r.Status)
		}
	}
}

func TestDequeueTimedOutEntry(t *testing.T) {
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

	fileContent := `
rebootTimes:
  - name: test1
    labelSelector:
      matchLabels:
        cke.cybozu.com/role: test1
    times:
      allow:
        - "* 0-7 * * 1-5"
  - name: test2
    labelSelector:
      matchLabels:
        cke.cybozu.com/role: test2
    times:
      allow:
        - "* 0-23 * * 1-5"
groupLabelKey: topology.kubernetes.io/zone
`

	config, err := LoadConfig(strings.NewReader(fileContent))
	if err != nil {
		t.Fatal(err)
	}
	rt, err := config.GetRebootTime()
	if err != nil {
		t.Fatal(err)
	}
	if len(rt) != 2 {
		t.Error("number of rebootTimes is not expected, actual ", len(rt))
	}
	c.rebootTimes = rt

	rebootListEntries := []*neco.RebootListEntry{
		// node1 is not in rebootTimes
		{
			Node:       "node1",
			Group:      "group1",
			RebootTime: "test1",
			Status:     neco.RebootListEntryStatusQueued,
		},
		// node2 is not in rebootTimes but it's rebootQueueEntries does not exist
		{
			Node:       "node2",
			Group:      "group1",
			RebootTime: "test1",
			Status:     neco.RebootListEntryStatusQueued,
		},
		// node3 is in rebootTimes but it is not queued
		{
			Node:       "node3",
			Group:      "group1",
			RebootTime: "test1",
			Status:     neco.RebootListEntryStatusPending,
		},
		// node4 is in rebootTimes and it is queued
		{
			Node:       "node4",
			Group:      "group1",
			RebootTime: "test2",
			Status:     neco.RebootListEntryStatusQueued,
		},
	}
	rebootQueueEntries := []*cke.RebootQueueEntry{
		{
			Node:   "node1",
			Status: cke.RebootStatusQueued,
		},
		{
			Node:   "node3",
			Status: cke.RebootStatusQueued,
		},
		{
			Node:   "node4",
			Status: cke.RebootStatusQueued,
		},
	}
	expectedRebootListEntries := map[string]string{
		"node1": neco.RebootListEntryStatusPending,
		"node2": neco.RebootListEntryStatusQueued,
		"node3": neco.RebootListEntryStatusPending,
		"node4": neco.RebootListEntryStatusQueued,
	}
	expectedRebootQueueEntries := map[string]cke.RebootStatus{
		"node1": cke.RebootStatusCancelled,
		"node3": cke.RebootStatusQueued,
		"node4": cke.RebootStatusQueued,
	}

	for _, r := range rebootListEntries {
		err = c.necoStorage.RegisterRebootListEntry(context.Background(), r)
		if err != nil {
			t.Fatal(err)
		}
	}
	for _, r := range rebootQueueEntries {
		err = c.ckeStorage.RegisterRebootsEntry(context.Background(), r)
		if err != nil {
			t.Fatal(err)
		}
	}
	rebootArgs := RebootArgs{
		rebootListEntries:  rebootListEntries,
		rebootQueueEntries: rebootQueueEntries,
		processingGroup:    "",
	}
	timeNowFunc = func() time.Time {
		return time.Date(2024, 1, 2, 9, 0, 0, 0, time.UTC) // 2024-01-02 (Tuesday) 09:00:00
	}
	err = c.dequeueTimedOutEntry(context.Background(), rebootArgs)
	if err != nil {
		t.Fatal(err)
	}

	rlEntries, err := c.necoStorage.GetRebootListEntries(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	for _, r := range rlEntries {
		if r.Status != expectedRebootListEntries[r.Node] {
			t.Errorf("rebootListEntry %s is not expected value, actual %s", r.Node, r.Status)
		}
	}
	rqEntries, err := c.ckeStorage.GetRebootsEntries(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range rqEntries {
		if r.Status != expectedRebootQueueEntries[r.Node] {
			t.Errorf("rebootQueueEntry %s is not expected value, actual %s", r.Node, r.Status)
		}
	}
}

func TestDequeueAndCancelEntry(t *testing.T) {
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

	rebootListEntries := []*neco.RebootListEntry{
		{
			Node:       "node1",
			Group:      "group1",
			RebootTime: "test1",
			Status:     neco.RebootListEntryStatusQueued,
		},
		{
			Node:       "node2",
			Group:      "group1",
			RebootTime: "test1",
			Status:     neco.RebootListEntryStatusPending,
		},
		{
			Node:       "node3",
			Group:      "group1",
			RebootTime: "test1",
			Status:     neco.RebootListEntryStatusQueued,
		},
	}
	rebootQueueEntries := []*cke.RebootQueueEntry{
		{
			Node:   "node1",
			Status: cke.RebootStatusQueued,
		},
	}
	expectedRebootListEntries := map[string]string{
		"node1": neco.RebootListEntryStatusPending,
		"node2": neco.RebootListEntryStatusPending,
		"node3": neco.RebootListEntryStatusQueued,
	}
	expectedRebootQueueEntries := map[string]cke.RebootStatus{
		"node1": cke.RebootStatusCancelled,
	}
	for _, r := range rebootListEntries {
		err = c.necoStorage.RegisterRebootListEntry(context.Background(), r)
		if err != nil {
			t.Fatal(err)
		}
	}
	for _, r := range rebootQueueEntries {
		err = c.ckeStorage.RegisterRebootsEntry(context.Background(), r)
		if err != nil {
			t.Fatal(err)
		}
	}
	rebootArgs := RebootArgs{
		rebootListEntries:  rebootListEntries,
		rebootQueueEntries: rebootQueueEntries,
		processingGroup:    "",
	}
	err = c.dequeueAndCancelEntry(context.Background(), rebootArgs)
	if err != nil {
		t.Fatal(err)
	}

	rlEntries, err := c.necoStorage.GetRebootListEntries(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	for _, r := range rlEntries {
		if r.Status != expectedRebootListEntries[r.Node] {
			t.Errorf("rebootListEntry %s is not expected value, actual %s", r.Node, r.Status)
		}
	}
	rqEntries, err := c.ckeStorage.GetRebootsEntries(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range rqEntries {
		if r.Status != expectedRebootQueueEntries[r.Node] {
			t.Errorf("rebootQueueEntry %s is not expected value, actual %s", r.Node, r.Status)
		}
	}
}

func TestAddRebootListEntry(t *testing.T) {
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

	fileContent := `
rebootTimes:
  - name: test1
    labelSelector:
      matchLabels:
        cke.cybozu.com/role: test1
    times:
      allow:
        - "* 0-23 * * 1-5"
  - name: test2
    labelSelector:
      matchLabels:
        cke.cybozu.com/role: test2
    times:
      deny:
        - "* 0-23 * * 1-5"
groupLabelKey: topology.kubernetes.io/zone
`

	config, err := LoadConfig(strings.NewReader(fileContent))
	if err != nil {
		t.Fatal(err)
	}
	rt, err := config.GetRebootTime()
	if err != nil {
		t.Fatal(err)
	}
	if len(rt) != 2 {
		t.Error("number of rebootTimes is not expected, actual ", len(rt))
	}
	c.rebootTimes = rt

	rebootListEntries := []*neco.RebootListEntry{
		{
			Node:       "node1",
			Group:      "group1",
			RebootTime: "test1",
			Status:     neco.RebootListEntryStatusPending,
		},
		{
			Node:       "node2",
			Group:      "group1",
			RebootTime: "test1",
			Status:     neco.RebootListEntryStatusPending,
		},
		{
			Node:       "node3",
			Group:      "group1",
			RebootTime: "test1",
			Status:     neco.RebootListEntryStatusQueued,
		},
		{
			Node:       "node4",
			Group:      "group2",
			RebootTime: "test1",
			Status:     neco.RebootListEntryStatusPending,
		},
		{
			Node:       "node5",
			Group:      "group1",
			RebootTime: "test2",
			Status:     neco.RebootListEntryStatusPending,
		},
	}
	rebootQueueEntries := []*cke.RebootQueueEntry{
		{
			Node:   "node2",
			Status: cke.RebootStatusRebooting,
		},
	}
	expectedRebootListEntries := map[string]string{
		"node1": neco.RebootListEntryStatusQueued,
		"node2": neco.RebootListEntryStatusQueued,
		"node3": neco.RebootListEntryStatusQueued,
		"node4": neco.RebootListEntryStatusPending,
		"node5": neco.RebootListEntryStatusPending,
	}
	expectedRebootQueueEntries := map[string]cke.RebootStatus{
		"node1": cke.RebootStatusQueued,
		"node2": cke.RebootStatusRebooting,
	}

	for _, r := range rebootListEntries {
		err = c.necoStorage.RegisterRebootListEntry(context.Background(), r)
		if err != nil {
			t.Fatal(err)
		}
	}
	for _, r := range rebootQueueEntries {
		err = c.ckeStorage.RegisterRebootsEntry(context.Background(), r)
		if err != nil {
			t.Fatal(err)
		}
	}
	rebootArgs := RebootArgs{
		rebootListEntries:  rebootListEntries,
		rebootQueueEntries: rebootQueueEntries,
		processingGroup:    "group1",
	}
	timeNowFunc = func() time.Time {
		return time.Date(2024, 1, 2, 1, 0, 0, 0, time.UTC) // 2024-01-02 (Tuesday) 01:00:00
	}

	err = c.addRebootListEntry(context.Background(), rebootArgs)
	if err != nil {
		t.Fatal(err)
	}

	rlEntries, err := c.necoStorage.GetRebootListEntries(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	for _, r := range rlEntries {
		if r.Status != expectedRebootListEntries[r.Node] {
			t.Errorf("rebootListEntry %s is not expected value, actual %s", r.Node, r.Status)
		}
	}
	rqEntries, err := c.ckeStorage.GetRebootsEntries(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(rqEntries) != 2 {
		t.Errorf("number of rebootQueueEntries is not expected, actual %d", len(rqEntries))
	}
	for _, r := range rqEntries {
		if r.Status != expectedRebootQueueEntries[r.Node] {
			t.Errorf("rebootQueueEntry %s is not expected value, actual %s", r.Node, r.Status)
		}
	}
}

func TestMoveToNextGroup(t *testing.T) {
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

	fileContent := `
rebootTimes:
  - name: test1
    labelSelector:
      matchLabels:
        cke.cybozu.com/role: test1
    times:
      allow:
        - "* 0-23 * * 1-5"
  - name: test2
    labelSelector:
      matchLabels:
        cke.cybozu.com/role: test2
    times:
      deny:
        - "* 0-23 * * 1-5"
groupLabelKey: topology.kubernetes.io/zone
`

	config, err := LoadConfig(strings.NewReader(fileContent))
	if err != nil {
		t.Fatal(err)
	}
	rt, err := config.GetRebootTime()
	if err != nil {
		t.Fatal(err)
	}
	if len(rt) != 2 {
		t.Error("number of rebootTimes is not expected, actual ", len(rt))
	}
	c.rebootTimes = rt

	rebootListEntries := []*neco.RebootListEntry{
		{
			Node:       "node1",
			Group:      "group1",
			RebootTime: "test1",
			Status:     neco.RebootListEntryStatusPending,
		},
		{
			Node:       "node2",
			Group:      "group1",
			RebootTime: "test2",
			Status:     neco.RebootListEntryStatusPending,
		},
		// group2 has no rebootable nodes
		{
			Node:       "node3",
			Group:      "group2",
			RebootTime: "test2",
			Status:     neco.RebootListEntryStatusPending,
		},
		// group3 has no rebootable nodes
		{
			Node:       "node3",
			Group:      "group3",
			RebootTime: "test2",
			Status:     neco.RebootListEntryStatusPending,
		},
	}
	rebootQueueEntries := []*cke.RebootQueueEntry{}
	rebootArgs := RebootArgs{
		rebootListEntries:  rebootListEntries,
		rebootQueueEntries: rebootQueueEntries,
		processingGroup:    "group3",
	}
	next, err := c.moveToNextGroup(context.Background(), rebootArgs)
	if err != nil {
		t.Fatal(err)
	}
	if next != "group1" {
		t.Errorf("next group is not expected value, actual %s", next)
	}
	nextEtcd, err := c.necoStorage.GetProcessingGroup(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if nextEtcd != next {
		t.Errorf("processing group is not expected value, actual %s, expected %s", nextEtcd, next)
	}
}

func TestIsRebootable(t *testing.T) {
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

	fileContent := `
rebootTimes:
  - name: test1
    labelSelector:
      matchLabels:
        cke.cybozu.com/role: test1
    times:
      allow:
        - "* 0-23 * * 1-5"
  - name: test2
    labelSelector:
      matchLabels:
        cke.cybozu.com/role: test2
    times:
      deny:
        - "* 0-23 * * 1-5"
groupLabelKey: topology.kubernetes.io/zone
`

	config, err := LoadConfig(strings.NewReader(fileContent))
	if err != nil {
		t.Fatal(err)
	}
	rt, err := config.GetRebootTime()
	if err != nil {
		t.Fatal(err)
	}
	if len(rt) != 2 {
		t.Error("number of rebootTimes is not expected, actual ", len(rt))
	}
	c.rebootTimes = rt
	rebootListEntries := []*neco.RebootListEntry{
		{
			Node:       "node1",
			Group:      "group1",
			RebootTime: "test1",
			Status:     neco.RebootListEntryStatusPending,
		},
		{
			Node:       "node2",
			Group:      "group1",
			RebootTime: "test2",
			Status:     neco.RebootListEntryStatusPending,
		},
		//test3 is not in rebootTimes
		{
			Node:       "node2",
			Group:      "group1",
			RebootTime: "test3",
			Status:     neco.RebootListEntryStatusPending,
		},
	}
	expectedResult := map[string]bool{
		"node1": true,
		"node2": false,
		"node3": false,
	}
	timeNowFunc = func() time.Time {
		return time.Date(2024, 1, 2, 1, 0, 0, 0, time.UTC) // 2024-01-02 (Tuesday) 01:00:00
	}
	for _, r := range rebootListEntries {
		result := c.isRebootable(r)
		if result != expectedResult[r.Node] {
			t.Errorf("node %s is not expected value, actual %t", r.Node, result)
		}
	}
}

func TestFindRebootableNodeInGroup(t *testing.T) {
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

	fileContent := `
rebootTimes:
  - name: test1
    labelSelector:
      matchLabels:
        cke.cybozu.com/role: test1
    times:
      allow:
        - "* 0-23 * * 1-5"
  - name: test2
    labelSelector:
      matchLabels:
        cke.cybozu.com/role: test2
    times:
      deny:
        - "* 0-23 * * 1-5"
groupLabelKey: topology.kubernetes.io/zone
`

	config, err := LoadConfig(strings.NewReader(fileContent))
	if err != nil {
		t.Fatal(err)
	}
	rt, err := config.GetRebootTime()
	if err != nil {
		t.Fatal(err)
	}
	if len(rt) != 2 {
		t.Error("number of rebootTimes is not expected, actual ", len(rt))
	}
	c.rebootTimes = rt
	rebootListEntries := []*neco.RebootListEntry{
		{
			Node:       "node1",
			Group:      "group1",
			RebootTime: "test1",
			Status:     neco.RebootListEntryStatusPending,
		},
		{
			Node:       "node2",
			Group:      "group1",
			RebootTime: "test2",
			Status:     neco.RebootListEntryStatusPending,
		},
	}
	timeNowFunc = func() time.Time {
		return time.Date(2024, 1, 2, 1, 0, 0, 0, time.UTC) // 2024-01-02 (Tuesday) 01:00:00
	}
	node := c.findRebootableNodeInGroup(rebootListEntries, "group1")
	if len(node) != 1 {
		t.Errorf("number of rebootable node is not expected, actual %d", len(node))
	}
	if node[0].Node != "node1" {
		t.Errorf("result is not expected, actual %s", node[0].Node)
	}
}
