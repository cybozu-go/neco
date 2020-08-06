package cmd

import (
	"context"
	"errors"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/neco/gcp/functions"
	"github.com/cybozu-go/well"
	"github.com/spf13/cobra"
)

var necotestDeleteInstanceCmd = &cobra.Command{
	Use:   "delete-instance",
	Short: "Delete instance",
	Long:  `Delete instance.`,
	Args:  cobra.ExactArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		if len(instanceName) == 0 {
			log.ErrorExit(errors.New("instance name is required"))
		}
		well.Go(func(ctx context.Context) error {
			cc, err := functions.NewComputeClient(ctx, projectID, zone)
			if err != nil {
				log.Error("failed to create compute client", map[string]interface{}{
					log.FnError: err,
				})
				return err
			}
			log.Info("start deleting instance", map[string]interface{}{
				"project": projectID,
				"zone":    zone,
				"name":    instanceName,
			})
			return cc.Delete(instanceName)
		})

		well.Stop()
		err := well.Wait()
		if err != nil {
			log.ErrorExit(err)
		}
	},
}

func init() {
	necotestDeleteInstanceCmd.Flags().StringVarP(&projectID, "project-id", "p", "neco-test", "Project ID for GCP")
	necotestDeleteInstanceCmd.Flags().StringVarP(&zone, "zone", "z", "asia-northeast1-c", "Zone name for GCP")
	necotestDeleteInstanceCmd.Flags().StringVarP(&instanceName, "instance-name", "n", "", "Instance name")
	necotestCmd.AddCommand(necotestDeleteInstanceCmd)
}
