package cmd

import (
	"context"
	"fmt"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/neco"
	"github.com/cybozu-go/neco/storage"
	"github.com/cybozu-go/well"
	"github.com/spf13/cobra"
)

var bmcConfigGetIPMIUserCmd = &cobra.Command{
	Use:   "ipmi-user",
	Short: "show the current IPMI username",
	Long:  `show the current IPMI username.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		etcd, err := neco.EtcdClient()
		if err != nil {
			log.ErrorExit(err)
		}
		defer etcd.Close()
		st := storage.NewStorage(etcd)
		well.Go(func(ctx context.Context) error {
			data, err := st.GetBMCIPMIUser(ctx)
			if err != nil {
				return err
			}
			fmt.Println(data)
			return nil
		})
		well.Stop()
		err = well.Wait()
		if err != nil {
			log.ErrorExit(err)
		}
	},
}

func init() {
	bmcConfigGetCmd.AddCommand(bmcConfigGetIPMIUserCmd)
}
