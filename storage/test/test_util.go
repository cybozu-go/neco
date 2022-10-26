package test

import (
	"os"
	"testing"
	"time"

	"github.com/cybozu-go/etcdutil"
	clientv3 "go.etcd.io/etcd/client/v3"
)

var (
	now        = time.Now().String()
	clientPort = os.Getenv("CLIENT_PORT")
)

// NewEtcdClient creates new etcd client for test server:
//
//	etcd := test.NewEtcdClient(t)
func NewEtcdClient(t *testing.T) *clientv3.Client {
	var clientURL string
	if len(clientPort) == 0 {
		clientURL = "http://localhost:2379"
	} else {
		clientURL = "http://localhost:" + clientPort
	}

	cfg := etcdutil.NewConfig(now + "/" + t.Name() + "/")
	cfg.Endpoints = []string{clientURL}

	etcd, err := etcdutil.NewClient(cfg)
	if err != nil {
		t.Fatal(err)
	}
	return etcd
}
