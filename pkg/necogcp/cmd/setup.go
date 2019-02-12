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

// setupCmd is setup command for instances.
var setupCmd = &cobra.Command{
	Use:   "setup",
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
				return gcp.SetupVMXEnabled(ctx, cfg.Common.Project, cfg.Compute.VMXEnabled.OptionalPackages)
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
	rootCmd.AddCommand(setupCmd)
}
