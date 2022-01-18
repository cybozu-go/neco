package setup

import (
	"context"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/neco"
	"github.com/cybozu-go/neco/progs/etcd"
	clientv3 "go.etcd.io/etcd/client/v3"
)

func enableEtcdAuth(ctx context.Context, ec *clientv3.Client) error {
	err := etcd.UserAdd(ctx, ec, "root", "")
	if err != nil {
		return err
	}
	err = etcd.UserAdd(ctx, ec, "backup", "")
	if err != nil {
		return err
	}
	err = etcd.UserAdd(ctx, ec, "vault", neco.VaultPrefix)
	if err != nil {
		return err
	}
	err = etcd.UserAdd(ctx, ec, "neco", "")
	if err != nil {
		return err
	}

	_, err = ec.AuthEnable(ctx)
	if err != nil {
		return err
	}

	log.Info("etcd: auth enabled", nil)
	return nil
}
