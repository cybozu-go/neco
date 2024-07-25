package cmd

import (
	"context"
	"time"

	"github.com/spf13/cobra"
)

var isoRebootCmd = &cobra.Command{
	Use:   "isoreboot ISO_FILE",
	Short: "initiate reboot from a ISO image",
	Long: `Connect iso file to virtual DVD and schedule reboot of workers.

This uses CKE's function of graceful reboot for the nodes used by CKE.
As for the other nodes, this reboots them immediately.
If some nodes are already powered off, this command does not do anything to those nodes.`,

	Args: cobra.ExactArgs(1),
	Run:  isoRebootRun,
}

var isoRebootGetOpts sabakanMachinesGetOpts
var isoRebootTimeoutOption time.Duration

func isoRebootRun(cmd *cobra.Command, args []string) {
	ctx := context.Background()

	uploadAssetsAndRunCommandOnWorkers(ctx, &isoRebootGetOpts, args, []string{"docker", "exec", "setup-hw", "setup-isoreboot"}, isoRebootTimeoutOption, true)
}

func init() {
	rootCmd.AddCommand(isoRebootCmd)
	addSabakanMachinesGetOpts(isoRebootCmd, &isoRebootGetOpts)
	isoRebootCmd.Flags().DurationVar(&isoRebootTimeoutOption, "timeout", 30*time.Second, "timeout")
}
