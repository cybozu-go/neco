package storage

import (
	"context"
	"testing"

	"github.com/coreos/etcd/clientv3/concurrency"

	"github.com/cybozu-go/neco/storage/test"
	"github.com/google/go-cmp/cmp"
)

func TestInfo(t *testing.T) {
	t.Parallel()

	etcd := test.NewEtcdClient(t)
	defer etcd.Close()

	ctx := context.Background()
	st := NewStorage(etcd)

	version, lrns, rev, err := st.GetInfo(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if rev == 0 {
		t.Error(`rev == 0`, rev)
	}
	if version != "" {
		t.Error("version should be empty")
	}
	if len(lrns) != 0 {
		t.Error("boot servers should be empty")
	}

	err = st.RegisterBootserver(ctx, 0)
	if err != nil {
		t.Fatal(err)
	}
	err = st.RegisterBootserver(ctx, 2)
	if err != nil {
		t.Fatal(err)
	}
	err = st.RegisterBootserver(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}

	version, lrns, rev, err = st.GetInfo(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if rev == 0 {
		t.Error(`rev == 0`, rev)
	}
	if !cmp.Equal(lrns, []int{0, 1, 2}) {
		t.Error("unexpected lrns", lrns)
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
	err = st.UpdateNecoRelease(ctx, "1.1.0", leaderKey)
	if err != nil {
		t.Fatal(err)
	}

	version, lrns, rev2, err := st.GetInfo(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if rev2 == 0 {
		t.Error(`rev2 == 0`, rev2)
	}
	if rev == rev2 {
		t.Error(`rev == rev2`, rev, rev2)
	}
	if !cmp.Equal(lrns, []int{0, 1, 2}) {
		t.Error("unexpected lrns", lrns)
	}
	if version != "1.1.0" {
		t.Error(`version != "1.1.0"`, version)
	}

	err = st.DeleteBootServer(ctx, 0)
	if err != nil {
		t.Fatal(err)
	}

	_, lrns, _, err = st.GetInfo(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !cmp.Equal(lrns, []int{1, 2}) {
		t.Error("unexpected lrns", lrns)
	}

	err = st.DeleteBootServer(ctx, 0)
	if err != ErrNotFound {
		t.Error("error should be ErrNotFound", err)
	}
}
