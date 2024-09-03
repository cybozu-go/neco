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
	rebootListEntry := neco.RebootListEntry{
		Node:       "node1",
		Group:      "group1",
		RebootTime: "test1",
		Status:     neco.RebootListEntryStatusCancelled,
	}
	rebootQueueEntry := cke.RebootQueueEntry{
		Node:   "node1",
		Status: cke.RebootStatusQueued,
	}
	entrySet := []EntrySet{
		{
			rebootListEntry:  &rebootListEntry,
			rebootQueueEntry: &rebootQueueEntry,
		},
	}
	err = c.necoStorage.RegisterRebootListEntry(context.Background(), &rebootListEntry)
	if err != nil {
		t.Fatal(err)
	}
	err = c.ckeStorage.RegisterRebootsEntry(context.Background(), &rebootQueueEntry)
	if err != nil {
		t.Fatal(err)
	}
	err = c.removeCancelledEntry(context.Background(), entrySet)
	if err != nil {
		t.Fatal(err)
	}
	rlEntries, err := c.necoStorage.GetRebootListEntries(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(rlEntries) != 0 {
		t.Error("rebootListEntry not removed")
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

	rebootListEntry := neco.RebootListEntry{
		Node:       "node1",
		Group:      "group1",
		RebootTime: "test1",
		Status:     neco.RebootListEntryStatusQueued,
	}
	err = c.necoStorage.RegisterRebootListEntry(context.Background(), &rebootListEntry)
	if err != nil {
		t.Fatal(err)
	}

	rebootListEntries := []*neco.RebootListEntry{&rebootListEntry}
	err = c.removeCompletedEntry(context.Background(), rebootListEntries)
	if err != nil {
		t.Fatal(err)
	}
	rlEntries, err := c.necoStorage.GetRebootListEntries(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(rlEntries) != 0 {
		t.Error("removeCompletedEntry failed")
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

	rebootListEntry := neco.RebootListEntry{
		Node:       "node1",
		Group:      "group1",
		RebootTime: "test1",
		Status:     neco.RebootListEntryStatusQueued,
	}
	rebootQueueEntry := cke.RebootQueueEntry{
		Node:   "node1",
		Status: cke.RebootStatusQueued,
	}
	entrySet := []EntrySet{
		{
			rebootListEntry:  &rebootListEntry,
			rebootQueueEntry: &rebootQueueEntry,
		},
	}
	err = c.necoStorage.RegisterRebootListEntry(context.Background(), &rebootListEntry)
	if err != nil {
		t.Fatal(err)
	}
	err = c.ckeStorage.RegisterRebootsEntry(context.Background(), &rebootQueueEntry)
	if err != nil {
		t.Fatal(err)
	}

	err = c.dequeueAndCancelEntry(context.Background(), entrySet)
	if err != nil {
		t.Fatal(err)
	}
	rlEntries, err := c.necoStorage.GetRebootListEntries(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	rqEntries, err := c.ckeStorage.GetRebootsEntries(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(rlEntries) != 1 {
		t.Error("dequeueAndCancelEntry failed")
	}
	if rlEntries[0].Status != neco.RebootListEntryStatusPending {
		t.Error("dequeueAndCancelEntry failed")
	}
	if len(rqEntries) != 1 {
		t.Error("dequeueAndCancelEntry failed")
	}
	if rqEntries[0].Status != cke.RebootStatusCancelled {
		t.Error("dequeueAndCancelEntry failed")
	}
}

func TestRemoveOrphanedEntry(t *testing.T) {
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

	rebootListEntry := neco.RebootListEntry{
		Node:       "node1",
		Group:      "group1",
		RebootTime: "test1",
		Status:     neco.RebootListEntryStatusPending,
	}
	rebootQueueEntry := []*cke.RebootQueueEntry{
		{
			Node:   "node1",
			Status: cke.RebootStatusQueued,
		},
		{
			Node:   "node2",
			Status: cke.RebootStatusQueued,
		},
	}
	entrySet := []EntrySet{
		{
			rebootListEntry:  &rebootListEntry,
			rebootQueueEntry: rebootQueueEntry[0],
		},
		{
			rebootQueueEntry: rebootQueueEntry[1],
		},
	}

	err = c.necoStorage.RegisterRebootListEntry(context.Background(), &rebootListEntry)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range rebootQueueEntry {
		err = c.ckeStorage.RegisterRebootsEntry(context.Background(), entry)
		if err != nil {
			t.Fatal(err)
		}
	}

	err = c.RemoveOrphanedEntry(context.Background(), entrySet)
	if err != nil {
		t.Fatal(err)
	}
	rlEntries, err := c.necoStorage.GetRebootListEntries(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	rqEntries, err := c.ckeStorage.GetRebootsEntries(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(rlEntries) != 1 {
		t.Error("RemoveOrphanedEntry failed")
	}
	for _, entry := range rqEntries {
		if entry.Status != cke.RebootStatusCancelled {
			t.Error("RemoveOrphanedEntry failed")
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

	rebootListEntry := neco.RebootListEntry{
		Node:       "node1",
		Group:      "group1",
		RebootTime: "test1",
		Status:     neco.RebootListEntryStatusPending,
	}

	err = c.necoStorage.RegisterRebootListEntry(context.Background(), &rebootListEntry)
	if err != nil {
		t.Fatal(err)
	}
	rebootListEntries := []*neco.RebootListEntry{&rebootListEntry}
	err = c.addRebootListEntry(context.Background(), rebootListEntries)
	if err != nil {
		t.Fatal(err)
	}
	rlEntries, err := c.necoStorage.GetRebootListEntries(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	rqEntries, err := c.ckeStorage.GetRebootsEntries(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(rlEntries) != 1 {
		t.Error("addRebootListEntry failed")
	}
	if rlEntries[0].Status != neco.RebootListEntryStatusQueued {
		t.Error("addRebootListEntry failed")
	}
	if len(rqEntries) != 1 {
		t.Error("addRebootListEntry failed")
	}
	if rqEntries[0].Status != cke.RebootStatusQueued {
		t.Error("addRebootListEntry failed")
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
	candidate := []string{}
	allGroups := getAllGroups(rebootListEntries)
	for _, group := range allGroups {
		rebootableEntries := c.findRebootableNodeInGroup(rebootListEntries, group)
		if len(rebootableEntries) > 0 {
			candidate = append(candidate, group)
		}
	}
	next, err := c.moveToNextGroup(context.Background(), candidate, "")
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

func TestCollectEntries(t *testing.T) {
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
	timeNowFunc = func() time.Time {
		return time.Date(2024, 1, 2, 1, 0, 0, 0, time.UTC) // 2024-01-02 (Tuesday) 01:00:00
	}
	rebootListEntries := []*neco.RebootListEntry{
		{ //Cancelled node
			Node:       "node1",
			Group:      "group1",
			RebootTime: "test1",
			Status:     neco.RebootListEntryStatusCancelled,
		},
		{ //TimedOut node
			Node:       "node2",
			Group:      "group1",
			RebootTime: "test2",
			Status:     neco.RebootListEntryStatusQueued,
		},
		{ // Completed node
			Node:       "node3",
			Group:      "group1",
			RebootTime: "test1",
			Status:     neco.RebootListEntryStatusQueued,
		},
		{ // will be added node
			Node:       "node4",
			Group:      "group1",
			RebootTime: "test1",
			Status:     neco.RebootListEntryStatusPending,
		},
		{ // Queued node
			Node:       "node5",
			Group:      "group1",
			RebootTime: "test1",
			Status:     neco.RebootListEntryStatusQueued,
		},
		{ // orphaned node
			Node:       "node6",
			Group:      "group1",
			RebootTime: "test1",
			Status:     neco.RebootListEntryStatusPending,
		},
	}
	rebootQueueEntry := []*cke.RebootQueueEntry{
		{
			Node:   "node1",
			Status: cke.RebootStatusQueued,
		},
		{
			Node:   "node2",
			Status: cke.RebootStatusQueued,
		},
		{
			Node:   "node5",
			Status: cke.RebootStatusQueued,
		},
		{ // orphaned node
			Node:   "node6",
			Status: cke.RebootStatusQueued,
		},
		{ // orphaned node
			Node:   "node7",
			Status: cke.RebootStatusQueued,
		},
	}
	collection := c.collectEntries(rebootListEntries, rebootQueueEntry, "group1")
	if len(collection.CancelledEntry) != 1 {
		t.Error("number of CancelledEntry is not expected value, actual ", len(collection.CancelledEntry))
	}
	if collection.CancelledEntry[0].rebootListEntry.Node != "node1" {
		t.Error("CancelledEntry is not expected value")
	}
	if collection.CancelledEntry[0].rebootListEntry.Status != neco.RebootListEntryStatusCancelled {
		t.Error("CancelledEntry is not expected value")
	}
	if collection.CancelledEntry[0].rebootListEntry.Node != collection.CancelledEntry[0].rebootQueueEntry.Node {
		t.Error("CancelledEntry is not expected value")
	}

	if len(collection.TimedOutEntry) != 1 {
		t.Error("number of TimedOutEntry is not expected value, actual ", len(collection.TimedOutEntry))
	}
	if collection.TimedOutEntry[0].rebootListEntry.Node != "node2" {
		t.Error("TimedOutEntry is not expected value")
	}
	if collection.TimedOutEntry[0].rebootListEntry.Status != neco.RebootListEntryStatusQueued {
		t.Error("TimedOutEntry is not expected value")
	}
	if collection.TimedOutEntry[0].rebootListEntry.Node != collection.TimedOutEntry[0].rebootQueueEntry.Node {
		t.Error("TimedOutEntry is not expected value")
	}

	if len(collection.CompletedEntry) != 1 {
		t.Error("number of CompletedEntry is not expected value, actual ", len(collection.CompletedEntry))
	}
	if collection.CompletedEntry[0].Node != "node3" {
		t.Error("CompletedEntry is not expected value")
	}
	if collection.CompletedEntry[0].Status != neco.RebootListEntryStatusQueued {
		t.Error("CompletedEntry is not expected value")
	}

	if len(collection.NewEntry) != 1 {
		t.Error("number of NewEntry is not expected value, actual ", len(collection.NewEntry))
	}
	if collection.NewEntry[0].Node != "node4" {
		t.Error("NewEntry is not expected value")
	}
	if collection.NewEntry[0].Status != neco.RebootListEntryStatusPending {
		t.Error("NewEntry is not expected value")
	}
	if collection.NewEntry[0].Group != "group1" {
		t.Error("NewEntry is not expected value")
	}

	if len(collection.QueuedEntry) != 2 {
		t.Error("number of QueuedEntry is not expected value, actual ", len(collection.QueuedEntry))
	}
	for _, es := range collection.QueuedEntry {
		if es.rebootListEntry.Node != "node2" && es.rebootListEntry.Node != "node5" {
			t.Error("QueuedEntry is not expected value")
		}
		if es.rebootListEntry.Status != neco.RebootListEntryStatusQueued {
			t.Error("QueuedEntry is not expected value")
		}
		if es.rebootListEntry.Node != es.rebootQueueEntry.Node {
			t.Error("QueuedEntry is not expected value")
		}
	}

	if len(collection.OrphanedEntry) != 2 {
		t.Error("number of OrphanedEntry is not expected value, actual ", len(collection.OrphanedEntry))
	}
	for _, es := range collection.OrphanedEntry {
		if es.rebootQueueEntry.Node != "node6" && es.rebootQueueEntry.Node != "node7" {
			t.Error("OrphanedEntry is not expected value, actual ", es.rebootQueueEntry.Node)
		}
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
