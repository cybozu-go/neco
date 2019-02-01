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

var bmcConfigGetBMCUserCmd = &cobra.Command{
	Use:   "bmc-user",
	Short: "show the current bmc-user.json contents",
	Long:  `Show the current bmc-user.json contents.`,
	Args:  cobra.ExactArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		etcd, err := neco.EtcdClient()
		if err != nil {
			log.ErrorExit(err)
		}
		defer etcd.Close()
		st := storage.NewStorage(etcd)
		well.Go(func(ctx context.Context) error {
			data, err := st.GetBMCBMCUser(ctx)
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
	bmcConfigGetCmd.AddCommand(bmcConfigGetBMCUserCmd)
}
