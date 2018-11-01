package setup

import (
	"context"
	"sort"

	"github.com/cybozu-go/neco"
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

	err = setupEtcd(ctx, mylrn, lrns)
	if err != nil {
		return err
	}

	if isLeader {
		err = setupVault(ctx, mylrn, lrns)
		if err != nil {
			return err
		}
		err = bootVault(ctx, pems)
		if err != nil {
			return err
		}
	} else {
		// TODO
	}

	pems = pems
	return nil
}
