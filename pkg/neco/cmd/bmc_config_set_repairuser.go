package cmd

import (
	"context"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/neco"
	"github.com/cybozu-go/neco/storage"
	"github.com/cybozu-go/well"
	"github.com/spf13/cobra"
)

var bmcConfigSetRepairUserCmd = &cobra.Command{
	Use:   "repair-user USERNAME",
	Short: "store BMC username for repair operations.",
	Long:  `Store BMC username for repair operations.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		etcd, err := neco.EtcdClient()
		if err != nil {
			log.ErrorExit(err)
		}
		defer etcd.Close()
		st := storage.NewStorage(etcd)
		well.Go(func(ctx context.Context) error {
			return st.PutBMCRepairUser(ctx, args[0])
		})
		well.Stop()
		err = well.Wait()
		if err != nil {
			log.ErrorExit(err)
		}
	},
}

func init() {
	bmcConfigSetCmd.AddCommand(bmcConfigSetRepairUserCmd)
}
