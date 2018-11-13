package setup

import (
	"context"
	"sort"
	"time"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/neco"
	"github.com/cybozu-go/neco/progs/vault"
	"github.com/cybozu-go/neco/storage"
	"github.com/cybozu-go/well"
	"github.com/hashicorp/vault/api"
)

// Setup installs and configures etcd and vault cluster.
func Setup(ctx context.Context, lrns []int, revoke bool) error {
	err := neco.FetchContainer(ctx, "etcd")
	if err != nil {
		return err
	}

	err = neco.FetchContainer(ctx, "vault")
	if err != nil {
		return err
	}

	sort.Ints(lrns)

	mylrn, err := neco.MyLRN()
	if err != nil {
		return err
	}

	isLeader := mylrn == lrns[0]

	err = installNecoBin()
	if err != nil {
		return err
	}

	pems, err := prepareCA(ctx, isLeader, mylrn, lrns)
	if err != nil {
		return err
	}

	ec, err := setupEtcd(ctx, mylrn, lrns)
	if err != nil {
		return err
	}
	defer ec.Close()

	var vc *api.Client

	if isLeader {
		err = setupVault(ctx, mylrn, lrns)
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
		err = setupVault(ctx, mylrn, lrns)
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

	err = st.Finish(ctx, mylrn)
	if err != nil {
		return err
	}

	for {
		log.Info("waiting for all servers to finish", nil)
		finished, err := st.GetFinished(ctx)
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
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(1 * time.Second):
		}

		ec, err = waitEtcd(ctx)
		if err == nil {
			break
		}
	}

	err = neco.RestartService(ctx, "vault")
	if err != nil {
		return err
	}
	time.Sleep(1 * time.Second)

	if isLeader {
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
	}

	well.CommandContext(ctx, "sync").Run()

	log.Info("setup: completed", nil)

	return nil
}

// PrepareFiles prepares certificates and files for new boot server
func PrepareFiles(ctx context.Context, vc *api.Client, mylrn int, lrns []int) error {
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

	return st.RegisterBootserver(ctx, mylrn)
}
