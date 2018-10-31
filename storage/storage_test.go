package storage

import (
	"context"
	"testing"

	"github.com/coreos/etcd/clientv3/concurrency"
	"github.com/cybozu-go/neco"
	"github.com/google/go-cmp/cmp"
)

func testArtifactSet(t *testing.T) {
	t.Parallel()

	etcd := newEtcdClient(t)
	defer etcd.Close()
	ctx := context.Background()
	st := NewStorage(etcd)

	bs, err := st.GetBootservers(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(bs) != 0 {
		t.Error(`len(bs) != 0`, bs)
	}

	err = st.DumpArtifactSet(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}

	_, err = st.GetArtifactSet(ctx, 0)
	if err != ErrNotFound {
		t.Error("lrn 0 should not exist")
	}

	as, err := st.GetArtifactSet(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}

	if !cmp.Equal(*as, neco.CurrentArtifacts) {
		t.Error(`as != CurrentArtifacts, as=`, as)
	}

	bs, err = st.GetBootservers(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !cmp.Equal(bs, []int{1}) {
		t.Error(`!cmp.Equal(bs, []int{1})`, bs)
	}
}

func testRequest(t *testing.T) {
	t.Parallel()

	etcd := newEtcdClient(t)
	defer etcd.Close()
	ctx := context.Background()
	st := NewStorage(etcd)

	_, err := st.GetRequest(ctx)
	if err != ErrNotFound {
		t.Error("request should not exist")
	}

	sess, err := concurrency.NewSession(etcd)
	if err != nil {
		t.Fatal(err)
	}
	defer sess.Close()
	e := concurrency.NewElection(sess, KeyLeader)
	err = e.Campaign(ctx, "test")
	if err != nil {
		t.Fatal(err)
	}
	leaderKey := e.Key()

	req := neco.UpdateRequest{Version: "1.0.0", Servers: []int{0, 1}}
	err = st.PutRequest(ctx, req, leaderKey)
	if err != nil {
		t.Fatal(err)
	}

	req2, err := st.GetRequest(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !cmp.Equal(req, *req2) {
		t.Errorf("unexpected request. expected=%#v, actual=%#v", req, *req2)
	}

	err = e.Resign(ctx)
	if err != nil {
		t.Fatal(err)
	}

	err = st.PutRequest(ctx, req, leaderKey)
	if err != ErrNoLeader {
		t.Error("shold lost leadership")
	}
}

func testStatus(t *testing.T) {
	t.Parallel()

	etcd := newEtcdClient(t)
	defer etcd.Close()
	ctx := context.Background()
	st := NewStorage(etcd)

	_, err := st.GetStatus(ctx, 1)
	if err != ErrNotFound {
		t.Error("status should not exist")
	}

	status := neco.UpdateStatus{
		Version:  "1.0.0",
		Finished: true,
		Error:    true,
		Message:  "aaa",
	}
	err = st.PutStatus(ctx, 1, status)
	if err != nil {
		t.Fatal(err)
	}

	status2, err := st.GetStatus(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}

	if !cmp.Equal(status, *status2) {
		t.Errorf("unexpected status. expect=%#v, actual=%#v", status, *status2)
	}
}

func testClearStatus(t *testing.T) {
	t.Parallel()

	etcd := newEtcdClient(t)
	defer etcd.Close()
	ctx := context.Background()
	st := NewStorage(etcd)

	err := st.ClearStatus(ctx)
	if err != ErrNotFound {
		t.Error("unexpected error", err)
	}

	sess, err := concurrency.NewSession(etcd)
	if err != nil {
		t.Fatal(err)
	}
	defer sess.Close()
	e := concurrency.NewElection(sess, KeyLeader)
	err = e.Campaign(ctx, "test")
	if err != nil {
		t.Fatal(err)
	}
	leaderKey := e.Key()

	req := neco.UpdateRequest{Version: "1.0.0", Servers: []int{0, 1}}
	err = st.PutRequest(ctx, req, leaderKey)
	if err != nil {
		t.Fatal(err)
	}

	err = st.ClearStatus(ctx)
	if err != ErrNotStopped {
		t.Error("unexpected error", err)
	}

	req.Stop = true
	err = st.PutRequest(ctx, req, leaderKey)
	if err != nil {
		t.Fatal(err)
	}

	status := neco.UpdateStatus{
		Version:  "1.0.0",
		Finished: true,
		Error:    true,
		Message:  "aaa",
	}
	err = st.PutStatus(ctx, 1, status)
	if err != nil {
		t.Fatal(err)
	}

	err = st.ClearStatus(ctx)
	if err != nil {
		t.Error("ClearStatus should succeed", err)
	}

	_, err = st.GetStatus(ctx, 1)
	if err != ErrNotFound {
		t.Error("worker status should have been cleared", err)
	}
}

func testNotificationConfig(t *testing.T) {
	t.Parallel()

	etcd := newEtcdClient(t)
	defer etcd.Close()
	ctx := context.Background()
	st := NewStorage(etcd)

	_, err := st.GetNotificationConfig(ctx)
	if err != ErrNotFound {
		t.Error("notification config should not be found")
	}

	err = st.PutNotificationConfig(ctx, neco.NotificationConfig{Slack: "http://slack.com/aaa"})
	if err != nil {
		t.Fatal(err)
	}

	nc, err := st.GetNotificationConfig(ctx)
	if err != nil {
		t.Fatal(err)
	}

	if nc.Slack != "http://slack.com/aaa" {
		t.Error(`nc.Slack != "http://slack.com/aaa"`, nc.Slack)
	}
}

func testVault(t *testing.T) {
	t.Parallel()

	etcd := newEtcdClient(t)
	defer etcd.Close()
	ctx := context.Background()
	st := NewStorage(etcd)

	err := st.PutVaultUnsealKey(ctx, "key")
	if err != nil {
		t.Fatal(err)
	}

	resp, err := etcd.Get(ctx, KeyVaultUnsealKey)
	if err != nil {
		t.Fatal(err)
	}
	if resp.Count != 1 {
		t.Fatal(`resp.Count != 1`, resp.Count)
	}
	if string(resp.Kvs[0].Value) != "key" {
		t.Error("wrong vault unseal key")
	}

	err = st.PutVaultRootToken(ctx, "root")
	if err != nil {
		t.Fatal(err)
	}

	resp, err = etcd.Get(ctx, KeyVaultRootToken)
	if err != nil {
		t.Fatal(err)
	}
	if resp.Count != 1 {
		t.Fatal(`resp.Count != 1`, resp.Count)
	}
	if string(resp.Kvs[0].Value) != "root" {
		t.Error("wrong vault root token")
	}
}

func TestStorage(t *testing.T) {
	t.Run("ArtifactSet", testArtifactSet)
	t.Run("Request", testRequest)
	t.Run("Status", testStatus)
	t.Run("ClearStatus", testClearStatus)
	t.Run("NotificationConfig", testNotificationConfig)
	t.Run("Vault", testVault)
}
