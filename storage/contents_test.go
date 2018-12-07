package storage

import (
	"context"
	"testing"

	"github.com/coreos/etcd/clientv3/concurrency"
	"github.com/cybozu-go/neco"
	"github.com/cybozu-go/neco/storage/test"
	"github.com/google/go-cmp/cmp"
)

func testSabakanContentsStatus(t *testing.T) {
	t.Parallel()

	etcd := test.NewEtcdClient(t)
	defer etcd.Close()
	ctx := context.Background()
	st := NewStorage(etcd)

	_, err := st.GetSabakanContentsStatus(ctx)
	if err != ErrNotFound {
		t.Error("SabakanContentsStatus should not exist")
	}

	sess, err := concurrency.NewSession(etcd)
	if err != nil {
		t.Fatal(err)
	}
	defer sess.Close()
	e := concurrency.NewElection(sess, KeyWorkerLeader)
	err = e.Campaign(ctx, "test")
	if err != nil {
		t.Fatal(err)
	}
	leaderKey := e.Key()

	status := neco.ContentsUpdateStatus{
		Version: "1.0.0",
		Success: true,
	}
	err = st.PutSabakanContentsStatus(ctx, &status, leaderKey)
	if err != nil {
		t.Fatal(err)
	}

	status2, err := st.GetSabakanContentsStatus(ctx)
	if err != nil {
		t.Fatal(err)
	}

	if !cmp.Equal(status, *status2) {
		t.Errorf("unexpected status. expect=%#v, actual=%#v", status, *status2)
	}

	err = e.Resign(ctx)
	if err != nil {
		t.Fatal(err)
	}

	err = st.PutSabakanContentsStatus(ctx, &status, leaderKey)
	if err != ErrNoLeader {
		t.Error("should lost leadership")
	}
}

func TestContents(t *testing.T) {
	t.Run("Sabakan", testSabakanContentsStatus)
}
