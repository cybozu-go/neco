package cmd

import (
	"context"
	"errors"
	"fmt"

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
	necoBranch         string
	necoAppsBranch     string
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
		builder := functions.NewNecoStartupScriptBuilder().WithFluentd()
		if len(necoBranch) > 0 {
			log.Info("run neco", map[string]interface{}{
				"branch": necoBranch,
			})
			builder.WithNeco(necoBranch)
		}
		if len(necoAppsBranch) > 0 {
			log.Info("run neco-apps", map[string]interface{}{
				"branch": necoAppsBranch,
			})
			_, err := builder.WithNecoApps(necoBranch)
			if err != nil {
				log.ErrorExit(fmt.Errorf("failed to create startup script: %v", err))
			}
		}

		well.Go(func(ctx context.Context) error {
			cc, err := gcp.NewComputeClient(ctx, projectID, zone)
			if err != nil {
				log.Error("failed to create compute client: %v", err)
				return err
			}
			log.Info("start creating instance", map[string]interface{}{
				"project":        projectID,
				"zone":           zone,
				"name":           instanceName,
				"serviceaccount": serviceAccountName,
				"machinetype":    machineType,
				"necobranch":     necoBranch,
				"necoappsbranch": necoAppsBranch,
			})
			return cc.Create(
				instanceName,
				serviceAccountName,
				machineType,
				functions.MakeVMXEnabledImageURL(projectID),
				builder.Build(),
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
	necotestCreateInstanceCmd.Flags().StringVar(&necoBranch, "neco-branch", "release", "Branch of neco to run")
	necotestCreateInstanceCmd.Flags().StringVar(&necoAppsBranch, "neco-apps-branch", "", "Branch of neco-apps to run")
	necotestCmd.AddCommand(necotestCreateInstanceCmd)
}
