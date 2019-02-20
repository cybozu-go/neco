package cmd

import (
	"context"
	"io/ioutil"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/neco"
	"github.com/cybozu-go/neco/ext"
	"github.com/cybozu-go/neco/storage"
	sabakan "github.com/cybozu-go/sabakan/v2/client"
	"github.com/cybozu-go/well"
	"github.com/spf13/cobra"
)

var bmcConfigSetBMCUserCmd = &cobra.Command{
	Use:   "bmc-user FILE",
	Short: "store bmc-user.json contents",
	Long:  `Store bmc-user.json contents, and upload it to sabakan`,
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

		localClient := ext.LocalHTTPClient()

		well.Go(func(ctx context.Context) error {
			err := st.PutBMCBMCUser(ctx, string(data))
			if err != nil {
				return err
			}

			saba, err := sabakan.NewClient(neco.SabakanLocalEndpoint, localClient)
			if err != nil {
				return err
			}
			_, err = saba.AssetsUpload(ctx, "bmc-user.json", args[0], nil)
			return err
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
