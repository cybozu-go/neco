package cmd

import (
	"context"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/neco/gcp"
	"github.com/cybozu-go/well"
	"github.com/spf13/cobra"
)

var createSnapshotCmd = &cobra.Command{
	Use:   "create-snapshot",
	Short: "Create home volume snapshot",
	Long:  `Create home volume snapshot manually.`,
	Args:  cobra.ExactArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		cc := gcp.NewComputeClient(cfg, "host-vm")
		well.Go(func(ctx context.Context) error {
			err := cc.CreateVolumeSnapshot(ctx)
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
	rootCmd.AddCommand(createSnapshotCmd)
}
