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

var bmcConfigGetIPMIPasswordCmd = &cobra.Command{
	Use:   "ipmi-password",
	Short: "show the current IPMI password",
	Long:  `Show the current IPMI password.`,
	Args:  cobra.ExactArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		etcd, err := neco.EtcdClient()
		if err != nil {
			log.ErrorExit(err)
		}
		defer etcd.Close()
		st := storage.NewStorage(etcd)
		well.Go(func(ctx context.Context) error {
			data, err := st.GetBMCIPMIPassword(ctx)
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
	bmcConfigGetCmd.AddCommand(bmcConfigGetIPMIPasswordCmd)
}
