package setup

import (
	"context"
	"os"
	"time"

	"github.com/coreos/etcd/clientv3"
	"github.com/cybozu-go/etcdutil"
	"github.com/cybozu-go/log"
	"github.com/cybozu-go/neco"
	"github.com/cybozu-go/neco/progs/etcd"
)

func etcdClient() (*clientv3.Client, error) {
	cfg := etcdutil.NewConfig("")
	cfg.Endpoints = []string{"127.0.0.1:2379"}
	cfg.TLSCertFile = neco.VaultCertFile
	cfg.TLSKeyFile = neco.VaultKeyFile
	return etcdutil.NewClient(cfg)
}

func setupEtcd(ctx context.Context, mylrn int, lrns []int) error {
	err := neco.RunContainer(ctx, "etcd",
		[]neco.Bind{{Name: "host", Source: "/usr/local/bin", Dest: "/host"}},
		[]string{"--user=0", "--group=0", "--exec=/usr/local/etcd/install-tools"})
	if err != nil {
		return err
	}

	f, err := os.OpenFile(neco.EtcdConfFile, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	err = etcd.GenerateConf(f, mylrn, lrns)
	if err != nil {
		return err
	}
	err = f.Sync()
	if err != nil {
		return err
	}

	err = os.MkdirAll(neco.EtcdDataDir, 0700)
	if err != nil {
		return err
	}
	err = os.Chown(neco.EtcdDataDir, neco.EtcdUID, neco.EtcdGID)
	if err != nil {
		return err
	}

	g, err := os.OpenFile(neco.ServiceFile(neco.EtcdService), os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer g.Close()

	err = etcd.GenerateService(g)
	if err != nil {
		return err
	}
	err = g.Sync()
	if err != nil {
		return err
	}

	err = neco.StartService(ctx, neco.EtcdService)
	if err != nil {
		return err
	}

	log.Info("etcd: waiting cluster...", nil)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(1 * time.Second):
		}

		client, err := etcdClient()
		if err != nil {
			continue
		}

		_, err = client.MemberList(ctx)
		client.Close()
		if err == nil {
			break
		}
	}

	log.Info("etcd: setup completed", nil)

	return nil
}
