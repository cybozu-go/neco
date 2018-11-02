package setup

import (
	"context"
	"sort"

	"github.com/cybozu-go/neco"
	"github.com/cybozu-go/neco/storage"
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
		err = unsealVault(vc, unsealKey)
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

	err = reissueCerts(ctx, vc, mylrn, rootToken)
	if err != nil {
		return err
	}

	return nil
}
