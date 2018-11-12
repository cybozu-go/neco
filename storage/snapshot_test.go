package storage

import (
	"context"
	"testing"
	"time"

	"github.com/cybozu-go/neco"
	"github.com/cybozu-go/neco/storage/test"
	"github.com/google/go-cmp/cmp"
)

func TestSnapshot(t *testing.T) {
	t.Parallel()

	etcd := test.NewEtcdClient(t)
	defer etcd.Close()
	ctx := context.Background()
	st := NewStorage(etcd)

	snap, err := st.NewSnapshot(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if snap.Revision == 0 {
		t.Error(`snap.Revision == 0`, snap.Revision)
	}
	if snap.Request != nil {
		t.Error(`snap.Request != nil`, snap.Request)
	}
	if len(snap.Statuses) != 0 {
		t.Error(`len(snap.Statuses) != 0`, len(snap.Statuses))
	}
	if snap.Latest != "" {
		t.Error(`snap.Latest != ""`, snap.Latest)
	}
	if len(snap.Servers) != 0 {
		t.Error(`len(snap.Servers) != 0`)
	}

	leaderKey := "test/leader"
	_, err = st.etcd.Put(ctx, leaderKey, "aaa")
	if err != nil {
		t.Fatal(err)
	}

	req := neco.UpdateRequest{
		Version:   "1.1.0",
		Servers:   []int{0, 1},
		StartedAt: time.Now(),
	}
	err = st.PutRequest(ctx, req, leaderKey)
	if err != nil {
		t.Fatal(err)
	}

	status1 := neco.UpdateStatus{
		Version: "1.1.0",
		Step:    1,
		Cond:    neco.CondRunning,
	}
	status2 := neco.UpdateStatus{
		Version: "1.0.0",
		Step:    3,
		Cond:    neco.CondComplete,
	}
	err = st.PutStatus(ctx, 0, status1)
	if err != nil {
		t.Fatal(err)
	}
	err = st.PutStatus(ctx, 1, status2)
	if err != nil {
		t.Fatal(err)
	}

	err = st.RegisterBootserver(ctx, 0)
	if err != nil {
		t.Fatal(err)
	}
	err = st.RegisterBootserver(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}
	err = st.RegisterBootserver(ctx, 2)
	if err != nil {
		t.Fatal(err)
	}
	err = st.UpdateNecoRelease(ctx, "1.1.1", leaderKey)
	if err != nil {
		t.Fatal(err)
	}

	snap2, err := st.NewSnapshot(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if snap2.Revision == 0 {
		t.Error(`snap2.Revision == 0`, snap2.Revision)
	}
	if snap2.Revision == snap.Revision {
		t.Error(`snap2.Revision == snap.Revision`, snap2.Revision, ",", snap.Revision)
	}
	if snap2.Request == nil {
		t.Fatal(`snap2.Request == nil`)
	}
	if !cmp.Equal(*snap2.Request, req) {
		t.Error(`!cmp.Equal(*snap2.Request, req)`, *snap2.Request)
	}
	if len(snap2.Statuses) != 2 {
		t.Error(`len(snap2.Statuses) != 2`, len(snap2.Statuses))
	}
	if snap2.Latest != "1.1.1" {
		t.Error(`snap2.Latest != "1.1.1"`, snap2.Latest)
	}
	if len(snap2.Servers) != 3 {
		t.Error(`len(snap2.Servers) != 3`)
	}
}
