package cmd

import (
	"context"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/neco/gcp"
	"github.com/cybozu-go/well"
	"github.com/spf13/cobra"
)

var necotestExtendCmd = &cobra.Command{
	Use:   "extend INSTANCE_NAME",
	Short: "Extend 1 hour given instance on neco-test to prevent auto deletion",
	Long:  `Extend 1 hour given instance on neco-test to prevent auto deletion.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		necotestCfg := gcp.NecoTestConfig()
		necotestCfg.Common.ServiceAccount = cfg.Common.ServiceAccount
		cc := gcp.NewComputeClient(necotestCfg, args[0])
		well.Go(func(ctx context.Context) error {
			err := cc.ExtendInstance(ctx)
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
	necotestCmd.AddCommand(necotestExtendCmd)
}
