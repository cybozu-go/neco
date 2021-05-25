package worker

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/cybozu-go/neco"
	"github.com/cybozu-go/neco/storage"
	"github.com/cybozu-go/neco/storage/test"
	"github.com/cybozu-go/well"
	"go.etcd.io/etcd/clientv3"
)

func TestEtcdCompaction(t *testing.T) {
	if os.Getenv("RUN_COMPACTION_TEST") == "" {
		t.Skip("RUN_COMPACTION_TEST is not set")
	}
	ec := test.NewEtcdClient(t)
	defer ec.Close()
	req := neco.UpdateRequest{Version: "1", Servers: []int{1}, Stop: true, StartedAt: time.Now()}
	reqStr, err := json.Marshal(req)
	if err != nil {
		t.Fatal(err)
	}

	_, err = ec.Put(context.Background(), storage.KeyCurrent, string(reqStr))
	if err != nil {
		t.Fatal(err)
	}

	presp := &clientv3.PutResponse{}
	for i := 0; i < 2; i++ {
		presp, err = ec.Put(context.Background(), "TestEtcdCompaction", string(reqStr))
		if err != nil {
			t.Fatal(err)
		}
	}
	_, err = ec.Compact(context.Background(), presp.Header.Revision)
	if err != nil {
		t.Fatal(err)
	}

	worker := NewWorker(ec, newMock(false, 0), "1.0.0", 0)

	env := well.NewEnvironment(context.Background())
	env.Go(func(ctx context.Context) error {
		ctx, cancel := context.WithTimeout(ctx, time.Second)
		defer cancel()
		return worker.Run(ctx)
	})
	env.Stop()
	err = env.Wait()
	if strings.Contains(err.Error(), "compacted") {
		t.Fatal(err)
	}
	if err != nil {
		t.Log(err)
	}
}
