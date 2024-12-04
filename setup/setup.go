package setup

import (
	"context"
	"io"
	"sort"
	"time"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/neco"
	"github.com/cybozu-go/neco/progs/etcd"
	"github.com/cybozu-go/neco/progs/vault"
	"github.com/cybozu-go/neco/storage"
	"github.com/cybozu-go/well"
	"github.com/hashicorp/vault/api"
)

const (
	stageBeforeRestart = iota
	stageAfterRestart
)

// Setup installs and configures etcd and vault cluster.
func Setup(ctx context.Context, lrns []int, revoke bool, proxy, ghToken string) error {
	rt, err := neco.GetContainerRuntime(proxy)
	if err != nil {
		return err
	}

	etcdImage, err := neco.CurrentArtifacts.FindContainerImage("etcd")
	if err != nil {
		return err
	}
	if err := rt.Pull(ctx, etcdImage); err != nil {
		return err
	}
	err = etcd.InstallTools(ctx, rt)
	if err != nil {
		return err
	}

	vaultImage, err := neco.CurrentArtifacts.FindContainerImage("vault")
	if err != nil {
		return err
	}
	if err := rt.Pull(ctx, vaultImage); err != nil {
		return err
	}
	err = vault.InstallTools(ctx, rt)
	if err != nil {
		return err
	}

	sort.Ints(lrns)

	mylrn, err := neco.MyLRN()
	if err != nil {
		return err
	}

	isLeader := mylrn == lrns[0]

	pems, err := prepareCA(ctx, isLeader, mylrn, lrns)
	if err != nil {
		return err
	}

	ec, err := etcd.Setup(ctx, rt, func(w io.Writer) error {
		return etcd.GenerateConf(w, mylrn, lrns)
	})
	if err != nil {
		return err
	}
	defer ec.Close()

	var vc *api.Client

	if isLeader {
		err = setupVault(ctx, rt, mylrn, lrns)
		if err != nil {
			return err
		}
		vc, err = bootVault(ctx, pems, ec)
		if err != nil {
			return err
		}
	} else {
		unsealKey, err := waitVault(ctx, ec)
		if err != nil {
			return err
		}
		err = setupVault(ctx, rt, mylrn, lrns)
		if err != nil {
			return err
		}
		cfg := api.DefaultConfig()
		vc, err = api.NewClient(cfg)
		if err != nil {
			return err
		}
		err = vault.Unseal(vc, unsealKey)
		if err != nil {
			return err
		}
	}

	st := storage.NewStorage(ec)
	rootToken, err := st.GetVaultRootToken(ctx)
	if err != nil {
		return err
	}
	vc.SetToken(rootToken)

	err = reissueCerts(ctx, vc, mylrn)
	if err != nil {
		return err
	}

	err = setupEtcdBackup(ctx, vc)
	if err != nil {
		return err
	}

	err = setupNecoFiles(ctx, vc, lrns)
	if err != nil {
		return err
	}

	err = st.RegisterBootserver(ctx, mylrn)
	if err != nil {
		return err
	}
	err = st.RecordContainerTag(ctx, mylrn, "etcd")
	if err != nil {
		return err
	}
	err = st.RecordContainerTag(ctx, mylrn, "vault")
	if err != nil {
		return err
	}

	err = st.Finish(ctx, mylrn, stageBeforeRestart)
	if err != nil {
		return err
	}

	for {
		log.Info("waiting for all servers to finish", nil)
		finished, err := st.GetFinished(ctx, stageBeforeRestart)
		if err != nil {
			return err
		}
		if len(finished) == len(lrns) {
			break
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(500 * time.Millisecond):
		}
	}

	log.Info("restart etcd and Vault", nil)

	// this sleep is a must to avoid failure in the above st.GetFinished
	time.Sleep(1 * time.Second)

	err = neco.RestartService(ctx, "etcd-container")
	if err != nil {
		return err
	}

	ec.Close()
	ec, err = etcd.WaitEtcdForVault(ctx)
	if err != nil {
		return err
	}

	err = neco.RestartService(ctx, "vault")
	if err != nil {
		return err
	}
	err = neco.WaitVaultLeader(ctx, vc)
	if err != nil {
		return err
	}

	st = storage.NewStorage(ec)
	err = st.Finish(ctx, mylrn, stageAfterRestart)
	if err != nil {
		return err
	}

	nRetries := 0
	for {
		log.Info("waiting for all servers to restart vault", nil)
		finished, err := st.GetFinished(ctx, stageAfterRestart)
		if err != nil {
			if nRetries < 10 {
				log.Warn("checking finish status failed", map[string]interface{}{
					log.FnError: err,
				})
				time.Sleep(10 * time.Second)
				nRetries++
				continue
			}
			return err
		}
		if len(finished) == len(lrns) {
			break
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(500 * time.Millisecond):
		}
	}

	if isLeader {
		ver, err := neco.GetDebianVersion(neco.NecoPackageName)
		if err != nil {
			return err
		}

		err = st.UpdateNecoRelease(ctx, ver, storage.KeyVaultUnsealKey)
		if err != nil {
			return err
		}

		if len(proxy) > 0 {
			err = st.PutProxyConfig(ctx, proxy)
			if err != nil {
				return err
			}
		}

		req := neco.UpdateRequest{
			Version:   ver,
			Servers:   lrns,
			StartedAt: time.Now(),
		}
		err = st.PutRequest(ctx, req, storage.KeyVaultUnsealKey)
		if err != nil {
			return err
		}
		if revoke {
			err = revokeRootToken(ctx, vc, ec)
			if err != nil {
				return err
			}
		}
		err = enableEtcdAuth(ctx, ec)
		if err != nil {
			return err
		}

		if ghToken != "" {
			if err := st.PutGitHubToken(ctx, ghToken); err != nil {
				return err
			}
		}
	}

	well.CommandContext(ctx, "sync").Run()

	err = neco.StartService(ctx, "neco-worker")
	if err != nil {
		return err
	}
	err = neco.StartService(ctx, "neco-updater")
	if err != nil {
		return err
	}

	err = setupBootIP(ctx, mylrn, lrns)
	if err != nil {
		return err
	}
	err = neco.RestartService(ctx, "systemd-networkd")
	if err != nil {
		return err
	}
	// TODO: Wait for the boot netdev to be ready?

	log.Info("setup: completed", nil)

	return nil
}

// Join prepares certificates and files for new boot server, start
// neco-updater and neco-worker, then register the server with etcd.
func Join(ctx context.Context, vc *api.Client, mylrn int, lrns []int) error {
	err := reissueCerts(ctx, vc, mylrn)
	if err != nil {
		return err
	}

	err = setupEtcdBackup(ctx, vc)
	if err != nil {
		return err
	}

	err = setupNecoFiles(ctx, vc, lrns)
	if err != nil {
		return err
	}

	// etcd client can be created only after setupNecoFiles
	etcd, err := neco.EtcdClient()
	if err != nil {
		return err
	}
	defer etcd.Close()
	st := storage.NewStorage(etcd)

	err = neco.StartService(ctx, "neco-worker")
	if err != nil {
		return err
	}
	err = neco.StartService(ctx, "neco-updater")
	if err != nil {
		return err
	}

	err = setupBootIP(ctx, mylrn, lrns)
	if err != nil {
		return err
	}
	err = neco.RestartService(ctx, "systemd-networkd")
	if err != nil {
		return err
	}
	// TODO: Wait for the boot netdev to be ready?

	return st.RegisterBootserver(ctx, mylrn)
}
