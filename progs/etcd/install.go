package etcd

import (
	"context"
	"io"
	"os"
	"time"

	"github.com/coreos/etcd/clientv3"
	"github.com/cybozu-go/etcdutil"
	"github.com/cybozu-go/log"
	"github.com/cybozu-go/neco"
	version "github.com/hashicorp/go-version"
)

var leastClusterVersion = version.Must(version.NewVersion("3.1.0"))

// InstallTools install etcdctl under /usr/local/bin.
func InstallTools(ctx context.Context, rt neco.ContainerRuntime) error {
	img, err := neco.CurrentArtifacts.FindContainerImage("etcd")
	if err != nil {
		return err
	}
	return rt.Run(ctx, img,
		[]neco.Bind{{Name: "host", Source: "/usr/local/bin", Dest: "/host"}},
		[]string{"/usr/local/etcd/install-tools"})
}

func etcdClient() (*clientv3.Client, error) {
	cfg := etcdutil.NewConfig(neco.NecoPrefix)
	cfg.Endpoints = []string{"127.0.0.1:2379"}

	// during bootstrap, certificate is not yet available
	_, err := os.Stat(neco.NecoCertFile)
	switch {
	case os.IsNotExist(err):
		cfg.TLSCertFile = neco.VaultCertFile
		cfg.TLSKeyFile = neco.VaultKeyFile
	case err == nil:
		cfg.TLSCertFile = neco.NecoCertFile
		cfg.TLSKeyFile = neco.NecoKeyFile
	default:
		return nil, err
	}
	return etcdutil.NewClient(cfg)
}

// WaitEtcdForVault waits for etcd cluster to stabilize.
// It returns etcd client connected to etcd server running on localhost.
func WaitEtcdForVault(ctx context.Context) (*clientv3.Client, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(1 * time.Second):
		}

		client, err := tryWait(ctx)
		if err != nil {
			continue
		}

		return client, nil
	}
}

func tryWait(ctx context.Context) (*clientv3.Client, error) {
	client, err := etcdClient()
	if err != nil {
		return nil, err
	}
	defer func() {
		if client != nil {
			client.Close()
		}
	}()

	_, err = client.MemberList(ctx)
	if err != nil {
		return nil, err
	}

	// Vault requires cluster version >= 3.1.0, but etcd starts with
	// cluster version 3.0.0, then gradually updates the version.
	// See https://github.com/etcd-io/etcd/issues/10038
	//
	// To avoid vault failure, we need to wait.
	resp, err := client.Status(ctx, "127.0.0.1:2379")
	if err != nil {
		return nil, err
	}

	ver, err := version.NewVersion(resp.Version)
	if err != nil {
		return nil, err
	}

	if ver.LessThan(leastClusterVersion) {
		return nil, err
	}

	c := client
	client = nil
	return c, nil
}

// Setup installs and starts etcd.
// It returns etcd client connected to etcd server running on localhost.
func Setup(ctx context.Context, rt neco.ContainerRuntime, generator func(io.Writer) error) (*clientv3.Client, error) {
	f, err := os.OpenFile(neco.EtcdConfFile, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	err = generator(f)
	if err != nil {
		return nil, err
	}
	err = f.Sync()
	if err != nil {
		return nil, err
	}

	err = os.MkdirAll(neco.EtcdDataDir, 0700)
	if err != nil {
		return nil, err
	}
	err = os.Chown(neco.EtcdDataDir, neco.EtcdUID, neco.EtcdGID)
	if err != nil {
		return nil, err
	}

	g, err := os.OpenFile(neco.ServiceFile(neco.EtcdService), os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return nil, err
	}
	defer g.Close()

	err = GenerateService(g, rt)
	if err != nil {
		return nil, err
	}
	err = g.Sync()
	if err != nil {
		return nil, err
	}

	err = neco.StartService(ctx, neco.EtcdService)
	if err != nil {
		return nil, err
	}

	log.Info("etcd: waiting cluster...", nil)
	client, err := WaitEtcdForVault(ctx)
	if err != nil {
		return nil, err
	}

	log.Info("etcd: setup completed", nil)
	return client, nil
}
