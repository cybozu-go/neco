package cmd

import (
	"context"
	"fmt"

	"github.com/cybozu-go/neco/gcp"
	"github.com/cybozu-go/well"
	"github.com/spf13/cobra"
)

var (
	destZone string
)

var restoreSnapshotCmd = &cobra.Command{
	Use:   "restore-snapshot",
	Short: "Restore home volume snapshot",
	Long:  `Restore home volume snapshot manually.`,
	Args:  cobra.ExactArgs(0),
	RunE: func(cmd *cobra.Command, args []string) error {
		if destZone == "" {
			return fmt.Errorf("setting --dest-zone flag is required")
		}
		cc := gcp.NewComputeClient(cfg, "host-vm")
		well.Go(func(ctx context.Context) error {
			err := cc.RestoreVolumeFromSnapshot(ctx, destZone)
			if err != nil {
				return err
			}
			return nil
		})
		well.Stop()
		err := well.Wait()
		return err
	},
}

func init() {
	restoreSnapshotCmd.Flags().StringVar(&destZone, "dest-zone", "", "zone name for creating a new volume")
	rootCmd.AddCommand(restoreSnapshotCmd)
}
