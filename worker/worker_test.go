package worker

import (
	"context"
	"testing"

	"github.com/cybozu-go/neco/storage/test"
)

func TestWorker(t *testing.T) {

	worker, err := NewWorker(context.Background(), test.NewEtcdClient(t), "1.0.0")
	if err != nil {
		t.Fatal(err)
	}

	worker.Run()
}
