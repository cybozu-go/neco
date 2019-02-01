package cmd

import (
	"context"
	"io/ioutil"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/neco"
	"github.com/cybozu-go/neco/storage"
	"github.com/cybozu-go/well"
	"github.com/spf13/cobra"
)

var bmcConfigSetBMCUserCmd = &cobra.Command{
	Use:   "bmc-user FILE",
	Short: "store bmc-user.json contents",
	Long:  `Store bmc-user.json contents.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		etcd, err := neco.EtcdClient()
		if err != nil {
			log.ErrorExit(err)
		}
		defer etcd.Close()
		st := storage.NewStorage(etcd)
		data, err := ioutil.ReadFile(args[0])
		if err != nil {
			log.ErrorExit(err)
		}

		well.Go(func(ctx context.Context) error {
			return st.PutBMCBMCUser(ctx, string(data))
		})
		well.Stop()
		err = well.Wait()
		if err != nil {
			log.ErrorExit(err)
		}
	},
}

func init() {
	bmcConfigSetCmd.AddCommand(bmcConfigSetBMCUserCmd)
}
