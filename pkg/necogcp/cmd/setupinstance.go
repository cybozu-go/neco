package cmd

import (
	"context"
	"errors"
	"os"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/neco/gcp"
	"github.com/cybozu-go/well"
	"github.com/spf13/cobra"
)

// setupInstanceCmd is setup command for instances.
var setupInstanceCmd = &cobra.Command{
	Use:   "setup-instance",
	Short: "setup instance",
	Long: `setup instance.

Please run this command on vmx-enabled or host-vm instance.`,
	Args: cobra.ExactArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		well.Go(func(ctx context.Context) error {
			hostname, err := os.Hostname()
			if err != nil {
				return err
			}
			switch hostname {
			case "vmx-enabled":
				options := cfg.Compute.OptionalPackages
				if len(cfg.Compute.VMXEnabled.OptionalPackages) > 0 {
					options = append(options, cfg.Compute.VMXEnabled.OptionalPackages...)
				}
				return gcp.SetupVMXEnabled(ctx, cfg.Common.Project, options)
			case "host-vm":
				return gcp.SetupHostVM(ctx)
			}
			return errors.New("this host is not supported")
		})
		well.Stop()
		err := well.Wait()
		if err != nil {
			log.ErrorExit(err)
		}
	},
}

func init() {
	rootCmd.AddCommand(setupInstanceCmd)
}
