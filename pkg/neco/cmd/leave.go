package cmd

import (
	"context"
	"errors"
	"strconv"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/neco"
	"github.com/cybozu-go/neco/storage"
	"github.com/cybozu-go/well"
	"github.com/spf13/cobra"
)

var leaveParams struct {
	lrn int
}

// leaveCmd represents the leave command
var leaveCmd = &cobra.Command{
	Use:   "leave LRN",
	Short: "Unregister LRN of the boot server from etcd.",
	Long:  `Unregister LRN of the boot server from etcd.`,

	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			return errors.New("expected exact one argument")
		}
		num, err := strconv.ParseUint(args[0], 10, 32)
		if err != nil {
			return err
		}
		leaveParams.lrn = int(num)
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		etcd, err := neco.EtcdClient()
		if err != nil {
			log.ErrorExit(err)
		}
		defer etcd.Close()
		st := storage.NewStorage(etcd)

		well.Go(func(ctx context.Context) error {
			return st.DeleteBootServer(ctx, leaveParams.lrn)
		})

		well.Stop()
		err = well.Wait()
		if err != nil {
			log.ErrorExit(err)
		}
	},
}

func init() {
	rootCmd.AddCommand(leaveCmd)
}
