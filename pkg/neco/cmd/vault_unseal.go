package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/neco"
	"github.com/cybozu-go/neco/progs/vault"
	"github.com/cybozu-go/neco/storage"
	"github.com/cybozu-go/well"
	"github.com/hashicorp/vault/api"
	"github.com/spf13/cobra"
)

func waitVault(ctx context.Context, vc *api.Client) error {
	for {
		_, err := vc.Sys().Health()
		if err == nil {
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(1 * time.Second):
		}
	}
}

func subMain(ctx context.Context) error {
	etcd, err := neco.EtcdClient()
	if err != nil {
		return err
	}
	defer etcd.Close()

	st := storage.NewStorage(etcd)
	key, err := st.GetVaultUnsealKey(ctx)
	if err == storage.ErrNotFound {
		fmt.Fprintln(os.Stderr, "no unseal key")
		return nil
	}
	if err != nil {
		return err
	}

	vc, err := api.NewClient(api.DefaultConfig())
	if err != nil {
		return err
	}

	// wait for vault container runs the server
	tctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	err = waitVault(tctx, vc)
	if err != nil {
		return err
	}

	return vault.Unseal(vc, key)
}

// vaultUnsealCmd implements "vault unseal".
var vaultUnsealCmd = &cobra.Command{
	Use:   "unseal",
	Short: "Unseal vault using the initial unseal key",
	Long: `Unseal local vault server using the initial unseal key.

The initial unseal key is stored in etcd.
If the initial unseal key was removed by remove-unseal-key,
this does nothing and exits with status 0.`,

	Run: func(cmd *cobra.Command, args []string) {
		_, err := os.Stat(neco.NecoConfFile)
		if err != nil && os.IsNotExist(err) {
			// setup is not completed
			return
		}

		well.Go(subMain)
		well.Stop()
		err = well.Wait()
		if err != nil {
			log.ErrorExit(err)
		}
	},
}

func init() {
	vaultCmd.AddCommand(vaultUnsealCmd)
}
