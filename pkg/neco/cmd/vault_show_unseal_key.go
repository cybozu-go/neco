package cmd

import (
	"context"
	"fmt"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/neco"
	"github.com/cybozu-go/neco/storage"
	"github.com/spf13/cobra"
)

// vaultShowUnsealKeyCmd implements "vault show-unseal-key".
var vaultShowUnsealKeyCmd = &cobra.Command{
	Use:   "show-unseal-key",
	Short: "Show the initial unseal key",
	Long: `Show the initial unseal key if not removed.

The initial unseal key is stored in etcd.`,

	Run: func(cmd *cobra.Command, args []string) {
		etcd, err := neco.EtcdClient()
		if err != nil {
			log.ErrorExit(err)
		}
		defer etcd.Close()

		st := storage.NewStorage(etcd)
		key, err := st.GetVaultUnsealKey(context.Background())
		if err != nil {
			log.ErrorExit(err)
		}

		fmt.Println(key)
	},
}

func init() {
	vaultCmd.AddCommand(vaultShowUnsealKeyCmd)
}
