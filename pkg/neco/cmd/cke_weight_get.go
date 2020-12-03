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

var ckeWeightGetCmd = &cobra.Command{
	Use:   "get ROLE",
	Short: "show the current weight of given role",
	Long:  `show the current weight of given role.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		etcd, err := neco.EtcdClient()
		if err != nil {
			log.ErrorExit(err)
		}
		defer etcd.Close()
		st := storage.NewStorage(etcd)
		well.Go(func(ctx context.Context) error {
			data, err := st.GetCKEWeight(ctx)
			if err != nil {
				return err
			}

			val, ok := data[args[0]]
			if !ok {
				return fmt.Errorf("no such weight found for role %s", args[0])
			}

			fmt.Println(val)
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
	ckeWeightCmd.AddCommand(ckeWeightGetCmd)
}
