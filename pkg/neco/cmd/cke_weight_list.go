package cmd

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/neco"
	"github.com/cybozu-go/neco/storage"
	"github.com/cybozu-go/well"
	"github.com/spf13/cobra"
)

var ckeWeightListCmd = &cobra.Command{
	Use:   "list",
	Short: "list the current weights",
	Long:  `list the current weights.`,
	Args:  cobra.ExactArgs(0),
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

			output, err := json.Marshal(data)
			if err != nil {
				return err
			}

			fmt.Println(string(output))
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
	ckeWeightCmd.AddCommand(ckeWeightListCmd)
}
