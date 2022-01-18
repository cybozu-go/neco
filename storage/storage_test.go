package storage

import (
	"context"
	"testing"

	"github.com/cybozu-go/neco"
	"github.com/cybozu-go/neco/storage/test"
	"github.com/google/go-cmp/cmp"
	"go.etcd.io/etcd/client/v3/concurrency"
)

func testContainerTag(t *testing.T) {
	t.Parallel()

	etcd := test.NewEtcdClient(t)
	defer etcd.Close()
	ctx := context.Background()
	st := NewStorage(etcd)

	_, err := st.GetContainerTag(ctx, 0, "etcd")
	if err != ErrNotFound {
		t.Error("unexpected error", err)
	}

	err = st.RecordContainerTag(ctx, 0, "etcd")
	if err != nil {
		t.Fatal(err)
	}

	tag, err := st.GetContainerTag(ctx, 0, "etcd")
	if err != nil {
		t.Fatal(err)
	}

	img, err := neco.CurrentArtifacts.FindContainerImage("etcd")
	if err != nil {
		t.Fatal(err)
	}
	if tag != img.Tag {
		t.Error("unexpected tag", tag)
	}
}

func testDebVersion(t *testing.T) {
	t.Parallel()

	etcd := test.NewEtcdClient(t)
	defer etcd.Close()
	ctx := context.Background()
	st := NewStorage(etcd)

	_, err := st.GetDebVersion(ctx, 1, "etcdpasswd")
	if err != ErrNotFound {
		t.Error("unexpected error", err)
	}

	err = st.RecordDebVersion(ctx, 1, "etcdpasswd")
	if err != nil {
		t.Fatal(err)
	}

	release, err := st.GetDebVersion(ctx, 1, "etcdpasswd")
	if err != nil {
		t.Fatal(err)
	}

	deb, err := neco.CurrentArtifacts.FindDebianPackage("etcdpasswd")
	if err != nil {
		t.Fatal(err)
	}
	if release != deb.Release {
		t.Error("unexpected version", release)
	}
}

func testRequest(t *testing.T) {
	t.Parallel()

	etcd := test.NewEtcdClient(t)
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
	e := concurrency.NewElection(sess, KeyUpdaterLeader)
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
		t.Error("should lost leadership")
	}
}

func testStatus(t *testing.T) {
	t.Parallel()

	etcd := test.NewEtcdClient(t)
	defer etcd.Close()
	ctx := context.Background()
	st := NewStorage(etcd)

	_, err := st.GetStatus(ctx, 1)
	if err != ErrNotFound {
		t.Error("status should not exist")
	}

	status := neco.UpdateStatus{
		Version: "1.0.0",
		Cond:    neco.CondComplete,
		Message: "aaa",
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

func testClearStatusAndContents(t *testing.T) {
	t.Parallel()

	etcd := test.NewEtcdClient(t)
	defer etcd.Close()
	ctx := context.Background()
	st := NewStorage(etcd)

	err := st.ClearStatusAndContents(ctx)
	if err != ErrNotFound {
		t.Error("unexpected error", err)
	}

	sess, err := concurrency.NewSession(etcd)
	if err != nil {
		t.Fatal(err)
	}
	defer sess.Close()
	e := concurrency.NewElection(sess, KeyUpdaterLeader)
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

	err = st.ClearStatusAndContents(ctx)
	if err != ErrNotStopped {
		t.Error("unexpected error", err)
	}

	req.Stop = true
	err = st.PutRequest(ctx, req, leaderKey)
	if err != nil {
		t.Fatal(err)
	}

	status := neco.UpdateStatus{
		Version: "1.0.0",
		Cond:    neco.CondComplete,
		Message: "aaa",
	}
	err = st.PutStatus(ctx, 1, status)
	if err != nil {
		t.Fatal(err)
	}

	reqContents := neco.ContentsUpdateStatus{Version: "1.0.0", Success: false}
	err = st.PutSabakanContentsStatus(ctx, &reqContents, leaderKey)
	if err != nil {
		t.Fatal(err)
	}

	err = st.ClearStatusAndContents(ctx)
	if err != nil {
		t.Error("ClearStatus should succeed", err)
	}

	_, err = st.GetStatus(ctx, 1)
	if err != ErrNotFound {
		t.Error("worker status should have been cleared", err)
	}

	_, err = st.GetSabakanContentsStatus(ctx)
	if err != ErrNotFound {
		t.Error("worker contents should have been cleared", err)
	}
}

func testFinish(t *testing.T) {
	t.Parallel()

	etcd := test.NewEtcdClient(t)
	defer etcd.Close()
	ctx := context.Background()
	st := NewStorage(etcd)

	lrns, err := st.GetFinished(ctx, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(lrns) != 0 {
		t.Error("lrns should be empty", lrns)
	}

	err = st.Finish(ctx, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	err = st.Finish(ctx, 1, 1)
	if err != nil {
		t.Fatal(err)
	}

	lrns, err = st.GetFinished(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}
	if !cmp.Equal(lrns, []int{1}) {
		t.Error("unexpected lrns", lrns)
	}

	err = st.Finish(ctx, 0, 1)
	if err != nil {
		t.Fatal(err)
	}

	lrns, err = st.GetFinished(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}
	if !cmp.Equal(lrns, []int{0, 1}) {
		t.Error("unexpected lrns", lrns)
	}
}

func TestStorage(t *testing.T) {
	t.Run("ContainerTag", testContainerTag)
	t.Run("DebVersion", testDebVersion)
	t.Run("Request", testRequest)
	t.Run("Status", testStatus)
	t.Run("ClearStatus", testClearStatusAndContents)
	t.Run("Finish", testFinish)
	t.Run("SabakanContentsStatus", testSabakanContentsStatus)
}
