package cmd

import (
	"context"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/neco/gcp"
	"github.com/cybozu-go/well"
	"github.com/spf13/cobra"
)

var deleteHostVMCommand = &cobra.Command{
	Use:   "host-vm",
	Short: "Delete host-vm instance",
	Long:  `Delete host-vm instance manually.`,
	Args:  cobra.ExactArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		cc := gcp.NewComputeClient(cfg, "host-vm")
		well.Go(func(ctx context.Context) error {
			err := cc.DeleteInstance(ctx)
			if err != nil {
				return err
			}

			return nil
		})
		well.Stop()
		err := well.Wait()
		if err != nil {
			log.ErrorExit(err)
		}
	},
}

func init() {
	deleteCmd.AddCommand(deleteHostVMCommand)
}
