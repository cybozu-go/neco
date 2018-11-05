package cmd

import (
	"context"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/neco"
	"github.com/cybozu-go/neco/storage"
	"github.com/spf13/cobra"
)

// vaultRemoveUnsealKeyCmd implements "vault remove-unseal-key".
var vaultRemoveUnsealKeyCmd = &cobra.Command{
	Use:   "remove-unseal-key",
	Short: "Remove the initial unseal key",
	Long:  `Remove the initial unseal key from etcd.`,

	Run: func(cmd *cobra.Command, args []string) {
		etcd, err := neco.EtcdClient()
		if err != nil {
			log.ErrorExit(err)
		}
		defer etcd.Close()

		st := storage.NewStorage(etcd)
		err = st.DeleteVaultUnsealKey(context.Background())
		if err != nil {
			log.ErrorExit(err)
		}
	},
}

func init() {
	vaultCmd.AddCommand(vaultRemoveUnsealKeyCmd)
}
