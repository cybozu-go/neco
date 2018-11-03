package cmd

import (
	"context"
	"fmt"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/neco"
	"github.com/cybozu-go/neco/storage"
	"github.com/spf13/cobra"
)

// vaultShowRootTokenCmd implements "vault show-root-token".
var vaultShowRootTokenCmd = &cobra.Command{
	Use:   "show-root-token",
	Short: "Show the initial root token",
	Long: `Show the initial root token if not revoked.

Normally, the root token was revoked during "neco setup".

It is not revoked only for testing purpose.
Therefore this command is only for testing too.`,

	Run: func(cmd *cobra.Command, args []string) {
		etcd, err := neco.EtcdClient()
		if err != nil {
			log.ErrorExit(err)
		}
		defer etcd.Close()

		st := storage.NewStorage(etcd)
		key, err := st.GetVaultRootToken(context.Background())
		if err != nil {
			log.ErrorExit(err)
		}

		fmt.Println(key)
	},
}

func init() {
	vaultCmd.AddCommand(vaultShowRootTokenCmd)
}
