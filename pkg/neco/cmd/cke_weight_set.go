package cmd

import (
	"context"
	"strconv"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/neco"
	"github.com/cybozu-go/neco/storage"
	"github.com/cybozu-go/well"
	"github.com/spf13/cobra"
)

var ckeWeightSetCmd = &cobra.Command{
	Use:   "set ROLE WEIGHT",
	Short: "set a weight of given role",
	Long:  `set a weight weight of given role.`,
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		etcd, err := neco.EtcdClient()
		if err != nil {
			log.ErrorExit(err)
		}
		defer etcd.Close()
		st := storage.NewStorage(etcd)
		well.Go(func(ctx context.Context) error {
			data, err := st.GetCKEWeight(ctx)
			if err != nil && err != storage.ErrNotFound {
				return err
			}

			weight, err := strconv.ParseFloat(args[1], 64)
			if err != nil {
				return err
			}
			data[args[0]] = weight

			return st.PutCKEWeight(ctx, data)
		})
		well.Stop()
		err = well.Wait()
		if err != nil {
			log.ErrorExit(err)
		}
	},
}

func init() {
	ckeWeightCmd.AddCommand(ckeWeightSetCmd)
}
