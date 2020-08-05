package cmd

import (
	"context"
	"errors"

	"github.com/cybozu-go/log"
	functions "github.com/cybozu-go/neco/pkg/necogcp-functions"
	"github.com/cybozu-go/neco/pkg/necogcp-functions/gcp"
	"github.com/cybozu-go/well"
	"github.com/spf13/cobra"
)

var (
	projectID          string
	zone               string
	serviceAccountName string
	machineType        string
	instanceName       string
)

var necotestCreateInstanceCmd = &cobra.Command{
	Use:   "create-instance",
	Short: "Create dctest env for neco (and neco-apps)",
	Long:  `Create dctest env for neco (and neco-apps).`,
	Args:  cobra.ExactArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		if len(instanceName) == 0 {
			log.ErrorExit(errors.New("instance name is required"))
		}

		well.Go(func(ctx context.Context) error {
			cc, err := gcp.NewComputeClient(ctx, projectID, zone)
			if err != nil {
				return err
			}
			log.Info("now creating instance", map[string]interface{}{
				"project":        projectID,
				"zone":           zone,
				"name":           instanceName,
				"serviceaccount": serviceAccountName,
				"machinetype":    machineType,
			})
			return cc.Create(
				instanceName,
				serviceAccountName,
				machineType,
				functions.MakeVMXEnabledImageURL(projectID),
				functions.MakeStartupScript("", "", ""),
			)
		})

		well.Stop()
		err := well.Wait()
		if err != nil {
			log.ErrorExit(err)
		}
	},
}

func init() {
	necotestCreateInstanceCmd.Flags().StringVarP(&projectID, "project-id", "p", "neco-test", "Project ID for GCP")
	necotestCreateInstanceCmd.Flags().StringVarP(&zone, "zone", "z", "asia-northeast1-c", "Zone name for GCP")
	necotestCreateInstanceCmd.Flags().StringVarP(&serviceAccountName, "service-account", "a",
		"815807730957-compute@developer.gserviceaccount.com", "Service account to obtain account.json in Secret Manager")
	necotestCreateInstanceCmd.Flags().StringVarP(&machineType, "machine-type", "t", "n1-standard-32", "Machine type")
	necotestCreateInstanceCmd.Flags().StringVarP(&instanceName, "instance-name", "n", "", "Instance name")
	necotestCmd.AddCommand(necotestCreateInstanceCmd)
}
