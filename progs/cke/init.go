package cke

import (
	"context"

	"github.com/coreos/etcd/clientv3"
	"github.com/cybozu-go/neco"
	"github.com/cybozu-go/neco/progs/etcd"
)

// Init initialize cke for cluster
func Init(ctx context.Context, ec *clientv3.Client) error {
	return etcd.UserAdd(ctx, ec, "cke", neco.CKEPrefix)
}
