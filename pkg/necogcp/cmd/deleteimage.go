package cmd

import (
	"context"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/neco/gcp"
	"github.com/cybozu-go/well"
	"github.com/spf13/cobra"
)

var deleteImageCmd = &cobra.Command{
	Use:   "delete-image",
	Short: "Delete vmx-enabled image",
	Long:  `Delete vmx-enabled image.`,
	Args:  cobra.ExactArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		cc := gcp.NewComputeClient(cfg, "vmx-enabled")
		well.Go(func(ctx context.Context) error {
			err := cc.DeleteVMXEnabledImage(ctx)
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
	rootCmd.AddCommand(deleteImageCmd)
}
